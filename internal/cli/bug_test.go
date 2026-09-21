package cli

import (
	"os"
	"strings"
	"testing"
)

// Bugs and their replies are handled by the code that handles questions and their
// answers (thread.go). These tests are for what is particular to bugs, and for
// keeping the two kinds apart.

const (
	idBug1 = "7f3a2b1c09d84e6fa5b17c2d3e4f5a60"
	idBug2 = "8e4b3c2d10e95f70b6c28d3e4f5a6b71"
	idBug3 = "b2c3d4e5f6a74b8c9d0e1f2a3b4c5d6e"
	idRep1 = "3d8e4a0b12c94f77b6a08d1e5f2c9b34"
	idRep2 = "4c9d5b1e23fa4a88c7b19e2f6a3d0c45"
	idRep3 = "d4e5f6a7b8c94d0e1f2a3b4c5d6e7f80"
)

func bugLine(id, text, author, ts string) string {
	return lineBy(map[string]any{"id": id, "op": "create", "type": "bug", "status": "open", "text": text, "ts": ts}, "human", author)
}

func replyLine(id, re, text, author, kind, ts string) string {
	return lineBy(map[string]any{"id": id, "op": "create", "type": "bug", "re": re, "text": text, "ts": ts}, kind, author)
}

func TestAddBug(t *testing.T) {
	t.Run("a bug, and only its ID is printed", func(t *testing.T) {
		for _, kind := range []string{"bug", "b"} {
			h := initialized(t)
			code, out, errOut := h.run(kind, "add", "Parser", "crashes", "on", "empty", "input")
			wantExit(t, code, 0, out, errOut)
			if !shortIDPattern.MatchString(out) || errOut != "" {
				t.Errorf("%s: stdout %q, stderr %q", kind, out, errOut)
			}
			journal := h.readJournal()
			if !strings.Contains(journal, `"type":"bug","status":"open","text":"Parser crashes on empty input"`) || strings.Contains(journal, `"re"`) {
				t.Errorf("%s: journal = %s", kind, journal)
			}
		}
	})

	t.Run("a bug does not read the journal", func(t *testing.T) {
		h := initialized(t)
		h.setJournal("this is not json")
		code, out, errOut := h.run("b", "add", "Crash", "on", "start")
		wantExit(t, code, 0, out, errOut)
		if errOut != "" {
			t.Errorf("stderr %q: the journal was read", errOut)
		}
	})

	t.Run("a first word that is not an ID makes a bug, and quotes make one of any text", func(t *testing.T) {
		for _, words := range [][]string{{"Add", "bad", "text"}, {"abc", "x"}, {"a8ec What does this do?"}, {"日本語", "の", "不具合"}} {
			h := initialized(t)
			code, out, errOut := h.run(append([]string{"b", "add"}, words...)...)
			wantExit(t, code, 0, out, errOut)
			if !strings.Contains(h.readJournal(), `"type":"bug","status":"open"`) || strings.Contains(h.readJournal(), `"re"`) {
				t.Errorf("%v: journal = %s", words, h.readJournal())
			}
		}
	})

	t.Run("the editor opens with no arguments at all", func(t *testing.T) {
		h := initialized(t)
		h.vars["EDITOR"] = "myeditor"
		h.env.RunEditor = func(argv []string) error {
			return os.WriteFile(argv[len(argv)-1], []byte("Crash on start\nwith an empty file\n"), 0o600)
		}
		code, out, errOut := h.run("b", "add")
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(h.readJournal(), `"type":"bug","status":"open","text":"Crash on start\nwith an empty file"`) {
			t.Errorf("journal = %s", h.readJournal())
		}
	})
}

