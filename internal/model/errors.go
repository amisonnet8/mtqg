package model

import (
	"errors"
	"fmt"
)

// The model returns kinds of error, not sentences: the wording shown to people
// belongs to the entry points (.claude/rules/cli-output.md). The Error methods
// are for logs and tests.

// ErrNotFound means no record matches an ID that was typed.
var ErrNotFound = errors.New("model: no record matches")

// ErrAmbiguous means more than one record matches an ID that was typed.
var ErrAmbiguous = errors.New("model: the ID matches more than one record")

// ErrWrongKind means the record is not of the kind the command is for.
var ErrWrongKind = errors.New("model: the record is of another kind")

// ErrNoState means the record has no state to change.
var ErrNoState = errors.New("model: the record has no state")

// ErrNoChange means the record is already in the state that was asked for.
var ErrNoChange = errors.New("model: the record is already in that state")

// ErrEmptyText means the text of a record is empty.
var ErrEmptyText = errors.New("model: the text is empty")

// NotFoundError says which ID matched nothing.
type NotFoundError struct{ Prefix string }

func (e *NotFoundError) Error() string { return fmt.Sprintf("model: no record matches %q", e.Prefix) }

// Is makes errors.Is(err, ErrNotFound) true.
func (e *NotFoundError) Is(target error) bool { return target == ErrNotFound }

// AmbiguousError lists the records that an ID matched, oldest first.
type AmbiguousError struct {
	Prefix     string
	Candidates []*Record
}

func (e *AmbiguousError) Error() string {
	return fmt.Sprintf("model: %q matches %d records", e.Prefix, len(e.Candidates))
}

// Is makes errors.Is(err, ErrAmbiguous) true.
func (e *AmbiguousError) Is(target error) bool { return target == ErrAmbiguous }

// WrongKindError says what the record is and what the command wanted.
type WrongKindError struct {
	Record *Record
	Want   string // a Kind constant
}

func (e *WrongKindError) Error() string {
	return fmt.Sprintf("model: %s is a %s, not a %s", e.Record.ID, e.Record.Kind(), e.Want)
}

// Is makes errors.Is(err, ErrWrongKind) true.
func (e *WrongKindError) Is(target error) bool { return target == ErrWrongKind }

// NoStateError names the record that has no state to change.
type NoStateError struct{ Record *Record }

func (e *NoStateError) Error() string {
	return fmt.Sprintf("model: a %s has no state", e.Record.Kind())
}

// Is makes errors.Is(err, ErrNoState) true.
func (e *NoStateError) Is(target error) bool { return target == ErrNoState }
