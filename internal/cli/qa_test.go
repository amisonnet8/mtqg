package cli

import (
	"os"
	"strings"
	"testing"
)

// The fixtures of the questions, answers and glossary entries. The IDs of the
// questions do not share their first four digits with anything else.
const (
	idQ2  = "1012f037b64c44228c38fb2918f135d2"
	idQ3  = "95e761d177314f10b06bf2efc6f87718"
	idQ4  = "70430f77ff91c2e04a8b33f1d7e6a025"
	idA1  = "ae2eb1547f00411c8e9d3a7b5c6d7e81"
	idA2  = "a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1"
	idA3  = "c0ffee00c0ffee00c0ffee00c0ffee00"
	idW1  = "f28c105d1fb14c2390c192cfd3ac94af"
	idW2  = "0cb1e29c65a04b7d8f3e1c2d4b5a6978"
	idW3  = "7a3c1d9e027b4f6a8c5d0e1f2a3b4c5d"
	nameC = "claude-code"
)

// lineBy makes a line like record does, for an author of the given kind.
func lineBy(ev map[string]any, kind, author string) string {
	ev["v"] = 0
	ev["author"] = map[string]string{"kind": kind, "name": author}
	return marshal(ev)
}

func question(id, text, author, ts string) string {
	return lineBy(map[string]any{"id": id, "op": "create", "type": "qa", "status": "open", "text": text, "ts": ts}, "human", author)
}

func answerLine(id, re, text, author, kind, ts string) string {
	return lineBy(map[string]any{"id": id, "op": "create", "type": "qa", "re": re, "text": text, "ts": ts}, kind, author)
}

func entryLine(id, word, text, author, ts string) string {
	return lineBy(map[string]any{"id": id, "op": "create", "type": "glossary", "word": word, "text": text, "ts": ts}, "human", author)
}

func deleteLine(id, author, ts string) string {
	return lineBy(map[string]any{"id": id, "op": "delete", "ts": ts}, "human", author)
}

func TestAddQuestion(t *testing.T) {
	t.Run("a question, and only its ID is printed", func(t *testing.T) {
		h := initialized(t)
		code, out, errOut := h.run("q", "add", "Should", "nested", "block", "comments", "be", "supported?")
		wantExit(t, code, 0, out, errOut)
		if !shortIDPattern.MatchString(out) || errOut != "" {
			t.Errorf("stdout %q, stderr %q", out, errOut)
		}
		journal := h.readJournal()
		if !strings.Contains(journal, `"type":"qa","status":"open","text":"Should nested block comments be supported?"`) || strings.Contains(journal, `"re"`) {
			t.Errorf("journal = %s", journal)
		}
	})

	t.Run("a question does not read the journal", func(t *testing.T) {
		// A line that cannot be read would be warned about if the journal were read.
		h := initialized(t)
		h.setJournal("this is not json")
		code, out, errOut := h.run("q", "add", "Which", "parser?")
		wantExit(t, code, 0, out, errOut)
		if errOut != "" {
			t.Errorf("stderr %q: the journal was read", errOut)
		}
	})

	t.Run("a first word that is not an ID makes a question", func(t *testing.T) {
		// add and bad are hex digits, but fewer than four of them. Nothing here looks
		// like an ID, and the text is a question in full.
		for _, first := range []string{"Add", "bad", "abc", "Should", "日本語", "a8ec-1", "a8eg"} {
			h := initialized(t)
			code, out, errOut := h.run("q", "add", first, "is", "the", "first", "word")
			wantExit(t, code, 0, out, errOut)
			if !strings.Contains(h.readJournal(), `"text":"`+first+` is the first word"`) || strings.Contains(h.readJournal(), `"re"`) {
				t.Errorf("%q: journal = %s", first, h.readJournal())
			}
		}
	})

	t.Run("a text in quotes is one word, and never an ID", func(t *testing.T) {
		h := initialized(t)
		code, out, errOut := h.run("q", "add", "a8ec What does this mean?")
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(h.readJournal(), `"text":"a8ec What does this mean?"`) || strings.Contains(h.readJournal(), `"re"`) {
			t.Errorf("journal = %s", h.readJournal())
		}
	})

	t.Run("standard input, and the editor with no arguments at all", func(t *testing.T) {
		h := initialized(t)
		h.stdin = "face\n"
		code, out, errOut := h.run("q", "add", "-")
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(h.readJournal(), `"text":"face"`) || strings.Contains(h.readJournal(), `"re"`) {
			t.Errorf("a question of one word that looks like an ID can come from standard input: %s", h.readJournal())
		}

		h = initialized(t)
		h.vars["EDITOR"] = "myeditor"
		h.env.RunEditor = func(argv []string) error {
			return os.WriteFile(argv[len(argv)-1], []byte("A long question\nover two lines?\n"), 0o600)
		}
		code, out, errOut = h.run("q", "add")
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(h.readJournal(), `"text":"A long question\nover two lines?"`) {
			t.Errorf("journal = %s", h.readJournal())
		}
	})
}

