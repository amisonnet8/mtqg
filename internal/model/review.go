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

// ConcurrentStatusChanges returns the records in view that have concurrent status
// changes, in the order they were created.
//
// The status changes of a record are taken in the order they are applied in and
// followed from the state the record was created in. A change whose from is not the
// state the record is in at that point was written by someone who had not seen the
// change before it: two branches that each closed the todo, merged later. Counting
// changes with the same from would call a plain open, done, open, done a conflict.
// A change that has no from is not judged, and a state that is not open or done is
// not a change (the record ignores it too).
func (s *State) ConcurrentStatusChanges() []StatusConflict {
	var out []StatusConflict
	for _, rec := range s.records {
		if !s.visible(rec) || !rec.HasState() {
			continue
		}
		// The state it was created in. The create is not always the first of the
		// events: a change written by a clock that runs behind is earlier than it.
		current := journal.StatusOpen
		for _, ev := range rec.Events {
			if ev.Op == journal.OpCreate {
				if ev.Status == journal.StatusDone {
					current = journal.StatusDone
				}
				break
			}
		}
		var changes []Entry
		conflict := false
		for _, ev := range rec.Events {
			if ev.Op != journal.OpStatus || (ev.Status != journal.StatusOpen && ev.Status != journal.StatusDone) {
				continue
			}
			if ev.From != "" && ev.From != current {
				conflict = true
			}
			current = ev.Status
			changes = append(changes, Entry{Event: ev, At: parseTime(ev.TS)})
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
