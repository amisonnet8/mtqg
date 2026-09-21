package cli

import (
	"regexp"
	"strings"
	"testing"
)

// lines splits the journal into its lines.
func journalLines(h *harness) []string {
	content := h.readJournal()
	if content == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(content, "\n"), "\n")
}

// as sets who and where the next commands write as.
func (h *harness) as(tty, kind, name string) {
	h.vars["MTQG_TTY"] = tty
	h.vars["MTQG_AUTHOR_KIND"] = kind
	h.vars["MTQG_AUTHOR_NAME"] = name
}

func TestUndoRemovesTheLastLineAndSaysWhat(t *testing.T) {
	t.Run("a question added by mistake", func(t *testing.T) {
		h := initialized(t)
		h.run("m", "add", "an earlier memo")
		before := journalLines(h)
		_, id, _ := h.run("q", "add", "Not in the first version. Revisit if there is demand")
		id = strings.TrimSpace(id)

		code, out, errOut := h.run("undo")
		wantExit(t, code, 0, out, errOut)
		want := "Undone: qa add \"Not in the first version. Revisit if there is demand\" (" + id + ")\n"
		if out != want || errOut != "" {
			t.Errorf("stdout %q\nwant   %q\nstderr %q", out, want, errOut)
		}
		if got := journalLines(h); strings.Join(got, "\n") != strings.Join(before, "\n") {
			t.Errorf("journal after undo:\n%s\nwant what it was before:\n%s", strings.Join(got, "\n"), strings.Join(before, "\n"))
		}
	})

	t.Run("one step at a time", func(t *testing.T) {
		h := initialized(t)
		h.run("m", "add", "first")
		h.run("m", "add", "second")
		h.run("m", "add", "third")
		h.run("undo")
		if got := h.readJournal(); strings.Contains(got, "third") || !strings.Contains(got, "second") || !strings.Contains(got, "first") {
			t.Errorf("after one undo:\n%s", got)
		}
		h.run("undo")
		if got := h.readJournal(); strings.Contains(got, "second") || !strings.Contains(got, "first") {
			t.Errorf("after two undos:\n%s", got)
		}
		h.run("undo")
		code, out, errOut := h.run("undo")
		wantExit(t, code, 1, out, errOut)
		if errOut != "Nothing to undo: .mtqg/journal.jsonl has no line written by tester from this terminal\n" || out != "" {
			t.Errorf("stderr %q", errOut)
		}
	})

	t.Run("a change of state, an edit and a delete are undone the same way", func(t *testing.T) {
		h := initialized(t)
		_, id, _ := h.run("t", "add", "Skip block comments")
		id = strings.TrimSpace(id)

		h.run("t", "done", id)
		_, out, _ := h.run("undo")
		if out != "Undone: todo done \"Skip block comments\" ("+id+")\n" {
			t.Errorf("undo of done: %q", out)
		}
		if _, list, _ := h.run("t", "list"); !strings.Contains(list, "Skip block comments") {
			t.Errorf("the todo should be open again:\n%s", list)
		}

		h.run("t", "done", id)
		h.run("t", "reopen", id)
		_, out, _ = h.run("undo")
		if out != "Undone: todo reopen \"Skip block comments\" ("+id+")\n" {
			t.Errorf("undo of reopen: %q", out)
		}
		if _, list, _ := h.run("t", "list", "--all"); !strings.Contains(list, "done") {
			t.Errorf("the todo should be done again:\n%s", list)
		}
		h.run("undo") // the done, so the todo is as it was

		h.run("edit", id, "Skip all comments")
		_, out, _ = h.run("undo")
		if out != "Undone: todo edit \"Skip all comments\" ("+id+")\n" {
			t.Errorf("undo of edit: %q", out)
		}
		if _, list, _ := h.run("t", "list"); !strings.Contains(list, "Skip block comments") || strings.Contains(list, "all comments") {
			t.Errorf("the old text should be back:\n%s", list)
		}

		h.run("delete", id)
		_, out, _ = h.run("undo")
		if out != "Undone: todo delete \"Skip block comments\" ("+id+")\n" {
			t.Errorf("undo of delete: %q", out)
		}
		if _, list, _ := h.run("t", "list"); !strings.Contains(list, "Skip block comments") {
			t.Errorf("the todo should be back in view:\n%s", list)
		}
	})

	t.Run("a glossary entry is named with its word", func(t *testing.T) {
		h := initialized(t)
		_, id, _ := h.run("g", "add", "token", "The smallest unit")
		_, out, _ := h.run("undo")
		if out != "Undone: glossary add \"token: The smallest unit\" ("+strings.TrimSpace(id)+")\n" {
			t.Errorf("stdout %q", out)
		}
	})
}