func TestAddAnswer(t *testing.T) {
	setup := func(t *testing.T) *harness {
		h := initialized(t)
		h.setJournal(
			question(idQ2, "Should nested block comments be supported?", "yamada", "2026-09-17T09:10:00Z"),
			question(idQ3, "Which license?", "yamada", "2026-09-17T09:20:00Z"),
			change(idQ3, "open", "done", "yamada", "2026-09-17T09:30:00Z"),
			record(idM, "memo", "a memo", "yamada", "2026-09-17T09:40:00Z"),
			answerLine(idA1, idQ2, "Supporting them is generally preferable", nameC, "ai", "2026-09-17T09:15:00Z"),
		)
		return h
	}

	t.Run("an answer holds the full ID of its question", func(t *testing.T) {
		for _, typed := range []string{idQ2[:4], idQ2[:10], strings.ToUpper(idQ2[:6]), idQ2} {
			h := setup(t)
			before := h.readJournal()
			code, out, errOut := h.run("q", "add", typed, "Not", "in", "the", "first", "version")
			wantExit(t, code, 0, out, errOut)
			if !shortIDPattern.MatchString(out) || errOut != "" {
				t.Errorf("%s: stdout %q, stderr %q", typed, out, errOut)
			}
			added := strings.TrimPrefix(h.readJournal(), before)
			if !strings.Contains(added, `"type":"qa","re":"`+idQ2+`","text":"Not in the first version"`) || strings.Contains(added, `"status"`) {
				t.Errorf("%s: the new line = %s", typed, added)
			}
		}
	})

	t.Run("the text can come from standard input", func(t *testing.T) {
		h := setup(t)
		h.stdin = "A long answer\nover two lines\n"
		code, out, errOut := h.run("q", "add", idQ2[:4], "-")
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(h.readJournal(), `"re":"`+idQ2+`","text":"A long answer\nover two lines"`) {
			t.Errorf("journal = %s", h.readJournal())
		}
	})

	t.Run("a closed question can still be answered", func(t *testing.T) {
		h := setup(t)
		code, out, errOut := h.run("q", "add", idQ3[:4], "Apache-2.0", "was", "chosen")
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(h.readJournal(), `"re":"`+idQ3+`"`) {
			t.Errorf("journal = %s", h.readJournal())
		}
	})

	t.Run("the ID and no text is a mistake, and the editor is not opened", func(t *testing.T) {
		h := setup(t)
		h.vars["EDITOR"] = "myeditor"
		opened := false
		h.env.RunEditor = func([]string) error { opened = true; return nil }
		before := h.readJournal()
		code, out, errOut := h.run("q", "add", idQ2[:6])
		wantExit(t, code, 2, out, errOut)
		if !strings.Contains(errOut, "Missing argument. Usage: mtqg qa add") || out != "" || opened || h.readJournal() != before {
			t.Errorf("stdout %q, stderr %q, editor opened: %v", out, errOut, opened)
		}
	})

	t.Run("an ID that matches nothing stops, and says how to ask the question", func(t *testing.T) {
		for _, tt := range []struct {
			words []string
			word  string
		}{
			{[]string{"a8ec", "What", "does", "this", "mean?"}, "a8ec"},
			{[]string{"face", "detection", "is", "slow.", "Why?"}, "face"},
			{[]string{"1012f037XX", "x"}, ""}, // not hex at all: a question
			{[]string{idQ2[:10][:9] + "0", "typo"}, idQ2[:9] + "0"},
		} {
			h := setup(t)
			before := h.readJournal()
			code, out, errOut := h.run(append([]string{"q", "add"}, tt.words...)...)
			if tt.word == "" {
				wantExit(t, code, 0, out, errOut) // 1012f037XX is not made of hex digits
				continue
			}
			wantExit(t, code, 1, out, errOut)
			want := "No record matches \"" + tt.word + "\". A first word of 4 or more hex digits is read as the ID of the question to answer.\n" +
				"To ask a question that starts with it, put the whole text in quotes: mtqg qa add \"" + tt.word + " ...\"\n"
			if errOut != want || out != "" || h.readJournal() != before {
				t.Errorf("%v:\nstderr %q\nwant   %q\nstdout %q, journal changed: %v", tt.words, errOut, want, out, h.readJournal() != before)
			}
		}
	})

	t.Run("a record that is not a question is named", func(t *testing.T) {
		h := setup(t)
		before := h.readJournal()
		code, out, errOut := h.run("q", "add", idM[:6], "an", "answer")
		wantExit(t, code, 1, out, errOut)
		if errOut != "81e74ef5e8 is a memo, not a question\n" || h.readJournal() != before {
			t.Errorf("stderr %q", errOut)
		}
		code, out, errOut = h.run("q", "add", idA1[:6], "an", "answer", "to", "an", "answer")
		wantExit(t, code, 1, out, errOut)
		if errOut != "ae2eb1547f is an answer, not a question\n" || h.readJournal() != before {
			t.Errorf("stderr %q", errOut)
		}
	})

	t.Run("an ambiguous ID lists the candidates with their full IDs", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			question("6cad4a268d0f4e2f8c1b7a3d5e9f0a11", "First?", "yamada", "2026-09-17T09:10:00Z"),
			question("6cad4a26ffff4e2f8c1b7a3d5e9f0a22", "Second?", "yamada", "2026-09-17T09:20:00Z"),
		)
		before := h.readJournal()
		code, out, errOut := h.run("q", "add", "6cad", "an", "answer")
		wantExit(t, code, 1, out, errOut)
		if !strings.Contains(errOut, `Ambiguous ID "6cad" matches 2 records:`) ||
			!strings.Contains(errOut, "6cad4a268d0f4e2f8c1b7a3d5e9f0a11") || !strings.Contains(errOut, "6cad4a26ffff4e2f8c1b7a3d5e9f0a22") ||
			h.readJournal() != before {
			t.Errorf("stderr %q", errOut)
		}
	})

	t.Run("a deleted question cannot be answered", func(t *testing.T) {
		h := setup(t)
		h.setJournal(
			question(idQ2, "Should nested block comments be supported?", "yamada", "2026-09-17T09:10:00Z"),
			deleteLine(idQ2, "yamada", "2026-09-17T09:12:00Z"),
		)
		code, _, errOut := h.run("q", "add", idQ2[:6], "late", "answer")
		if code != 1 || !strings.Contains(errOut, `No record matches "`+idQ2[:6]+`"`) {
			t.Errorf("exit %d, stderr %q", code, errOut)
		}
	})

	t.Run("no .mtqg/ or no author is reported before anything is read", func(t *testing.T) {
		h := newHarness(t)
		code, _, errOut := h.run("q", "add", "a8ec", "x")
		if code != 1 || !strings.Contains(errOut, "No .mtqg/ found") {
			t.Errorf("exit %d, stderr %q", code, errOut)
		}
	})
}

