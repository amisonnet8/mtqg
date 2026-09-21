package model

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/amisonnet8/mtqg/internal/journal"
)

const (
	idTodoA = "6b0d549b6f03475a8600a35a099950d8"
	idTodoB = "6b0d549bffffffffffffffffffffffff" // shares the first 8 digits with idTodoA
	idMemo  = "1012f037b64c44228c38fb2918f135d2"
	idQ     = "95e761d177314f10b06bf2efc6f87718"
	idAns   = "a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1"
	idWord  = "f28c105d1fb14c2390c192cfd3ac94af"
)

var (
	human = journal.Author{Kind: journal.AuthorHuman, Name: "yamada"}
	agent = journal.Author{Kind: journal.AuthorAI, Name: "claude-code"}
)

// at returns a ts for a minute of 2026-09-17, so that tests read in order.
func at(minute int) string {
	return time.Date(2026, 9, 17, 10, minute, 0, 0, time.UTC).Format(time.RFC3339)
}

func create(id, typ, text string, minute int) journal.Event {
	ev := journal.Event{ID: id, Op: journal.OpCreate, Type: typ, Text: text, TS: at(minute), Author: human}
	if typ == journal.TypeTodo || typ == journal.TypeQA {
		ev.Status = journal.StatusOpen
	}
	return ev
}

func status(id, from, to string, minute int, author journal.Author) journal.Event {
	return journal.Event{ID: id, Op: journal.OpStatus, From: from, Status: to, TS: at(minute), Author: author}
}

func TestBuildCreatesRecords(t *testing.T) {
	answer := create(idAns, journal.TypeQA, "Not in the first version", 3)
	answer.Re = idQ
	answer.Status = ""
	word := create(idWord, journal.TypeGlossary, "The smallest unit produced by lexing", 4)
	word.Word = "token"

	state := Build([]journal.Event{
		create(idTodoA, journal.TypeTodo, "Skip block comments", 0),
		create(idMemo, journal.TypeMemo, "Use English for errors", 1),
		create(idQ, journal.TypeQA, "Nested block comments?", 2),
		answer,
		word,
	})

	tests := []struct {
		id        string
		kind      string
		status    string
		hasState  bool
		wantText  string
		wantWord  string
		wantRe    string
		wantAuthr journal.Author
	}{
		{idTodoA, KindTodo, journal.StatusOpen, true, "Skip block comments", "", "", human},
		{idMemo, KindMemo, "", false, "Use English for errors", "", "", human},
		{idQ, KindQuestion, journal.StatusOpen, true, "Nested block comments?", "", "", human},
		{idAns, KindAnswer, "", false, "Not in the first version", "", idQ, human},
		{idWord, KindGlossary, "", false, "The smallest unit produced by lexing", "token", "", human},
	}
	for _, tt := range tests {
		rec := state.Record(tt.id)
		if rec == nil {
			t.Fatalf("no record %s", tt.id)
		}
		if rec.Kind() != tt.kind || rec.Status != tt.status || rec.HasState() != tt.hasState ||
			rec.Text != tt.wantText || rec.Word != tt.wantWord || rec.Re != tt.wantRe || rec.Author != tt.wantAuthr {
			t.Errorf("%s: got %+v", tt.kind, rec)
		}
	}
}

func TestBuildAppliesChangesInOrder(t *testing.T) {
	events := []journal.Event{
		create(idTodoA, journal.TypeTodo, "Skip block comments", 0),
		status(idTodoA, "open", "done", 5, human),
		status(idTodoA, "done", "open", 6, agent),
		{ID: idTodoA, Op: journal.OpEdit, Text: "Skip block comments /* */", TS: at(7), Author: human},
	}
	rec := Build(events).Record(idTodoA)
	if rec.Status != journal.StatusOpen || rec.Text != "Skip block comments /* */" {
		t.Errorf("got status %q text %q", rec.Status, rec.Text)
	}
	if len(rec.Events) != 4 {
		t.Errorf("got %d events, want 4", len(rec.Events))
	}
	if !rec.Created.Equal(time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC)) || !rec.Updated.Equal(time.Date(2026, 9, 17, 10, 7, 0, 0, time.UTC)) {
		t.Errorf("created %s, updated %s", rec.Created, rec.Updated)
	}
}

