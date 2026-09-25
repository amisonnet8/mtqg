package model

import (
	"sort"
	"time"

	"github.com/amisonnet8/mtqg/internal/journal"
)

// The kinds of record as people speak of them. A qa record is a question or an
// answer, and a bug record is a bug or a reply; the journal tells them apart by re.
// The two have the same shape: a parent that can be replied to, and its replies.
const (
	KindMemo     = "memo"
	KindTodo     = "todo"
	KindQuestion = "question"
	KindAnswer   = "answer"
	KindBug      = "bug"
	KindReply    = "reply"
	KindGlossary = "glossary entry"
	KindRule     = "rule"
)

// Record is one memo, todo, question, answer, bug, reply or glossary entry, as
// its events leave it.
type Record struct {
	ID string

	// Type is the type field of the format: memo, todo, qa, bug or glossary.
	Type string

	// Text is the body. For a glossary entry it is the definition.
	Text string

	// Word is the term of a glossary entry.
	Word string

	// Re is the ID of the question or bug that an answer or a reply belongs to.
	Re string

	// Status is open or done for a todo, a question and a bug. It is empty for
	// records that have no state.
	Status string

	// Deleted records are hidden: no list shows them and no ID matches them. So is
	// an answer or a reply whose question or bug is deleted (State.visible).
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

// Kind returns how to call the record: memo, todo, question, answer, bug, reply
// or glossary entry.
func (r *Record) Kind() string { return kindOf(r.Type, r.Re != "") }

// kindOf is the kind of a record of a type. reply says that it has re, which only
// makes a difference for the types that have replies (qa and bug).
func kindOf(typ string, reply bool) string {
	switch typ {
	case journal.TypeMemo:
		return KindMemo
	case journal.TypeTodo:
		return KindTodo
	case journal.TypeGlossary:
		return KindGlossary
	case journal.TypeRule:
		return KindRule
	case journal.TypeBug:
		if reply {
			return KindReply
		}
		return KindBug
	default:
		if reply {
			return KindAnswer
		}
		return KindQuestion
	}
}

// ParentKind is the kind of a record of this type that is not a reply: what a
// create of the type starts (a question for qa, a bug for bug; for memo, todo and
// glossary, the kind itself).
func ParentKind(typ string) string { return kindOf(typ, false) }

// ReplyKind is the kind of a record of this type that has re: an answer for qa, a
// reply for bug. The other types have no replies, and it is their own kind.
func ReplyKind(typ string) string { return kindOf(typ, true) }

// HasState reports whether the record has a state to change: todos, questions and
// bugs do; memos, answers, replies and glossary entries do not.
func (r *Record) HasState() bool {
	k := r.Kind()
	return k == KindTodo || k == KindQuestion || k == KindBug
}

// CanHaveReplies reports whether the record can be answered or replied to: a
// question or a bug. An answer or a reply cannot.
func (r *Record) CanHaveReplies() bool {
	k := r.Kind()
	return k == KindQuestion || k == KindBug
}

// IsReply reports whether the record is an answer or a reply.
func (r *Record) IsReply() bool {
	k := r.Kind()
	return k == KindAnswer || k == KindReply
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
// belong to one record and have the same time are told apart by what they do
// (an edit before a delete), and after that by the order they were given in,
// which is the one place where the order of lines counts.
//
// A record starts with its create, whatever the times say. A change whose time
// is earlier than the create of its record still counts, applied after it, in
// order with the other changes: the clock of the machine that made the change
// may run behind the one that made the record, and dropping the change would
// lose what that person did.
//
// An event for an ID that no create started is ignored (its record may have
// been archived). A second create of an ID is ignored. Events that are repeated
// change nothing the second time. None of these is an error: what the journal
// holds is shown as it is.
func Build(events []journal.Event) *State {
	ordered := sortEvents(events)
	state := &State{byID: make(map[string]*Record, len(ordered))}

	// First the records: the earliest create of each ID.
	for _, ev := range ordered {
		if ev.Op != journal.OpCreate {
			continue
		}
		if _, exists := state.byID[ev.ID]; exists {
			continue
		}
		rec := &Record{
			ID: ev.ID, Type: ev.Type, Text: ev.Text, Word: ev.Word, Re: ev.Re,
			Author: ev.Author, Created: parseTime(ev.TS),
		}
		if rec.HasState() {
			rec.Status = ev.Status
			if rec.Status != journal.StatusDone {
				rec.Status = journal.StatusOpen
			}
		}
		state.byID[ev.ID] = rec
		state.records = append(state.records, rec)
	}

	// Then every event, in order, is added to its record and applied. The create
	// that started the record is already applied; another create is skipped.
	created := make(map[string]bool, len(state.records))
	for _, ev := range ordered {
		rec := state.byID[ev.ID]
		if rec == nil {
			continue
		}
		at := parseTime(ev.TS)
		switch ev.Op {
		case journal.OpCreate:
			if created[ev.ID] {
				continue
			}
			created[ev.ID] = true
		case journal.OpStatus:
			if rec.HasState() && (ev.Status == journal.StatusOpen || ev.Status == journal.StatusDone) {
				rec.Status = ev.Status
			}
		case journal.OpEdit:
			rec.Text = ev.Text
		case journal.OpDelete:
			rec.Deleted = true
		default:
			continue
		}
		rec.record(ev, at)
	}
	return state
}

func (r *Record) record(ev journal.Event, at time.Time) {
	r.Events = append(r.Events, ev)
	r.Updated = at
}

// Record returns the record with this full ID, deleted or not, or nil.
func (s *State) Record(id string) *Record { return s.byID[id] }

// visible reports whether a record is in view: it is not deleted, and if it is
// an answer or a reply, its question or bug is not deleted either (deleting a
// question or a bug hides its answers or replies). An answer whose question is not
// in the journal at all is in view: what is missing is not the same as what was
// deleted.
func (s *State) visible(r *Record) bool {
	if r.Deleted {
		return false
	}
	if p := s.parentOf(r); p != nil && p.Deleted {
		return false
	}
	return true
}

// parentOf returns the question or bug that an answer or a reply belongs to, or
// nil if it is not one, or there is no such record. A reply has the same type as
// its parent, and only a question or a bug can be one: an re that names a record of
// another type (or an answer) does not make a reply of it, and is not an error.
func (s *State) parentOf(r *Record) *Record {
	if !r.IsReply() {
		return nil
	}
	p := s.byID[r.Re]
	if p == nil || p.Type != r.Type || !p.CanHaveReplies() {
		return nil
	}
	return p
}

// All returns every record that is in view, of every kind, oldest first.
func (s *State) All() []*Record {
	return s.pick(func(*Record) bool { return true })
}

// Todos returns the todos that are in view, oldest first. Done ones are included
// only when includeDone is set.
func (s *State) Todos(includeDone bool) []*Record {
	return s.pick(func(r *Record) bool {
		return r.Kind() == KindTodo && (includeDone || r.Status == journal.StatusOpen)
	})
}

// Memos returns the memos that are in view, oldest first.
func (s *State) Memos() []*Record {
	return s.pick(func(r *Record) bool { return r.Kind() == KindMemo })
}

// Rules returns the rules that are in view, oldest first.
func (s *State) Rules() []*Record {
	return s.pick(func(r *Record) bool { return r.Kind() == KindRule })
}

// Parents returns the records of a type that can be replied to and are in view,
// oldest first: the questions of type qa, the bugs of type bug. Closed ones are
// included only when includeDone is set.
func (s *State) Parents(typ string, includeDone bool) []*Record {
	kind := ParentKind(typ)
	return s.pick(func(r *Record) bool {
		return r.HasState() && r.Kind() == kind && (includeDone || r.Status == journal.StatusOpen)
	})
}

// Replies returns the answers or replies in view to the question or bug with this
// full ID, oldest first. Only a question or a bug has them, and a reply has the
// type of its parent: a record whose re names a record of another type is not one.
func (s *State) Replies(parentID string) []*Record {
	parent := s.byID[parentID]
	if parent == nil || !parent.CanHaveReplies() {
		return nil
	}
	return s.pick(func(r *Record) bool { return r.IsReply() && r.Re == parentID && r.Type == parent.Type })
}

// HasParent reports whether the question or bug of an answer or a reply is in the
// journal. One without it (its parent may be in an archive) is left out of the
// lists of questions and bugs.
func (s *State) HasParent(reply *Record) bool { return s.parentOf(reply) != nil }

// Parent returns the question or bug that an answer or a reply belongs to, the
// same as HasParent decides it (nil for anything else, or if there is none).
func (s *State) Parent(reply *Record) *Record { return s.parentOf(reply) }

// Glossary returns the glossary entries that are in view, oldest first.
func (s *State) Glossary() []*Record {
	return s.pick(func(r *Record) bool { return r.Kind() == KindGlossary })
}

// DuplicateWords returns the words that are defined more than once, each with its
// entries in the order they were written, the words in the order of their first
// entry. Two words are the same only if they are the same character for
// character: mtqg does not decide that Token and token mean one thing. Nothing
// here says which definition is right.
func (s *State) DuplicateWords() [][]*Record {
	var order []string
	byWord := make(map[string][]*Record)
	for _, r := range s.Glossary() {
		if _, seen := byWord[r.Word]; !seen {
			order = append(order, r.Word)
		}
		byWord[r.Word] = append(byWord[r.Word], r)
	}
	var groups [][]*Record
	for _, word := range order {
		if len(byWord[word]) > 1 {
			groups = append(groups, byWord[word])
		}
	}
	return groups
}

func (s *State) pick(keep func(*Record) bool) []*Record {
	var out []*Record
	for _, r := range s.records {
		if s.visible(r) && keep(r) {
			out = append(out, r)
		}
	}
	return out
}

// Summary counts what a person wants to know at a glance.
type Summary struct {
	OpenTodos int

	// OpenQuestions are the questions that are not closed. The ones awaiting
	// confirmation are those of them that have an answer. Likewise for the bugs and
	// their replies.
	OpenQuestions                 int
	QuestionsAwaitingConfirmation int
	OpenBugs                      int
	BugsAwaitingConfirmation      int

	// GlossaryEntries counts the entries, so that two definitions of one word are
	// two. DuplicateWords counts the words that have more than one.
	GlossaryEntries int
	DuplicateWords  int

	// ConcurrentStatusChanges counts the records that have them (ConcurrentStatusChanges).
	ConcurrentStatusChanges int
}

// Summary counts the records that are in view.
func (s *State) Summary() Summary {
	sum := Summary{
		OpenTodos:       len(s.Todos(false)),
		GlossaryEntries: len(s.Glossary()),
		DuplicateWords:  len(s.DuplicateWords()),

		ConcurrentStatusChanges: len(s.ConcurrentStatusChanges()),
	}
	sum.OpenQuestions, sum.QuestionsAwaitingConfirmation = s.openParents(journal.TypeQA)
	sum.OpenBugs, sum.BugsAwaitingConfirmation = s.openParents(journal.TypeBug)
	return sum
}

// openParents counts the open questions or bugs, and the ones of them that have an
// answer or a reply.
func (s *State) openParents(typ string) (open, awaiting int) {
	for _, p := range s.Parents(typ, false) {
		open++
		if len(s.Replies(p.ID)) > 0 {
			awaiting++
		}
	}
	return open, awaiting
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
	order := EventOrder(events)
	out := make([]journal.Event, len(order))
	for i, at := range order {
		out[i] = events[at]
	}
	return out
}

// EventOrder returns the indexes of events in the order they are applied in and
// shown in: by time, by ID for the same time, a create before the changes of a
// record and a delete after them, and for the rest in the order they were given.
// Anything that shows events from any source in time order (format) uses this, so
// that there is one order.
func EventOrder(events []journal.Event) []int {
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
	order := make([]int, len(items))
	for i, item := range items {
		order[i] = item.index
	}
	return order
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