func TestListQA(t *testing.T) {
	t.Run("the open questions, each with its state and the latest answer under it", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			question(idQ2, "Nested?", "yamada", "2026-09-17T09:10:00Z"),
			answerLine(idA1, idQ2, "Preferable", nameC, "ai", "2026-09-17T09:15:00Z"),
			answerLine(idA2, idQ2, "Not now", "yamada", "human", "2026-09-17T09:41:00Z"),
			question(idQ, "Which parser?", nameC, "2026-09-17T11:05:00Z"),
			question(idQ4, "Is this needed?", "yamada", "2026-09-15T10:00:00Z"),
			change(idQ4, "open", "done", "yamada", "2026-09-15T10:30:00Z"),
		)
		code, out, errOut := h.run("q", "list")
		wantExit(t, code, 0, out, errOut)
		want := "1012f037b6  Nested?        yamada       09:10  2 answers, awaiting confirmation\n" +
			"          └ Not now        yamada       09:41\n" +
			"2217beaddb  Which parser?  claude-code  11:05  unanswered\n" +
			"2 open (show done: --all)\n"
		if out != want || errOut != "" {
			t.Errorf("stdout:\n%s\nwant:\n%s\nstderr %q", out, want, errOut)
		}
	})

	t.Run("--all shows the closed ones with their states", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			question(idQ4, "Is this needed?", "yamada", "2026-09-15T10:00:00Z"),
			change(idQ4, "open", "done", "yamada", "2026-09-15T10:30:00Z"),
			question(idQ3, "Which license?", "yamada", "2026-09-16T08:00:00Z"),
			answerLine(idA3, idQ3, "MIT", nameC, "ai", "2026-09-16T08:30:00Z"),
			change(idQ3, "open", "done", "yamada", "2026-09-16T09:00:00Z"),
			question(idQ, "Which parser?", nameC, "2026-09-17T11:05:00Z"),
		)
		code, out, errOut := h.run("q", "list", "--all")
		wantExit(t, code, 0, out, errOut)
		want := "70430f77ff  Is this needed?  yamada       2026-09-15  done without answers\n" +
			"95e761d177  Which license?   yamada       2026-09-16  1 answer, done\n" +
			"          └ MIT              claude-code  2026-09-16\n" +
			"2217beaddb  Which parser?    claude-code  11:05       unanswered\n" +
			"1 open, 2 done\n"
		if out != want || errOut != "" {
			t.Errorf("stdout:\n%s\nwant:\n%s\nstderr %q", out, want, errOut)
		}
	})

	t.Run("the latest answer is the newest by time, not the last line", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			question(idQ2, "Nested?", "yamada", "2026-09-17T09:10:00Z"),
			answerLine(idA2, idQ2, "The newer one", "yamada", "human", "2026-09-17T09:41:00Z"),
			answerLine(idA1, idQ2, "The older one", nameC, "ai", "2026-09-17T09:15:00Z"), // a merge put it after
		)
		_, out, _ := h.run("q", "list")
		if !strings.Contains(out, "The newer one") || strings.Contains(out, "The older one") {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("answers of a deleted question, and answers without a question, are not listed", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			question(idQ2, "Nested?", "yamada", "2026-09-17T09:10:00Z"),
			answerLine(idA1, idQ2, "An answer of a deleted question", nameC, "ai", "2026-09-17T09:15:00Z"),
			deleteLine(idQ2, "yamada", "2026-09-17T09:20:00Z"),
			answerLine(idA2, idQ3, "An answer whose question is somewhere else", nameC, "ai", "2026-09-17T09:30:00Z"),
			question(idQ, "Which parser?", nameC, "2026-09-17T11:05:00Z"),
		)
		code, out, errOut := h.run("q", "list", "--all")
		wantExit(t, code, 0, out, errOut)
		if out != "2217beaddb  Which parser?  claude-code  11:05  unanswered\n1 open, 0 done\n" {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("Japanese texts line up by display width", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			question(idQ2, "ブロックコメントの入れ子に対応する？", "yamada", "2026-09-17T09:10:00Z"),
			answerLine(idA2, idQ2, "初版では非対応。需要が出たら再検討", "yamada", "human", "2026-09-17T09:41:00Z"),
			question(idQ, "エラー位置は行と列の両方を出しますか？", nameC, "2026-09-17T11:05:00Z"),
		)
		_, out, _ := h.run("q", "list")
		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		if len(lines) != 4 {
			t.Fatalf("stdout %q", out)
		}
		var cols []int
		for i, author := range []string{"yamada", "yamada", nameC} {
			line := lines[i]
			cols = append(cols, displayWidth(line[:strings.Index(line, author)]))
		}
		if cols[0] != cols[1] || cols[1] != cols[2] {
			t.Errorf("the author starts at columns %v:\n%s", cols, out)
		}
		// The answer's text starts where the question's text starts.
		if a, q := displayWidth(lines[1][:strings.Index(lines[1], "初版")]), displayWidth(lines[0][:strings.Index(lines[0], "ブロック")]); a != q {
			t.Errorf("the answer starts at column %d, the question at %d:\n%s", a, q, out)
		}
	})

	t.Run("on a terminal the text is cut to the window, and the answer too", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			question(idQ2, strings.Repeat("Q", 80), "yamada", "2026-09-17T09:10:00Z"),
			answerLine(idA2, idQ2, strings.Repeat("A", 80), "yamada", "human", "2026-09-17T09:41:00Z"),
		)
		h.env.StdoutIsTerminal, h.env.StdoutWidth = true, 90
		_, out, _ := h.run("q", "list")
		for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
			if displayWidth(line) > 89 {
				t.Errorf("a line is %d columns: %q", displayWidth(line), line)
			}
		}
		if !strings.Contains(out, "...") {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("an empty list", func(t *testing.T) {
		h := initialized(t)
		code, out, errOut := h.run("q", "list")
		wantExit(t, code, 0, out, errOut)
		if out != "0 open (show done: --all)\n" {
			t.Errorf("stdout %q", out)
		}
	})
}

