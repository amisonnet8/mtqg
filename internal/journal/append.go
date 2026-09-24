package journal

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"
)

// Append writes one event to the end of journal.jsonl and returns it as written.
//
// The caller gives only the meaning of the event: op, type, re, from, status,
// word, text, at, and the id of the record for every op except create. The
// journal layer fills in the rest and overwrites anything the caller put there:
// the id of a new record, v, ts, author and tty.
//
// The steps, all under the write lock:
//
//  1. Refuse if journal.jsonl still holds conflict markers.
//  2. If the file does not end with a line feed (a line was cut off by a crash,
//     or the file was edited by hand), start with one, so that the new line does
//     not get glued to the broken one.
//  3. Write the line with a single write. There is no fsync: the speed of
//     writing a note down comes first.
//
// Nothing is written when an error is returned, except that a failed write may
// leave a cut-off line, which step 2 then keeps apart from the next one.
func (j *Journal) Append(ev Event) (Event, error) {
	if err := j.validate(ev); err != nil {
		return Event{}, err
	}

	release, err := j.lock()
	if err != nil {
		return Event{}, err
	}
	defer release()

	// The time is taken once the lock is held, so that the order of the lines
	// and the order of the times agree as far as they can.
	if ev.Op == OpCreate {
		id, err := newID()
		if err != nil {
			return Event{}, err
		}
		ev.ID = id
	}
	ev.V = j.version
	ev.TS = j.now().UTC().Format(time.RFC3339)
	ev.Author = j.opts.Author
	ev.TTY = j.opts.TTY

	line, err := encodeLine(ev)
	if err != nil {
		return Event{}, err
	}

	root, err := j.openRoot()
	if err != nil {
		return Event{}, err
	}
	defer func() { _ = root.Close() }()

	existing, err := root.ReadFile(journalName)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Event{}, fmt.Errorf("journal: %w", err)
	}
	if hasConflictMarkers(existing) {
		return Event{}, ErrConflictMarkers
	}
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		line = append([]byte{'\n'}, line...)
	}

	f, err := root.OpenFile(journalName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return Event{}, fmt.Errorf("journal: %w", err)
	}
	if _, err := f.Write(line); err != nil {
		_ = f.Close()
		return Event{}, fmt.Errorf("journal: write %s: %w", journalName, err)
	}
	if err := f.Close(); err != nil {
		return Event{}, fmt.Errorf("journal: write %s: %w", journalName, err)
	}
	return ev, nil
}

// validate checks what the caller controls, before the lock is taken. It checks
// that the event can be a line of the format, not what it means: whether a memo
// can be done is for the layer above.
func (j *Journal) validate(ev Event) error {
	bad := func(field, reason string) error { return &InvalidEventError{Field: field, Reason: reason} }

	switch ev.Op {
	case OpCreate:
		if ev.ID != "" {
			return bad("id", "the id of a new record is chosen by the journal layer")
		}
		if !oneOf(ev.Type, TypeMemo, TypeTodo, TypeQA, TypeBug, TypeGlossary) {
			return bad("type", "must be memo, todo, qa, bug or glossary")
		}
		if ev.Re != "" && !isID(ev.Re) {
			return bad("re", "must be a full ID: 32 lowercase hex digits")
		}
	case OpStatus, OpEdit, OpDelete:
		if !isID(ev.ID) {
			return bad("id", "must be a full ID: 32 lowercase hex digits")
		}
		if ev.Re != "" {
			return bad("re", "only a created answer or reply has re")
		}
	default:
		return bad("op", "must be create, status, edit or delete")
	}

	if ev.Status != "" && !oneOf(ev.Status, StatusOpen, StatusDone) {
		return bad("status", "must be open or done")
	}
	if ev.From != "" && !oneOf(ev.From, StatusOpen, StatusDone) {
		return bad("from", "must be open or done")
	}
	if ev.Basis < 0 {
		return bad("basis", "must not be negative")
	}
	if !oneOf(j.opts.Author.Kind, AuthorHuman, AuthorAI) {
		return bad("author.kind", "must be human or ai")
	}
	if j.opts.Author.Name == "" {
		return bad("author.name", "must not be empty")
	}
	return nil
}

func oneOf(s string, options ...string) bool {
	for _, o := range options {
		if s == o {
			return true
		}
	}
	return false
}

// conflictMarkerStarts are what every conflict marker line begins with.
var conflictMarkerStarts = [][]byte{
	[]byte("<<<<<<<"), []byte(">>>>>>>"), []byte("======="), []byte("|||||||"),
}

// hasConflictMarkers reports whether data holds a merge conflict marker line.
// Most journals hold none of the marker strings anywhere, and that is settled
// with a plain search; only when one turns up (it may be inside the text of a
// note) are the lines looked at one by one.
func hasConflictMarkers(data []byte) bool {
	found := false
	for _, start := range conflictMarkerStarts {
		if bytes.Contains(data, start) {
			found = true
			break
		}
	}
	if !found {
		return false
	}
	for len(data) > 0 {
		line := data
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			line, data = data[:i], data[i+1:]
		} else {
			data = nil
		}
		if isConflictMarker(bytes.TrimSuffix(line, []byte("\r"))) {
			return true
		}
	}
	return false
}
