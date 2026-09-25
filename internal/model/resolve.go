package model

import (
	"strings"

	"github.com/amisonnet8/mtqg/internal/journal"
)

// MinIDDigits is how short a typed ID may be. Anything shorter is refused, so
// that a word such as add or bad is never taken for the ID of a record, and
// because a digit or two matches nearly everything.
const MinIDDigits = 4

// IsIDLike reports whether a word is made of hex digits only, and at least
// MinIDDigits of them: the shape of an ID that was typed (upper case counts as
// lower case). Whether such a word names a record is for Resolve to say.
func IsIDLike(word string) bool {
	return len(word) >= MinIDDigits && isHex(strings.ToLower(word))
}

func isHex(s string) bool { return s != "" && strings.Trim(s, "0123456789abcdef") == "" }

// Resolve finds the record that a typed ID stands for. The ID is a prefix of the
// full ID of at least MinIDDigits digits, as long as it is unique, the way git
// takes a prefix of a hash. Case does not matter. A record that is not in view
// (deleted, or an answer or a reply to a deleted question or bug) is never
// matched, so it can neither be found nor make another ID ambiguous.
//
// It returns a *NotFoundError, a *TooShortError, or an *AmbiguousError that lists
// the candidates.
func (s *State) Resolve(prefix string) (*Record, error) {
	p := strings.ToLower(strings.TrimSpace(prefix))
	if !isHex(p) {
		return nil, &NotFoundError{Prefix: prefix}
	}
	if len(p) < MinIDDigits {
		return nil, &TooShortError{Prefix: prefix}
	}
	var matches []*Record
	for _, rec := range s.records {
		if s.visible(rec) && strings.HasPrefix(rec.ID, p) {
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

// RuleCreate returns the event that creates a rule.
func RuleCreate(text string) (journal.Event, error) {
	if strings.TrimSpace(text) == "" {
		return journal.Event{}, ErrEmptyText
	}
	return journal.Event{Op: journal.OpCreate, Type: journal.TypeRule, Text: text}, nil
}

// TodoCreate returns the event that creates a todo, which starts open.
func TodoCreate(text string) (journal.Event, error) {
	if strings.TrimSpace(text) == "" {
		return journal.Event{}, ErrEmptyText
	}
	return journal.Event{Op: journal.OpCreate, Type: journal.TypeTodo, Status: journal.StatusOpen, Text: text}, nil
}

// ParentCreate returns the event that creates a question (type qa) or a bug (type
// bug), which starts open. Any other type has nothing to be replied to.
func ParentCreate(typ, text string) (journal.Event, error) {
	if typ != journal.TypeQA && typ != journal.TypeBug {
		return journal.Event{}, &journal.InvalidEventError{Field: "type", Reason: "must be qa or bug"}
	}
	if strings.TrimSpace(text) == "" {
		return journal.Event{}, ErrEmptyText
	}
	return journal.Event{Op: journal.OpCreate, Type: typ, Status: journal.StatusOpen, Text: text}, nil
}

// ReplyCreate returns the event that creates an answer to a question, or a reply
// to a bug. It has the type of its parent, and holds the full ID of the parent
// (re), whatever length of it was typed. Only a question or a bug can be replied
// to, and an answer or a reply has no state.
func ReplyCreate(parent *Record, text string) (journal.Event, error) {
	if !parent.CanHaveReplies() {
		return journal.Event{}, &NoRepliesError{Record: parent}
	}
	if strings.TrimSpace(text) == "" {
		return journal.Event{}, ErrEmptyText
	}
	return journal.Event{Op: journal.OpCreate, Type: parent.Type, Re: parent.ID, Text: text}, nil
}

// GlossaryCreate returns the event that defines a word. The word is kept as it
// was given: two entries are of one word only if the words are the same
// character for character.
func GlossaryCreate(word, text string) (journal.Event, error) {
	if strings.TrimSpace(word) == "" {
		return journal.Event{}, ErrEmptyWord
	}
	if strings.TrimSpace(text) == "" {
		return journal.Event{}, ErrEmptyText
	}
	return journal.Event{Op: journal.OpCreate, Type: journal.TypeGlossary, Word: word, Text: text}, nil
}

// SetStatus returns the event that puts a todo, a question or a bug into a state. It
// records the state the record was in, as the writer saw it (from), and how many
// events the record had (basis): together, that is what shows a change that was
// made without knowing of another one to the same record (review).
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
	return journal.Event{ID: rec.ID, Op: journal.OpStatus, From: rec.Status, Status: status, Basis: len(rec.Events)}, nil
}

// EditText returns the event that replaces the text of a record. Only the text:
// the word of a glossary entry and the state of a todo, a question or a bug are
// not touched. A text that is empty is ErrEmptyText. A text that is the same as the
// record has now is ErrNoChange, and nothing should be written for it.
//
// It records how many events the record had (basis), the same way SetStatus
// records from: what shows an edit that was made without knowing of another
// change to the same record, whichever field that other change touched (review).
func EditText(rec *Record, text string) (journal.Event, error) {
	if strings.TrimSpace(text) == "" {
		return journal.Event{}, ErrEmptyText
	}
	if rec.Text == text {
		return journal.Event{}, ErrNoChange
	}
	return journal.Event{ID: rec.ID, Op: journal.OpEdit, Text: text, Basis: len(rec.Events)}, nil
}

// Delete returns the event that hides a record. Its answers or replies are hidden
// with it: that is how the record is read (State.visible), not a second event.
func Delete(rec *Record) (journal.Event, error) {
	return journal.Event{ID: rec.ID, Op: journal.OpDelete}, nil
}