func TestDoneAndReopenAQuestion(t *testing.T) {
	setup := func(t *testing.T) *harness {
		h := initialized(t)
		h.setJournal(
			question(idQ, "Which parser?", nameC, "2026-09-17T11:05:00Z"),
			record(idA, "todo", "Skip block comments", "yamada", "2026-09-17T10:18:00Z"),
			record(idM, "memo", "a memo", "yamada", "2026-09-17T09:40:00Z"),
			answerLine(idA1, idQ, "PEG", nameC, "ai", "2026-09-17T11:10:00Z"),
		)
		return h
	}

	t.Run("done, again, and reopen", func(t *testing.T) {
		h := setup(t)
		code, out, errOut := h.run("q", "done", idQ[:4])
		wantExit(t, code, 0, out, errOut)
		if out != "Done: 2217beaddb  Which parser?\n" {
			t.Errorf("stdout %q", out)
		}
		if !strings.Contains(h.readJournal(), `"id":"`+idQ+`","op":"status","from":"open","status":"done"`) {
			t.Errorf("journal = %s", h.readJournal())
		}
		before := h.readJournal()
		code, out, errOut = h.run("q", "done", idQ[:4])
		wantExit(t, code, 0, out, errOut)
		if out != "Already done: 2217beaddb  Which parser?\n" || h.readJournal() != before {
			t.Errorf("stdout %q", out)
		}
		code, out, errOut = h.run("q", "reopen", idQ[:4])
		wantExit(t, code, 0, out, errOut)
		if out != "Reopened: 2217beaddb  Which parser?\n" {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("a record of another kind names the right command", func(t *testing.T) {
		h := setup(t)
		code, out, errOut := h.run("q", "done", idA[:6])
		wantExit(t, code, 1, out, errOut)
		if errOut != "6cad4a268d is a todo, not a question; use `mtqg todo done 6cad4a268d`\n" {
			t.Errorf("stderr %q", errOut)
		}
		code, out, errOut = h.run("t", "done", idQ[:6])
		wantExit(t, code, 1, out, errOut)
		if errOut != "2217beaddb is a question, not a todo; use `mtqg qa done 2217beaddb`\n" {
			t.Errorf("stderr %q", errOut)
		}
		code, out, errOut = h.run("q", "done", idM[:6])
		wantExit(t, code, 1, out, errOut)
		if errOut != "81e74ef5e8 is a memo, not a question\n" {
			t.Errorf("stderr %q", errOut)
		}
		// An answer has no state either.
		code, out, errOut = h.run("q", "done", idA1[:6])
		wantExit(t, code, 1, out, errOut)
		if errOut != "ae2eb1547f is an answer, not a question\n" {
			t.Errorf("stderr %q", errOut)
		}
	})

	t.Run("an ID of fewer than four digits is too short", func(t *testing.T) {
		h := setup(t)
		for _, args := range [][]string{{"q", "done", "221"}, {"t", "done", "6"}} {
			code, out, errOut := h.run(args...)
			wantExit(t, code, 1, out, errOut)
			if !strings.Contains(errOut, "is too short: give at least 4 digits") {
				t.Errorf("%v: stderr %q", args, errOut)
			}
		}
	})
}

func TestGlossary(t *testing.T) {
	t.Run("the first word is the term, and the rest its definition", func(t *testing.T) {
		h := initialized(t)
		code, out, errOut := h.run("g", "add", "token", "The", "smallest", "unit", "produced", "by", "lexing")
		wantExit(t, code, 0, out, errOut)
		if !shortIDPattern.MatchString(out) || errOut != "" {
			t.Errorf("stdout %q, stderr %q", out, errOut)
		}
		if !strings.Contains(h.readJournal(), `"type":"glossary","word":"token","text":"The smallest unit produced by lexing"`) {
			t.Errorf("journal = %s", h.readJournal())
		}
		// A term of several words needs quotes.
		code, out, errOut = h.run("g", "add", "block comment", "A", "comment", "enclosed", "in", "/*", "and", "*/")
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(h.readJournal(), `"word":"block comment","text":"A comment enclosed in /* and */"`) {
			t.Errorf("journal = %s", h.readJournal())
		}
	})

	t.Run("a definition from standard input", func(t *testing.T) {
		h := initialized(t)
		h.stdin = "Reading source\nand making tokens\n"
		code, out, errOut := h.run("g", "add", "lexing", "-")
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(h.readJournal(), `"word":"lexing","text":"Reading source\nand making tokens"`) {
			t.Errorf("journal = %s", h.readJournal())
		}
	})

	t.Run("a missing part is a mistake, and the editor is not opened", func(t *testing.T) {
		h := initialized(t)
		h.vars["EDITOR"] = "myeditor"
		opened := false
		h.env.RunEditor = func([]string) error { opened = true; return nil }
		for _, args := range [][]string{{"g", "add"}, {"g", "add", "token"}} {
			code, out, errOut := h.run(args...)
			wantExit(t, code, 2, out, errOut)
			if !strings.Contains(errOut, "Missing argument. Usage: mtqg glossary add <word> <definition>") || out != "" {
				t.Errorf("%v: stdout %q, stderr %q", args, out, errOut)
			}
		}
		if opened || h.readJournal() != "" {
			t.Errorf("editor opened: %v, journal %q", opened, h.readJournal())
		}
	})

	t.Run("an empty word or definition writes nothing", func(t *testing.T) {
		h := initialized(t)
		code, out, errOut := h.run("g", "add", " ", "a definition")
		wantExit(t, code, 1, out, errOut)
		if errOut != "Aborting: the word is empty\n" {
			t.Errorf("stderr %q", errOut)
		}
		h.stdin = "\n"
		code, out, errOut = h.run("g", "add", "token", "-")
		wantExit(t, code, 1, out, errOut)
		if errOut != "Aborting: the text is empty\n" || h.readJournal() != "" {
			t.Errorf("stderr %q, journal %q", errOut, h.readJournal())
		}
	})

	t.Run("the list shows every entry, and says which words are defined twice", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			entryLine(idW1, "token", "The smallest unit produced by lexing", "yamada", "2026-09-17T09:00:00Z"),
			entryLine(idW2, "block comment", "A comment enclosed in /* and */", "yamada", "2026-09-17T09:30:00Z"),
			entryLine(idW3, "block comment", "A comment that can span lines", nameC, "2026-09-17T10:00:00Z"),
			entryLine("d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1d1", "Token", "The case differs: another word", "yamada", "2026-09-16T10:00:00Z"),
		)
		code, out, errOut := h.run("g", "list")
		wantExit(t, code, 0, out, errOut)
		want := "d1d1d1d1d1  Token          The case differs: another word        yamada       2026-09-16\n" +
			"f28c105d1f  token          The smallest unit produced by lexing  yamada       09:00\n" +
			"0cb1e29c65  block comment  A comment enclosed in /* and */       yamada       09:30\n" +
			"7a3c1d9e02  block comment  A comment that can span lines         claude-code  10:00\n" +
			"4 terms (1 with duplicate definitions)\n"
		if out != want || errOut != "" {
			t.Errorf("stdout:\n%s\nwant:\n%s\nstderr %q", out, want, errOut)
		}
	})

	t.Run("no duplicates, one term, none", func(t *testing.T) {
		h := initialized(t)
		_, out, _ := h.run("g", "list")
		if out != "0 terms\n" {
			t.Errorf("stdout %q", out)
		}
		h.setJournal(entryLine(idW1, "token", "d", "yamada", "2026-09-17T09:00:00Z"))
		_, out, _ = h.run("g", "list")
		if out != "f28c105d1f  token  d  yamada  09:00\n1 term\n" {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("Japanese words and definitions line up", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			entryLine(idW1, "トークン", "字句解析で切り出す最小単位", "yamada", "2026-09-17T09:00:00Z"),
			entryLine(idW2, "lexer", "字句解析を行う実装", nameC, "2026-09-17T09:30:00Z"),
		)
		_, out, _ := h.run("g", "list")
		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		a := displayWidth(lines[0][:strings.Index(lines[0], "字句")])
		b := displayWidth(lines[1][:strings.Index(lines[1], "字句")])
		if a != b {
			t.Errorf("the definitions start at columns %d and %d:\n%s", a, b, out)
		}
	})
}

func TestStatusCountsQuestionsAndGlossary(t *testing.T) {
	h := initialized(t)
	h.setJournal(
		question(idQ, "Which parser?", nameC, "2026-09-17T11:05:00Z"),
		question(idQ2, "Nested?", "yamada", "2026-09-17T09:10:00Z"),
		answerLine(idA1, idQ2, "Preferable", nameC, "ai", "2026-09-17T09:15:00Z"),
		question(idQ3, "Which license?", "yamada", "2026-09-16T08:00:00Z"),
		change(idQ3, "open", "done", "yamada", "2026-09-16T09:00:00Z"),
		entryLine(idW1, "token", "d1", "yamada", "2026-09-17T09:00:00Z"),
		entryLine(idW2, "block comment", "d2", "yamada", "2026-09-17T09:30:00Z"),
		entryLine(idW3, "block comment", "d3", nameC, "2026-09-17T10:00:00Z"),
	)
	code, out, errOut := h.run("status")
	wantExit(t, code, 0, out, errOut)
	want := "Open todos          0\n" +
		"Open questions      2  (1 awaiting confirmation)\n" +
		"Glossary            3  (1 with duplicate definitions)\n" +
		"\n" +
		"Uncommitted records 7\n"
	if out != want {
		t.Errorf("stdout:\n%s\nwant:\n%s", out, want)
	}
}
