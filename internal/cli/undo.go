package cli

import (
	"bytes"
	"errors"

	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

// runUndo removes the last line that this author wrote from this terminal. It is
// the only command besides archive that removes a line, so it is careful: it says
// what it removed, and it refuses, writing nothing, when the line is the creation
// of a record that other events are about.
func runUndo(c *ctx) int {
	j, author, tty, err := c.writerAs()
	if err != nil {
		return c.fail(err)
	}

	var (
		removed journal.Event
		record  *model.Record // as it was before the line was removed
	)
	err = j.Rewrite(func(lines []journal.Line) ([]journal.Line, error) {
		i, err := model.UndoTarget(lines, author, tty)
		if err != nil {
			return nil, err
		}
		target := lines[i]
		state := model.Build(eventsOf(lines))
		if err := state.CanUndo(*target.Event); err != nil {
			return nil, err
		}
		removed, record = *target.Event, state.Record(target.Event.ID)

		// A line that is in the file more than once is one event (reading counts it
		// once), so all of its copies go.
		kept := make([]journal.Line, 0, len(lines))
		for _, l := range lines {
			if !bytes.Equal(l.Raw, target.Raw) {
				kept = append(kept, l)
			}
		}
		return kept, nil
	})
	if errors.Is(err, model.ErrNothingToUndo) {
		return c.failWith(errorReport{kind: kindNothingToUndo, lines: []string{msgNothingToUndo(oneLine(author.Name))}})
	}
	if err != nil {
		return c.fail(err)
	}

	if c.inv.json {
		out := jsonUndo{Command: c.inv.cmd.label(), Event: removed}
		if record != nil {
			r := recordJSON(record)
			out.Record = &r
		}
		return c.emit(out)
	}
	typ, what, text := describeLine(removed, record)
	c.println(msgUndone(typ, what, oneLine(text), shortID(removed.ID, c.inv.fullID)))
	return exitOK
}

// eventsOf is the events of lines that could be read, each once: a line that is in
// the file twice is one event, as it is when reading.
func eventsOf(lines []journal.Line) []journal.Event {
	seen := make(map[string]struct{}, len(lines))
	var events []journal.Event
	for _, l := range lines {
		if l.Event == nil {
			continue
		}
		if _, dup := seen[string(l.Raw)]; dup {
			continue
		}
		seen[string(l.Raw)] = struct{}{}
		events = append(events, *l.Event)
	}
	return events
}

// describeLine says what a line was, for "Undone:": the type of its record, what
// it did (add, done, reopen, edit, delete) and the text it holds, or else the text
// of its record.
func describeLine(ev journal.Event, rec *model.Record) (typ, what, text string) {
	typ = ev.Type
	if typ == "" && rec != nil {
		typ = rec.Type
	}
	if typ == "" {
		typ = "record"
	}
	switch ev.Op {
	case journal.OpCreate:
		what = "add"
	case journal.OpStatus:
		what = "reopen"
		if ev.Status == journal.StatusDone {
			what = "done"
		}
	default:
		what = ev.Op // edit, delete
	}

	switch {
	case ev.Op == journal.OpCreate && ev.Word != "":
		text = ev.Word + ": " + ev.Text
	case ev.Text != "":
		text = ev.Text
	case rec != nil:
		text = rec.Text
	}
	return typ, what, text
}