func TestUndoNeverTouchesAnotherTerminalOrAnotherAuthor(t *testing.T) {
	t.Run("another terminal", func(t *testing.T) {
		h := initialized(t)
		h.as("terminal-a", "", "")
		h.run("m", "add", "written at a")
		h.as("terminal-b", "", "")
		h.run("m", "add", "written at b") // the last line of the file

		h.as("terminal-a", "", "")
		code, out, errOut := h.run("undo")
		wantExit(t, code, 0, out, errOut)
		if got := h.readJournal(); strings.Contains(got, "written at a") || !strings.Contains(got, "written at b") {
			t.Errorf("undo at terminal a removed the wrong line:\n%s", got)
		}

		// Terminal b still has its line, and a has none left.
		h.as("terminal-a", "", "")
		code, out, errOut = h.run("undo")
		wantExit(t, code, 1, out, errOut)
		if !strings.Contains(h.readJournal(), "written at b") {
			t.Errorf("the line of terminal b is gone:\n%s", h.readJournal())
		}
	})

	t.Run("another author", func(t *testing.T) {
		h := initialized(t)
		h.run("m", "add", "written by tester")
		h.as("", "ai", "claude-code")
		h.run("m", "add", "written by the agent") // the last line of the file
		h.as("", "", "")

		code, out, errOut := h.run("undo")
		wantExit(t, code, 0, out, errOut)
		if got := h.readJournal(); strings.Contains(got, "written by tester") || !strings.Contains(got, "written by the agent") {
			t.Errorf("the human's undo removed the wrong line:\n%s", got)
		}
	})

	t.Run("the same name as another kind is another author", func(t *testing.T) {
		h := initialized(t)
		h.as("", "ai", "tester")
		h.run("m", "add", "written by an AI called tester")
		h.as("", "", "") // the human tester
		before := h.readJournal()
		code, out, errOut := h.run("undo")
		wantExit(t, code, 1, out, errOut)
		if h.readJournal() != before {
			t.Errorf("the journal changed:\n%s", h.readJournal())
		}
	})

	t.Run("a line with a terminal and a line without one are not each other's target", func(t *testing.T) {
		h := initialized(t)
		h.as("terminal-a", "", "")
		h.run("m", "add", "at a terminal")
		h.as("", "", "")
		h.run("m", "add", "through a pipe") // no tty; the last line

		h.as("terminal-a", "", "")
		h.run("undo")
		if got := h.readJournal(); strings.Contains(got, "at a terminal") || !strings.Contains(got, "through a pipe") {
			t.Errorf("undo at a terminal removed the wrong line:\n%s", got)
		}

		h.as("", "", "")
		h.run("undo")
		if strings.Contains(h.readJournal(), "through a pipe") {
			t.Errorf("undo without a terminal did not remove its line:\n%s", h.readJournal())
		}
	})

	t.Run("nothing to undo when the journal is empty", func(t *testing.T) {
		h := initialized(t)
		code, out, errOut := h.run("undo")
		wantExit(t, code, 1, out, errOut)
		if !strings.HasPrefix(errOut, "Nothing to undo: ") {
			t.Errorf("stderr %q", errOut)
		}
	})
}

