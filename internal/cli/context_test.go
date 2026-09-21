package cli

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	for _, tt := range []struct {
		s    string
		want int
	}{
		{"", 0},
		{"abc", 1}, // ASCII: 4 characters to a token, rounded up
		{"abcd", 1},
		{"abcde", 2},
		{strings.Repeat("a", 400), 100},
		{"\n\n\n\n\n", 2},       // a line feed is a character
		{"あいうえお", 5},            // every character that is not ASCII is a token
		{"aあb", 2},              // 2 ASCII characters round up to 1, and the other is 1
		{"日本語 text 日本語", 6 + 2}, // 6 characters, and 6 of ASCII
	} {
		if got := estimateTokens(tt.s); got != tt.want {
			t.Errorf("estimateTokens(%q) = %d, want %d", tt.s, got, tt.want)
		}
	}
}

func TestFitToBudget(t *testing.T) {
	// A cost that falls by 10 with each cut, from 100 to 0 in 10 cuts.
	cost := func(n int) int { return 100 - 10*n }
	for _, tt := range []struct{ budget, want int }{
		{0, 0},   // no limit
		{100, 0}, // fits
		{99, 1},
		{50, 5},
		{51, 5},
		{10, 9},
		{1, 10}, // cost(9) is 10 and cost(10) is 0
	} {
		if got := fitToBudget(10, tt.budget, cost); got != tt.want {
			t.Errorf("budget %d: %d cuts, want %d", tt.budget, got, tt.want)
		}
	}
	if got := fitToBudget(4, 5, func(n int) int { return 1000 }); got != 4 {
		t.Errorf("never fits: %d cuts, want all 4", got)
	}
}

// contextHarness is a repository with a journal that has every kind in it. The
// clock is 2026-09-17 12:00 UTC, so the times of the fixture are shown as HH:MM.
func contextHarness(t *testing.T) *harness {
	t.Helper()
	h := initialized(t)
	h.setJournal(
		record(idA, "todo", "Skip block comments /* */", nameC, "2026-09-17T10:18:00Z"),
		record(idC, "todo", "Show error positions as line and column", "yamada", "2026-09-17T11:06:00Z"),
		record(idB, "todo", "Ignore // inside string literals", "yamada", "2026-09-17T08:00:00Z"),
		change(idB, "open", "done", "yamada", "2026-09-17T08:30:00Z"),
		record(idM, "memo", "Policy: use English for all error messages", "yamada", "2026-09-17T10:32:00Z"),
		question(idQ2, "Should nested block comments be supported?", nameC, "2026-09-17T09:10:00Z"),
		answerLine(idA2, idQ2, "Not in the first version\nRevisit if there is demand", "yamada", "human", "2026-09-17T09:41:00Z"),
		question(idQ3, "Should error positions show both line and column?", nameC, "2026-09-17T11:05:00Z"),
		bugLine(idBug1, "Parser crashes on empty input", "yamada", "2026-09-17T10:41:00Z"),
		replyLine(idRep1, idBug1, "Reproduced on macOS too", nameC, "ai", "2026-09-17T10:45:00Z"),
		entryLine(idW1, "block comment", "A comment enclosed in /* and */", "yamada", "2026-09-17T09:30:00Z"),
		entryLine(idW2, "block comment", "A comment that can span multiple lines", nameC, "2026-09-17T10:00:00Z"),
		entryLine(idW3, "lexing", "Reading source and turning it into tokens", nameC, "2026-09-17T11:24:00Z"),
	)
	return h
}

