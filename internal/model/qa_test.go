package model

import (
	"errors"
	"testing"

	"github.com/amisonnet8/mtqg/internal/journal"
)

const (
	idQ2    = "2217beaddb1f4b6e9c0d1e2f3a4b5c66"
	idAns2  = "70430f77ff91c2e04a8b33f1d7e6a025"
	idAns3  = "c0ffee00c0ffee00c0ffee00c0ffee00"
	idWord2 = "0cb1e29c65a04b7d8f3e1c2d4b5a6978"
	idWord3 = "7a3c1d9e027b4f6a8c5d0e1f2a3b4c5d"
)

func answer(id, question, text string, minute int, author journal.Author) journal.Event {
	ev := create(id, journal.TypeQA, text, minute)
	ev.Re, ev.Status, ev.Author = question, "", author
	return ev
}

func entry(id, word, text string, minute int, author journal.Author) journal.Event {
	ev := create(id, journal.TypeGlossary, text, minute)
	ev.Word, ev.Author = word, author
	return ev
}

func del(id string, minute int) journal.Event {
	return journal.Event{ID: id, Op: journal.OpDelete, TS: at(minute), Author: human}
}

func ids(records []*Record) []string {
	out := make([]string, len(records))
	for i, r := range records {
		out[i] = r.ID
	}
	return out
}

func sameIDs(t *testing.T, what string, got []*Record, want ...string) {
	t.Helper()
	g := ids(got)
	if len(g) != len(want) {
		t.Errorf("%s: got %v, want %v", what, g, want)
		return
	}
	for i := range g {
		if g[i] != want[i] {
			t.Errorf("%s: got %v, want %v", what, g, want)
			return
		}
	}
}

func TestQuestionsAndAnswers(t *testing.T) {
	state := Build([]journal.Event{
		create(idQ, journal.TypeQA, "Nested block comments?", 0),
		create(idQ2, journal.TypeQA, "Error positions?", 1),
		answer(idAns, idQ, "Supporting them is preferable", 2, agent),
		answer(idAns2, idQ, "Not in the first version", 3, human),
		status(idQ2, "open", "done", 4, human),
		create(idMemo, journal.TypeMemo, "m", 5),
	})

	sameIDs(t, "open questions", state.Questions(false), idQ)
	sameIDs(t, "all questions", state.Questions(true), idQ, idQ2)
	sameIDs(t, "answers, oldest first", state.Answers(idQ), idAns, idAns2)
	sameIDs(t, "answers of a question without any", state.Answers(idQ2))
	sameIDs(t, "answers of a memo", state.Answers(idMemo))

	// Answers are not questions, and a question is not among the todos.
	for _, r := range state.Questions(true) {
		if r.Kind() != KindQuestion {
			t.Errorf("%s is a %s", r.ID, r.Kind())
		}
	}
	sameIDs(t, "todos", state.Todos(true))

	if !state.HasQuestion(state.Record(idAns)) {
		t.Error("an answer of a question in the journal has its question")
	}
}

func TestAnswerWhoseQuestionIsNotInTheJournal(t *testing.T) {
	// The question may have been archived. What is missing is not deleted: the
	// answer is in view, and it is for the lists to leave it out of the questions.
	state := Build([]journal.Event{answer(idAns, idQ, "an answer", 0, human)})
	rec := state.Record(idAns)
	if rec == nil || rec.Kind() != KindAnswer {
		t.Fatalf("record = %+v", rec)
	}
	if state.HasQuestion(rec) {
		t.Error("HasQuestion should be false")
	}
	sameIDs(t, "All", state.All(), idAns)
	if got, err := state.Resolve("a1a1"); err != nil || got.ID != idAns {
		t.Errorf("the answer should resolve: %v, %v", got, err)
	}
}