func TestUndoOfTheSameSecondRemovesTheLastLineOfTheFile(t *testing.T) {
	// The IDs are random, so they are no help. Try both orders.
	for _, ids := range [][2]string{{idA, idM}, {idM, idA}} {
		h := initialized(t)
		h.setJournal(
			record(ids[0], "memo", "written first", "tester", "2026-09-17T09:00:00Z"),
			record(ids[1], "memo", "written second", "tester", "2026-09-17T09:00:00Z"),
		)
		code, out, errOut := h.run("undo")
		wantExit(t, code, 0, out, errOut)
		if got := h.readJournal(); strings.Contains(got, "written second") || !strings.Contains(got, "written first") {
			t.Errorf("ids %v: removed the wrong line:\n%s", ids, got)
		}
	}
}

func TestUndoRefusesToLeaveEventsWithoutTheirRecord(t *testing.T) {
	setup := func(t *testing.T) (*harness, string) {
		h := initialized(t)
		h.setJournal(
			record(idA, "todo", "Skip block comments", "tester", "2026-09-17T09:00:00Z"),
			change(idA, "open", "done", nameC, "2026-09-17T09:30:00Z"),
			change(idA, "done", "open", "yamada", "2026-09-17T09:40:00Z"),
		)
		return h, h.readJournal()
	}

	t.Run("the creation of a todo that was changed", func(t *testing.T) {
		h, before := setup(t)
		code, out, errOut := h.run("undo")
		wantExit(t, code, 1, out, errOut)
		want := "Cannot undo: todo " + idA[:10] + " has 2 other events, and undoing its creation would leave them without a record\n" +
			"To hide it instead: mtqg delete " + idA[:10] + "\n"
		if errOut != want || out != "" {
			t.Errorf("stderr %q\nwant   %q\nstdout %q", errOut, want, out)
		}
		if h.readJournal() != before {
			t.Errorf("the journal changed:\n%s", h.readJournal())
		}
	})

	t.Run("as JSON, on standard error", func(t *testing.T) {
		h, before := setup(t)
		code, out, errOut := h.run("--json", "undo")
		wantExit(t, code, 1, out, errOut)
		obj := jsonObject(t, oneLineOfJSONBody(t, errOut))
		e := field(t, obj, "error").(map[string]any)
		rec := field(t, e, "record").(map[string]any)
		if out != "" || e["kind"] != "has_later_events" || rec["id"] != idA || h.readJournal() != before {
			t.Errorf("stdout %q, error %v", out, e)
		}
	})

	t.Run("a question that was answered", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			question(idQ2, "Should nested block comments be supported?", "tester", "2026-09-17T10:00:00Z"),
			answerLine(idA1, idQ2, "Supporting them is preferable", nameC, "ai", "2026-09-17T10:05:00Z"),
		)
		before := h.readJournal()
		code, out, errOut := h.run("undo")
		wantExit(t, code, 1, out, errOut)
		if !strings.Contains(errOut, "Cannot undo: question "+idQ2[:10]+" has 1 other event, and") || h.readJournal() != before {
			t.Errorf("stderr %q", errOut)
		}
	})

	t.Run("the change of state is removed whatever follows it", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			record(idA, "todo", "Skip block comments", nameC, "2026-09-17T09:00:00Z"),
			change(idA, "open", "done", "tester", "2026-09-17T09:30:00Z"),
			change(idA, "done", "open", "yamada", "2026-09-17T09:40:00Z"),
		)
		code, out, errOut := h.run("undo")
		wantExit(t, code, 0, out, errOut)
		if !strings.HasPrefix(out, "Undone: todo done ") || strings.Contains(h.readJournal(), `"from":"open","status":"done"`) {
			t.Errorf("stdout %q, journal\n%s", out, h.readJournal())
		}
	})

	t.Run("the creation of a memo that nothing else is about", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(record(idM, "memo", "alone", "tester", "2026-09-17T09:00:00Z"))
		code, out, errOut := h.run("undo")
		wantExit(t, code, 0, out, errOut)
		if h.readJournal() != "" {
			t.Errorf("journal = %q", h.readJournal())
		}
	})
}

