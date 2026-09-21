package cli

import (
	"strings"
	"testing"
)

func TestShow(t *testing.T) {
	question2 := func(h *harness) {
		h.setJournal(
			question(idQ2, "Should nested block comments be supported?", nameC, "2026-09-17T09:10:00Z"),
			answerLine(idA1, idQ2, "Supporting them is generally preferable", nameC, "ai", "2026-09-17T09:15:00Z"),
			answerLine(idA2, idQ2, "Not in the first version.\nRevisit if there is demand", "yamada", "human", "2026-09-17T09:41:00Z"),
			change(idQ2, "open", "done", "yamada", "2026-09-17T10:00:00Z"),
		)
	}

	t.Run("a question with its answers and its history", func(t *testing.T) {
		h := initialized(t)
		question2(h)
		code, out, errOut := h.run("show", idQ2[:6])
		wantExit(t, code, 0, out, errOut)
		want := "question  1012f037b6  done\n" +
			"by claude-code (human), 2026-09-17 09:10\n" +
			"\n" +
			"  Should nested block comments be supported?\n" +
			"\n" +
			"Answers (2)\n" +
			"  ae2eb1547f  claude-code (ai)  2026-09-17 09:15\n" +
			"    Supporting them is generally preferable\n" +
			"  a1a1a1a1a1  yamada (human)    2026-09-17 09:41\n" +
			"    Not in the first version.\n" +
			"    Revisit if there is demand\n" +
			"\n" +
			"Events\n" +
			"  2026-09-17 09:10  create  claude-code (human)\n" +
			"  2026-09-17 09:15  create  claude-code (ai)     answer ae2eb1547f\n" +
			"  2026-09-17 09:41  create  yamada (human)       answer a1a1a1a1a1\n" +
			"  2026-09-17 10:00  status  yamada (human)       open -> done\n"
		if out != want || errOut != "" {
			t.Errorf("stdout:\n%s\nwant:\n%s\nstderr %q", out, want, errOut)
		}
	})

	t.Run("an answer says which question it is for", func(t *testing.T) {
		h := initialized(t)
		question2(h)
		code, out, errOut := h.run("show", idA1[:6])
		wantExit(t, code, 0, out, errOut)
		want := "answer  ae2eb1547f\n" +
			"by claude-code (ai), 2026-09-17 09:15\n" +
			"to question 1012f037b6  Should nested block comments be supported?\n" +
			"\n" +
			"  Supporting them is generally preferable\n" +
			"\n" +
			"Events\n" +
			"  2026-09-17 09:15  create  claude-code (ai)\n"
		if out != want {
			t.Errorf("stdout:\n%s\nwant:\n%s", out, want)
		}
	})

	t.Run("a glossary entry gives its word before its definition", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(entryLine(idW1, "block comment", "A comment enclosed in /* and */", "yamada", "2026-09-17T09:00:00Z"))
		code, out, errOut := h.run("show", idW1[:6])
		wantExit(t, code, 0, out, errOut)
		want := "glossary  f28c105d1f\n" +
			"by yamada (human), 2026-09-17 09:00\n" +
			"\n" +
			"Word: block comment\n" +
			"\n" +
			"  A comment enclosed in /* and */\n" +
			"\n" +
			"Events\n" +
			"  2026-09-17 09:00  create  yamada (human)\n"
		if out != want {
			t.Errorf("stdout:\n%s\nwant:\n%s", out, want)
		}
	})

	t.Run("a memo and a todo", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			record(idM, "memo", "Use English for all error messages", "yamada", "2026-09-16T10:32:00Z"),
			record(idA, "todo", "Skip block comments", "yamada", "2026-09-17T10:18:00Z"),
		)
		_, out, _ := h.run("show", idM[:6])
		if !strings.HasPrefix(out, "memo  81e74ef5e8\nby yamada (human), 2026-09-16 10:32\n\n  Use English for all error messages\n") {
			t.Errorf("stdout %q", out)
		}
		_, out, _ = h.run("show", idA[:6])
		if !strings.HasPrefix(out, "todo  6cad4a268d  open\n") {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("the text is shown in full, all its lines, on a terminal as well", func(t *testing.T) {
		long := strings.Repeat("word ", 40) + "end"
		h := initialized(t)
		h.setJournal(record(idM, "memo", "first line\n\nthird line after a blank one\r\n"+long, "yamada", "2026-09-17T09:00:00Z"))
		h.env.StdoutIsTerminal, h.env.StdoutWidth = true, 40
		_, out, _ := h.run("show", idM[:6])
		if !strings.Contains(out, "  first line\n\n  third line after a blank one\n  "+long+"\n") {
			t.Errorf("stdout %q", out)
		}
		if strings.Contains(out, "...") || strings.Contains(out, "�") {
			t.Errorf("nothing is cut, and a CR LF is one line ending: %q", out)
		}
	})

	t.Run("control characters are replaced, in every place they are shown", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			lineBy(map[string]any{"id": idM, "op": "create", "type": "memo", "text": "a\x1b[2Jb", "ts": "2026-09-17T09:00:00Z"}, "human", "ev\x1bil"),
		)
		_, out, _ := h.run("show", idM[:6])
		if strings.Contains(out, "\x1b") || !strings.Contains(out, "a�[2Jb") || !strings.Contains(out, "ev�il (human)") {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("the changes of state and the edits are in the history", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			record(idA, "todo", "Skip block comments", "yamada", "2026-09-17T10:18:00Z"),
			change(idA, "open", "done", nameC, "2026-09-17T10:30:00Z"),
			change(idA, "done", "open", "yamada", "2026-09-17T10:40:00Z"),
			lineBy(map[string]any{"id": idA, "op": "edit", "text": "Skip block comments /* */", "ts": "2026-09-17T10:50:00Z"}, "human", "yamada"),
		)
		_, out, _ := h.run("show", idA[:6])
		want := "Events\n" +
			"  2026-09-17 10:18  create  yamada (human)\n" +
			"  2026-09-17 10:30  status  claude-code (human)  open -> done\n" +
			"  2026-09-17 10:40  status  yamada (human)       done -> open\n" +
			"  2026-09-17 10:50  edit    yamada (human)\n"
		if !strings.HasSuffix(out, want) || !strings.Contains(out, "  Skip block comments /* */\n") {
			t.Errorf("stdout:\n%s", out)
		}
	})

	t.Run("--full-id gives full IDs", func(t *testing.T) {
		h := initialized(t)
		question2(h)
		_, out, _ := h.run("show", "--full-id", idQ2[:6])
		if !strings.HasPrefix(out, "question  "+idQ2+"  done\n") || !strings.Contains(out, "  "+idA1+"  claude-code (ai)") {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("a record that is not found, too short, ambiguous, or deleted", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			question(idQ2, "Nested?", "yamada", "2026-09-17T09:10:00Z"),
			answerLine(idA1, idQ2, "An answer", nameC, "ai", "2026-09-17T09:15:00Z"),
			deleteLine(idQ2, "yamada", "2026-09-17T09:20:00Z"),
		)
		for _, tt := range []struct{ arg, want string }{
			{"ffff", `No record matches "ffff"`},
			{"ab", `The ID "ab" is too short: give at least 4 digits`},
			{idQ2[:6], "No record matches"}, // deleted
			{idA1[:6], "No record matches"}, // hidden with its question
		} {
			code, out, errOut := h.run("show", tt.arg)
			wantExit(t, code, 1, out, errOut)
			if !strings.Contains(errOut, tt.want) || out != "" {
				t.Errorf("%s: stdout %q, stderr %q", tt.arg, out, errOut)
			}
		}
		code, out, errOut := h.run("show")
		wantExit(t, code, 2, out, errOut)
	})
}

func TestShowAnAnswerThatBelongsToNothing(t *testing.T) {
	const idGone = "deadbeefdeadbeefdeadbeefdeadbeef"
	h := initialized(t)
	h.setJournal(
		record(idA, "todo", "a todo", "yamada", "2026-09-17T09:00:00Z"),
		answerLine(idA1, idGone, "an answer to a question that is not in the journal", nameC, "ai", "2026-09-17T09:10:00Z"),
		answerLine(idA2, idA, "an answer to a todo", nameC, "ai", "2026-09-17T09:20:00Z"),
		replyLine(idRep1, idA1, "a reply to an answer", nameC, "ai", "2026-09-17T09:30:00Z"),
	)
	for _, tt := range []struct{ id, want string }{
		{idA1, "\nto question " + idGone[:10] + "  (no such record)\n"},
		{idA2, "\nto " + idA[:10] + "  (a todo, not a question)\n"},
		{idRep1, "\nto " + idA1[:10] + "  (an answer, not a bug)\n"},
	} {
		code, out, errOut := h.run("show", tt.id[:6])
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(out, tt.want) {
			t.Errorf("show %s:\n%s\nwant a line %q", tt.id[:6], out, tt.want)
		}
	}
}