// The order of lines in the file means nothing: a merge does not keep it. Every
// order of the same events must give the same state.
func TestBuildDoesNotDependOnTheOrderOfEvents(t *testing.T) {
	events := []journal.Event{
		create(idTodoA, journal.TypeTodo, "first", 0),
		create(idMemo, journal.TypeMemo, "a memo", 1),
		status(idTodoA, "open", "done", 2, human),
		{ID: idMemo, Op: journal.OpEdit, Text: "a memo, edited", TS: at(3), Author: human},
		status(idTodoA, "done", "open", 4, agent),
		create(idTodoB, journal.TypeTodo, "second", 5),
	}
	want := describe(Build(events))

	permutations(len(events), func(order []int) {
		shuffled := make([]journal.Event, len(events))
		for i, from := range order {
			shuffled[i] = events[from]
		}
		if got := describe(Build(shuffled)); got != want {
			t.Fatalf("order %v gave\n%s\nwant\n%s", order, got, want)
		}
	})
}

func describe(s *State) string {
	var b strings.Builder
	for _, r := range s.records {
		fmt.Fprintf(&b, "%s|%s|%s|%s|%v|%d events\n", r.ID, r.Kind(), r.Status, r.Text, r.Deleted, len(r.Events))
	}
	return b.String()
}

// permutations calls f with every order of 0..n-1.
func permutations(n int, f func([]int)) {
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	var rec func(k int)
	rec = func(k int) {
		if k == n {
			f(order)
			return
		}
		for i := k; i < n; i++ {
			order[k], order[i] = order[i], order[k]
			rec(k + 1)
			order[k], order[i] = order[i], order[k]
		}
	}
	rec(0)
}

func TestBuildToleratesWhatIsOddInAJournal(t *testing.T) {
	t.Run("an event that is repeated changes nothing the second time", func(t *testing.T) {
		done := status(idTodoA, "open", "done", 5, human)
		once := Build([]journal.Event{create(idTodoA, journal.TypeTodo, "x", 0), done})
		twice := Build([]journal.Event{create(idTodoA, journal.TypeTodo, "x", 0), create(idTodoA, journal.TypeTodo, "x", 0), done, done})
		if once.Record(idTodoA).Status != journal.StatusDone || twice.Record(idTodoA).Status != journal.StatusDone {
			t.Error("the todo should be done either way")
		}
		if len(twice.Todos(true)) != 1 {
			t.Errorf("got %d todos, want 1", len(twice.Todos(true)))
		}
	})

	t.Run("an event for a record that was never created is ignored", func(t *testing.T) {
		state := Build([]journal.Event{
			create(idMemo, journal.TypeMemo, "m", 0),
			status(idTodoA, "open", "done", 1, human),
			{ID: idTodoB, Op: journal.OpDelete, TS: at(2), Author: human},
		})
		if len(state.records) != 1 || state.Record(idTodoA) != nil {
			t.Errorf("got %d records", len(state.records))
		}
	})

	t.Run("a create at the same time as its changes comes first", func(t *testing.T) {
		// Created and done within one second: the same ts.
		done := status(idTodoA, "open", "done", 0, human)
		state := Build([]journal.Event{done, create(idTodoA, journal.TypeTodo, "x", 0)})
		if rec := state.Record(idTodoA); rec == nil || rec.Status != journal.StatusDone {
			t.Errorf("got %+v, want a done todo", rec)
		}
	})

	t.Run("a ts that cannot be read does not stop anything", func(t *testing.T) {
		broken := create(idMemo, journal.TypeMemo, "broken ts", 0)
		broken.TS = "yesterday"
		state := Build([]journal.Event{create(idTodoA, journal.TypeTodo, "x", 1), broken})
		if len(state.records) != 2 {
			t.Fatalf("got %d records", len(state.records))
		}
		if !state.Record(idMemo).Created.IsZero() {
			t.Error("the time of an unreadable ts should be the zero time")
		}
	})

	t.Run("a status change on a record without a state is not applied", func(t *testing.T) {
		state := Build([]journal.Event{
			create(idMemo, journal.TypeMemo, "m", 0),
			status(idMemo, "open", "done", 1, human),
		})
		if got := state.Record(idMemo).Status; got != "" {
			t.Errorf("a memo got the status %q", got)
		}
	})

	t.Run("a todo without a status starts open", func(t *testing.T) {
		ev := create(idTodoA, journal.TypeTodo, "x", 0)
		ev.Status = ""
		if got := Build([]journal.Event{ev}).Record(idTodoA).Status; got != journal.StatusOpen {
			t.Errorf("status = %q", got)
		}
	})
}

