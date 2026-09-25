package model

import (
	"fmt"
	"testing"

	"github.com/amisonnet8/mtqg/internal/journal"
)

// contextIDs gives the records of a fixture IDs that are easy to tell apart.
func cid(n int) string { return fmt.Sprintf("%032x", n) }

// contextFixture is a journal for context. By the time of creation: the todos
// t1 t2 t3 (minutes 0 1 2, and t4 that is done), the question q1 (3, two answers), the bug b1 (4, one
// reply), the question q2 (5), then a closed question, ten memos, a bug b2 (20)
// and the words a, b and b again.
func contextFixture() *State {
	events := []journal.Event{
		create(cid(1), journal.TypeTodo, "t1", 0),
		create(cid(2), journal.TypeTodo, "t2", 1),
		create(cid(3), journal.TypeTodo, "t3", 2),
		create(cid(4), journal.TypeTodo, "t4, done", 8),
		status(cid(4), "open", "done", 9, human),
		create(cid(10), journal.TypeQA, "q1", 3),
		answer(cid(11), cid(10), "answer one", 10, agent),
		answer(cid(12), cid(10), "answer two", 11, human),
		create(cid(20), journal.TypeBug, "b1", 4),
		reply(journal.TypeBug, cid(21), cid(20), "reply one", 12, agent),
		create(cid(13), journal.TypeQA, "q2", 5),
		create(cid(14), journal.TypeQA, "q closed", 6),
		status(cid(14), "open", "done", 7, human),
		create(cid(22), journal.TypeBug, "b2", 20),
		entry(cid(30), "a", "definition of a", 40, human),
		entry(cid(31), "b", "first definition of b", 41, human),
		entry(cid(32), "b", "second definition of b", 42, agent),
	}
	for i := 0; i < 10; i++ {
		events = append(events, create(cid(100+i), journal.TypeMemo, fmt.Sprintf("memo %d", i), 30+i))
	}
	return Build(events)
}

func TestContextGathersWhatIsOpen(t *testing.T) {
	d := contextFixture().Context(4)

	sameIDs(t, "todos", d.Todos, cid(1), cid(2), cid(3))
	if d.TodosTotal != 3 || d.QuestionsTotal != 2 || d.BugsTotal != 2 {
		t.Errorf("totals %d %d %d", d.TodosTotal, d.QuestionsTotal, d.BugsTotal)
	}

	// Open ones only, oldest first, each with how many answers it has and the latest.
	if len(d.Questions) != 2 || d.Questions[0].Parent.ID != cid(10) || d.Questions[1].Parent.ID != cid(13) {
		t.Fatalf("questions %v", d.Questions)
	}
	if q := d.Questions[0]; q.ReplyCount != 2 || q.Latest == nil || q.Latest.ID != cid(12) {
		t.Errorf("q1: %d replies, latest %v", q.ReplyCount, q.Latest)
	}
	if q := d.Questions[1]; q.ReplyCount != 0 || q.Latest != nil {
		t.Errorf("q2: %d replies, latest %v", q.ReplyCount, q.Latest)
	}
	if b := d.Bugs[0]; b.Parent.ID != cid(20) || b.ReplyCount != 1 || b.Latest.ID != cid(21) || b.Latest.Kind() != KindReply {
		t.Errorf("b1: %+v", b)
	}

	// The newest records of every kind come first, ten of them, and the total says
	// how many there are.
	if len(d.Recent) != 10 || d.RecentTotal != 25 { // 4 todos, 3 questions and 2 answers, 2 bugs and a reply, 3 words, 10 memos
		t.Errorf("%d recent of %d", len(d.Recent), d.RecentTotal)
	}
	sameIDs(t, "the newest", d.Recent[:4], cid(32), cid(31), cid(30), cid(109))

	// The glossary is every entry, and the words with more than one definition are
	// named.
	if d.GlossaryTotal != 3 || len(d.Glossary) != 3 || d.Glossary[2].Definitions != 2 || d.Glossary[0].Definitions != 1 || d.Glossary[2].Entry == nil {
		t.Errorf("glossary %+v", d.Glossary)
	}
	if len(d.DuplicateWords) != 1 || d.DuplicateWords[0] != "b" || d.Uncommitted != 4 {
		t.Errorf("attention: %v, %d", d.DuplicateWords, d.Uncommitted)
	}
	if d.DefinitionsLeft || d.RepliesLeft {
		t.Error("nothing is left out to begin with")
	}
}

