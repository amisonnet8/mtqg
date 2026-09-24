package cli

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// editFixture: a todo that is done, a memo, a question with two answers by two
// authors, a bug with one reply, and a glossary entry.
func editFixture(t *testing.T) *harness {
	t.Helper()
	h := initialized(t)
	h.setJournal(
		record(idA, "todo", "Skip block comments", "yamada", "2026-09-17T09:00:00Z"),
		change(idA, "open", "done", nameC, "2026-09-17T09:30:00Z"),
		record(idM, "memo", "Use English for all error messages", "yamada", "2026-09-17T09:40:00Z"),
		question(idQ2, "Should nested block comments be supported?", "yamada", "2026-09-17T10:00:00Z"),
		answerLine(idA1, idQ2, "Supporting them is generally preferable", nameC, "ai", "2026-09-17T10:05:00Z"),
		answerLine(idA2, idQ2, "Not in the first version", "yamada", "human", "2026-09-17T10:10:00Z"),
		bugLine(idBug1, "Parser crashes on empty input", "yamada", "2026-09-17T10:20:00Z"),
		replyLine(idRep1, idBug1, "Reproduced on macOS too", nameC, "ai", "2026-09-17T10:25:00Z"),
		entryLine(idW1, "token", "The smallest unit produced by lexing", "yamada", "2026-09-17T11:00:00Z"),
	)
	return h
}

