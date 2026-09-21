package model

import (
	"sort"
	"time"

	"github.com/amisonnet8/mtqg/internal/journal"
)

// The kinds of record as people speak of them. A qa record is a question or an
// answer, and the journal tells them apart by re.
const (
	KindMemo     = "memo"
	KindTodo     = "todo"
	KindQuestion = "question"
	KindAnswer   = "answer"
	KindGlossary = "glossary entry"
)

// Record is one memo, todo, question, answer or glossary entry, as its events
// leave it.
type Record struct {
	ID string

	// Type is the type field of the format: memo, todo, qa or glossary.
	Type string

	// Text is the body. For a glossary entry it is the definition.
	Text string

	// Word is the term of a glossary entry.
	Word string

	// Re is the ID of the question that an answer belongs to.
	Re string

	// Status is open or done for a todo and a question. It is empty for records
	// that have no state.
	Status string

	// Deleted records are hidden: no list shows them and no ID matches them.
	Deleted bool

	// Author wrote the create event.
	Author journal.Author

	// Created and Updated are the times of the first and of the last event. They
	// are the zero time if the ts of an event could not be read.
	Created time.Time
	Updated time.Time

	// Events are the events of the record in order.
	Events []journal.Event
}

// Kind returns how to call the record: memo, todo, question, answer or glossary
// entry.
func (r *Record) Kind() string {
	switch {
	case r.Type == journal.TypeMemo:
		return KindMemo
	case r.Type == journal.TypeTodo:
		return KindTodo
	case r.Type == journal.TypeGlossary:
		return KindGlossary
	case r.Re != "":
		return KindAnswer
	default:
		return KindQuestion
	}
}

// HasState reports whether the record has a state to change: todos and
// questions do; memos, answers and glossary entries do not.
func (r *Record) HasState() bool {
	return r.Kind() == KindTodo || r.Kind() == KindQuestion
}

// State is the records that a journal's events amount to.
type State struct {
	records []*Record // in the order they were created
	byID    map[string]*Record
}

// Build puts events in order and applies them, record by record.
//
// The order is by time (ts) and, for the same time, by ID: the order of lines in
// the file carries no meaning, because a merge does not keep it. Events that
// belong to one record and have the same time are told apart by what they do (a
// create comes before a change), and after that by the order they were given in,
// which is the one place where the order of lines counts.
//
// An event for an ID that no create started is ignored (its record may have
// been archived). A second create of an ID is ignored. Events that are repeated
// change nothing the second time. None of these is an error: what the journal
// holds is shown as it is.
func Build(events []journal.Event) *State {
	ordered := sortEvents(events)
	state := &State{byID: make(map[string]*Record, len(ordered))}
	for _, ev := range ordered {
		at := parseTime(ev.TS)
		switch ev.Op {
		case journal.OpCreate:
			if _, exists := state.byID[ev.ID]; exists {
				continue
			}
			rec := &Record{
				ID: ev.ID, Type: ev.Type, Text: ev.Text, Word: ev.Word, Re: ev.Re,
				Author: ev.Author, Created: at,
			}
			if rec.HasState() {
				rec.Status = ev.Status
				if rec.Status != journal.StatusDone {
					rec.Status = journal.StatusOpen
				}
			}
			state.byID[ev.ID] = rec
			state.records = append(state.records, rec)
			rec.record(ev, at)
		case journal.OpStatus:
			if rec := state.byID[ev.ID]; rec != nil {
				if rec.HasState() && (ev.Status == journal.StatusOpen || ev.Status == journal.StatusDone) {
					rec.Status = ev.Status
				}
				rec.record(ev, at)
			}
		case journal.OpEdit:
			if rec := state.byID[ev.ID]; rec != nil {
				rec.Text = ev.Text
				rec.record(ev, at)
			}
		case journal.OpDelete:
			if rec := state.byID[ev.ID]; rec != nil {
				rec.Deleted = true
				rec.record(ev, at)
			}
		}
	}
	return state
}

func (r *Record) record(ev journal.Event, at time.Time) {
	r.Events = append(r.Events, ev)
	r.Updated = at
}

// Record returns the record with this full ID, deleted or not, or nil.
func (s *State) Record(id string) *Record { return s.byID[id] }

// Todos returns the todos that are not deleted, oldest first. Done ones are
// included only when includeDone is set.
func (s *State) Todos(includeDone bool) []*Record {
	return s.pick(func(r *Record) bool {
		return r.Kind() == KindTodo && (includeDone || r.Status == journal.StatusOpen)
	})
}

// Memos returns the memos that are not deleted, oldest first.
func (s *State) Memos() []*Record {
	return s.pick(func(r *Record) bool { return r.Kind() == KindMemo })
}

func (s *State) pick(keep func(*Record) bool) []*Record {
	var out []*Record
	for _, r := range s.records {
		if !r.Deleted && keep(r) {
			out = append(out, r)
		}
	}
	return out
}

// Summary counts what a person wants to know at a glance.
type Summary struct {
	OpenTodos int
}

// Summary counts the records that are not deleted.
func (s *State) Summary() Summary {
	return Summary{OpenTodos: len(s.Todos(false))}
}

// UncommittedRecords counts the records that the given events belong to. Give it
// the events that are not in the last commit: a record counts once, however many
// of its events there are.
func UncommittedRecords(events []journal.Event) int {
	ids := make(map[string]struct{}, len(events))
	for _, ev := range events {
		ids[ev.ID] = struct{}{}
	}
	return len(ids)
}

// sortEvents returns the events in the order they are applied in.
func sortEvents(events []journal.Event) []journal.Event {
	type keyed struct {
		ev    journal.Event
		at    time.Time
		index int
	}
	items := make([]keyed, len(events))
	for i, ev := range events {
		items[i] = keyed{ev: ev, at: parseTime(ev.TS), index: i}
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if !a.at.Equal(b.at) {
			return a.at.Before(b.at)
		}
		if a.ev.TS != b.ev.TS {
			return a.ev.TS < b.ev.TS
		}
		if a.ev.ID != b.ev.ID {
			return a.ev.ID < b.ev.ID
		}
		if ra, rb := opRank(a.ev.Op), opRank(b.ev.Op); ra != rb {
			return ra < rb
		}
		return a.index < b.index
	})
	out := make([]journal.Event, len(items))
	for i, item := range items {
		out[i] = item.ev
	}
	return out
}

// opRank puts a create before the changes of the same record and a delete after
// them, for events that have the same time.
func opRank(op string) int {
	switch op {
	case journal.OpCreate:
		return 0
	case journal.OpDelete:
		return 2
	default:
		return 1
	}
}

// parseTime reads a ts. A ts that cannot be read is the zero time, so that it
// sorts first and never stops anything.
func parseTime(ts string) time.Time {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}