func TestContextLeavesOutWhatIsHidden(t *testing.T) {
	state := Build([]journal.Event{
		create(cid(1), journal.TypeTodo, "gone", 0),
		del(cid(1), 1),
		create(cid(2), journal.TypeQA, "asked", 2),
		answer(cid(3), cid(2), "answered", 3, agent),
		del(cid(2), 4),
		create(cid(4), journal.TypeMemo, "kept", 5),
	})
	d := state.Context(0)
	if len(d.Todos) != 0 || len(d.Questions) != 0 || d.RecentTotal != 1 || d.Recent[0].ID != cid(4) {
		t.Errorf("%+v", d)
	}
	if e := Build(nil).Context(0); e.Steps() != 5 || len(e.Recent) != 0 || e.RecentTotal != 0 {
		t.Errorf("empty: %+v", e)
	}
}

func TestContextReducedCutsInTheOrderOfTheSpec(t *testing.T) {
	d := contextFixture().Context(0)
	if d.Steps() != 3+2+3 { // no question or bug is beyond the newest 3
		t.Fatalf("steps = %d", d.Steps())
	}

	type shape struct{ recent, glossary, questions, bugs, todos int }
	got := func(n int) (shape, *ContextData) {
		r := d.Reduced(n)
		return shape{len(r.Recent), len(r.Glossary), len(r.Questions), len(r.Bugs), len(r.Todos)}, r
	}
	for _, tt := range []struct {
		n    int
		want shape
		note string
	}{
		{0, shape{10, 3, 2, 2, 3}, "everything"},
		{1, shape{5, 3, 2, 2, 3}, "recent records to 5"},
		{2, shape{3, 3, 2, 2, 3}, "to 3"},
		{3, shape{0, 3, 2, 2, 3}, "to none"},
		{4, shape{0, 2, 2, 2, 3}, "the definitions: each word once"},
		{5, shape{0, 2, 2, 2, 3}, "the latest answers and replies"},
		{6, shape{0, 2, 2, 2, 2}, "the oldest todo: the questions and bugs are all among the newest 3"},
		{7, shape{0, 2, 2, 2, 1}, "the next"},
		{8, shape{0, 2, 2, 2, 0}, "all the todos"},
		{9, shape{0, 2, 2, 2, 0}, "more cuts than steps make no difference"},
		{1000, shape{0, 2, 2, 2, 0}, "the same"},
	} {
		if s, _ := got(tt.n); s != tt.want {
			t.Errorf("after %d cuts (%s): %+v, want %+v", tt.n, tt.note, s, tt.want)
		}
	}

	// What is left is the newest, and the totals still say what there was.
	_, r := got(8)
	if len(r.Questions) != 2 || len(r.Bugs) != 2 || r.QuestionsTotal != 2 || r.BugsTotal != 2 || r.TodosTotal != 3 || r.RecentTotal != d.RecentTotal || r.GlossaryTotal != 3 {
		t.Errorf("totals: %+v", r)
	}
	_, r = got(6)
	sameIDs(t, "todos left after one cut", r.Todos, cid(2), cid(3))

	// The words are the words: b has two definitions and no entry.
	_, r = got(4)
	if !r.DefinitionsLeft || r.Glossary[0].Word != "a" || r.Glossary[1].Word != "b" || r.Glossary[1].Definitions != 2 || r.Glossary[1].Entry != nil {
		t.Errorf("words %+v", r.Glossary)
	}
	// Leaving out the latest answers keeps how many there were.
	_, r = got(5)
	if !r.RepliesLeft || r.Questions[0].Latest != nil || r.Questions[0].ReplyCount != 2 || r.Bugs[0].Latest != nil {
		t.Errorf("questions %+v", r.Questions)
	}
}