// oneLineOfJSONBody is standard error that must be one line of JSON, as it is.
func oneLineOfJSONBody(t *testing.T, errOut string) string {
	t.Helper()
	if strings.Count(errOut, "\n") != 1 {
		t.Fatalf("stderr is not one line: %q", errOut)
	}
	return errOut
}

func TestUndoKeepsEveryOtherLineByteForByte(t *testing.T) {
	h := initialized(t)
	// Lines that a rewrite could damage: unknown fields, hand-made spacing, a
	// line that is not JSON, and another author's line.
	keep := []string{
		`{"id":"` + idM + `","op":"create","type":"memo","text":"hand made","v":0,"ts":"2026-09-17T09:00:00Z","author":{"kind":"human","name":"yamada"},"future":{"a":1}}`,
		`this is not json`,
		`{ "id" : "` + idC + `" , "op":"create", "type":"memo", "text":"spaced out", "v":0, "ts":"2026-09-17T09:10:00Z", "author":{"kind":"ai","name":"claude-code"} }`,
	}
	h.setJournal(append(keep, record(idA, "memo", "to undo", "tester", "2026-09-17T09:20:00Z"))...)

	code, out, errOut := h.run("undo")
	wantExit(t, code, 0, out, errOut)
	if got, want := h.readJournal(), strings.Join(keep, "\n")+"\n"; got != want {
		t.Errorf("the other lines changed:\n got %q\nwant %q", got, want)
	}
	// The line that could not be read is still reported when reading.
	if _, _, warn := h.run("log"); !strings.Contains(warn, "line 2") {
		t.Errorf("warnings: %q", warn)
	}
}

func TestUndoRemovesEveryCopyOfALine(t *testing.T) {
	// A merge can bring the same line in twice. Reading counts it once, so undoing
	// it once must leave none.
	h := initialized(t)
	line := record(idM, "memo", "twice", "tester", "2026-09-17T09:00:00Z")
	h.setJournal(record(idC, "memo", "other", "yamada", "2026-09-17T08:00:00Z"), line, line)
	code, out, errOut := h.run("undo")
	wantExit(t, code, 0, out, errOut)
	if strings.Contains(h.readJournal(), "twice") || !strings.Contains(h.readJournal(), "other") {
		t.Errorf("journal:\n%s", h.readJournal())
	}
}

func TestUndoJSON(t *testing.T) {
	h := initialized(t)
	h.run("m", "add", "to undo")
	code, out, errOut := h.run("--json", "undo")
	wantExit(t, code, 0, out, errOut)
	obj := jsonObject(t, out)
	ev := field(t, obj, "event").(map[string]any)
	rec := field(t, obj, "record").(map[string]any)
	if obj["command"] != "undo" || ev["op"] != "create" || ev["type"] != "memo" || ev["text"] != "to undo" ||
		rec["kind"] != "memo" || rec["text"] != "to undo" || rec["id"] != ev["id"] {
		t.Errorf("object = %v", obj)
	}
	if ok, _ := regexp.MatchString(`^[0-9a-f]{32}$`, ev["id"].(string)); !ok {
		t.Errorf("the event holds the ID %q, want the full ID", ev["id"])
	}

	code, out, errOut = h.run("--json", "undo")
	wantExit(t, code, 1, out, errOut)
	e := field(t, jsonObject(t, errOut), "error").(map[string]any)
	if out != "" || e["kind"] != "nothing_to_undo" {
		t.Errorf("stdout %q, error %v", out, e)
	}
}

func TestUndoWithoutAnAuthorOrARepositoryFailsLikeWriting(t *testing.T) {
	h := initialized(t)
	h.vars["MTQG_AUTHOR_KIND"] = "ai"
	code, out, errOut := h.run("undo")
	wantExit(t, code, 1, out, errOut)
	if !strings.Contains(errOut, "MTQG_AUTHOR_NAME") {
		t.Errorf("stderr %q", errOut)
	}

	h2 := newHarness(t)
	code, out, errOut = h2.run("undo")
	wantExit(t, code, 1, out, errOut)
	if !strings.Contains(errOut, "No .mtqg/ found") {
		t.Errorf("stderr %q", errOut)
	}
}
