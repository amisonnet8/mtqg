package model

import (
	"testing"

	"github.com/amisonnet8/mtqg/internal/journal"
)

// conflictIDs are the IDs of the records that ConcurrentStatusChanges finds.
func conflictIDs(s *State) []string {
	var out []string
	for _, c := range s.ConcurrentStatusChanges() {
		out = append(out, c.Record.ID)
	}
	return out
}

func edit(id, text string, minute int, author journal.Author) journal.Event {
	return journal.Event{ID: id, Op: journal.OpEdit, Text: text, TS: at(minute), Author: author}
}

func withBasis(ev journal.Event, n int) journal.Event {
	ev.Basis = n
	return ev
}

func TestConcurrentStatusChanges(t *testing.T) {
	other := journal.Author{Kind: journal.AuthorHuman, Name: "sato"}
	todo := func(id string) journal.Event { return create(id, journal.TypeTodo, "a todo", 0) }

	tests := []struct {
		name    string
		events  []journal.Event
		want    bool // the record is in conflict
		changes int  // how many changes are listed
	}{
		{
			name:   "one change is not a conflict",
			events: []journal.Event{todo(idTodoA), status(idTodoA, "open", "done", 5, agent)},
		},
		{
			name: "two branches each close the todo",
			events: []journal.Event{
				todo(idTodoA),
				status(idTodoA, "open", "done", 5, agent),
				status(idTodoA, "open", "done", 8, human),
			},
			want: true, changes: 2,
		},
		{
			// The judgment that counts changes with the same from gets this one wrong:
			// there are two changes from open, and one after the other.
			name: "open, done, open, done, each written after seeing the one before",
			events: []journal.Event{
				todo(idTodoA),
				status(idTodoA, "open", "done", 5, agent),
				status(idTodoA, "done", "open", 6, human),
				status(idTodoA, "open", "done", 7, agent),
			},
		},
		{
			name: "a chain, and then a change that had not seen the last one",
			events: []journal.Event{
				todo(idTodoA),
				status(idTodoA, "open", "done", 5, agent),
				status(idTodoA, "done", "open", 6, human),
				status(idTodoA, "open", "done", 7, agent),
				status(idTodoA, "open", "done", 9, other),
			},
			want: true, changes: 4,
		},
		{
			name: "two people reopen it",
			events: []journal.Event{
				todo(idTodoA),
				status(idTodoA, "open", "done", 5, agent),
				status(idTodoA, "done", "open", 6, human),
				status(idTodoA, "done", "open", 7, other),
			},
			want: true, changes: 3,
		},
		{
			name: "a change with no from is not judged",
			events: []journal.Event{
				todo(idTodoA),
				status(idTodoA, "", "done", 5, agent),
				status(idTodoA, "", "done", 6, human),
			},
		},
		{
			name: "changes of the same minute, each from the state it finds",
			events: []journal.Event{
				todo(idTodoA),
				status(idTodoA, "open", "done", 5, agent),
				status(idTodoA, "done", "open", 5, agent),
			},
		},
		{
			name: "a change written by a clock that runs behind is still in order",
			events: []journal.Event{
				create(idTodoA, journal.TypeTodo, "a todo", 10),
				status(idTodoA, "open", "done", 5, agent), // earlier than its create
			},
		},
		{
			name: "a todo that was created done, and closed again",
			events: func() []journal.Event {
				c := todo(idTodoA)
				c.Status = journal.StatusDone
				return []journal.Event{c, status(idTodoA, "open", "done", 5, agent)}
			}(),
			want: true, changes: 1,
		},
		{
			name: "a record that was deleted is not reported",
			events: []journal.Event{
				todo(idTodoA),
				status(idTodoA, "open", "done", 5, agent),
				status(idTodoA, "open", "done", 8, human),
				del(idTodoA, 9),
			},
		},
	}
	for _, tt := range tests {
		state := Build(tt.events)
		got := state.ConcurrentStatusChanges()
		if (len(got) == 1) != tt.want || len(got) > 1 {
			t.Errorf("%s: %d records in conflict (%v), want conflict: %v", tt.name, len(got), conflictIDs(state), tt.want)
			continue
		}
		if tt.want && len(got[0].Changes) != tt.changes {
			t.Errorf("%s: %d changes listed, want %d", tt.name, len(got[0].Changes), tt.changes)
		}
	}
}