func TestDeletingAQuestionHidesItsAnswers(t *testing.T) {
	state := Build([]journal.Event{
		create(idQ, journal.TypeQA, "Nested block comments?", 0),
		create(idQ2, journal.TypeQA, "Error positions?", 1),
		answer(idAns, idQ, "one", 2, agent),
		answer(idAns2, idQ, "two", 3, human),
		answer(idAns3, idQ2, "three", 4, human),
		del(idQ, 5),
	})
	sameIDs(t, "questions", state.Questions(true), idQ2)
	sameIDs(t, "answers of the deleted question", state.Answers(idQ))
	sameIDs(t, "answers of the other question", state.Answers(idQ2), idAns3)
	sameIDs(t, "All", state.All(), idQ2, idAns3)

	// They can no longer be found by ID, and do not make another ID ambiguous.
	for _, id := range []string{idQ, idAns, idAns2} {
		if _, err := state.Resolve(id); !errors.Is(err, ErrNotFound) {
			t.Errorf("Resolve(%s): err = %v, want ErrNotFound", id, err)
		}
	}
	// The events are still there for whoever asks for the record itself.
	if rec := state.Record(idAns); rec == nil || rec.Deleted {
		t.Errorf("the answer itself is not deleted, only hidden: %+v", rec)
	}

	// Deleting one answer hides that answer only.
	state = Build([]journal.Event{
		create(idQ, journal.TypeQA, "q", 0),
		answer(idAns, idQ, "one", 1, agent),
		answer(idAns2, idQ, "two", 2, human),
		del(idAns, 3),
	})
	sameIDs(t, "questions", state.Questions(true), idQ)
	sameIDs(t, "answers", state.Answers(idQ), idAns2)
}

func TestAllIsEveryRecordInViewOldestFirst(t *testing.T) {
	state := Build([]journal.Event{
		create(idTodoA, journal.TypeTodo, "t", 0),
		create(idMemo, journal.TypeMemo, "m", 1),
		create(idQ, journal.TypeQA, "q", 2),
		answer(idAns, idQ, "a", 3, human),
		entry(idWord, "token", "d", 4, human),
		create(idTodoB, journal.TypeTodo, "gone", 5),
		del(idTodoB, 6),
	})
	sameIDs(t, "All", state.All(), idTodoA, idMemo, idQ, idAns, idWord)
}

func TestGlossaryAndDuplicateWords(t *testing.T) {
	state := Build([]journal.Event{
		entry(idWord, "token", "The smallest unit produced by lexing", 0, human),
		entry(idWord2, "block comment", "A comment enclosed in /* and */", 1, human),
		entry(idWord3, "block comment", "A comment that can span lines", 2, agent),
		entry("d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1", "Token", "Not the same word: the case differs", 3, human),
		entry("e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2", "block comment", "A third definition", 4, human),
		create(idMemo, journal.TypeMemo, "m", 5),
	})

	if got := len(state.Glossary()); got != 5 {
		t.Errorf("the glossary has %d entries, want 5", got)
	}
	groups := state.DuplicateWords()
	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1 (Token and token are two words)", len(groups))
	}
	sameIDs(t, "the entries of the word, in the order written", groups[0],
		idWord2, idWord3, "e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2")
	for _, r := range groups[0] {
		if r.Word != "block comment" {
			t.Errorf("%s has the word %q", r.ID, r.Word)
		}
	}
	// Nothing is chosen: every entry is still in the glossary, with its own text.
	if state.Record(idWord2).Text == state.Record(idWord3).Text {
		t.Error("the definitions should be kept apart")
	}
}

func TestNoDuplicatesWithoutTwoOfAWord(t *testing.T) {
	state := Build([]journal.Event{
		entry(idWord, "token", "d", 0, human),
		entry(idWord2, "lexer", "d", 1, human),
	})
	if groups := state.DuplicateWords(); len(groups) != 0 {
		t.Errorf("got %d groups", len(groups))
	}
	// A deleted entry no longer counts as a definition.
	state = Build([]journal.Event{
		entry(idWord, "token", "d", 0, human),
		entry(idWord2, "token", "d2", 1, human),
		del(idWord2, 2),
	})
	if groups := state.DuplicateWords(); len(groups) != 0 {
		t.Errorf("a deleted entry still counted: %d groups", len(groups))
	}
}

