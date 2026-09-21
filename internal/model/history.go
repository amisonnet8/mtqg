package model

import (
	"sort"
	"time"

	"github.com/amisonnet8/mtqg/internal/journal"
)

// Entry is one line of the history of a record.
type Entry struct {
	Event journal.Event
	At    time.Time // the time of the event, or the zero time if it could not be read

	// Answer is set on the create event of an answer or a reply, in the history of
	// its question or bug. It is nil for the events of the record itself.
	Answer *Record
}

// History returns what happened to a record, oldest first: its own events (the
// create, the changes of state, the edits, the delete) and, for a question or a
// bug, the creation of each of its answers or replies that is in view. Events with
// the same time keep the order the record has them in, and the record's own events
// come before the answers that arrived at the same moment.
func (s *State) History(rec *Record) []Entry {
	entries := make([]Entry, 0, len(rec.Events))
	for _, ev := range rec.Events {
		entries = append(entries, Entry{Event: ev, At: parseTime(ev.TS)})
	}
	if rec.CanHaveReplies() {
		for _, answer := range s.Replies(rec.ID) {
			if len(answer.Events) == 0 {
				continue
			}
			ev := answer.Events[0] // its create: the record starts with it
			entries = append(entries, Entry{Event: ev, At: parseTime(ev.TS), Answer: answer})
		}
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].At.Before(entries[j].At) })
	return entries
}

// EventTime is the time of an event, or the zero time if its ts cannot be read.
func EventTime(ev journal.Event) time.Time { return parseTime(ev.TS) }