func TestReplyToABug(t *testing.T) {
	setup := func(t *testing.T) *harness {
		h := initialized(t)
		h.setJournal(
			bugLine(idBug1, "Parser crashes on empty input", "yamada", "2026-09-17T09:10:00Z"),
			bugLine(idBug2, "Error position is off by one", "yamada", "2026-09-17T09:20:00Z"),
			change(idBug2, "open", "done", "yamada", "2026-09-17T09:30:00Z"),
			question(idQ2, "Should nested block comments be supported?", "yamada", "2026-09-17T09:35:00Z"),
			replyLine(idRep1, idBug1, "Reproduced on macOS too", nameC, "ai", "2026-09-17T09:15:00Z"),
		)
		return h
	}

	t.Run("a reply is a bug record with the full ID of its bug in re, and no status", func(t *testing.T) {
		for _, typed := range []string{idBug1[:4], idBug1[:10], strings.ToUpper(idBug1[:6]), idBug1} {
			h := setup(t)
			before := h.readJournal()
			code, out, errOut := h.run("b", "add", typed, "The", "empty", "file", "has", "no", "first", "token")
			wantExit(t, code, 0, out, errOut)
			if !shortIDPattern.MatchString(out) || errOut != "" {
				t.Errorf("%s: stdout %q, stderr %q", typed, out, errOut)
			}
			added := strings.TrimPrefix(h.readJournal(), before)
			if !strings.Contains(added, `"type":"bug","re":"`+idBug1+`","text":"The empty file has no first token"`) || strings.Contains(added, `"status"`) {
				t.Errorf("%s: the new line = %s", typed, added)
			}
		}
	})

	t.Run("a closed bug can still be replied to", func(t *testing.T) {
		h := setup(t)
		code, out, errOut := h.run("b", "add", idBug2[:4], "Fixed", "in", "the", "next", "build")
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(h.readJournal(), `"re":"`+idBug2+`"`) {
			t.Errorf("journal = %s", h.readJournal())
		}
	})

	t.Run("the ID and no text opens the editor for the reply", func(t *testing.T) {
		h := setup(t)
		h.vars["EDITOR"] = "myeditor"
		h.env.RunEditor = func(argv []string) error {
			return os.WriteFile(argv[len(argv)-1], []byte("Reproduced on macOS\nas well\n"), 0o600)
		}
		code, out, errOut := h.run("b", "add", idBug1[:6])
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(h.readJournal(), `"re":"`+idBug1+`","text":"Reproduced on macOS\nas well"`) {
			t.Errorf("journal = %s", h.readJournal())
		}
	})

	t.Run("an ID that matches nothing stops before the editor is opened", func(t *testing.T) {
		h := setup(t)
		h.vars["EDITOR"] = "myeditor"
		opened := false
		h.env.RunEditor = func([]string) error { opened = true; return nil }
		before := h.readJournal()
		code, out, errOut := h.run("b", "add", "a8ec")
		wantExit(t, code, 1, out, errOut)
		if opened || out != "" || h.readJournal() != before || !strings.Contains(errOut, `No record matches "a8ec"`) {
			t.Errorf("stdout %q, stderr %q, editor opened: %v", out, errOut, opened)
		}
	})

	t.Run("an ID that matches nothing stops, and says how to report the bug", func(t *testing.T) {
		h := setup(t)
		before := h.readJournal()
		code, out, errOut := h.run("bug", "add", "a8ec", "Crashes", "on", "empty", "input")
		wantExit(t, code, 1, out, errOut)
		want := "No record matches \"a8ec\". A first word of 4 or more hex digits is read as the ID of the bug to reply to.\n" +
			"To report a bug that starts with it, put the whole text in quotes: mtqg bug add \"a8ec ...\"\n"
		if errOut != want || out != "" || h.readJournal() != before {
			t.Errorf("stderr %q\nwant   %q\nstdout %q, journal changed: %v", errOut, want, out, h.readJournal() != before)
		}
	})

	t.Run("a question is not a bug, and a bug is not a question", func(t *testing.T) {
		h := setup(t)
		before := h.readJournal()
		code, out, errOut := h.run("b", "add", idQ2[:6], "a", "reply")
		wantExit(t, code, 1, out, errOut)
		if errOut != "1012f037b6 is a question, not a bug\n" || h.readJournal() != before {
			t.Errorf("stderr %q", errOut)
		}
		code, out, errOut = h.run("q", "add", idBug1[:6], "an", "answer")
		wantExit(t, code, 1, out, errOut)
		if errOut != "7f3a2b1c09 is a bug, not a question\n" || h.readJournal() != before {
			t.Errorf("stderr %q", errOut)
		}
	})

	t.Run("a reply cannot be replied to", func(t *testing.T) {
		h := setup(t)
		before := h.readJournal()
		code, out, errOut := h.run("b", "add", idRep1[:6], "a", "reply", "to", "a", "reply")
		wantExit(t, code, 1, out, errOut)
		if errOut != "3d8e4a0b12 is a reply, not a bug\n" || h.readJournal() != before {
			t.Errorf("stderr %q", errOut)
		}
	})

	t.Run("a deleted bug cannot be replied to", func(t *testing.T) {
		h := setup(t)
		h.setJournal(
			bugLine(idBug1, "Parser crashes on empty input", "yamada", "2026-09-17T09:10:00Z"),
			deleteLine(idBug1, "yamada", "2026-09-17T09:12:00Z"),
		)
		code, _, errOut := h.run("b", "add", idBug1[:6], "late", "reply")
		if code != 1 || !strings.Contains(errOut, `No record matches "`+idBug1[:6]+`"`) {
			t.Errorf("exit %d, stderr %q", code, errOut)
		}
	})
}

