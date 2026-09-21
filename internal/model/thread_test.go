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

// reply is an answer (type qa) or a reply (type bug) to the record with the ID
// parent.
func reply(typ, id, parent, text string, minute int, author journal.Author) journal.Event {
	ev := create(id, typ, text, minute)
	ev.Re, ev.Status, ev.Author = parent, "", author
	return ev
}

func answer(id, question, text string, minute int, author journal.Author) journal.Event {
	return reply(journal.TypeQA, id, question, text, minute, author)
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

	sameIDs(t, "open questions", state.Parents(journal.TypeQA, false), idQ)
	sameIDs(t, "all questions", state.Parents(journal.TypeQA, true), idQ, idQ2)
	sameIDs(t, "answers, oldest first", state.Replies(idQ), idAns, idAns2)
	sameIDs(t, "answers of a question without any", state.Replies(idQ2))
	sameIDs(t, "answers of a memo", state.Replies(idMemo))

	// Answers are not questions, and a question is not among the todos.
	for _, r := range state.Parents(journal.TypeQA, true) {
		if r.Kind() != KindQuestion {
			t.Errorf("%s is a %s", r.ID, r.Kind())
		}
	}
	sameIDs(t, "todos", state.Todos(true))

	if !state.HasParent(state.Record(idAns)) {
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
	if state.HasParent(rec) {
		t.Error("HasParent should be false")
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
	sameIDs(t, "questions", state.Parents(journal.TypeQA, true), idQ2)
	sameIDs(t, "answers of the deleted question", state.Replies(idQ))
	sameIDs(t, "answers of the other question", state.Replies(idQ2), idAns3)
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
	sameIDs(t, "questions", state.Parents(journal.TypeQA, true), idQ)
	sameIDs(t, "answers", state.Replies(idQ), idAns2)
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
	want := Summary{OpenTodos: 1, OpenQuestions: 2, QuestionsAwaitingConfirmation: 1, GlossaryEntries: 3, DuplicateWords: 1}
	if got := state.Summary(); got != want {
		t.Errorf("Summary() = %+v, want %+v", got, want)
	}

	// When the only answer is deleted, the question is not awaiting confirmation.
	state = Build([]journal.Event{
		create(idQ, journal.TypeQA, "q", 0),
		answer(idAns, idQ, "a", 1, agent),
		del(idAns, 2),
	})
	if got := state.Summary(); got.OpenQuestions != 1 || got.QuestionsAwaitingConfirmation != 0 {
		t.Errorf("Summary() = %+v", got)
	}
	// A question that is deleted is not open, and its answers are not waited for.
	state = Build([]journal.Event{
		create(idQ, journal.TypeQA, "q", 0),
		answer(idAns, idQ, "a", 1, agent),
		del(idQ, 2),
	})
	if got := state.Summary(); got.OpenQuestions != 0 || got.QuestionsAwaitingConfirmation != 0 {
		t.Errorf("Summary() = %+v", got)
	}
}

func TestQuestionAnswerAndGlossaryEvents(t *testing.T) {
	q, err := ParentCreate(journal.TypeQA, "Nested block comments?")
	if err != nil || q.Op != journal.OpCreate || q.Type != journal.TypeQA || q.Status != journal.StatusOpen || q.Re != "" || q.ID != "" {
		t.Errorf("question = %+v, %v", q, err)
	}

	state := Build([]journal.Event{
		create(idQ, journal.TypeQA, "q", 0),
		create(idMemo, journal.TypeMemo, "m", 1),
		answer(idAns, idQ, "a", 2, human),
	})
	a, err := ReplyCreate(state.Record(idQ), "Not in the first version")
	// The re is the full ID of the question, whatever was typed to find it.
	if err != nil || a.Op != journal.OpCreate || a.Type != journal.TypeQA || a.Re != idQ || a.Status != "" || a.Text != "Not in the first version" {
		t.Errorf("answer = %+v, %v", a, err)
	}
	for _, rec := range []*Record{state.Record(idMemo), state.Record(idAns)} {
		var none *NoRepliesError
		if _, err := ReplyCreate(rec, "x"); !errors.As(err, &none) || !errors.Is(err, ErrNoReplies) {
			t.Errorf("ReplyCreate on a %s: err = %v, want a NoRepliesError", rec.Kind(), err)
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
		if _, err := ParentCreate(journal.TypeQA, text); !errors.Is(err, ErrEmptyText) {
			t.Errorf("ParentCreate(qa, %q): err = %v", text, err)
		}
		if _, err := ReplyCreate(state.Record(idQ), text); !errors.Is(err, ErrEmptyText) {
			t.Errorf("ReplyCreate(%q): err = %v", text, err)
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

func TestHistory(t *testing.T) {
	state := Build([]journal.Event{
		create(idQ, journal.TypeQA, "Nested block comments?", 0),
		answer(idAns, idQ, "one", 2, agent),
		status(idQ, "open", "done", 4, human),
		answer(idAns2, idQ, "two", 3, human),
		{ID: idQ, Op: journal.OpEdit, Text: "Nested block comments, please?", TS: at(5), Author: human},
		create(idMemo, journal.TypeMemo, "m", 6),
	})

	// A question: its own events and the creation of its answers, by time.
	var got []string
	for _, e := range state.History(state.Record(idQ)) {
		what := e.Event.Op
		if e.Answer != nil {
			what += " " + e.Answer.ID[:4]
		}
		got = append(got, what)
	}
	want := []string{"create", "create a1a1", "create 7043", "status", "edit"}
	if len(got) != len(want) {
		t.Fatalf("history = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("history = %v, want %v", got, want)
		}
	}
	for _, e := range state.History(state.Record(idQ)) {
		if e.At.IsZero() {
			t.Errorf("an entry has no time: %+v", e)
		}
	}

	// An answer or a memo has only its own events.
	if h := state.History(state.Record(idAns)); len(h) != 1 || h[0].Answer != nil {
		t.Errorf("history of an answer = %+v", h)
	}
	if h := state.History(state.Record(idMemo)); len(h) != 1 {
		t.Errorf("history of a memo = %+v", h)
	}

	// An answer that was deleted is not part of the question's history.
	deleted := Build([]journal.Event{
		create(idQ, journal.TypeQA, "q", 0),
		answer(idAns, idQ, "one", 1, agent),
		del(idAns, 2),
	})
	if h := deleted.History(deleted.Record(idQ)); len(h) != 1 {
		t.Errorf("history = %+v, want the question's create only", h)
	}
}

// Bugs and their replies have the shape of questions and their answers. What
// follows is what differs, or what must not be mixed up.

const (
	idBug  = "7f3a2b1c09d84e6fa5b17c2d3e4f5a60"
	idBug2 = "8e4b3c2d10e95f70b6c28d3e4f5a6b71"
	idRep  = "3d8e4a0b12c94f77b6a08d1e5f2c9b34"
	idRep2 = "4c9d5b1e23fa4a88c7b19e2f6a3d0c45"
)

func replyTo(id, bug, text string, minute int, author journal.Author) journal.Event {
	return reply(journal.TypeBug, id, bug, text, minute, author)
}

func TestKindsOfTheTypesThatHaveReplies(t *testing.T) {
	for _, tt := range []struct {
		typ, parent, reply string
	}{
		{journal.TypeQA, KindQuestion, KindAnswer},
		{journal.TypeBug, KindBug, KindReply},
		// The types that have no replies are their own kind.
		{journal.TypeMemo, KindMemo, KindMemo},
		{journal.TypeTodo, KindTodo, KindTodo},
		{journal.TypeGlossary, KindGlossary, KindGlossary},
	} {
		if got := ParentKind(tt.typ); got != tt.parent {
			t.Errorf("ParentKind(%s) = %s, want %s", tt.typ, got, tt.parent)
		}
		if got := ReplyKind(tt.typ); got != tt.reply {
			t.Errorf("ReplyKind(%s) = %s, want %s", tt.typ, got, tt.reply)
		}
	}

	state := Build([]journal.Event{
		create(idBug, journal.TypeBug, "Parser crashes on empty input", 0),
		replyTo(idRep, idBug, "Reproduced on macOS too", 1, agent),
		create(idQ, journal.TypeQA, "Nested block comments?", 2),
		answer(idAns, idQ, "Not in the first version", 3, human),
		create(idMemo, journal.TypeMemo, "m", 4),
	})
	for _, tt := range []struct {
		id                     string
		kind                   string
		hasState, canBeReplied bool
		isReply                bool
	}{
		{idBug, KindBug, true, true, false},
		{idRep, KindReply, false, false, true},
		{idQ, KindQuestion, true, true, false},
		{idAns, KindAnswer, false, false, true},
		{idMemo, KindMemo, false, false, false},
	} {
		r := state.Record(tt.id)
		if r.Kind() != tt.kind || r.HasState() != tt.hasState || r.CanHaveReplies() != tt.canBeReplied || r.IsReply() != tt.isReply {
			t.Errorf("%s: kind %s, HasState %v, CanHaveReplies %v, IsReply %v; want %s, %v, %v, %v",
				tt.id[:4], r.Kind(), r.HasState(), r.CanHaveReplies(), r.IsReply(),
				tt.kind, tt.hasState, tt.canBeReplied, tt.isReply)
		}
	}
	// A bug starts open, like a question.
	if r := state.Record(idBug); r.Status != journal.StatusOpen {
		t.Errorf("a bug starts %q", r.Status)
	}
}

func TestBugsAndReplies(t *testing.T) {
	state := Build([]journal.Event{
		create(idBug, journal.TypeBug, "Parser crashes on empty input", 0),
		create(idBug2, journal.TypeBug, "Error position is off by one", 1),
		replyTo(idRep, idBug, "Reproduced on macOS too", 2, agent),
		replyTo(idRep2, idBug, "The empty file has no first token", 3, human),
		status(idBug2, "open", "done", 4, human),
		create(idQ, journal.TypeQA, "Nested block comments?", 5),
		answer(idAns, idQ, "Not in the first version", 6, human),
	})

	sameIDs(t, "open bugs", state.Parents(journal.TypeBug, false), idBug)
	sameIDs(t, "all bugs", state.Parents(journal.TypeBug, true), idBug, idBug2)
	sameIDs(t, "replies, oldest first", state.Replies(idBug), idRep, idRep2)
	sameIDs(t, "replies of a bug without any", state.Replies(idBug2))

	// The two do not mix: a bug is not a question, and a reply is not an answer.
	sameIDs(t, "open questions", state.Parents(journal.TypeQA, false), idQ)
	sameIDs(t, "answers", state.Replies(idQ), idAns)
	sameIDs(t, "todos", state.Todos(true))
	if !state.HasParent(state.Record(idRep)) {
		t.Error("a reply to a bug in the journal has its bug")
	}

	// A closed bug can still be replied to, and closing is a state of the bug
	// only: the replies do not change.
	ev, err := SetStatus(state.Record(idBug2), journal.StatusOpen)
	if err != nil || ev.From != journal.StatusDone || ev.Status != journal.StatusOpen {
		t.Errorf("SetStatus on a bug = %+v, %v", ev, err)
	}
	if _, err := SetStatus(state.Record(idRep), journal.StatusDone); !errors.Is(err, ErrNoState) {
		t.Errorf("a reply has no state: err = %v", err)
	}

	// IDs find them, and a command for bugs does not take a question.
	if rec, err := state.ResolveKind("7f3a", KindBug); err != nil || rec.ID != idBug {
		t.Errorf("ResolveKind(bug) = %v, %v", rec, err)
	}
	var wrong *WrongKindError
	if _, err := state.ResolveKind("95e7", KindBug); !errors.As(err, &wrong) || wrong.Record.Kind() != KindQuestion {
		t.Errorf("a question is not a bug: err = %v", err)
	}
	if _, err := state.ResolveKind("7f3a", KindQuestion); !errors.As(err, &wrong) || wrong.Record.Kind() != KindBug {
		t.Errorf("a bug is not a question: err = %v", err)
	}
}

func TestAReplyHasTheTypeOfItsParent(t *testing.T) {
	// Lines written by another tool can name a record of another type in re. That
	// is not an error, and it does not make a reply: the record is a reply to
	// nothing, and stays in view.
	state := Build([]journal.Event{
		create(idBug, journal.TypeBug, "a bug", 0),
		create(idQ, journal.TypeQA, "a question", 1),
		replyTo(idRep, idQ, "type bug, but re is a question", 2, human),
		answer(idAns, idBug, "type qa, but re is a bug", 3, human),
		replyTo(idRep2, idAns, "re is an answer, which cannot be replied to", 4, human),
	})

	sameIDs(t, "replies of the bug", state.Replies(idBug))
	sameIDs(t, "answers of the question", state.Replies(idQ))
	sameIDs(t, "replies of an answer", state.Replies(idAns))
	for _, id := range []string{idRep, idAns, idRep2} {
		if state.HasParent(state.Record(id)) {
			t.Errorf("%s has a parent", id[:4])
		}
	}
	sameIDs(t, "All", state.All(), idBug, idQ, idRep, idAns, idRep2)
	// Their kind still says what they claim to be.
	if k := state.Record(idRep).Kind(); k != KindReply {
		t.Errorf("kind = %s", k)
	}

	// Deleting the record that re names hides nothing of another type.
	state = Build([]journal.Event{
		create(idBug, journal.TypeBug, "a bug", 0),
		create(idQ, journal.TypeQA, "a question", 1),
		replyTo(idRep, idQ, "type bug, but re is a question", 2, human),
		del(idQ, 3),
	})
	sameIDs(t, "All", state.All(), idBug, idRep)
}

func TestDeletingABugHidesItsReplies(t *testing.T) {
	state := Build([]journal.Event{
		create(idBug, journal.TypeBug, "a bug", 0),
		create(idBug2, journal.TypeBug, "another bug", 1),
		replyTo(idRep, idBug, "one", 2, agent),
		replyTo(idRep2, idBug2, "two", 3, human),
		del(idBug, 4),
	})
	sameIDs(t, "bugs", state.Parents(journal.TypeBug, true), idBug2)
	sameIDs(t, "replies of the deleted bug", state.Replies(idBug))
	sameIDs(t, "All", state.All(), idBug2, idRep2)
	for _, id := range []string{idBug, idRep} {
		if _, err := state.Resolve(id); !errors.Is(err, ErrNotFound) {
			t.Errorf("Resolve(%s): err = %v, want ErrNotFound", id[:4], err)
		}
	}

	// Deleting one reply hides that reply only.
	state = Build([]journal.Event{
		create(idBug, journal.TypeBug, "a bug", 0),
		replyTo(idRep, idBug, "one", 1, agent),
		replyTo(idRep2, idBug, "two", 2, human),
		del(idRep, 3),
	})
	sameIDs(t, "replies", state.Replies(idBug), idRep2)
}

func TestSummaryCountsBugsApartFromQuestions(t *testing.T) {
	state := Build([]journal.Event{
		create(idQ, journal.TypeQA, "a question", 0),
		answer(idAns, idQ, "an answer", 1, human),
		create(idBug, journal.TypeBug, "a bug with a reply", 2),
		replyTo(idRep, idBug, "a reply", 3, agent),
		create(idBug2, journal.TypeBug, "a bug without one", 4),
		create("b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3", journal.TypeBug, "a closed bug", 5),
		status("b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3", "open", "done", 6, human),
	})
	want := Summary{OpenQuestions: 1, QuestionsAwaitingConfirmation: 1, OpenBugs: 2, BugsAwaitingConfirmation: 1}
	if got := state.Summary(); got != want {
		t.Errorf("Summary() = %+v, want %+v", got, want)
	}

	// A reply that is deleted no longer awaits confirmation; a bug that is deleted
	// is not open.
	state = Build([]journal.Event{
		create(idBug, journal.TypeBug, "a bug", 0),
		replyTo(idRep, idBug, "a reply", 1, agent),
		del(idRep, 2),
		create(idBug2, journal.TypeBug, "another bug", 3),
		del(idBug2, 4),
	})
	want = Summary{OpenBugs: 1}
	if got := state.Summary(); got != want {
		t.Errorf("Summary() = %+v, want %+v", got, want)
	}
}

func TestBugAndReplyEvents(t *testing.T) {
	b, err := ParentCreate(journal.TypeBug, "Parser crashes on empty input")
	if err != nil || b.Op != journal.OpCreate || b.Type != journal.TypeBug || b.Status != journal.StatusOpen || b.Re != "" || b.ID != "" {
		t.Errorf("bug = %+v, %v", b, err)
	}

	state := Build([]journal.Event{
		create(idBug, journal.TypeBug, "a bug", 0),
		create(idQ, journal.TypeQA, "a question", 1),
		replyTo(idRep, idBug, "a reply", 2, human),
	})
	// The reply has the type of its parent, and the full ID of it in re.
	r, err := ReplyCreate(state.Record(idBug), "Reproduced on macOS too")
	if err != nil || r.Op != journal.OpCreate || r.Type != journal.TypeBug || r.Re != idBug || r.Status != "" || r.Text != "Reproduced on macOS too" {
		t.Errorf("reply = %+v, %v", r, err)
	}
	a, err := ReplyCreate(state.Record(idQ), "an answer")
	if err != nil || a.Type != journal.TypeQA || a.Re != idQ {
		t.Errorf("answer = %+v, %v", a, err)
	}

	// Only a question or a bug can be replied to: not a reply, not a memo.
	for _, id := range []string{idRep} {
		var none *NoRepliesError
		if _, err := ReplyCreate(state.Record(id), "x"); !errors.As(err, &none) {
			t.Errorf("ReplyCreate on a %s: err = %v", state.Record(id).Kind(), err)
		}
	}

	for _, text := range []string{"", "  ", "\n"} {
		if _, err := ParentCreate(journal.TypeBug, text); !errors.Is(err, ErrEmptyText) {
			t.Errorf("ParentCreate(bug, %q): err = %v", text, err)
		}
		if _, err := ReplyCreate(state.Record(idBug), text); !errors.Is(err, ErrEmptyText) {
			t.Errorf("ReplyCreate(%q): err = %v", text, err)
		}
	}

	// Only qa and bug start something that can be replied to.
	for _, typ := range []string{journal.TypeMemo, journal.TypeTodo, journal.TypeGlossary, "note"} {
		if _, err := ParentCreate(typ, "x"); !errors.Is(err, journal.ErrInvalidEvent) {
			t.Errorf("ParentCreate(%s): err = %v", typ, err)
		}
	}
}

func TestHistoryOfABug(t *testing.T) {
	state := Build([]journal.Event{
		create(idBug, journal.TypeBug, "a bug", 0),
		replyTo(idRep, idBug, "one", 1, agent),
		status(idBug, "open", "done", 3, human),
		replyTo(idRep2, idBug, "two", 2, human),
		create(idQ, journal.TypeQA, "a question", 4),
		replyTo("5d0e6c2f34ab4b99d8c2af307b4e1d56", idQ, "type bug, re is a question", 5, human),
	})
	var got []string
	for _, e := range state.History(state.Record(idBug)) {
		what := e.Event.Op
		if e.Answer != nil {
			what += " " + e.Answer.ID[:4]
		}
		got = append(got, what)
	}
	want := []string{"create", "create 3d8e", "create 4c9d", "status"}
	if len(got) != len(want) {
		t.Fatalf("history = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("history = %v, want %v", got, want)
		}
	}
	// What re names is not part of the history of another type.
	if h := state.History(state.Record(idQ)); len(h) != 1 {
		t.Errorf("history of the question = %+v, want its create only", h)
	}
}
