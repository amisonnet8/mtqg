package model

import (
	"time"

	"github.com/amisonnet8/mtqg/internal/journal"
)

// ArchivePlan is what archive does to the records of a journal for a range of
// time: what moves out of view and what has its last event in the range and stays.
type ArchivePlan struct {
	// Move is every record whose lines move, oldest first. An answer or a reply that follows its question or bug is here beside
	// it, and a deleted one is too.
	Move []*Record

	// Skipped are the records whose last event is in the range and that stay:
	// todos, questions and bugs that are open, and glossary entries. Answers and
	// replies are not counted apart from their question or bug.
	Skipped []*Record
}

// ArchiveTargets picks the items to archive for the times from up to, but not
// including, to (docs/reference/schema.md "Archive"). The caller decides what the
// dates of a range mean; this compares times.
//
// An item is a record together with the answers or replies that follow it. Its
// last event is the latest of all of their events, deleted ones included, so that
// every line that moves is dated within the range. An item is archived when its
// last event is in the range and
//
//   - it was deleted, whatever it is: it is out of view already; or
//   - it is a memo, or an answer or a reply with no question or bug in the journal
//     to follow; or
//   - it is a todo, a question or a bug that is done.
//
// A glossary entry that is not deleted is never archived. An item whose events
// have no time that can be read is not in any range, so it stays.
func (s *State) ArchiveTargets(from, to time.Time) ArchivePlan {
	// The answers and replies of each question or bug, whether they are in view or not.
	followers := make(map[string][]*Record)
	for _, r := range s.records {
		if p := s.parentOf(r); p != nil {
			followers[p.ID] = append(followers[p.ID], r)
		}
	}

	move := make(map[*Record]bool)
	var plan ArchivePlan
	for _, r := range s.records {
		if s.parentOf(r) != nil {
			continue // it goes with its question or bug
		}
		last := latestEvent(r)
		for _, f := range followers[r.ID] {
			if at := latestEvent(f); at.After(last) {
				last = at
			}
		}
		if last.IsZero() || last.Before(from) || !last.Before(to) {
			continue
		}
		if archivable(r) {
			move[r] = true
			for _, f := range followers[r.ID] {
				move[f] = true
			}
		} else {
			plan.Skipped = append(plan.Skipped, r)
		}
	}
	for _, r := range s.records {
		if move[r] {
			plan.Move = append(plan.Move, r)
		}
	}
	return plan
}

// archivable says whether an item that is in the range is one that archive moves.
func archivable(r *Record) bool {
	switch {
	case r.Deleted:
		return true
	case r.Type == journal.TypeGlossary:
		return false
	case r.HasState():
		return r.Status == journal.StatusDone
	default:
		return true // a memo, or an answer or a reply that follows nothing
	}
}

// latestEvent is the time of the latest event of a record, or the zero time if none
// of them has a time that can be read.
func latestEvent(r *Record) time.Time {
	var last time.Time
	for _, ev := range r.Events {
		if at := parseTime(ev.TS); at.After(last) {
			last = at
		}
	}
	return last
}