func TestListBugs(t *testing.T) {
	fixture := func(h *harness) {
		h.setJournal(
			bugLine(idBug1, "Parser crashes on empty input", "yamada", "2026-09-17T09:10:00Z"),
			replyLine(idRep1, idBug1, "Reproduced on macOS too", nameC, "ai", "2026-09-17T09:15:00Z"),
			replyLine(idRep2, idBug1, "The empty file has no first token", "yamada", "human", "2026-09-17T09:41:00Z"),
			bugLine(idBug2, "Error position is off by one", nameC, "2026-09-17T11:05:00Z"),
			bugLine(idBug3, "Crash in the linter", "yamada", "2026-09-15T10:00:00Z"),
			change(idBug3, "open", "done", "yamada", "2026-09-15T10:30:00Z"),
			bugLine(idA, "A closed bug with a reply", "yamada", "2026-09-16T08:00:00Z"),
			replyLine(idRep3, idA, "Not a bug after all", nameC, "ai", "2026-09-16T08:30:00Z"),
			change(idA, "open", "done", "yamada", "2026-09-16T09:00:00Z"),
			// Questions are not bugs.
			question(idQ2, "Should nested block comments be supported?", "yamada", "2026-09-17T09:35:00Z"),
			answerLine(idA1, idQ2, "Not in the first version", "yamada", "human", "2026-09-17T09:50:00Z"),
		)
	}

	t.Run("the open bugs, each with its state and the latest reply under it", func(t *testing.T) {
		h := initialized(t)
		fixture(h)
		code, out, errOut := h.run("bug", "list")
		wantExit(t, code, 0, out, errOut)
		want := "7f3a2b1c09  Parser crashes on empty input      yamada       09:10  2 replies, awaiting confirmation\n" +
			"          └ The empty file has no first token  yamada       09:41\n" +
			"8e4b3c2d10  Error position is off by one       claude-code  11:05  no replies\n" +
			"2 open (show done: --all)\n"
		if out != want || errOut != "" {
			t.Errorf("stdout:\n%s\nwant:\n%s\nstderr %q", out, want, errOut)
		}
	})

	t.Run("--all shows the closed ones, with their states", func(t *testing.T) {
		h := initialized(t)
		fixture(h)
		code, out, errOut := h.run("b", "list", "--all")
		wantExit(t, code, 0, out, errOut)
		want := "b2c3d4e5f6  Crash in the linter                yamada       2026-09-15  done without replies\n" +
			"6cad4a268d  A closed bug with a reply          yamada       2026-09-16  1 reply, done\n" +
			"          └ Not a bug after all                claude-code  2026-09-16\n" +
			"7f3a2b1c09  Parser crashes on empty input      yamada       09:10       2 replies, awaiting confirmation\n" +
			"          └ The empty file has no first token  yamada       09:41\n" +
			"8e4b3c2d10  Error position is off by one       claude-code  11:05       no replies\n" +
			"2 open, 2 done\n"
		if out != want || errOut != "" {
			t.Errorf("stdout:\n%s\nwant:\n%s\nstderr %q", out, want, errOut)
		}
	})

	t.Run("bugs and questions stay apart", func(t *testing.T) {
		h := initialized(t)
		fixture(h)
		_, out, _ := h.run("q", "list", "--all")
		if strings.Contains(out, "Parser crashes") || strings.Contains(out, "replies") || !strings.Contains(out, "1 answer, awaiting confirmation") {
			t.Errorf("qa list:\n%s", out)
		}
		_, out, _ = h.run("b", "list", "--all")
		if strings.Contains(out, "nested block comments") || strings.Contains(out, "answer") {
			t.Errorf("bug list:\n%s", out)
		}
	})

	t.Run("replies of a deleted bug, and replies of nothing, are not listed", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			bugLine(idBug1, "A bug", "yamada", "2026-09-17T09:10:00Z"),
			replyLine(idRep1, idBug1, "hidden with its bug", nameC, "ai", "2026-09-17T09:15:00Z"),
			deleteLine(idBug1, "yamada", "2026-09-17T09:20:00Z"),
			// A reply whose bug is not in the journal (it may have been archived).
			replyLine(idRep2, idBug2, "an orphan", nameC, "ai", "2026-09-17T09:30:00Z"),
			// A record of type bug whose re names a question is a reply to nothing.
			question(idQ2, "A question", "yamada", "2026-09-17T09:35:00Z"),
			replyLine(idRep3, idQ2, "type bug, re a question", nameC, "ai", "2026-09-17T09:40:00Z"),
		)
		code, out, errOut := h.run("b", "list", "--all")
		wantExit(t, code, 0, out, errOut)
		if out != "0 open, 0 done\n" || errOut != "" {
			t.Errorf("stdout %q", out)
		}
		_, out, _ = h.run("q", "list", "--all")
		if !strings.Contains(out, "unanswered") || strings.Contains(out, "type bug") {
			t.Errorf("qa list:\n%s", out)
		}
	})

	t.Run("an empty list", func(t *testing.T) {
		h := initialized(t)
		code, out, errOut := h.run("b", "list")
		wantExit(t, code, 0, out, errOut)
		if out != "0 open (show done: --all)\n" {
			t.Errorf("stdout %q", out)
		}
	})
}