func TestEdit(t *testing.T) {
	t.Run("replaces the text and writes an edit of the text only", func(t *testing.T) {
		h := editFixture(t)
		code, out, errOut := h.run("edit", idA[:6], "Skip", "block", "and", "line", "comments")
		wantExit(t, code, 0, out, errOut)
		if out != "Edited: "+idA[:10]+"  Skip block and line comments\n" || errOut != "" {
			t.Errorf("stdout %q, stderr %q", out, errOut)
		}
		lines := strings.Split(strings.TrimSuffix(h.readJournal(), "\n"), "\n")
		last := lines[len(lines)-1]
		// The event holds the full ID, the text and the basis (2: the todo's create
		// and its status change, both before this edit), and no word, no state, no
		// from. (The time is the clock of the journal layer, not the one that the
		// display uses, so it is matched, not compared.)
		want := regexp.MustCompile(`^\{"id":"` + idA + `","op":"edit","basis":2,"text":"Skip block and line comments","v":0,"ts":"\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ","author":\{"kind":"human","name":"tester"\}\}$`)
		if !want.MatchString(last) {
			t.Errorf("last line\n got %s", last)
		}
		// The state is not touched by an edit.
		_, list, _ := h.run("todo", "list", "--all")
		if !strings.Contains(list, "Skip block and line comments") || !strings.Contains(list, "done") {
			t.Errorf("todo list --all =\n%s", list)
		}
	})

	t.Run("edits any kind: an answer, a reply, a definition", func(t *testing.T) {
		h := editFixture(t)
		for _, tt := range []struct{ id, text string }{
			{idA1, "Supporting them is preferable"},
			{idRep1, "Reproduced on Linux too"},
			{idW1, "The smallest unit lexing produces"},
			{idM, "Use English for every message"},
			{idQ2, "Should nested block comments work?"},
		} {
			code, out, errOut := h.run("edit", tt.id[:8], tt.text)
			wantExit(t, code, 0, out, errOut)
			if !strings.HasPrefix(out, "Edited: "+tt.id[:10]+"  "+tt.text) {
				t.Errorf("edit %s: stdout %q", tt.id[:8], out)
			}
		}
		// The word of the entry and the answers of the question are as they were.
		_, glossary, _ := h.run("glossary", "list")
		if !strings.Contains(glossary, "token") || !strings.Contains(glossary, "The smallest unit lexing produces") {
			t.Errorf("glossary list =\n%s", glossary)
		}
		_, qa, _ := h.run("qa", "list")
		if !strings.Contains(qa, "Should nested block comments work?") || !strings.Contains(qa, "2 answers, awaiting confirmation") {
			t.Errorf("qa list =\n%s", qa)
		}
	})

	t.Run("the same text writes nothing", func(t *testing.T) {
		h := editFixture(t)
		before := h.readJournal()
		code, out, errOut := h.run("edit", idA[:6], "Skip block comments")
		wantExit(t, code, 0, out, errOut)
		if out != "Unchanged: "+idA[:10]+"  Skip block comments\n" || h.readJournal() != before {
			t.Errorf("stdout %q, journal changed: %v", out, h.readJournal() != before)
		}
	})

	t.Run("the text from standard input", func(t *testing.T) {
		h := editFixture(t)
		h.stdin = "First line\nsecond line\n"
		code, out, errOut := h.run("edit", idM[:6], "-")
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(h.readJournal(), `"op":"edit","basis":1,"text":"First line\nsecond line"`) || out != "Edited: "+idM[:10]+"  First line\n" {
			t.Errorf("stdout %q, journal %s", out, h.readJournal())
		}
	})

	t.Run("with no text the editor opens on the text the record has now", func(t *testing.T) {
		h := editFixture(t)
		h.setJournal(record(idM, "memo", "Line one\nLine two", "yamada", "2026-09-17T09:40:00Z"))
		h.vars["EDITOR"] = "myeditor"
		var seen string
		h.env.RunEditor = func(argv []string) error {
			data, err := os.ReadFile(argv[len(argv)-1])
			if err != nil {
				return err
			}
			seen = string(data)
			return os.WriteFile(argv[len(argv)-1], []byte("Line one\nLine two\nLine three\n"), 0o600)
		}
		code, out, errOut := h.run("edit", idM[:6])
		wantExit(t, code, 0, out, errOut)
		if seen != "Line one\nLine two\n" {
			t.Errorf("the editor was given %q", seen)
		}
		if !strings.Contains(h.readJournal(), `"op":"edit","basis":1,"text":"Line one\nLine two\nLine three"`) {
			t.Errorf("journal = %s", h.readJournal())
		}
	})

	t.Run("saving the editor without a change writes nothing", func(t *testing.T) {
		h := editFixture(t)
		h.vars["EDITOR"] = "myeditor"
		h.env.RunEditor = func([]string) error { return nil } // leaves the file as it was
		before := h.readJournal()
		code, out, errOut := h.run("edit", idA[:6])
		wantExit(t, code, 0, out, errOut)
		if !strings.HasPrefix(out, "Unchanged: ") || h.readJournal() != before {
			t.Errorf("stdout %q, journal changed: %v", out, h.readJournal() != before)
		}
	})

	t.Run("an ID that matches nothing stops before the editor opens", func(t *testing.T) {
		h := editFixture(t)
		h.vars["EDITOR"] = "myeditor"
		opened := false
		h.env.RunEditor = func([]string) error { opened = true; return nil }
		code, out, errOut := h.run("edit", "a8ec")
		wantExit(t, code, 1, out, errOut)
		if opened || out != "" || errOut != "No record matches \"a8ec\"\n" {
			t.Errorf("stdout %q, stderr %q, editor opened: %v", out, errOut, opened)
		}
	})

	t.Run("an empty text is refused", func(t *testing.T) {
		h := editFixture(t)
		before := h.readJournal()
		h.stdin = "\n"
		code, out, errOut := h.run("edit", idA[:6], "-")
		wantExit(t, code, 1, out, errOut)
		if errOut != "Aborting: the text is empty\n" || h.readJournal() != before {
			t.Errorf("stderr %q, journal changed: %v", errOut, h.readJournal() != before)
		}
	})

	t.Run("no ID is a mistake in the command line", func(t *testing.T) {
		h := editFixture(t)
		code, out, errOut := h.run("edit")
		wantExit(t, code, 2, out, errOut)
		if !strings.Contains(errOut, "Missing argument. Usage: mtqg edit <id> [<text>]") {
			t.Errorf("stderr %q", errOut)
		}
	})

	t.Run("a record that was deleted cannot be edited", func(t *testing.T) {
		h := editFixture(t)
		wantExit(t, mustCode(h.run("delete", idM[:6])), 0, "", "")
		code, out, errOut := h.run("edit", idM[:6], "x")
		wantExit(t, code, 1, out, errOut)
	})
}

func mustCode(code int, _, _ string) int { return code }

func TestEditJSON(t *testing.T) {
	h := editFixture(t)
	code, out, errOut := h.run("--json", "edit", idA[:6], "Skip block and line comments")
	wantExit(t, code, 0, out, errOut)
	obj := jsonObject(t, out)
	rec := field(t, obj, "record").(map[string]any)
	if obj["command"] != "edit" || obj["changed"] != true || rec["id"] != idA ||
		rec["text"] != "Skip block and line comments" || rec["status"] != "done" {
		t.Errorf("object = %v", obj)
	}
	// It was updated after the last change in the fixture (the times are UTC, so
	// they compare as text).
	if updated, _ := rec["updated"].(string); updated <= "2026-09-17T09:30:00Z" {
		t.Errorf("updated = %q, want the time of the edit", updated)
	}

	// The same again: nothing is written, and it says so.
	before := h.readJournal()
	_, out, _ = h.run("--json", "edit", idA[:6], "Skip block and line comments")
	obj = jsonObject(t, out)
	if obj["changed"] != false || h.readJournal() != before {
		t.Errorf("object = %v, journal changed: %v", obj, h.readJournal() != before)
	}
}