// TestBasis covers what from cannot: basis judges a status change or an edit by
// how many events the record actually had, so a change that raced with a change
// to a different field is caught too, and old data written before basis existed
// is not judged by it.
func TestBasis(t *testing.T) {
	tests := []struct {
		name    string
		events  []journal.Event
		want    bool
		changes int
	}{
		{
			name: "a status change and an edit made at once, from the same base",
			events: []journal.Event{
				create(idTodoA, journal.TypeTodo, "a todo", 0), // 1 event so far: the create
				withBasis(status(idTodoA, "open", "done", 5, agent), 1),
				withBasis(edit(idTodoA, "edited text", 6, human), 1),
			},
			want: true, changes: 2,
		},
		{
			// An edit is applied first (basis correctly says 1, no conflict from it),
			// then a status change whose basis (1) does not match how many events the
			// record actually had by then (2): it did not know of the edit, even though
			// its from ("open") still matches (the edit did not touch status, so from
			// alone would miss this).
			name: "the status change is the one that raced, and its from still agrees",
			events: []journal.Event{
				create(idTodoA, journal.TypeTodo, "a todo", 0),
				withBasis(edit(idTodoA, "edited text", 5, human), 1),
				withBasis(status(idTodoA, "open", "done", 6, agent), 1),
			},
			want: true, changes: 2,
		},
		{
			name: "two edits racing from the same base",
			events: []journal.Event{
				create(idTodoA, journal.TypeTodo, "a todo", 0),
				withBasis(edit(idTodoA, "a", 5, agent), 1),
				withBasis(edit(idTodoA, "b", 6, human), 1),
			},
			want: true, changes: 2,
		},
		{
			name: "a sequential edit, each seeing the one before, is not a conflict",
			events: []journal.Event{
				create(idTodoA, journal.TypeTodo, "a todo", 0),
				withBasis(edit(idTodoA, "a", 5, agent), 1),
				withBasis(edit(idTodoA, "b", 6, human), 2),
			},
		},
		{
			name: "an edit with no basis is not judged (data written before the field existed)",
			events: []journal.Event{
				create(idTodoA, journal.TypeTodo, "a todo", 0),
				edit(idTodoA, "a", 5, agent),
				edit(idTodoA, "b", 6, human),
			},
		},
		{
			name: "a memo's edits can conflict too: basis does not require HasState",
			events: []journal.Event{
				create(idMemo, journal.TypeMemo, "a memo", 0),
				withBasis(edit(idMemo, "a", 5, agent), 1),
				withBasis(edit(idMemo, "b", 6, human), 1),
			},
			want: true, changes: 2,
		},
	}
	for _, tt := range tests {
		state := Build(tt.events)
		got := state.ConcurrentStatusChanges()
		if (len(got) == 1) != tt.want || len(got) > 1 {
			t.Errorf("%s: %d records in conflict, want conflict: %v", tt.name, len(got), tt.want)
			continue
		}
		if tt.want && len(got[0].Changes) != tt.changes {
			t.Errorf("%s: %d changes listed, want %d", tt.name, len(got[0].Changes), tt.changes)
		}
	}
}

