//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// todoLine is the line of a todo written by an AI on a day long ago, for a journal that
// needs an ID that mtqg would not make.
func todoLine(id, ts, text string) string {
	return fmt.Sprintf(`{"id":"%s","op":"create","type":"todo","status":"open","text":"%s","v":0,"ts":"%s","author":{"kind":"ai","name":"claude-code"}}`, id, text, ts)
}

// A prefix that fits two records is not a guess: mtqg stops, writes nothing, and lists
// the candidates with their full IDs, from which any length can be typed again.
func TestAnAmbiguousIDListsCandidates(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")
	const (
		memoID = "70430f77ff91c2e04a8b33f1d7e6a025"
		todoID = "70430f77ff4b475185d5cae12dff1a17"
	)
	r.appendToJournal(
		oldLine(memoID, "2026-03-02T09:00:00Z", "Parser now skips // at line end"),
		todoLine(todoID, "2026-09-21T10:18:00Z", "Skip block comments"),
	)
	before := r.journal()

	for _, prefix := range []string{"70430f77ff", "7043"} {
		res := r.run(nil, "", "t", "done", prefix)
		if res.code != 1 || res.stdout != "" || !strings.HasPrefix(res.stderr, "Ambiguous ID \""+prefix+"\" matches 2 records:\n") {
			t.Errorf("t done %s: %+v", prefix, res)
		}
		for _, id := range []string{memoID, todoID} {
			if !strings.Contains(res.stderr, id) {
				t.Errorf("t done %s does not list the full ID %s:\n%s", prefix, id, res.stderr)
			}
		}
	}
	if got := r.journal(); got != before {
		t.Errorf("the journal changed:\n%s", got)
	}

	// For a program, the candidates are records.
	res := r.run(nil, "", "show", "70430f77ff", "--json")
	var failure struct {
		Error struct {
			Kind       string
			Candidates []struct{ ID string }
		}
	}
	if res.code != 1 || res.stdout != "" || json.Unmarshal([]byte(res.stderr), &failure) != nil || failure.Error.Kind != "ambiguous" || len(failure.Error.Candidates) != 2 {
		t.Fatalf("show --json: %+v", res)
	}
	got := []string{failure.Error.Candidates[0].ID, failure.Error.Candidates[1].ID}
	slices.Sort(got)
	if want := []string{todoID, memoID}; !slices.Equal(got, want) { // 4b47... sorts before 91c2...
		t.Errorf("candidates = %v, want %v", got, want)
	}

	// One more digit tells them apart, and fewer than four are not an ID at all.
	if got := r.mtqg("t", "done", "70430f77ff4"); !strings.HasPrefix(got, "Done: 70430f77ff  Skip block comments") {
		t.Errorf("t done with the longer prefix: %q", got)
	}
	res = r.run(nil, "", "show", "704")
	if res.code != 1 || !strings.Contains(res.stderr, `The ID "704" is too short: give at least 4 digits`) {
		t.Errorf("show 704: %+v", res)
	}
}

// An ID that names the wrong kind of record stops the command, whichever way round, and
// says what the record is (and which command is the right one, if there is one).
func TestTheWrongKindOfIDStops(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")
	question := trimmed(r.mtqg("q", "add", "Should nested block comments be supported?"))
	bug := trimmed(r.mtqg("b", "add", "Parser crashes on empty input"))
	answer := trimmed(r.mtqg("q", "add", question, "Not in the first version"))
	reply := trimmed(r.mtqg("b", "add", bug, "Reproduced on macOS too"))
	todo := trimmed(r.mtqg("t", "add", "Skip block comments"))
	before := r.journal()

	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"b", "add", question, "x"}, question + " is a question, not a bug"},
		{[]string{"q", "add", bug, "x"}, bug + " is a bug, not a question"},
		{[]string{"q", "add", todo, "x"}, todo + " is a todo, not a question"},
		{[]string{"q", "add", answer, "x"}, answer + " is an answer, not a question"}, // an answer is not answered
		{[]string{"b", "add", reply, "x"}, reply + " is a reply, not a bug"},
		{[]string{"q", "done", bug}, bug + " is a bug, not a question; use `mtqg bug done " + bug + "`"},
		{[]string{"b", "done", question}, question + " is a question, not a bug; use `mtqg qa done " + question + "`"},
		{[]string{"t", "done", question}, question + " is a question, not a todo; use `mtqg qa done " + question + "`"},
		{[]string{"t", "done", bug}, bug + " is a bug, not a todo; use `mtqg bug done " + bug + "`"},
		{[]string{"q", "done", todo}, todo + " is a todo, not a question; use `mtqg todo done " + todo + "`"},
		{[]string{"t", "done", answer}, answer + " is an answer, not a todo"},
		{[]string{"b", "reopen", reply}, reply + " is a reply, not a bug"},
	} {
		res := r.run(nil, "", c.args...)
		if res.code != 1 || res.stdout != "" || !strings.HasPrefix(res.stderr, c.want+"\n") {
			t.Errorf("mtqg %s:\n%+v\nwant %q", strings.Join(c.args, " "), res, c.want)
		}
	}
	if got := r.journal(); got != before {
		t.Errorf("a command that stopped wrote to the journal:\n%s", got)
	}

	// A line that another tool wrote, a reply whose re names a question, is not an error
	// either: it is read as a reply to nothing, and review says so.
	full := trimmed(r.mtqg("q", "add", "--full-id", "A question for the stray reply"))
	r.appendToJournal(fmt.Sprintf(`{"id":"9a8b7c6d5e4f4a3b2c1d0e9f8a7b6c5d","op":"create","type":"bug","re":"%s","text":"A reply that names a question","v":0,"ts":"2026-09-21T10:46:00Z","author":{"kind":"ai","name":"claude-code"}}`, full))
	if res := r.run(nil, "", "b", "list"); res.code != 0 || strings.Contains(res.stdout, "names a question") || res.stderr != "" {
		t.Errorf("b list: %+v", res)
	}
	review := r.mtqg("review")
	if !strings.HasPrefix(review, "Answers and replies with no parent (1)\n") || !strings.Contains(review, "9a8b7c6d5e  reply  claude-code  A reply that names a question") || !strings.Contains(review, "a question, not a bug") {
		t.Errorf("review:\n%s", review)
	}
}

