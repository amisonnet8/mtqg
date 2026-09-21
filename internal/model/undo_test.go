package model

import (
	"errors"
	"testing"

	"github.com/amisonnet8/mtqg/internal/journal"
)

func lineOf(ev journal.Event) journal.Line { return journal.Line{Event: &ev} }

func writtenBy(ev journal.Event, author journal.Author, tty string) journal.Event {
	ev.Author, ev.TTY = author, tty
	return ev
}

func TestUndoTarget(t *testing.T) {
	other := journal.Author{Kind: journal.AuthorAI, Name: "yamada"} // the same name, another kind
	lines := []journal.Line{
		lineOf(writtenBy(create(idTodoA, journal.TypeTodo, "a", 0), human, "t1")), // 0
		lineOf(writtenBy(create(idMemo, journal.TypeMemo, "b", 1), human, "t2")),  // 1: another terminal
		lineOf(writtenBy(create(idQ, journal.TypeQA, "c", 2), agent, "t1")),       // 2: another author
		lineOf(writtenBy(create(idWord, journal.TypeMemo, "d", 3), other, "t1")),  // 3: the same name, another kind
		lineOf(writtenBy(status(idTodoA, "open", "done", 4, human), human, "t1")), // 4: the last of yamada at t1
		lineOf(writtenBy(create(idAns, journal.TypeMemo, "e", 5), human, "")),     // 5: no terminal
		{Raw: []byte("not json")}, // 6: unreadable
	}

	tests := []struct {
		name   string
		author journal.Author
		tty    string
		want   int
	}{
		{"this author and this terminal", human, "t1", 4},
		{"another terminal of the same author", human, "t2", 1},
		{"a line without a terminal is its own terminal", human, "", 5},
		{"the AI at t1", agent, "t1", 2},
		{"the same name and another kind is another author", other, "t1", 3},
	}
	for _, tt := range tests {
		got, err := UndoTarget(lines, tt.author, tt.tty)
		if err != nil || got != tt.want {
			t.Errorf("%s: got %d, %v; want %d", tt.name, got, err, tt.want)
		}
	}

	for _, tt := range []struct {
		name   string
		author journal.Author
		tty    string
	}{
		{"an author with no lines", journal.Author{Kind: journal.AuthorHuman, Name: "nobody"}, "t1"},
		{"a terminal with no lines", human, "t9"},
		{"an AI with no line without a terminal", agent, ""},
	} {
		if got, err := UndoTarget(lines, tt.author, tt.tty); !errors.Is(err, ErrNothingToUndo) || got != -1 {
			t.Errorf("%s: got %d, %v; want ErrNothingToUndo", tt.name, got, err)
		}
	}
	if _, err := UndoTarget(nil, human, ""); !errors.Is(err, ErrNothingToUndo) {
		t.Errorf("no lines: err = %v", err)
	}
}

func TestUndoTargetIsTheLatestByTimeNotByPositionInTheFile(t *testing.T) {
	// A merge puts lines in any order: the line that is last in the file is not the
	// latest one.
	lines := []journal.Line{
		lineOf(create(idTodoA, journal.TypeTodo, "later", 5)),
		lineOf(create(idMemo, journal.TypeMemo, "earlier", 1)),
	}
	if got, err := UndoTarget(lines, human, ""); err != nil || got != 0 {
		t.Errorf("got %d, %v; want the line with the latest time (0)", got, err)
	}
}

func TestUndoTargetOfTheSameSecondIsTheLastInTheFile(t *testing.T) {
	// The IDs are random, so they cannot say which was written last. Both orders of
	// IDs must give the line that comes last in the file.
	for _, ids := range [][2]string{{idTodoA, idMemo}, {idMemo, idTodoA}} {
		lines := []journal.Line{
			lineOf(create(ids[0], journal.TypeMemo, "first", 3)),
			lineOf(create(ids[1], journal.TypeMemo, "second", 3)),
		}
		if got, err := UndoTarget(lines, human, ""); err != nil || got != 1 {
			t.Errorf("ids %v: got %d, %v; want the last line of the file (1)", ids, got, err)
		}
	}
}

func TestOrphaned(t *testing.T) {
	events := []journal.Event{
		create(idTodoA, journal.TypeTodo, "a todo that was changed", 0),
		status(idTodoA, "open", "done", 1, agent),
		create(idMemo, journal.TypeMemo, "a memo left alone", 2),
		create(idQ, journal.TypeQA, "a question", 3),
		answer(idAns, idQ, "an answer", 4, human),
		create(idQ2, journal.TypeQA, "a question with a deleted answer", 5),
		answer(idAns2, idQ2, "gone", 6, human),
		del(idAns2, 7),
		create(idWord, journal.TypeGlossary, "an entry that was edited", 8),
		{ID: idWord, Op: journal.OpEdit, Text: "edited", TS: at(9), Author: human},
	}
	state := Build(events)
	byID := map[string]journal.Event{}
	for _, ev := range events {
		if ev.Op == journal.OpCreate {
			byID[ev.ID] = ev
		}
	}

	tests := []struct {
		name string
		ev   journal.Event
		want int
	}{
		{"a todo that was closed", byID[idTodoA], 1},
		{"a memo with nothing else", byID[idMemo], 0},
		{"a question that was answered", byID[idQ], 1},
		{"an answer, which nothing depends on", byID[idAns], 0},
		{"a question whose answer was deleted: the answer is not in view, but its creation is still there", byID[idQ2], 1},
		{"an entry that was edited", byID[idWord], 1},
		{"the change of state of the todo", events[1], 0},
		{"the edit of the entry", events[9], 0},
		{"the delete of the answer", events[7], 0},
	}
	for _, tt := range tests {
		if got := state.Orphaned(tt.ev); len(got) != tt.want {
			t.Errorf("%s: %d events left without a record (%v), want %d", tt.name, len(got), got, tt.want)
		}
	}

	// The refusal names the record and what would be left.
	err := state.CanUndo(byID[idQ])
	var later *HasLaterEventsError
	if !errors.As(err, &later) || !errors.Is(err, ErrHasLaterEvents) || later.Record.ID != idQ || len(later.Events) != 1 || later.Events[0].ID != idAns {
		t.Errorf("CanUndo(question) = %v", err)
	}
	if err := state.CanUndo(byID[idMemo]); err != nil {
		t.Errorf("CanUndo(memo) = %v, want nil", err)
	}
	if err := state.CanUndo(events[1]); err != nil {
		t.Errorf("CanUndo(status change) = %v, want nil", err)
	}
}

func TestAnAnswerToAnotherTypeDoesNotHoldARecordUp(t *testing.T) {
	// A record whose re names a bug is not an answer to it (schema.md), so removing
	// the creation of the bug leaves nothing that was bound to it.
	stray := create(idAns, journal.TypeQA, "stray", 2)
	stray.Re = idBugForUndo
	state := Build([]journal.Event{create(idBugForUndo, journal.TypeBug, "a bug", 1), stray})
	if got := state.Orphaned(create(idBugForUndo, journal.TypeBug, "a bug", 1)); len(got) != 0 {
		t.Errorf("Orphaned = %v, want none", got)
	}
}

const idBugForUndo = "7f3a2b1c09d84e6fa5b17c2d3e4f5a60"