func TestDeleteHidesARecord(t *testing.T) {
	state := Build([]journal.Event{
		create(idTodoA, journal.TypeTodo, "gone", 0),
		create(idTodoB, journal.TypeTodo, "stays", 1),
		create(idMemo, journal.TypeMemo, "gone too", 2),
		{ID: idTodoA, Op: journal.OpDelete, TS: at(3), Author: human},
		{ID: idMemo, Op: journal.OpDelete, TS: at(4), Author: human},
	})
	if got := state.Todos(true); len(got) != 1 || got[0].ID != idTodoB {
		t.Errorf("todos = %+v", got)
	}
	if got := state.Memos(); len(got) != 0 {
		t.Errorf("memos = %+v", got)
	}
	if state.Summary().OpenTodos != 1 {
		t.Errorf("open todos = %d", state.Summary().OpenTodos)
	}
	if state.Record(idTodoA) == nil || !state.Record(idTodoA).Deleted {
		t.Error("the record stays known, marked deleted")
	}
}

func TestTodosAndMemosAreOldestFirst(t *testing.T) {
	state := Build([]journal.Event{
		create(idTodoB, journal.TypeTodo, "second", 5),
		create(idTodoA, journal.TypeTodo, "first", 1),
		status(idTodoA, "open", "done", 6, human),
		create(idMemo, journal.TypeMemo, "m", 2),
	})
	open := state.Todos(false)
	if len(open) != 1 || open[0].ID != idTodoB {
		t.Errorf("open = %+v", open)
	}
	all := state.Todos(true)
	if len(all) != 2 || all[0].ID != idTodoA || all[1].ID != idTodoB {
		t.Errorf("all = %+v", all)
	}
}

func TestResolve(t *testing.T) {
	state := Build([]journal.Event{
		create(idTodoA, journal.TypeTodo, "a", 0),
		create(idTodoB, journal.TypeTodo, "b", 1),
		create(idMemo, journal.TypeMemo, "m", 2),
	})

	t.Run("a unique prefix", func(t *testing.T) {
		for _, prefix := range []string{"1012", "1", idMemo, "1012F037", "  1012  "} {
			rec, err := state.Resolve(prefix)
			if err != nil || rec.ID != idMemo {
				t.Errorf("Resolve(%q) = %v, %v", prefix, rec, err)
			}
		}
	})

	t.Run("a prefix that matches two records", func(t *testing.T) {
		_, err := state.Resolve("6b0d549b")
		var amb *AmbiguousError
		if !errors.As(err, &amb) || !errors.Is(err, ErrAmbiguous) {
			t.Fatalf("err = %v, want an AmbiguousError", err)
		}
		if len(amb.Candidates) != 2 || amb.Candidates[0].ID != idTodoA || amb.Candidates[1].ID != idTodoB {
			t.Errorf("candidates = %+v", amb.Candidates)
		}
		// The full IDs tell them apart.
		if rec, err := state.Resolve(idTodoA); err != nil || rec.ID != idTodoA {
			t.Errorf("the full ID must resolve: %v, %v", rec, err)
		}
	})

	t.Run("nothing matches", func(t *testing.T) {
		for _, prefix := range []string{"ffff", "", "  ", "xyz", "6b0d-549"} {
			if _, err := state.Resolve(prefix); !errors.Is(err, ErrNotFound) {
				t.Errorf("Resolve(%q): err = %v, want ErrNotFound", prefix, err)
			}
		}
	})

	t.Run("a deleted record is not matched and does not make another ambiguous", func(t *testing.T) {
		withDeleted := Build([]journal.Event{
			create(idTodoA, journal.TypeTodo, "a", 0),
			create(idTodoB, journal.TypeTodo, "b", 1),
			{ID: idTodoB, Op: journal.OpDelete, TS: at(2), Author: human},
		})
		if rec, err := withDeleted.Resolve("6b0d549b"); err != nil || rec.ID != idTodoA {
			t.Errorf("got %v, %v; want the record that is left", rec, err)
		}
		if _, err := withDeleted.Resolve(idTodoB); !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound for the deleted one", err)
		}
	})
}