func TestContext(t *testing.T) {
	t.Run("every section, in order, with IDs", func(t *testing.T) {
		h := contextHarness(t)
		code, out, errOut := h.run("context")
		wantExit(t, code, 0, out, errOut)
		want := "# mtqg context — " + filepath.Base(h.root) + " (main)\n" + `
This is the process record of this project. Read the following before you start working.
- Respect what has been decided (answered questions, memos stating a policy)
- Do not decide open questions on your own; confirm them
- Use terms as defined in the glossary
- Record questions, decisions, findings, bugs, and todos with mtqg as they come up

## Attention
- Glossary term "block comment" has conflicting definitions (see mtqg glossary list)
- 12 mtqg records are not committed

## Open todos (2)
- 6cad4a268d Skip block comments /* */ (claude-code, 10:18)
- 1e27a1c08a Show error positions as line and column (yamada, 11:06)

## Open questions (2)
- 1012f037b6 Should nested block comments be supported? (awaiting confirmation, claude-code, 09:10)
    ` + "└" + ` Not in the first version (yamada, human)
- 95e761d177 Should error positions show both line and column? (unanswered, claude-code, 11:05)

## Open bugs (1)
- 7f3a2b1c09 Parser crashes on empty input (awaiting confirmation, yamada, 10:41)
    ` + "└" + ` Reproduced on macOS too (claude-code, ai)

## Recent records (newest first)
- 11:24  claude-code  glossary  7a3c1d9e02  lexing: Reading source and turning it into tokens
- 11:06  yamada       todo      1e27a1c08a  Show error positions as line and column
- 11:05  claude-code  question  95e761d177  Should error positions show both line and column?
- 10:45  claude-code  reply     3d8e4a0b12  Reproduced on macOS too (to 7f3a2b1c09)
- 10:41  yamada       bug       7f3a2b1c09  Parser crashes on empty input
- 10:32  yamada       memo      81e74ef5e8  Policy: use English for all error messages
- 10:18  claude-code  todo      6cad4a268d  Skip block comments /* */
- 10:00  claude-code  glossary  0cb1e29c65  block comment: A comment that can span multiple lines
- 09:41  yamada       answer    a1a1a1a1a1  Not in the first version (to 1012f037b6)
- 09:30  yamada       glossary  f28c105d1f  block comment: A comment enclosed in /* and */
- (2 more; see mtqg log)

## Glossary (3)
- f28c105d1f block comment: A comment enclosed in /* and */
- 0cb1e29c65 block comment: A comment that can span multiple lines
- 7a3c1d9e02 lexing: Reading source and turning it into tokens

---
Read full entries with mtqg show <id>.
`
		if out != want {
			t.Errorf("stdout:\n%s\nwant:\n%s", out, want)
		}
	})

	t.Run("nothing is written", func(t *testing.T) {
		h := contextHarness(t)
		before := h.readJournal()
		h.run("context")
		h.run("--json", "context")
		if h.readJournal() != before {
			t.Error("context wrote to the journal")
		}
	})

	t.Run("an empty journal has the header, the instructions and the last line", func(t *testing.T) {
		h := initialized(t)
		_, out, _ := h.run("context")
		for _, section := range []string{"## Attention", "## Open todos", "## Open questions", "## Open bugs", "## Recent records", "## Glossary"} {
			if strings.Contains(out, section) {
				t.Errorf("an empty section %q is shown:\n%s", section, out)
			}
		}
		if !strings.HasPrefix(out, "# mtqg context") || !strings.HasSuffix(out, "---\nRead full entries with mtqg show <id>.\n") || !strings.Contains(out, "Do not decide open questions on your own") {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("without a name for the branch: no branch, and nothing about the commits when git is gone", func(t *testing.T) {
		h := contextHarness(t)
		git(t, h.root, "add", ".")
		git(t, h.root, "commit", "-q", "-m", "records")
		git(t, h.root, "checkout", "-q", "--detach")
		_, out, _ := h.run("context")
		first := strings.SplitN(out, "\n", 2)[0]
		if first != "# mtqg context — "+filepath.Base(h.root) || strings.Contains(out, "not committed") {
			t.Errorf("first line %q, and\n%s", first, out)
		}

		t.Setenv("PATH", t.TempDir())
		code, out, errOut := h.run("context")
		wantExit(t, code, 0, "", errOut)
		if !strings.Contains(errOut, "warning: git could not be run") || strings.Contains(out, "not committed") ||
			!strings.HasPrefix(out, "# mtqg context — "+filepath.Base(h.root)+"\n") {
			t.Errorf("stdout %q, stderr %q", out, errOut)
		}
	})

	t.Run("committed records are not counted, and the branch is the one that is checked out", func(t *testing.T) {
		h := contextHarness(t)
		git(t, h.root, "add", ".")
		git(t, h.root, "commit", "-q", "-m", "records")
		git(t, h.root, "checkout", "-q", "-b", "feature/x")
		h.run("memo", "add", "one more")
		_, out, _ := h.run("context")
		if !strings.HasPrefix(out, "# mtqg context — "+filepath.Base(h.root)+" (feature/x)\n") || !strings.Contains(out, "- 1 mtqg record is not committed\n") {
			t.Errorf("stdout:\n%s", out)
		}
	})

	t.Run("text is one line, cut, and made safe; and so are the words", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			record(idA, "todo", "First line\x1b[31m red\nsecond line", "yamada", "2026-09-17T10:18:00Z"),
			record(idC, "todo", strings.Repeat("x", 150), "yamada", "2026-09-17T10:19:00Z"),
			record(idM, "memo", strings.Repeat("あ", 120), "yamada", "2026-09-17T10:20:00Z"),
		)
		_, out, _ := h.run("context")
		if strings.Contains(out, "\x1b") || strings.Contains(out, "second line") || !strings.Contains(out, "First line�[31m red (yamada, 10:18)") {
			t.Errorf("a text was not made one safe line:\n%s", out)
		}
		if !strings.Contains(out, strings.Repeat("x", 97)+"... (yamada") || strings.Contains(out, strings.Repeat("x", 98)) {
			t.Errorf("a text was not cut to 100 characters:\n%s", out)
		}
		if !strings.Contains(out, strings.Repeat("あ", 97)+"...") || strings.Contains(out, strings.Repeat("あ", 98)) {
			t.Errorf("Japanese was not cut to 100 characters:\n%s", out)
		}
	})

	t.Run("hidden records are not in it", func(t *testing.T) {
		h := contextHarness(t)
		h.setJournal(
			question(idQ2, "Asked and deleted", nameC, "2026-09-17T09:10:00Z"),
			answerLine(idA2, idQ2, "An answer of it", "yamada", "human", "2026-09-17T09:41:00Z"),
			deleteLine(idQ2, "yamada", "2026-09-17T09:50:00Z"),
			record(idM, "memo", "kept", "yamada", "2026-09-17T10:32:00Z"),
		)
		_, out, _ := h.run("context")
		if strings.Contains(out, "Asked and deleted") || strings.Contains(out, "An answer of it") || strings.Contains(out, "## Open questions") || !strings.Contains(out, "kept") {
			t.Errorf("stdout:\n%s", out)
		}
	})

	t.Run("mistakes in the options", func(t *testing.T) {
		h := contextHarness(t)
		for _, tt := range []struct {
			args []string
			want string
		}{
			{[]string{"context", "--max-tokens", "-1"}, "Option --max-tokens needs a whole number of 0 or more"},
			{[]string{"context", "--max-tokens=many"}, `not "many"`},
			{[]string{"context", "--max-tokens"}, "Option --max-tokens needs a value"},
			{[]string{"context", "extra"}, "Too many arguments"},
		} {
			code, out, errOut := h.run(tt.args...)
			wantExit(t, code, 2, out, errOut)
			if !strings.Contains(errOut, tt.want) || out != "" {
				t.Errorf("%v: stdout %q, stderr %q", tt.args, out, errOut)
			}
		}
	})
}

// What bigContextHarness holds: more than any budget in the tests will hold.
const (
	bigTodos     = 60
	bigQuestions = 16
	bigAnswered  = 12 // of the questions, the first ones
	bigBugs      = 10 // all with a reply
	bigMemos     = 60
	bigWords     = 30
	bigRecords   = bigTodos + bigQuestions + bigAnswered + 2*bigBugs + bigMemos + bigWords
)

// bigContextHarness is a repository with bigTodos todos, bigQuestions questions
// and so on.
func bigContextHarness(t *testing.T) *harness {
	t.Helper()
	h := initialized(t)
	id := func(kind, n int) string { return fmt.Sprintf("%02x%030x", kind, n) }
	ts := func(minute int) string { return fmt.Sprintf("2026-09-16T%02d:%02d:00Z", 1+minute/60, minute%60) }
	var lines []string
	minute := 0
	next := func() string { minute++; return ts(minute) }
	for i := 0; i < bigTodos; i++ {
		lines = append(lines, record(id(1, i), "todo", fmt.Sprintf("Todo number %02d has a text of some length to it", i), "yamada", next()))
	}
	for i := 0; i < bigQuestions; i++ {
		lines = append(lines, question(id(2, i), fmt.Sprintf("Question number %02d asks something of some length?", i), nameC, next()))
		if i < bigAnswered {
			lines = append(lines, answerLine(id(3, i), id(2, i), fmt.Sprintf("Answer to question %02d says something of some length", i), "yamada", "human", next()))
		}
	}
	for i := 0; i < bigBugs; i++ {
		lines = append(lines, bugLine(id(4, i), fmt.Sprintf("Bug number %02d breaks something of some length", i), "yamada", next()))
		lines = append(lines, replyLine(id(5, i), id(4, i), fmt.Sprintf("Reply to bug %02d says something of some length", i), nameC, "ai", next()))
	}
	for i := 0; i < bigMemos; i++ {
		lines = append(lines, record(id(6, i), "memo", fmt.Sprintf("Memo number %02d has a text of some length to it", i), "yamada", next()))
	}
	for i := 0; i < bigWords; i++ {
		lines = append(lines, entryLine(id(7, i), fmt.Sprintf("word%02d", i), fmt.Sprintf("The definition of word %02d, of some length", i), "yamada", next()))
	}
	h.setJournal(lines...)
	return h
}

func TestContextBudget(t *testing.T) {
	h := bigContextHarness(t)

	full := mustRun(h, "context", "--max-tokens", "0")
	fullTokens := estimateTokens(full)
	if fullTokens < 2500 {
		t.Fatalf("the fixture is too small to be a test: %d tokens", fullTokens)
	}

	headings := []string{
		fmt.Sprintf("## Open todos (%d)", bigTodos), fmt.Sprintf("## Open questions (%d)", bigQuestions),
		fmt.Sprintf("## Open bugs (%d)", bigBugs), fmt.Sprintf("## Glossary (%d)", bigWords),
	}

	t.Run("no limit leaves nothing out but the recent records past ten", func(t *testing.T) {
		for _, left := range []string{"older; see", "definitions left out", "left out; see mtqg show"} {
			if strings.Contains(full, left) {
				t.Errorf("%q in the full text", left)
			}
		}
		if !strings.Contains(full, fmt.Sprintf("(%d more; see mtqg log)", bigRecords-10)) {
			t.Errorf("the recent records:\n%s", full)
		}
		for _, heading := range headings {
			if !strings.Contains(full, heading) {
				t.Errorf("no %q", heading)
			}
		}
	})

	t.Run("the default is 2000 tokens", func(t *testing.T) {
		out := mustRun(h, "context")
		if got := estimateTokens(out); got > 2000 || got < 1500 {
			t.Errorf("%d tokens for the default budget of 2000", got)
		}
		if out == full {
			t.Error("the default did not cut anything")
		}
	})

	t.Run("it fits, and the less there is room for, the less is shown", func(t *testing.T) {
		prev := fullTokens
		for _, budget := range []int{3000, 2000, 1500, 1000, 700, 500, 400} {
			out := mustRun(h, "context", "--max-tokens", fmt.Sprint(budget))
			got := estimateTokens(out)
			if got > budget {
				t.Errorf("budget %d: %d tokens", budget, got)
			}
			if got > prev {
				t.Errorf("budget %d has more (%d) than a larger budget (%d)", budget, got, prev)
			}
			prev = got
			// What is never left out, and what the headings count.
			for _, must := range append(headings, "# mtqg context", "This is the process record", "## Recent records (newest first)", "Read full entries with mtqg show <id>.") {
				if !strings.Contains(out, must) {
					t.Errorf("budget %d: %q is missing", budget, must)
				}
			}
		}
	})

	t.Run("things are left out in the order of the spec, and each says so", func(t *testing.T) {
		// As the budget falls, look at every output for what is left out: none of a
		// later cut is made before all of the earlier ones.
		reached := map[string]bool{}
		for budget := fullTokens; budget >= 150; budget -= 25 {
			out := mustRun(h, "context", "--max-tokens", fmt.Sprint(budget))
			f := contextFacts(out)
			for name, made := range map[string]bool{
				"recent":      f.recentShown < 10,
				"definitions": f.definitionsLeft,
				"replies":     f.repliesLeft,
				"threads":     f.threadsLeft,
				"todos":       f.todosLeft,
			} {
				reached[name] = reached[name] || made
			}
			if f.definitionsLeft && f.recentShown != 0 {
				t.Errorf("budget %d: definitions are left out while %d recent records are shown", budget, f.recentShown)
			}
			if f.repliesLeft && !f.definitionsLeft {
				t.Errorf("budget %d: answers are left out before the definitions", budget)
			}
			if f.threadsLeft && (f.replyLines != 0 || !f.definitionsLeft) {
				t.Errorf("budget %d: questions or bugs are left out while %d answers are shown (definitions left out: %v)", budget, f.replyLines, f.definitionsLeft)
			}
			if f.todosLeft && f.threadsShown != 0 {
				t.Errorf("budget %d: todos are left out while %d questions and bugs are shown", budget, f.threadsShown)
			}
		}
		for _, name := range []string{"recent", "definitions", "replies", "threads", "todos"} {
			if !reached[name] {
				t.Errorf("no budget made the cut of %s: the test proves nothing about it", name)
			}
		}
	})

	t.Run("what is cut is the oldest, and the newest stays", func(t *testing.T) {
		// In this fixture every question is older than every bug, so the questions go
		// first, oldest first, and only when they are all gone do the bugs.
		checked := map[string]bool{}
		for budget := fullTokens; budget >= 300; budget -= 10 {
			out := mustRun(h, "context", "--max-tokens", fmt.Sprint(budget))
			if k := olderCount(out, "qa list"); k > 0 && k < bigQuestions {
				checked["questions"] = true
				if strings.Contains(out, fmt.Sprintf("Question number %02d ", k-1)) || !strings.Contains(out, fmt.Sprintf("Question number %02d ", k)) ||
					!strings.Contains(out, fmt.Sprintf("Question number %02d ", bigQuestions-1)) || olderCount(out, "bug list") != 0 {
					t.Errorf("budget %d: %d older questions are left out, and that is not the oldest %d:\n%s", budget, k, k, out)
				}
			}
			if k := olderCount(out, "bug list"); k > 0 && k < bigBugs {
				checked["bugs"] = true
				if olderCount(out, "qa list") != bigQuestions || strings.Contains(out, fmt.Sprintf("Bug number %02d ", k-1)) || !strings.Contains(out, fmt.Sprintf("Bug number %02d ", k)) {
					t.Errorf("budget %d: %d older bugs are left out, and that is not the oldest %d:\n%s", budget, k, k, out)
				}
			}
			if k := olderCount(out, "todo list"); k > 0 && k < bigTodos {
				checked["todos"] = true
				if strings.Contains(out, fmt.Sprintf("Todo number %02d ", k-1)) || !strings.Contains(out, fmt.Sprintf("Todo number %02d ", k)) ||
					!strings.Contains(out, fmt.Sprintf("Todo number %02d ", bigTodos-1)) || contextFacts(out).threadsShown != 0 {
					t.Errorf("budget %d: %d older todos are left out, and that is not the oldest %d", budget, k, k)
				}
			}
		}
		for _, what := range []string{"questions", "bugs", "todos"} {
			if !checked[what] {
				t.Errorf("no budget left out some %s and kept others: the test proves nothing", what)
			}
		}
	})

	t.Run("more than the fixed parts can hold is printed as it is", func(t *testing.T) {
		code, out, errOut := h.run("context", "--max-tokens", "1")
		wantExit(t, code, 0, "", errOut)
		if !strings.Contains(out, "This is the process record") || !strings.Contains(out, fmt.Sprintf("## Open todos (%d)", bigTodos)) || strings.Contains(out, "Todo number") {
			t.Errorf("stdout:\n%s", out)
		}
	})
}

// olderCount is how many are said to be left out of a section, in "(N older; see
// mtqg <list>)", or 0 if nothing is.
func olderCount(out, list string) int {
	suffix := " older; see mtqg " + list + ")"
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "- (") && strings.HasSuffix(line, suffix) {
			n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(line, "- ("), suffix))
			if err == nil {
				return n
			}
		}
	}
	return 0
}

