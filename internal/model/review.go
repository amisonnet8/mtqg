package model

import (
	"github.com/amisonnet8/mtqg/internal/journal"
)

// What review shows: places where the journal holds facts that do not agree. The
// model finds them and does not decide which fact is right (docs/reference/cli.md,
// review).

// StatusConflict is a record whose status changes were made without knowing of
// each other: the record, and all of its status changes, oldest first.
type StatusConflict struct {
	Record  *Record
	Changes []Entry
}

// ConcurrentStatusChanges returns the records in view that have changes made
// without knowing of one another, in the order they were created. Despite the
// name, an edit can be one of the changes listed (see below); it keeps the name
// (and the name of Summary's field, and of the --json field) it had before edits
// could be caught too, since that is a promise to whatever reads --json (only
// fields are added, cli-output.md).
//
// Two independent signals catch this, so that a change written before basis
// existed is still caught.
//
//  1. A status change whose from is not the state the record is in at that
//     point, taken in the order the events are applied in and followed from the
//     state the record was created in: it was written by someone who had not
//     seen the change before it (two branches that each closed the todo, merged
//     later). Counting changes with the same from would call a plain open, done,
//     open, done a conflict, so the comparison is against the state at that
//     point, not against other events' from. A change that has no from is not
//     judged this way, and a state that is not open or done is not a change (the
//     record ignores it too).
//  2. A status change or an edit whose basis does not match how many events
//     (including the create) the record actually had at that point: the writer
//     had not seen something that came before it, whichever field that something
//     touched. A change that has no basis is not judged this way (data written
//     before the field existed, or by another tool).
//
// Either signal, on any change, flags the whole record, which is then listed
// with all of its status changes and edits, oldest first.
func (s *State) ConcurrentStatusChanges() []StatusConflict {
	var out []StatusConflict
	for _, rec := range s.records {
		if !s.visible(rec) {
			continue
		}
		// The state it was created in. The create is not always the first of the
		// events: a change written by a clock that runs behind is earlier than it.
		current := journal.StatusOpen
		if rec.HasState() {
			for _, ev := range rec.Events {
				if ev.Op == journal.OpCreate {
					if ev.Status == journal.StatusDone {
						current = journal.StatusDone
					}
					break
				}
			}
		}
		seen := 0
		var changes []Entry
		conflict := false
		for _, ev := range rec.Events {
			switch ev.Op {
			case journal.OpStatus:
				if rec.HasState() && (ev.Status == journal.StatusOpen || ev.Status == journal.StatusDone) {
					if ev.From != "" && ev.From != current {
						conflict = true
					}
					if ev.Basis != 0 && ev.Basis != seen {
						conflict = true
					}
					current = ev.Status
					changes = append(changes, Entry{Event: ev, At: parseTime(ev.TS)})
				}
			case journal.OpEdit:
				if ev.Basis != 0 && ev.Basis != seen {
					conflict = true
				}
				changes = append(changes, Entry{Event: ev, At: parseTime(ev.TS)})
			}
			seen++
		}
		if conflict {
			out = append(out, StatusConflict{Record: rec, Changes: changes})
		}
	}
	return out
}

// UnattachedReplies returns the answers and replies in view that belong to no
// question or bug: the record their re names is not in the journal, or is of another
// kind. Such a record is not shown under any question or bug, so this is where it is
// found. They are in the order they were created.
func (s *State) UnattachedReplies() []*Record {
	return s.pick(func(r *Record) bool { return r.IsReply() && !s.HasParent(r) })
}
