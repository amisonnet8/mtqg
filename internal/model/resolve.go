package model

import (
	"strings"

	"github.com/amisonnet8/mtqg/internal/journal"
)

// Resolve finds the record that a typed ID stands for. The ID is a prefix of the
// full ID, as long as it is unique, the way git takes a prefix of a hash. Case
// does not matter. A deleted record is never matched, so it can neither be found
// nor make another ID ambiguous.
//
// It returns a *NotFoundError, or an *AmbiguousError that lists the candidates.
func (s *State) Resolve(prefix string) (*Record, error) {
	p := strings.ToLower(strings.TrimSpace(prefix))
	if p == "" || strings.Trim(p, "0123456789abcdef") != "" {
		return nil, &NotFoundError{Prefix: prefix}
	}
	var matches []*Record
	for _, rec := range s.records {
		if !rec.Deleted && strings.HasPrefix(rec.ID, p) {
			matches = append(matches, rec)
		}
	}
	switch len(matches) {
	case 0:
		return nil, &NotFoundError{Prefix: prefix}
	case 1:
		return matches[0], nil
	default:
		return nil, &AmbiguousError{Prefix: prefix, Candidates: matches}
	}
}

// ResolveKind is Resolve for a command that works on one kind of record: a
// record of another kind is a *WrongKindError that says what it is.
func (s *State) ResolveKind(prefix, kind string) (*Record, error) {
	rec, err := s.Resolve(prefix)
	if err != nil {
		return nil, err
	}
	if rec.Kind() != kind {
		return nil, &WrongKindError{Record: rec, Want: kind}
	}
	return rec, nil
}

// MemoCreate returns the event that creates a memo.
func MemoCreate(text string) (journal.Event, error) {
	if strings.TrimSpace(text) == "" {
		return journal.Event{}, ErrEmptyText
	}
	return journal.Event{Op: journal.OpCreate, Type: journal.TypeMemo, Text: text}, nil
}

// TodoCreate returns the event that creates a todo, which starts open.
func TodoCreate(text string) (journal.Event, error) {
	if strings.TrimSpace(text) == "" {
		return journal.Event{}, ErrEmptyText
	}
	return journal.Event{Op: journal.OpCreate, Type: journal.TypeTodo, Status: journal.StatusOpen, Text: text}, nil
}

// SetStatus returns the event that puts a todo or a question into a state. It
// records the state the record was in, as the writer saw it: that is what shows
// two changes that were made without knowing of each other.
//
// A record that has no state is a *NoStateError. A record that is in the state
// already is ErrNoChange, and nothing should be written for it.
func SetStatus(rec *Record, status string) (journal.Event, error) {
	if !rec.HasState() {
		return journal.Event{}, &NoStateError{Record: rec}
	}
	if status != journal.StatusOpen && status != journal.StatusDone {
		return journal.Event{}, &journal.InvalidEventError{Field: "status", Reason: "must be open or done"}
	}
	if rec.Status == status {
		return journal.Event{}, ErrNoChange
	}
	return journal.Event{ID: rec.ID, Op: journal.OpStatus, From: rec.Status, Status: status}, nil
}