func TestSummary(t *testing.T) {
	state := Build([]journal.Event{
		create(idTodoA, journal.TypeTodo, "t", 0),
		create(idTodoB, journal.TypeTodo, "t2", 1),
		status(idTodoB, "open", "done", 2, human),
		create(idQ, journal.TypeQA, "answered but not closed", 3),
		answer(idAns, idQ, "an answer", 4, agent),
		create(idQ2, journal.TypeQA, "unanswered", 5),
		create("b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3", journal.TypeQA, "closed", 6),
		status("b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3", "open", "done", 7, human),
		entry(idWord, "token", "d", 8, human),
		entry(idWord2, "token", "d2", 9, agent),
		entry(idWord3, "lexer", "d3", 10, human),
	})
	want := Summary{OpenTodos: 1, OpenQuestions: 2, AwaitingConfirmation: 1, GlossaryEntries: 3, DuplicateWords: 1}
	if got := state.Summary(); got != want {
		t.Errorf("Summary() = %+v, want %+v", got, want)
	}

	// When the only answer is deleted, the question is not awaiting confirmation.
	state = Build([]journal.Event{
		create(idQ, journal.TypeQA, "q", 0),
		answer(idAns, idQ, "a", 1, agent),
		del(idAns, 2),
	})
	if got := state.Summary(); got.OpenQuestions != 1 || got.AwaitingConfirmation != 0 {
		t.Errorf("Summary() = %+v", got)
	}
	// A question that is deleted is not open, and its answers are not waited for.
	state = Build([]journal.Event{
		create(idQ, journal.TypeQA, "q", 0),
		answer(idAns, idQ, "a", 1, agent),
		del(idQ, 2),
	})
	if got := state.Summary(); got.OpenQuestions != 0 || got.AwaitingConfirmation != 0 {
		t.Errorf("Summary() = %+v", got)
	}
}

func TestQuestionAnswerAndGlossaryEvents(t *testing.T) {
	q, err := QuestionCreate("Nested block comments?")
	if err != nil || q.Op != journal.OpCreate || q.Type != journal.TypeQA || q.Status != journal.StatusOpen || q.Re != "" || q.ID != "" {
		t.Errorf("question = %+v, %v", q, err)
	}

	state := Build([]journal.Event{
		create(idQ, journal.TypeQA, "q", 0),
		create(idMemo, journal.TypeMemo, "m", 1),
		answer(idAns, idQ, "a", 2, human),
	})
	a, err := AnswerCreate(state.Record(idQ), "Not in the first version")
	// The re is the full ID of the question, whatever was typed to find it.
	if err != nil || a.Op != journal.OpCreate || a.Type != journal.TypeQA || a.Re != idQ || a.Status != "" || a.Text != "Not in the first version" {
		t.Errorf("answer = %+v, %v", a, err)
	}
	for _, rec := range []*Record{state.Record(idMemo), state.Record(idAns)} {
		var wrong *WrongKindError
		if _, err := AnswerCreate(rec, "x"); !errors.As(err, &wrong) || wrong.Want != KindQuestion {
			t.Errorf("AnswerCreate on a %s: err = %v, want a WrongKindError", rec.Kind(), err)
		}
	}

	g, err := GlossaryCreate("block comment", "A comment enclosed in /* and */")
	if err != nil || g.Type != journal.TypeGlossary || g.Word != "block comment" || g.Text != "A comment enclosed in /* and */" || g.Status != "" {
		t.Errorf("glossary = %+v, %v", g, err)
	}
	// The word is kept as it was given.
	if g, _ := GlossaryCreate(" token ", "d"); g.Word != " token " {
		t.Errorf("the word was changed: %q", g.Word)
	}

	for _, text := range []string{"", "  ", "\n"} {
		if _, err := QuestionCreate(text); !errors.Is(err, ErrEmptyText) {
			t.Errorf("QuestionCreate(%q): err = %v", text, err)
		}
		if _, err := AnswerCreate(state.Record(idQ), text); !errors.Is(err, ErrEmptyText) {
			t.Errorf("AnswerCreate(%q): err = %v", text, err)
		}
		if _, err := GlossaryCreate("token", text); !errors.Is(err, ErrEmptyText) {
			t.Errorf("GlossaryCreate(text %q): err = %v", text, err)
		}
		if _, err := GlossaryCreate(text, "definition"); !errors.Is(err, ErrEmptyWord) {
			t.Errorf("GlossaryCreate(word %q): err = %v", text, err)
		}
	}
}

func TestIsIDLike(t *testing.T) {
	for _, tt := range []struct {
		word string
		want bool
	}{
		{"a8ec", true},
		{"a8ec051c41", true},
		{"A8EC", true}, // upper case is read as lower case
		{"a8ec051c4116436aba1e7845c66cc956", true},
		{"face", true}, // an English word: whether it names a record is for Resolve
		{"0000", true},
		{"abc", false}, // three digits
		{"a", false},
		{"", false},
		{"a8eg", false},
		{"a8ec-1", false},
		{"a8ec ", false}, // a quoted text with a space is never an ID
		{"a8ec はどういう意味ですか？", false},
		{"日本語", false},
		{"-", false},
	} {
		if got := IsIDLike(tt.word); got != tt.want {
			t.Errorf("IsIDLike(%q) = %v, want %v", tt.word, got, tt.want)
		}
	}
}