func TestDoneAndReopenABug(t *testing.T) {
	setup := func(t *testing.T) *harness {
		h := initialized(t)
		h.setJournal(
			bugLine(idBug1, "Parser crashes on empty input", "yamada", "2026-09-17T09:10:00Z"),
			replyLine(idRep1, idBug1, "Reproduced on macOS too", nameC, "ai", "2026-09-17T09:15:00Z"),
			question(idQ, "Which parser?", nameC, "2026-09-17T11:05:00Z"),
			record(idA, "todo", "Skip block comments", "yamada", "2026-09-17T10:18:00Z"),
		)
		return h
	}

	t.Run("done, again, and reopen", func(t *testing.T) {
		h := setup(t)
		code, out, errOut := h.run("b", "done", idBug1[:4])
		wantExit(t, code, 0, out, errOut)
		if out != "Done: 7f3a2b1c09  Parser crashes on empty input\n" {
			t.Errorf("stdout %q", out)
		}
		if !strings.Contains(h.readJournal(), `"id":"`+idBug1+`","op":"status","from":"open","status":"done"`) {
			t.Errorf("journal = %s", h.readJournal())
		}
		before := h.readJournal()
		code, out, errOut = h.run("b", "done", idBug1[:4])
		wantExit(t, code, 0, out, errOut)
		if out != "Already done: 7f3a2b1c09  Parser crashes on empty input\n" || h.readJournal() != before {
			t.Errorf("stdout %q", out)
		}
		code, out, errOut = h.run("b", "reopen", idBug1[:4])
		wantExit(t, code, 0, out, errOut)
		if out != "Reopened: 7f3a2b1c09  Parser crashes on empty input\n" {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("a record of another kind names the right command", func(t *testing.T) {
		h := setup(t)
		for _, tt := range []struct {
			args []string
			want string
		}{
			{[]string{"q", "done", idBug1[:6]}, "7f3a2b1c09 is a bug, not a question; use `mtqg bug done 7f3a2b1c09`\n"},
			{[]string{"b", "done", idQ[:6]}, "2217beaddb is a question, not a bug; use `mtqg qa done 2217beaddb`\n"},
			{[]string{"t", "done", idBug1[:6]}, "7f3a2b1c09 is a bug, not a todo; use `mtqg bug done 7f3a2b1c09`\n"},
			{[]string{"b", "reopen", idA[:6]}, "6cad4a268d is a todo, not a bug; use `mtqg todo reopen 6cad4a268d`\n"},
			// A reply has no state, and no command of its own to name.
			{[]string{"b", "done", idRep1[:6]}, "3d8e4a0b12 is a reply, not a bug\n"},
			{[]string{"t", "done", idRep1[:6]}, "3d8e4a0b12 is a reply, not a todo\n"},
		} {
			code, out, errOut := h.run(tt.args...)
			wantExit(t, code, 1, out, errOut)
			if errOut != tt.want {
				t.Errorf("%v: stderr %q, want %q", tt.args, errOut, tt.want)
			}
		}
	})
}

func TestShowABug(t *testing.T) {
	fixture := func(h *harness) {
		h.setJournal(
			bugLine(idBug1, "Parser crashes on empty input", "yamada", "2026-09-17T09:10:00Z"),
			replyLine(idRep1, idBug1, "Reproduced on macOS too", nameC, "ai", "2026-09-17T09:15:00Z"),
			replyLine(idRep2, idBug1, "The empty file has no first token", "yamada", "human", "2026-09-17T09:41:00Z"),
			change(idBug1, "open", "done", "yamada", "2026-09-17T10:00:00Z"),
			question(idQ2, "Should nested block comments be supported?", "yamada", "2026-09-17T10:10:00Z"),
			// Type bug, but it points at a question.
			replyLine(idRep3, idQ2, "type bug, re a question", nameC, "ai", "2026-09-17T10:20:00Z"),
		)
	}

	t.Run("a bug with its replies and its history", func(t *testing.T) {
		h := initialized(t)
		fixture(h)
		code, out, errOut := h.run("show", idBug1[:6])
		wantExit(t, code, 0, out, errOut)
		want := "bug  7f3a2b1c09  done\n" +
			"by yamada (human), 2026-09-17 09:10\n" +
			"\n" +
			"  Parser crashes on empty input\n" +
			"\n" +
			"Replies (2)\n" +
			"  3d8e4a0b12  claude-code (ai)  2026-09-17 09:15\n" +
			"    Reproduced on macOS too\n" +
			"  4c9d5b1e23  yamada (human)    2026-09-17 09:41\n" +
			"    The empty file has no first token\n" +
			"\n" +
			"Events\n" +
			"  2026-09-17 09:10  create  yamada (human)\n" +
			"  2026-09-17 09:15  create  claude-code (ai)  reply 3d8e4a0b12\n" +
			"  2026-09-17 09:41  create  yamada (human)    reply 4c9d5b1e23\n" +
			"  2026-09-17 10:00  status  yamada (human)    open -> done\n"
		if out != want || errOut != "" {
			t.Errorf("stdout:\n%s\nwant:\n%s\nstderr %q", out, want, errOut)
		}
	})

	t.Run("a reply says which bug it is for", func(t *testing.T) {
		h := initialized(t)
		fixture(h)
		code, out, errOut := h.run("show", idRep1[:6])
		wantExit(t, code, 0, out, errOut)
		want := "reply  3d8e4a0b12\n" +
			"by claude-code (ai), 2026-09-17 09:15\n" +
			"to bug 7f3a2b1c09  Parser crashes on empty input\n" +
			"\n" +
			"  Reproduced on macOS too\n" +
			"\n" +
			"Events\n" +
			"  2026-09-17 09:15  create  claude-code (ai)\n"
		if out != want {
			t.Errorf("stdout:\n%s\nwant:\n%s", out, want)
		}
	})

	t.Run("a reply that points at a record of another type says what that is, and does not quote it", func(t *testing.T) {
		h := initialized(t)
		fixture(h)
		_, out, _ := h.run("show", idRep3[:6])
		// Not "to bug 1012f037b6": the record is a question, and this is no reply to it.
		if !strings.Contains(out, "\nto 1012f037b6  (a question, not a bug)\n") || strings.Contains(out, "to bug") || strings.Contains(out, "Should nested") {
			t.Errorf("stdout:\n%s", out)
		}
		// And the question it points at does not count it as an answer.
		_, out, _ = h.run("show", idQ2[:6])
		if strings.Contains(out, "Answers") || strings.Contains(out, "type bug") {
			t.Errorf("stdout:\n%s", out)
		}
	})
}

func TestLogBugs(t *testing.T) {
	h := initialized(t)
	h.setJournal(
		bugLine(idBug1, "Parser crashes on empty input", "yamada", "2026-09-17T09:10:00Z"),
		replyLine(idRep1, idBug1, "Reproduced on macOS too", nameC, "ai", "2026-09-17T09:15:00Z"),
		question(idQ2, "Should nested block comments be supported?", "yamada", "2026-09-17T09:35:00Z"),
		answerLine(idA1, idQ2, "Not in the first version", "yamada", "human", "2026-09-17T09:50:00Z"),
		bugLine(idBug3, "Crash in the linter", "yamada", "2026-09-15T10:00:00Z"),
		change(idBug3, "open", "done", "yamada", "2026-09-15T10:30:00Z"),
	)

	t.Run("bugs and replies are kinds of their own", func(t *testing.T) {
		code, out, errOut := h.run("log")
		wantExit(t, code, 0, out, errOut)
		want := "09:50       answer    ae2eb1547f  (to 1012f037b6) Not in the first version    yamada\n" +
			"09:35       question  1012f037b6  Should nested block comments be supported?  yamada\n" +
			"09:15       reply     3d8e4a0b12  (to 7f3a2b1c09) Reproduced on macOS too     claude-code\n" +
			"09:10       bug       7f3a2b1c09  Parser crashes on empty input               yamada\n" +
			"2026-09-15  bug       b2c3d4e5f6  Crash in the linter                         yamada       done\n" +
			"5 records\n"
		if out != want || errOut != "" {
			t.Errorf("stdout:\n%s\nwant:\n%s\nstderr %q", out, want, errOut)
		}
	})

	t.Run("--kind bug shows bugs and their replies, and not questions", func(t *testing.T) {
		for _, k := range []string{"bug", "b"} {
			code, out, errOut := h.run("log", "--kind", k)
			wantExit(t, code, 0, out, errOut)
			if strings.Count(out, "\n") != 4 || strings.Contains(out, "question") || strings.Contains(out, "answer") ||
				!strings.Contains(out, "reply") || !strings.HasSuffix(out, "3 records\n") {
				t.Errorf("--kind %s:\n%s", k, out)
			}
		}
		code, out, errOut := h.run("log", "--kind", "qa")
		wantExit(t, code, 0, out, errOut)
		if strings.Contains(out, "bug") || strings.Contains(out, "reply") || !strings.HasSuffix(out, "2 records\n") {
			t.Errorf("--kind qa:\n%s", out)
		}
	})
}

func TestStatusCountsBugs(t *testing.T) {
	h := initialized(t)
	h.setJournal(
		bugLine(idBug1, "Parser crashes on empty input", "yamada", "2026-09-17T09:10:00Z"),
		replyLine(idRep1, idBug1, "Reproduced on macOS too", nameC, "ai", "2026-09-17T09:15:00Z"),
		bugLine(idBug2, "Error position is off by one", nameC, "2026-09-17T11:05:00Z"),
		bugLine(idBug3, "Crash in the linter", "yamada", "2026-09-15T10:00:00Z"),
		change(idBug3, "open", "done", "yamada", "2026-09-15T10:30:00Z"),
		question(idQ2, "Nested?", "yamada", "2026-09-17T09:35:00Z"),
	)
	code, out, errOut := h.run("status")
	wantExit(t, code, 0, out, errOut)
	want := "Open todos          0\n" +
		"Open questions      1\n" +
		"Open bugs           2  (1 awaiting confirmation)\n" +
		"Glossary            0\n" +
		"\n" +
		"Uncommitted records 5\n"
	if out != want {
		t.Errorf("stdout:\n%s\nwant:\n%s", out, want)
	}
}
