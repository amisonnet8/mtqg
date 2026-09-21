package model

import (
	"errors"
	"testing"

	"github.com/amisonnet8/mtqg/internal/journal"
)

func TestEditText(t *testing.T) {
	state := Build([]journal.Event{
		create(idTodoA, journal.TypeTodo, "Skip block comments", 0),
		entry(idWord, "token", "The smallest unit", 1, human),
	})
	todo, word := state.Record(idTodoA), state.Record(idWord)

	ev, err := EditText(todo, "Skip block and line comments")
	if err != nil || ev.Op != journal.OpEdit || ev.ID != idTodoA || ev.Text != "Skip block and line comments" {
		t.Fatalf("EditText = %+v, %v", ev, err)
	}
	// Only the text goes into the event: not the word, and not the state.
	if ev.Word != "" || ev.Status != "" || ev.From != "" || ev.Type != "" {
		t.Errorf("an edit carries more than the text: %+v", ev)
	}
	if ev, err := EditText(word, "The smallest unit produced by lexing"); err != nil || ev.Word != "" {
		t.Errorf("edit of a glossary entry = %+v, %v: the word must not be in it", ev, err)
	}

	// The same text is no change; a text that only differs by a space is one.
	if _, err := EditText(todo, "Skip block comments"); !errors.Is(err, ErrNoChange) {
		t.Errorf("the same text: err = %v, want ErrNoChange", err)
	}
	if _, err := EditText(todo, "Skip block comments "); err != nil {
		t.Errorf("a text that differs by a space: err = %v", err)
	}
	for _, text := range []string{"", "  ", "\n\t"} {
		if _, err := EditText(todo, text); !errors.Is(err, ErrEmptyText) {
			t.Errorf("EditText(%q): err = %v, want ErrEmptyText", text, err)
		}
	}
}

func TestAnEditReplacesTheTextAndNothingElse(t *testing.T) {
	events := []journal.Event{
		create(idTodoA, journal.TypeTodo, "Skip block comments", 0),
		status(idTodoA, journal.StatusOpen, journal.StatusDone, 1, agent),
	}
	edit, err := EditText(Build(events).Record(idTodoA), "Skip block and line comments")
	if err != nil {
		t.Fatal(err)
	}
	edit.TS, edit.Author = at(2), human

	rec := Build(append(events, edit)).Record(idTodoA)
	if rec.Text != "Skip block and line comments" || rec.Status != journal.StatusDone {
		t.Errorf("after the edit: text %q, status %q", rec.Text, rec.Status)
	}
}

func TestDelete(t *testing.T) {
	events := []journal.Event{
		create(idQ, journal.TypeQA, "Nested block comments?", 0),
		answer(idAns, idQ, "Not in the first version", 1, human),
	}
	state := Build(events)
	ev, err := Delete(state.Record(idQ))
	if err != nil || ev.Op != journal.OpDelete || ev.ID != idQ || ev.Text != "" {
		t.Fatalf("Delete = %+v, %v", ev, err)
	}
	ev.TS, ev.Author = at(2), human

	after := Build(append(events, ev))
	if inView(after, idQ) {
		t.Errorf("the question is still in view")
	}
	if got := after.Replies(idQ); len(got) != 0 {
		t.Errorf("the answers of a deleted question are still in view: %v", ids(got))
	}
}

// inView says whether the record is in view: it is there and not hidden.
func inView(s *State, id string) bool {
	r := s.Record(id)
	return r != nil && s.visible(r)
}

func TestSearch(t *testing.T) {
	entryWord := entry(idWord, "Delimited", "A comment enclosed in /* and */", 3, human)
	events := []journal.Event{
		create(idTodoA, journal.TypeTodo, "Skip block comments", 0),
		create(idMemo, journal.TypeMemo, "Use English for errors\nand a second LINE about Comments", 1),
		create(idQ, journal.TypeQA, "Nested BLOCK comments?", 2),
		entryWord,
		answer(idAns, idQ, "Not in the first version", 4, human),
		create(idQ2, journal.TypeQA, "A question that is deleted: block", 5),
		del(idQ2, 6),
	}
	state := Build(events)

	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{"case is ignored, in both directions", "block", []string{idTodoA, idQ}},
		{"upper case in the query", "BLOCK", []string{idTodoA, idQ}},
		{"a match in a later line of the text", "second line", []string{idMemo}},
		{"the word of a glossary entry, in any case", "DELIMITED", []string{idWord}},
		{"the text of a glossary entry", "enclosed", []string{idWord}},
		{"an answer", "first version", []string{idAns}},
		{"a deleted record is not found", "deleted", nil},
		{"nothing matches", "zebra", nil},
		{"no patterns", "b.ock", nil},
		{"no word boundaries", "ock com", []string{idTodoA, idQ}},
		{"a phrase with punctuation", "/* and */", []string{idWord}},
	}
	for _, tt := range tests {
		sameIDs(t, tt.name, state.Search(tt.query), tt.want...)
	}
}

func TestEventOrder(t *testing.T) {
	// Out of order on purpose, as a merge leaves them: a change before its create in
	// the file, the same second for two IDs, a delete and an edit of the same second.
	events := []journal.Event{
		status(idTodoA, "open", "done", 5, human), // 0
		create(idTodoA, journal.TypeTodo, "a", 1), // 1
		del(idMemo, 3), // 2
		{ID: idMemo, Op: journal.OpEdit, Text: "e", TS: at(3), Author: human}, // 3
		create(idMemo, journal.TypeMemo, "m", 3),                              // 4
		create(idQ, journal.TypeQA, "q", 3),                                   // 5
	}
	got := EventOrder(events)
	// 1 (minute 1); then minute 3 by ID: idMemo (1012...) before idQ (95e7...), each
	// as create, edit, delete; then minute 5.
	want := []int{1, 4, 3, 2, 5, 0}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	// The input is not changed.
	if events[0].Op != journal.OpStatus || events[1].Op != journal.OpCreate {
		t.Errorf("EventOrder changed its argument")
	}
}