// edit changes the words and nothing else, delete hides a record and what hangs from it,
// and search finds what is in view.
func TestEditDeleteAndSearch(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")
	agent := []string{"MTQG_AUTHOR_KIND=ai", "MTQG_AUTHOR_NAME=claude-code"}

	todo := trimmed(r.mtqg("t", "add", "Skip Block comments"))
	r.mtqg("t", "done", todo)
	word := trimmed(r.mtqg("g", "add", "token", "The smallest unit"))
	question := trimmed(r.mtqg("q", "add", "Should nested block comments be supported?"))
	if res := r.run(agent, "", "q", "add", question, "Supporting them is preferable"); res.code != 0 {
		t.Fatalf("%+v", res)
	}
	answer := trimmed(r.mtqg("q", "add", question, "Not in the first version"))

	// edit: the todo stays done, the entry keeps its word.
	if got := r.mtqg("edit", todo, "Skip", "block", "comments", "and", "line", "comments"); got != "Edited: "+todo+"  Skip block comments and line comments\n" {
		t.Errorf("edit: %q", got)
	}
	if got := r.mtqg("edit", todo, "Skip", "block", "comments", "and", "line", "comments"); !strings.HasPrefix(got, "Unchanged: ") {
		t.Errorf("edit with the same text: %q", got)
	}
	if got := r.mtqg("t", "list", "--all"); !strings.Contains(got, "Skip block comments and line comments") || !strings.HasSuffix(got, "0 open, 1 done\n") {
		t.Errorf("t list --all after the edit:\n%s", got)
	}
	r.mtqg("edit", word, "The smallest unit that lexing produces")
	if got := r.mtqg("g", "list"); !strings.Contains(got, "token  The smallest unit that lexing produces") {
		t.Errorf("g list after the edit:\n%s", got)
	}

	// search: case does not matter, a word of the glossary is found, and so is text of any kind.
	if got := r.mtqg("search", "BLOCK"); !strings.Contains(got, "Skip block comments and line comments") || !strings.Contains(got, "Should nested block comments") || !strings.HasSuffix(got, "2 records contain \"BLOCK\"\n") {
		t.Errorf("search BLOCK:\n%s", got)
	}
	if got := r.mtqg("search", "TOKEN"); !strings.Contains(got, "token: The smallest unit") || !strings.HasSuffix(got, "1 record contains \"TOKEN\"\n") {
		t.Errorf("search TOKEN:\n%s", got)
	}
	if res := r.run(nil, "", "search", "no such text"); res.code != 0 || res.stdout != "No records contain \"no such text\"\n" {
		t.Errorf("search with no match: %+v", res)
	}

	// delete an answer: only it is hidden, and the question stays open.
	if got := r.mtqg("delete", answer); !strings.HasPrefix(got, "Deleted: "+answer+"  Not in the first version\n") || strings.Contains(got, "also hidden") || !strings.HasSuffix(got, "The lines remain in the journal and in git history\n") {
		t.Errorf("delete an answer: %q", got)
	}
	if got := r.mtqg("q", "list"); !strings.Contains(got, "1 answer, awaiting confirmation") || strings.Contains(got, "Not in the first version") {
		t.Errorf("q list after the answer is deleted:\n%s", got)
	}

	// delete the question: what answered it goes too, and who wrote it is said.
	if got := r.mtqg("delete", question); !strings.Contains(got, "1 answer is also hidden (claude-code)\n") {
		t.Errorf("delete a question: %q", got)
	}
	for _, args := range [][]string{{"q", "list", "--all"}, {"log"}, {"search", "nested"}, {"search", "supporting"}} {
		if got := r.mtqg(args...); strings.Contains(got, "nested block comments") || strings.Contains(got, "Supporting them") {
			t.Errorf("mtqg %s still shows what was deleted:\n%s", strings.Join(args, " "), got)
		}
	}
	res := r.run(nil, "", "show", question)
	if res.code != 1 || !strings.Contains(res.stderr, "No record matches") {
		t.Errorf("show a deleted record: %+v", res)
	}
	// Nothing was removed: the lines are still in the file.
	if journal := r.journal(); !strings.Contains(journal, "Should nested block comments be supported?") || strings.Count(journal, `"op":"delete"`) != 2 {
		t.Errorf("the journal after the deletes:\n%s", journal)
	}
}