func TestContextReducedDoesNotChangeWhatItIsGiven(t *testing.T) {
	d := contextFixture().Context(2)
	before := *d
	for n := 0; n <= d.Steps()+1; n++ {
		d.Reduced(n)
	}
	if len(d.Recent) != 10 || len(d.Todos) != 3 || len(d.Questions) != 2 || len(d.Glossary) != 3 ||
		d.Questions[0].Latest == nil || d.DefinitionsLeft || d.RepliesLeft || d.Uncommitted != before.Uncommitted {
		t.Errorf("d was changed: %+v", d)
	}
	// Each cut takes away and never gives back.
	size := func(r *ContextData) int {
		return len(r.Recent) + len(r.Glossary) + len(r.Questions) + len(r.Bugs) + len(r.Todos)
	}
	prev := size(d)
	for n := 1; n <= d.Steps(); n++ {
		s := size(d.Reduced(n))
		if s > prev {
			t.Errorf("after %d cuts there is more (%d) than after %d (%d)", n, s, n-1, prev)
		}
		prev = s
	}
}

func TestContextRulesAreNeverReduced(t *testing.T) {
	state := Build([]journal.Event{
		create(cid(1), journal.TypeRule, "rule one", 0),
		create(cid(2), journal.TypeRule, "rule two", 1),
		create(cid(3), journal.TypeMemo, "a memo", 2),
	})
	d := state.Context(0)
	sameIDs(t, "rules", d.Rules, cid(1), cid(2))

	// Cutting to nothing (Steps() does not count Rules, so there is no step for
	// them) still leaves every rule, in full.
	r := d.Reduced(d.Steps() + 5)
	sameIDs(t, "rules after every cut", r.Rules, cid(1), cid(2))
	if len(r.Recent) != 0 {
		t.Errorf("recent after every cut = %v, want none", r.Recent)
	}
}

func TestContextKeepsTheNewestQuestionsAndBugs(t *testing.T) {
	// Six questions (minutes 0 2 4 6 8 10), five bugs (1 3 5 7 9) and four todos.
	var events []journal.Event
	for i := 0; i < 6; i++ {
		events = append(events, create(cid(10+i), journal.TypeQA, fmt.Sprintf("q%d", i), 2*i))
	}
	for i := 0; i < 5; i++ {
		events = append(events, create(cid(20+i), journal.TypeBug, fmt.Sprintf("b%d", i), 2*i+1))
	}
	for i := 0; i < 4; i++ {
		events = append(events, create(cid(30+i), journal.TypeTodo, fmt.Sprintf("t%d", i), 20+i))
	}
	d := Build(events).Context(0)

	// 3 questions and 2 bugs are beyond the newest 3 of each, and can go.
	if d.Steps() != 3+2+3+2+4 {
		t.Fatalf("steps = %d", d.Steps())
	}
	first := 3 + 2 // the steps before the questions and bugs
	type shape struct{ questions, bugs, todos int }
	for _, tt := range []struct {
		cuts int
		want shape
		note string
	}{
		{first, shape{6, 5, 4}, "before any of them"},
		{first + 1, shape{5, 5, 4}, "q0 (minute 0) is the oldest"},
		{first + 2, shape{5, 4, 4}, "b0 (minute 1)"},
		{first + 3, shape{4, 4, 4}, "q1 (minute 2)"},
		{first + 4, shape{4, 3, 4}, "b1 (minute 3)"},
		{first + 5, shape{3, 3, 4}, "q2 (minute 4); q3, b2 and the newer ones are the newest 3 and stay"},
		{first + 6, shape{3, 3, 3}, "and now the todos, oldest first"},
		{first + 9, shape{3, 3, 0}, "all the todos"},
		{first + 10, shape{3, 3, 0}, "and the newest 3 of each are still there"},
		{100, shape{3, 3, 0}, "however many cuts"},
	} {
		r := d.Reduced(tt.cuts)
		if got := (shape{len(r.Questions), len(r.Bugs), len(r.Todos)}); got != tt.want {
			t.Errorf("after %d cuts (%s): %+v, want %+v", tt.cuts, tt.note, got, tt.want)
		}
	}
	r := d.Reduced(100)
	if r.Questions[0].Parent.ID != cid(13) || r.Questions[2].Parent.ID != cid(15) || r.Bugs[0].Parent.ID != cid(22) || r.Bugs[2].Parent.ID != cid(24) {
		t.Errorf("what stays is not the newest: %v %v", r.Questions, r.Bugs)
	}
	if r.QuestionsTotal != 6 || r.BugsTotal != 5 {
		t.Errorf("totals %d %d", r.QuestionsTotal, r.BugsTotal)
	}
}

