package journal

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Where a rewrite builds the new file. It has to be on the same file system as
// journal.jsonl so that the last step is one replacement. .local/ is never
// committed.
const tmpName = "tmp"

// Rewrite replaces journal.jsonl with the lines fn returns. It is what undo is
// made of; undo and archive (Archive) are the only operations that remove or move
// lines.
//
// Under the write lock, fn is given every non-blank line of journal.jsonl, and
// returns the lines to keep, in the order to write them. A line fn gives back
// as it got it is written byte for byte, so unknown fields, lines that could
// not be read and hand-made formatting all survive; only what fn leaves out is
// gone. A line with a nil Raw is written from its Event by the writing rules.
// Lines with the same content are not merged, as they are when reading. Blank
// lines are dropped and CRLF becomes LF.
//
// The new file is written to .mtqg/.local/tmp/, synced, and then put in place
// of journal.jsonl, so an append that comes in the meantime waits for the lock
// instead of being lost, and a crash leaves either the old file or the new one.
//
// If fn returns an error, nothing is written. Rewriting is refused while
// journal.jsonl holds conflict markers. fn must not call Append or Rewrite: the
// lock is not re-entrant, so it would wait for itself until the timeout.
func (j *Journal) Rewrite(fn func(lines []Line) ([]Line, error)) error {
	release, err := j.lock()
	if err != nil {
		return err
	}
	defer release()

	root, err := j.openRoot()
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	lines, exists, err := readLocked(root)
	if err != nil {
		return err
	}
	kept, err := fn(lines)
	if err != nil {
		return err
	}
	out, err := renderLines(kept)
	if err != nil {
		return err
	}
	return replaceJournal(root, out, exists)
}

// readLocked reads the lines of journal.jsonl for a rewrite, which holds the
// lock. exists says whether the file is there: a missing file is an empty journal.
// Conflict markers are refused, as every write refuses them.
func readLocked(root *os.Root) (lines []Line, exists bool, err error) {
	data, err := root.ReadFile(journalName)
	exists = err == nil
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, false, fmt.Errorf("journal: %w", err)
	}
	if hasConflictMarkers(data) {
		return nil, false, ErrConflictMarkers
	}
	// A bytes.Reader cannot fail, so Scan has no error to report here.
	lines, _, _ = Scan(bytes.NewReader(data))
	return lines, exists, nil
}

// replaceJournal writes out to .local/tmp/, syncs it, and puts it in place of
// journal.jsonl, keeping the permissions of the file it replaces. Nothing is
// written when there was no journal.jsonl and there is nothing to put in it.
func replaceJournal(root *os.Root, out []byte, exists bool) error {
	if !exists && len(out) == 0 {
		return nil
	}

	perm := fs.FileMode(0o600)
	if exists {
		info, err := root.Stat(journalName)
		if err != nil {
			return fmt.Errorf("journal: %w", err)
		}
		perm = info.Mode().Perm()
	}
	tmp, err := writeTemp(root, out, perm)
	if err != nil {
		return err
	}
	if err := replaceFile(root, tmp, journalName); err != nil {
		_ = root.Remove(tmp)
		return fmt.Errorf("journal: replace %s: %w", journalName, err)
	}
	return nil
}

// renderLines returns the bytes of the file for lines: each one ends with LF.
func renderLines(lines []Line) ([]byte, error) {
	var out bytes.Buffer
	for _, line := range lines {
		switch {
		case line.Raw != nil:
			if bytes.ContainsAny(line.Raw, "\r\n") {
				return nil, &InvalidEventError{Field: "line", Reason: "a line must not contain a line ending"}
			}
			out.Write(line.Raw)
			out.WriteByte('\n')
		case line.Event != nil:
			encoded, err := encodeLine(*line.Event)
			if err != nil {
				return nil, err
			}
			out.Write(encoded)
		default:
			return nil, &InvalidEventError{Field: "line", Reason: "has neither raw bytes nor an event"}
		}
	}
	return out.Bytes(), nil
}

// writeTemp writes data to a fresh file in .local/tmp/ and returns its name
// relative to the root. Whatever an earlier rewrite left in .local/tmp/ is
// removed first: it holds only unfinished rewrites, and only one rewrite runs at
// a time, under the lock.
func writeTemp(root *os.Root, data []byte, perm fs.FileMode) (string, error) {
	dir := filepath.Join(localName, tmpName)
	if err := root.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("journal: %w", err)
	}
	if err := root.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("journal: %w", err)
	}
	name := filepath.Join(dir, journalName+".tmp")
	f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("journal: %w", err)
	}
	fail := func(err error) (string, error) {
		_ = f.Close()
		_ = root.Remove(name)
		return "", fmt.Errorf("journal: write %s: %w", name, err)
	}
	if _, err := f.Write(data); err != nil {
		return fail(err)
	}
	if err := f.Chmod(perm); err != nil {
		return fail(err)
	}
	// The data must be on disk before the name points at it.
	if err := f.Sync(); err != nil {
		return fail(err)
	}
	if err := f.Close(); err != nil {
		_ = root.Remove(name)
		return "", fmt.Errorf("journal: write %s: %w", name, err)
	}
	return name, nil
}
