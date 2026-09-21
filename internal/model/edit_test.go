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
