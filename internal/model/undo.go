package model

import (
	"time"

	"github.com/amisonnet8/mtqg/internal/journal"
)

// UndoTarget returns the index of the line that undo removes: of the lines
// written by this author from this terminal (tty, which is empty for a line
// without one), the one with the latest time; of those with the same time, the one
// that comes last in the file.
//
// The file order decides among lines of one second because their IDs are random
// and say nothing about which came first, while appends are made one after the
// other under the lock, so in one file the later line is the one that was written
// later. A merge can bring in lines of another machine, and if it wrote as the same
// author from the same tty it can be chosen: that is the case that MTQG_TTY is for.
//
// A line that could not be read as an event is never the target.
func UndoTarget(lines []journal.Line, author journal.Author, tty string) (int, error) {
	best := -1
	var latest time.Time // the time of the line at best
	for i, line := range lines {
		ev := line.Event
		if ev == nil || ev.Author != author || ev.TTY != tty {
			continue
		}
		// Not before: a line of the same time that comes later in the file wins.
		if at := parseTime(ev.TS); best < 0 || !at.Before(latest) {
			best, latest = i, at
		}
	}
	if best < 0 {
		return -1, ErrNothingToUndo
	}
	return best, nil
}

// Orphaned returns the events that removing ev would leave without a record: if
// ev creates a record, the other events of that record and the creation of each
// answer or reply to it. Removing the line of any other event leaves nothing
// behind, and the state that is read is as if the change had not been made.
func (s *State) Orphaned(ev journal.Event) []journal.Event {
	if ev.Op != journal.OpCreate {
		return nil
	}
	rec := s.byID[ev.ID]
	if rec == nil {
		return nil
	}
	var others []journal.Event
	for _, e := range rec.Events {
		if e.Op != journal.OpCreate {
			others = append(others, e)
		}
	}
	for _, r := range s.records {
		if r.Re == rec.ID && r.Type == rec.Type && len(r.Events) > 0 {
			others = append(others, r.Events[0]) // the creation of the reply
		}
	}
	return others
}

// CanUndo says whether the line of ev may be removed: a *HasLaterEventsError if it
// creates a record that has other events.
func (s *State) CanUndo(ev journal.Event) error {
	others := s.Orphaned(ev)
	if len(others) == 0 {
		return nil
	}
	return &HasLaterEventsError{Record: s.byID[ev.ID], Events: others}
}