// contextFacts reads from the text of context what was left out of it.
type facts struct {
	recentShown     int  // how many recent records are shown
	threadsShown    int  // how many questions and bugs are shown
	replyLines      int  // how many latest answers and replies are shown
	definitionsLeft bool // the definitions of the glossary are left out
	repliesLeft     bool // the latest answers and replies are left out
	threadsLeft     bool // some questions or bugs are left out
	todosLeft       bool // some todos are left out
}

func contextFacts(out string) facts {
	var f facts
	section := ""
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "## "):
			section = line
		case section == "## Recent records (newest first)" && strings.HasPrefix(line, "- ") && !strings.HasPrefix(line, "- ("):
			f.recentShown++
		case strings.Contains(line, "\u2514 "):
			f.replyLines++
		case strings.Contains(line, "(unanswered,") || strings.Contains(line, "(no replies,") || strings.Contains(line, "(awaiting confirmation,"):
			f.threadsShown++
		}
	}
	f.definitionsLeft = strings.Contains(out, "- (definitions left out; see mtqg glossary list)")
	f.repliesLeft = strings.Contains(out, "left out; see mtqg show <id>)")
	f.threadsLeft = strings.Contains(out, "older; see mtqg qa list)") || strings.Contains(out, "older; see mtqg bug list)")
	f.todosLeft = strings.Contains(out, "older; see mtqg todo list)")
	return f
}

