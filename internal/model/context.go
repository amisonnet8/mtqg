package model

import (
	"github.com/amisonnet8/mtqg/internal/journal"
)

// What context tells an agent, before it is put into words: what needs
// attention, what is open, what happened lately, which words are agreed. The
// model layer decides what is in it and what is left out first when there is too
// much; putting it into words and measuring it are the entry point's.

// recentShown is how many recent records there are to begin with, and
// recentSteps what that is cut to, one step at a time, when there is too much.
const recentShown = 10

var recentSteps = [...]int{5, 3, 0}

// Thread is a question or a bug, with its latest answer or reply.
type Thread struct {
	Parent     *Record
	ReplyCount int
	Latest     *Record // the latest answer or reply; nil if there is none, or it was left out
}

// GlossaryItem is a glossary entry, or when the definitions are left out, a word.
type GlossaryItem struct {
	Word        string
	Entry       *Record // nil when the definition is left out
	Definitions int     // how many entries the word has
}

// ContextData is what context shows. Everything in it is what is shown; the
// Total fields say how many there were, and so how many were left out.
type ContextData struct {
	// DuplicateWords are the words that have more than one definition.
	DuplicateWords []string
	// Uncommitted is how many records are not committed: 0 when none, or when
	// that is not known.
	Uncommitted int

	Todos      []*Record // open, oldest first
	TodosTotal int

	Questions      []Thread // open, oldest first
	QuestionsTotal int
	Bugs           []Thread
	BugsTotal      int
	// RepliesLeft says that the latest answers and replies were left out.
	RepliesLeft bool

	Recent      []*Record // newest first
	RecentTotal int

	Glossary      []GlossaryItem // the entries in the order they were written
	GlossaryTotal int            // the entries there are
	// DefinitionsLeft says that only the words are shown, each once.
	DefinitionsLeft bool
}

// Context gathers what context shows, all of it: Reduced cuts it. uncommitted is
// how many records are not committed (0 if none or not known).
func (s *State) Context(uncommitted int) *ContextData {
	d := &ContextData{Uncommitted: uncommitted}
	for _, group := range s.DuplicateWords() {
		d.DuplicateWords = append(d.DuplicateWords, group[0].Word)
	}

	d.Todos = s.Todos(false)
	d.TodosTotal = len(d.Todos)
	d.Questions = s.threads(journal.TypeQA)
	d.QuestionsTotal = len(d.Questions)
	d.Bugs = s.threads(journal.TypeBug)
	d.BugsTotal = len(d.Bugs)

	all := s.All()
	d.RecentTotal = len(all)
	for i := len(all) - 1; i >= 0 && len(d.Recent) < recentShown; i-- {
		d.Recent = append(d.Recent, all[i])
	}

	entries := s.Glossary()
	d.GlossaryTotal = len(entries)
	perWord := make(map[string]int)
	for _, e := range entries {
		perWord[e.Word]++
	}
	for _, e := range entries {
		d.Glossary = append(d.Glossary, GlossaryItem{Word: e.Word, Entry: e, Definitions: perWord[e.Word]})
	}
	return d
}

// threads returns the open questions or bugs, oldest first, each with its latest
// answer or reply.
func (s *State) threads(typ string) []Thread {
	var out []Thread
	for _, p := range s.Parents(typ, false) {
		replies := s.Replies(p.ID)
		t := Thread{Parent: p, ReplyCount: len(replies)}
		if len(replies) > 0 {
			t.Latest = replies[len(replies)-1]
		}
		out = append(out, t)
	}
	return out
}

// Steps is how many times Reduced can cut: the steps that are always there (the
// recent records in three steps, the definitions, the latest answers and replies),
// and one for each question, bug and todo. A step may cut nothing (when there are
// only 3 recent records, the step from 5 to 3 does nothing).
func (d *ContextData) Steps() int {
	return len(recentSteps) + 2 + len(d.Questions) + len(d.Bugs) + len(d.Todos)
}

// Reduced returns d with its first n cuts made, in this order: the recent records
// (to 5, to 3, to none), the definitions of the glossary (each word stays), the
// latest answers and replies, then the oldest questions and bugs, one at a time (the
// two together), then the oldest todos, one at a time. More cuts than Steps make
// no more difference. d itself is not changed, and the more cuts, the less there is.
func (d *ContextData) Reduced(n int) *ContextData {
	r := *d
	for _, limit := range recentSteps {
		if n == 0 {
			return &r
		}
		n--
		if len(r.Recent) > limit {
			r.Recent = r.Recent[:limit]
		}
	}

	if n == 0 {
		return &r
	}
	n--
	r.DefinitionsLeft = true
	r.Glossary = wordsOf(d.Glossary)

	if n == 0 {
		return &r
	}
	n--
	r.RepliesLeft = true
	r.Questions = withoutLatest(r.Questions)
	r.Bugs = withoutLatest(r.Bugs)

	// The oldest question or bug of the two sections goes first.
	drop := min(n, len(r.Questions)+len(r.Bugs))
	n -= drop
	q, b := 0, 0
	for i := 0; i < drop; i++ {
		switch {
		case q == len(r.Questions):
			b++
		case b == len(r.Bugs):
			q++
		case olderThan(r.Questions[q].Parent, r.Bugs[b].Parent):
			q++
		default:
			b++
		}
	}
	r.Questions, r.Bugs = r.Questions[q:], r.Bugs[b:]

	r.Todos = r.Todos[min(n, len(r.Todos)):]
	return &r
}

// wordsOf keeps each word once, at its first entry, with how many entries it has.
func wordsOf(items []GlossaryItem) []GlossaryItem {
	var out []GlossaryItem
	seen := make(map[string]bool)
	for _, it := range items {
		if !seen[it.Word] {
			seen[it.Word] = true
			out = append(out, GlossaryItem{Word: it.Word, Definitions: it.Definitions})
		}
	}
	return out
}

func withoutLatest(threads []Thread) []Thread {
	out := make([]Thread, len(threads))
	for i, t := range threads {
		t.Latest = nil
		out[i] = t
	}
	return out
}

// olderThan orders by the time of creation and, for the same time, by ID, as the
// events are ordered.
func olderThan(a, b *Record) bool {
	if !a.Created.Equal(b.Created) {
		return a.Created.Before(b.Created)
	}
	return a.ID < b.ID
}