func TestContextReducedWhenOneSectionIsOlder(t *testing.T) {
	// Every question is older than every bug: the questions go first, down to the
	// newest 3, and only then the bugs, and neither goes below 3.
	var events []journal.Event
	for i := 0; i < 5; i++ {
		events = append(events, create(cid(10+i), journal.TypeQA, fmt.Sprintf("q%d", i), i))
	}
	for i := 0; i < 5; i++ {
		events = append(events, create(cid(20+i), journal.TypeBug, fmt.Sprintf("b%d", i), 10+i))
	}
	d := Build(events).Context(0)
	const first = 3 + 2
	if d.Steps() != first+2+2 {
		t.Fatalf("steps = %d", d.Steps())
	}
	r := d.Reduced(first + 2)
	if len(r.Questions) != 3 || len(r.Bugs) != 5 || r.Questions[0].Parent.ID != cid(12) {
		t.Errorf("questions %d, bugs %d", len(r.Questions), len(r.Bugs))
	}
	r = d.Reduced(first + 3)
	if len(r.Questions) != 3 || len(r.Bugs) != 4 || r.Bugs[0].Parent.ID != cid(21) {
		t.Errorf("questions %d, bugs %d", len(r.Questions), len(r.Bugs))
	}

	// Bugs that are older than the questions: the one bug beyond the newest 3 goes
	// first, and then it is the questions, though the next bug is older than they are.
	events = nil
	for i := 0; i < 4; i++ {
		events = append(events, create(cid(20+i), journal.TypeBug, fmt.Sprintf("b%d", i), i))
	}
	for i := 0; i < 6; i++ {
		events = append(events, create(cid(10+i), journal.TypeQA, fmt.Sprintf("q%d", i), 10+i))
	}
	d = Build(events).Context(0)
	r = d.Reduced(first + 1 + 3)
	if len(r.Bugs) != 3 || len(r.Questions) != 3 || r.Bugs[0].Parent.ID != cid(21) || r.Questions[0].Parent.ID != cid(13) {
		t.Errorf("bugs %d, questions %d", len(r.Bugs), len(r.Questions))
	}

	// Only bugs, and no more than the newest 3: nothing to leave out.
	few := Build([]journal.Event{
		create(cid(1), journal.TypeBug, "b1", 0),
		create(cid(2), journal.TypeBug, "b2", 1),
		create(cid(3), journal.TypeBug, "b3", 2),
	}).Context(0)
	if few.Steps() != 5 || len(few.Reduced(100).Bugs) != 3 {
		t.Errorf("steps = %d, bugs left %d", few.Steps(), len(few.Reduced(100).Bugs))
	}
}