func TestContextJSON(t *testing.T) {
	h := contextHarness(t)

	t.Run("the same content, in sections that count what there is", func(t *testing.T) {
		code, out, errOut := h.run("--json", "context")
		wantExit(t, code, 0, out, errOut)
		obj := jsonObject(t, out)
		if obj["command"] != "context" || obj["repository"] != filepath.Base(h.root) || obj["branch"] != "main" || obj["truncated"] != false || obj["max_tokens"] != float64(2000) {
			t.Errorf("%v", obj)
		}
		if todos := field(t, obj, "open_todos").(map[string]any); todos["total"] != float64(2) || len(records(t, todos, "records")) != 2 {
			t.Errorf("open_todos %v", todos)
		}
		questions := records(t, field(t, obj, "open_questions").(map[string]any), "records")
		if len(questions) != 2 || questions[0]["id"] != idQ2 || questions[0]["reply_count"] != float64(1) || field(t, questions[0], "latest_reply", "id") != idA2 || questions[0]["status"] != "open" {
			t.Errorf("questions %v", questions)
		}
		if _, ok := questions[1]["latest_reply"]; ok || questions[1]["reply_count"] != float64(0) {
			t.Errorf("a question without answers: %v", questions[1])
		}
		bugs := records(t, field(t, obj, "open_bugs").(map[string]any), "records")
		if len(bugs) != 1 || field(t, bugs[0], "latest_reply", "kind") != "reply" {
			t.Errorf("bugs %v", bugs)
		}
		recent := field(t, obj, "recent").(map[string]any)
		if recent["total"] != float64(12) || len(records(t, recent, "records")) != 10 || records(t, recent, "records")[0]["id"] != idW3 {
			t.Errorf("recent %v", recent["total"])
		}
		glossary := records(t, field(t, obj, "glossary").(map[string]any), "records")
		if len(glossary) != 3 || glossary[0]["word"] != "block comment" || glossary[0]["definitions"] != float64(2) || glossary[0]["id"] != idW1 || glossary[2]["definitions"] != float64(1) {
			t.Errorf("glossary %v", glossary)
		}
		attention := records(t, obj, "attention")
		if len(attention) != 2 || attention[0]["kind"] != "duplicate_word" || attention[0]["word"] != "block comment" || attention[1]["kind"] != "uncommitted" || attention[1]["count"] != float64(12) {
			t.Errorf("attention %v", attention)
		}
	})

	t.Run("the estimate is the estimate of the text", func(t *testing.T) {
		text := mustRun(h, "context")
		obj := jsonObject(t, mustRun(h, "--json", "context"))
		if obj["estimated_tokens"] != float64(estimateTokens(text)) {
			t.Errorf("estimated_tokens = %v, the text is %d", obj["estimated_tokens"], estimateTokens(text))
		}
	})

	t.Run("no limit is null, and every section is there when it is empty", func(t *testing.T) {
		e := initialized(t)
		obj := jsonObject(t, mustRun(e, "--json", "context", "--max-tokens", "0"))
		if v, ok := obj["max_tokens"]; !ok || v != nil {
			t.Errorf("max_tokens = %v (present %v)", v, ok)
		}
		for _, key := range []string{"open_todos", "open_questions", "open_bugs", "recent", "glossary"} {
			s := field(t, obj, key).(map[string]any)
			if s["total"] != float64(0) || len(records(t, s, "records")) != 0 {
				t.Errorf("%s: %v", key, s)
			}
		}
		if a, ok := obj["attention"].([]any); !ok || len(a) != 0 {
			t.Errorf("attention %v", obj["attention"])
		}
	})

	t.Run("cut to the budget as the text is, and it says so", func(t *testing.T) {
		big := bigContextHarness(t)
		text := mustRun(big, "context", "--max-tokens", "600")
		obj := jsonObject(t, mustRun(big, "--json", "context", "--max-tokens", "600"))
		if obj["truncated"] != true || obj["max_tokens"] != float64(600) || obj["estimated_tokens"] != float64(estimateTokens(text)) || obj["estimated_tokens"].(float64) > 600 {
			t.Errorf("truncated %v, max %v, estimated %v", obj["truncated"], obj["max_tokens"], obj["estimated_tokens"])
		}
		todos := field(t, obj, "open_todos").(map[string]any)
		shown := len(records(t, todos, "records"))
		if todos["total"] != float64(bigTodos) || (shown < bigTodos && !strings.Contains(text, fmt.Sprintf("(%d older; see mtqg todo list)", bigTodos-shown))) {
			t.Errorf("todos: total %v, shown %d\n%s", todos["total"], shown, text)
		}
		// With the definitions left out a word has no id and no text.
		word := records(t, field(t, obj, "glossary").(map[string]any), "records")[0]
		if _, ok := word["id"]; ok {
			t.Errorf("a word without its definition has an id: %v", word)
		}
		if _, ok := word["text"]; ok || word["word"] == nil || word["definitions"] != float64(1) {
			t.Errorf("word %v", word)
		}
	})

	t.Run("text is as it was written", func(t *testing.T) {
		e := initialized(t)
		e.setJournal(record(idA, "todo", "A todo\x1b[31m with escapes\nand a second line "+strings.Repeat("x", 150), "yamada", "2026-09-17T10:18:00Z"))
		obj := jsonObject(t, mustRun(e, "--json", "context"))
		text := records(t, field(t, obj, "open_todos").(map[string]any), "records")[0]["text"].(string)
		if text != "A todo\x1b[31m with escapes\nand a second line "+strings.Repeat("x", 150) {
			t.Errorf("text %q", text)
		}
	})
}

func TestContextErrors(t *testing.T) {
	h := newHarness(t)
	code, out, errOut := h.run("context")
	wantExit(t, code, 1, out, errOut)
	if !strings.Contains(errOut, "No .mtqg/ found") {
		t.Errorf("stderr %q", errOut)
	}
	code, out, errOut = h.run("--json", "context")
	if code != 1 || out != "" || field(t, oneLineOfJSON(t, errOut), "error", "kind") != "not_initialized" {
		t.Errorf("exit %d, stdout %q, stderr %q", code, out, errOut)
	}
}