func TestConcurrentStatusChangesAreListedInOrderWithTheirTimes(t *testing.T) {
	state := Build([]journal.Event{
		create(idTodoA, journal.TypeTodo, "a todo", 0),
		status(idTodoA, "open", "done", 8, human), // written last in the file
		status(idTodoA, "open", "done", 5, agent),
	})
	got := state.ConcurrentStatusChanges()
	if len(got) != 1 || got[0].Record.ID != idTodoA || len(got[0].Changes) != 2 {
		t.Fatalf("got %+v", got)
	}
	first, second := got[0].Changes[0], got[0].Changes[1]
	if first.Event.Author != agent || second.Event.Author != human || !first.At.Before(second.At) {
		t.Errorf("changes = %+v, %+v: want the earlier one first", first, second)
	}
}

func TestQuestionsAndBugsCanHaveConcurrentChangesToo(t *testing.T) {
	state := Build([]journal.Event{
		create(idQ, journal.TypeQA, "a question", 0),
		status(idQ, "open", "done", 5, agent),
		status(idQ, "open", "done", 6, human),
		create(idBugForUndo, journal.TypeBug, "a bug", 1),
		status(idBugForUndo, "open", "done", 5, agent),
		// Memos and glossary entries have no state: a status line about one is not a
		// change of anything, and not a conflict.
		create(idMemo, journal.TypeMemo, "a memo", 2),
		status(idMemo, "open", "done", 5, agent),
		status(idMemo, "open", "done", 6, human),
	})
	got := conflictIDs(state)
	if len(got) != 1 || got[0] != idQ {
		t.Errorf("records in conflict = %v, want only the question", got)
	}
	if n := state.Summary().ConcurrentStatusChanges; n != 1 {
		t.Errorf("Summary.ConcurrentStatusChanges = %d, want 1", n)
	}
	if d := state.Context(0); len(d.Conflicts) != 1 || d.Conflicts[0].ID != idQ {
		t.Errorf("Context.Conflicts = %v", d.Conflicts)
	}
}

func TestUnattachedReplies(t *testing.T) {
	stray := func(id, typ, re string, minute int) journal.Event {
		ev := create(id, typ, "stray "+id[:4], minute)
		ev.Re, ev.Status = re, ""
		return ev
	}
	const (
		idGone = "deadbeefdeadbeefdeadbeefdeadbeef" // in no line of the journal
		idS1   = "a1000000000000000000000000000001"
		idS2   = "a2000000000000000000000000000002"
		idS3   = "a3000000000000000000000000000003"
		idS4   = "a4000000000000000000000000000004"
		idS5   = "a5000000000000000000000000000005"
		idS6   = "a6000000000000000000000000000006"
	)
	state := Build([]journal.Event{
		create(idQ, journal.TypeQA, "a question", 0),
		answer(idAns, idQ, "an answer that belongs to it", 1, human),
		create(idBugForUndo, journal.TypeBug, "a bug", 2),
		create(idTodoA, journal.TypeTodo, "a todo", 3),
		stray(idS1, journal.TypeQA, idGone, 4),       // to a record that is not in the journal
		stray(idS2, journal.TypeBug, idQ, 5),         // a reply whose re names a question
		stray(idS3, journal.TypeQA, idTodoA, 6),      // an answer to a todo
		stray(idS4, journal.TypeQA, idAns, 7),        // an answer to an answer
		stray(idS5, journal.TypeQA, idBugForUndo, 8), // an answer to a bug
		stray(idS6, journal.TypeQA, idGone, 9),       // deleted below
		del(idS6, 10),
		create(idQ2, journal.TypeQA, "a deleted question", 11),
		answer(idAns2, idQ2, "its answer", 12, human),
		del(idQ2, 13),
	})
	sameIDs(t, "UnattachedReplies", state.UnattachedReplies(), idS1, idS2, idS3, idS4, idS5)
	// None of them is shown under a question or a bug.
	for _, p := range []string{idQ, idBugForUndo, idTodoA, idAns} {
		for _, r := range state.Replies(p) {
			if r.ID != idAns {
				t.Errorf("%s is a reply of %s", r.ID, p[:4])
			}
		}
	}
}