func TestDelete(t *testing.T) {
	t.Run("hides a memo, and says the lines remain", func(t *testing.T) {
		h := editFixture(t)
		code, out, errOut := h.run("delete", idM[:6])
		wantExit(t, code, 0, out, errOut)
		want := "Deleted: " + idM[:10] + "  Use English for all error messages\n" +
			"The lines remain in the journal and in git history\n"
		if out != want || errOut != "" {
			t.Errorf("stdout %q\nwant   %q\nstderr %q", out, want, errOut)
		}
		_, list, _ := h.run("memo", "list")
		if strings.Contains(list, "English") || !strings.HasSuffix(list, "0 memos\n") {
			t.Errorf("memo list =\n%s", list)
		}
		// Its ID matches nothing any more.
		code, _, errOut = h.run("show", idM[:6])
		wantExit(t, code, 1, "", errOut)
		if !strings.Contains(errOut, "No record matches") {
			t.Errorf("show after delete: %q", errOut)
		}
		// The line is still in the file: nothing was removed.
		if !strings.Contains(h.readJournal(), `"text":"Use English for all error messages"`) {
			t.Errorf("the create line is gone:\n%s", h.readJournal())
		}
	})

	t.Run("a question takes its answers along, and the output says whose", func(t *testing.T) {
		h := editFixture(t)
		code, out, errOut := h.run("delete", idQ2[:6])
		wantExit(t, code, 0, out, errOut)
		want := "Deleted: " + idQ2[:10] + "  Should nested block comments be supported?\n" +
			"2 answers are also hidden (claude-code, yamada)\n" +
			"The lines remain in the journal and in git history\n"
		if out != want {
			t.Errorf("stdout %q\nwant   %q", out, want)
		}
		_, log, _ := h.run("log")
		if strings.Contains(log, "answer") || strings.Contains(log, "question") {
			t.Errorf("log still shows them:\n%s", log)
		}
	})

	t.Run("a bug takes its replies along", func(t *testing.T) {
		h := editFixture(t)
		code, out, errOut := h.run("delete", idBug1[:6])
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(out, "1 reply is also hidden (claude-code)\n") {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("an author who wrote several is named once", func(t *testing.T) {
		h := editFixture(t)
		h.setJournal(
			question(idQ2, "Q?", "yamada", "2026-09-17T10:00:00Z"),
			answerLine(idA1, idQ2, "one", nameC, "ai", "2026-09-17T10:05:00Z"),
			answerLine(idA2, idQ2, "two", "yamada", "human", "2026-09-17T10:10:00Z"),
			answerLine(idA3, idQ2, "three", nameC, "ai", "2026-09-17T10:15:00Z"),
		)
		_, out, _ := h.run("delete", idQ2[:6])
		if !strings.Contains(out, "3 answers are also hidden (claude-code, yamada)\n") {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("an answer is hidden alone, and the question is as it was", func(t *testing.T) {
		h := editFixture(t)
		code, out, errOut := h.run("delete", idA1[:6])
		wantExit(t, code, 0, out, errOut)
		if strings.Contains(out, "also hidden") || !strings.HasPrefix(out, "Deleted: "+idA1[:10]+"  Supporting them") {
			t.Errorf("stdout %q", out)
		}
		_, qa, _ := h.run("qa", "list")
		if !strings.Contains(qa, "1 answer, awaiting confirmation") || !strings.Contains(qa, "Not in the first version") {
			t.Errorf("qa list =\n%s", qa)
		}
	})

	t.Run("deleting twice: the second time the ID matches nothing", func(t *testing.T) {
		h := editFixture(t)
		h.run("delete", idM[:6])
		before := h.readJournal()
		code, out, errOut := h.run("delete", idM[:6])
		wantExit(t, code, 1, out, errOut)
		if h.readJournal() != before {
			t.Errorf("a second delete wrote a line")
		}
	})

	t.Run("no ID is a mistake in the command line", func(t *testing.T) {
		h := editFixture(t)
		code, out, errOut := h.run("delete")
		wantExit(t, code, 2, out, errOut)
	})
}

func TestDeleteJSON(t *testing.T) {
	h := editFixture(t)
	code, out, errOut := h.run("--json", "delete", idQ2[:6])
	wantExit(t, code, 0, out, errOut)
	obj := jsonObject(t, out)
	rec := field(t, obj, "record").(map[string]any)
	hidden := records(t, obj, "hidden_replies")
	if obj["command"] != "delete" || rec["id"] != idQ2 || len(hidden) != 2 || hidden[0]["id"] != idA1 || hidden[1]["id"] != idA2 {
		t.Errorf("object = %v", obj)
	}

	// A record with nothing under it has an empty list, not a missing one.
	_, out, _ = h.run("--json", "delete", idM[:6])
	if !strings.Contains(out, `"hidden_replies": []`) {
		t.Errorf("stdout %s", out)
	}
}