func TestResolveKind(t *testing.T) {
	answer := create(idAns, journal.TypeQA, "an answer", 3)
	answer.Re, answer.Status = idQ, ""
	state := Build([]journal.Event{
		create(idTodoA, journal.TypeTodo, "t", 0),
		create(idMemo, journal.TypeMemo, "m", 1),
		create(idQ, journal.TypeQA, "q?", 2),
		answer,
	})

	if rec, err := state.ResolveKind("6b0d", KindTodo); err != nil || rec.ID != idTodoA {
		t.Errorf("got %v, %v", rec, err)
	}
	for _, tt := range []struct{ prefix, want, isA string }{
		{"1012", KindTodo, KindMemo},
		{"95e7", KindTodo, KindQuestion},
		{"a1a1", KindQuestion, KindAnswer},
	} {
		_, err := state.ResolveKind(tt.prefix, tt.want)
		var wrong *WrongKindError
		if !errors.As(err, &wrong) || !errors.Is(err, ErrWrongKind) || wrong.Record.Kind() != tt.isA || wrong.Want != tt.want {
			t.Errorf("ResolveKind(%s, %s) = %v, want a WrongKindError for a %s", tt.prefix, tt.want, err, tt.isA)
		}
	}
	// The errors of Resolve pass through.
	if _, err := state.ResolveKind("ffff", KindTodo); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v", err)
	}
}

func TestCreateEvents(t *testing.T) {
	memo, err := MemoCreate("Use English for errors")
	if err != nil || memo.Op != journal.OpCreate || memo.Type != journal.TypeMemo || memo.Text != "Use English for errors" || memo.Status != "" || memo.ID != "" {
		t.Errorf("memo = %+v, %v", memo, err)
	}
	todo, err := TodoCreate("Skip block comments")
	if err != nil || todo.Type != journal.TypeTodo || todo.Status != journal.StatusOpen || todo.Text != "Skip block comments" {
		t.Errorf("todo = %+v, %v", todo, err)
	}
	for _, text := range []string{"", "   ", "\n\t "} {
		if _, err := MemoCreate(text); !errors.Is(err, ErrEmptyText) {
			t.Errorf("MemoCreate(%q): err = %v", text, err)
		}
		if _, err := TodoCreate(text); !errors.Is(err, ErrEmptyText) {
			t.Errorf("TodoCreate(%q): err = %v", text, err)
		}
	}
}

func TestSetStatus(t *testing.T) {
	state := Build([]journal.Event{
		create(idTodoA, journal.TypeTodo, "t", 0),
		create(idMemo, journal.TypeMemo, "m", 1),
	})
	todo, memo := state.Record(idTodoA), state.Record(idMemo)

	ev, err := SetStatus(todo, journal.StatusDone)
	if err != nil || ev.ID != idTodoA || ev.Op != journal.OpStatus || ev.From != journal.StatusOpen || ev.Status != journal.StatusDone {
		t.Errorf("event = %+v, %v", ev, err)
	}
	if _, err := SetStatus(todo, journal.StatusOpen); !errors.Is(err, ErrNoChange) {
		t.Errorf("err = %v, want ErrNoChange for a todo that is open already", err)
	}
	var noState *NoStateError
	if _, err := SetStatus(memo, journal.StatusDone); !errors.As(err, &noState) || !errors.Is(err, ErrNoState) {
		t.Errorf("err = %v, want a NoStateError for a memo", err)
	}
	if _, err := SetStatus(todo, "closed"); !errors.Is(err, journal.ErrInvalidEvent) {
		t.Errorf("err = %v, want an invalid status to be refused", err)
	}

	// What the writer saw is what is recorded: a done todo reopened.
	doneState := Build([]journal.Event{create(idTodoA, journal.TypeTodo, "t", 0), status(idTodoA, "open", "done", 1, human)})
	if ev, err := SetStatus(doneState.Record(idTodoA), journal.StatusOpen); err != nil || ev.From != journal.StatusDone {
		t.Errorf("event = %+v, %v; want from done", ev, err)
	}
}

func TestUncommittedRecords(t *testing.T) {
	events := []journal.Event{
		create(idTodoA, journal.TypeTodo, "t", 0),
		status(idTodoA, "open", "done", 1, human), // the same record again
		create(idMemo, journal.TypeMemo, "m", 2),
	}
	if got := UncommittedRecords(events); got != 2 {
		t.Errorf("got %d, want 2 records", got)
	}
	if got := UncommittedRecords(nil); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}
