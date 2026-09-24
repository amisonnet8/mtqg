package cli

import (
	"strings"
	"testing"
)

const idGone = "deadbeefdeadbeefdeadbeefdeadbeef" // in no line of the journal

// reviewFixture has one of each thing that review shows: a todo that two people
// closed without knowing, a word that is defined twice, and two answers that belong
// to no question.
func reviewFixture(t *testing.T) *harness {
	t.Helper()
	h := initialized(t)
	h.setJournal(
		record(idA, "todo", "Skip block comments", "yamada", "2026-09-17T09:00:00Z"),
		change(idA, "open", "done", nameC, "2026-09-17T09:30:00Z"),
		change(idA, "open", "done", "yamada", "2026-09-17T09:45:00Z"),
		bugLine(idBug1, "Parser crashes on empty input", "yamada", "2026-09-17T10:00:00Z"),
		entryLine(idW1, "token", "The smallest unit produced by lexing", "yamada", "2026-09-17T10:10:00Z"),
		entryLine(idW2, "token", "A second definition", nameC, "2026-09-17T10:20:00Z"),
		answerLine(idA1, idBug1, "an answer that names a bug", nameC, "ai", "2026-09-17T10:30:00Z"),
		answerLine(idA2, idGone, "an answer to a question that is not in the journal", "yamada", "human", "2026-09-17T10:40:00Z"),
		record(idM, "memo", "a memo, which review has nothing to say about", "yamada", "2026-09-17T10:50:00Z"),
	)
	return h
}

func TestReview(t *testing.T) {
	t.Run("one of each", func(t *testing.T) {
		h := reviewFixture(t)
		code, out, errOut := h.run("review")
		wantExit(t, code, 0, out, errOut)
		want := "Concurrent changes (1)\n" +
			"  todo " + idA[:10] + " \"Skip block comments\"\n" +
			"    2026-09-17 09:30  claude-code  open -> done\n" +
			"    2026-09-17 09:45  yamada       open -> done\n" +
			"\n" +
			"Duplicate glossary definitions (1)\n" +
			"  token\n" +
			"    " + idW1[:10] + "  yamada       The smallest unit produced by lexing\n" +
			"    " + idW2[:10] + "  claude-code  A second definition\n" +
			"\n" +
			"Answers and replies with no parent (2)\n" +
			"  " + idA1[:10] + "  answer  claude-code  an answer that names a bug\n" +
			"    re " + idBug1[:10] + ": a bug, not a question\n" +
			"  " + idA2[:10] + "  answer  yamada       an answer to a question that is not in the journal\n" +
			"    re " + idGone[:10] + ": not in the journal\n"
		if out != want || errOut != "" {
			t.Errorf("stdout:\n%s\nwant:\n%s\nstderr %q", out, want, errOut)
		}
	})

	t.Run("nothing to review", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(record(idA, "todo", "a todo", "yamada", "2026-09-17T09:00:00Z"))
		code, out, errOut := h.run("review")
		wantExit(t, code, 0, out, errOut)
		if out != "Nothing to review\n" {
			t.Errorf("stdout %q", out)
		}
		// An empty journal is the same.
		h.setJournal()
		_, out, _ = h.run("review")
		if out != "Nothing to review\n" {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("a section with nothing in it is left out, and no blank line is left", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			entryLine(idW1, "token", "one", "yamada", "2026-09-17T10:10:00Z"),
			entryLine(idW2, "token", "two", nameC, "2026-09-17T10:20:00Z"),
		)
		_, out, _ := h.run("review")
		if !strings.HasPrefix(out, "Duplicate glossary definitions (1)\n") || strings.Contains(out, "Concurrent") || strings.Contains(out, "\n\n") || strings.HasPrefix(out, "\n") {
			t.Errorf("stdout:\n%s", out)
		}
	})

	t.Run("changes written one after the other are not a conflict", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			record(idA, "todo", "a todo", "yamada", "2026-09-17T09:00:00Z"),
			change(idA, "open", "done", nameC, "2026-09-17T09:30:00Z"),
			change(idA, "done", "open", "yamada", "2026-09-17T09:40:00Z"),
			change(idA, "open", "done", nameC, "2026-09-17T09:50:00Z"),
		)
		_, out, _ := h.run("review")
		if out != "Nothing to review\n" {
			t.Errorf("stdout:\n%s", out)
		}
	})

	t.Run("the same line twice is one event, and not a conflict", func(t *testing.T) {
		h := initialized(t)
		done := change(idA, "open", "done", nameC, "2026-09-17T09:30:00Z")
		h.setJournal(record(idA, "todo", "a todo", "yamada", "2026-09-17T09:00:00Z"), done, done)
		_, out, _ := h.run("review")
		if out != "Nothing to review\n" {
			t.Errorf("stdout:\n%s", out)
		}
	})

	t.Run("a record that was deleted is not reviewed", func(t *testing.T) {
		h := reviewFixture(t)
		h.run("delete", idA[:6])
		_, out, _ := h.run("review")
		if strings.Contains(out, "Concurrent") {
			t.Errorf("stdout:\n%s", out)
		}
	})

	t.Run("--full-id", func(t *testing.T) {
		h := reviewFixture(t)
		_, out, _ := h.run("review", "--full-id")
		if !strings.Contains(out, "  todo "+idA+" ") || !strings.Contains(out, "re "+idBug1+": ") {
			t.Errorf("stdout:\n%s", out)
		}
	})
}

// A status change and an edit that raced (the stage-2 report's original
// scenario): basis catches it even though from and status never disagree
// (whoever wrote the status change had seen only the create, not the edit
// that -- once merged -- sorts before it; its basis says so even though its
// from does not).
func TestReviewCatchesAStatusChangeAndAnEditThatRaced(t *testing.T) {
	h := initialized(t)
	h.setJournal(
		record(idA, "todo", "Skip block comments", "yamada", "2026-09-17T09:00:00Z"),
		editLine(idA, "Skip block comments (updated)", 1, nameC, "2026-09-17T09:10:00Z"),
		marshal(map[string]any{
			"id": idA, "op": "status", "from": "open", "status": "done", "basis": 1,
			"v": 0, "ts": "2026-09-17T09:20:00Z", "author": map[string]string{"kind": "human", "name": "yamada"},
		}),
	)
	code, out, errOut := h.run("review")
	wantExit(t, code, 0, out, errOut)
	want := "Concurrent changes (1)\n" +
		"  todo " + idA[:10] + " \"Skip block comments (updated)\"\n" +
		"    2026-09-17 09:10  claude-code  edited\n" +
		"    2026-09-17 09:20  yamada       open -> done\n"
	if out != want || errOut != "" {
		t.Errorf("stdout:\n%s\nwant:\n%s\nstderr %q", out, want, errOut)
	}
}

func TestReviewJSON(t *testing.T) {
	h := reviewFixture(t)
	code, out, errOut := h.run("--json", "review")
	wantExit(t, code, 0, out, errOut)
	obj := jsonObject(t, out)
	conflicts := records(t, obj, "concurrent_status_changes")
	dups := records(t, obj, "duplicate_words")
	strays := records(t, obj, "unattached_replies")
	if obj["command"] != "review" || len(conflicts) != 1 || len(dups) != 1 || len(strays) != 2 {
		t.Fatalf("object = %v", obj)
	}
	changes := conflicts[0]["changes"].([]any)
	if field(t, conflicts[0], "record", "id") != idA || len(changes) != 2 ||
		changes[0].(map[string]any)["op"] != "status" || changes[1].(map[string]any)["author"].(map[string]any)["name"] != "yamada" {
		t.Errorf("conflicts = %v", conflicts)
	}
	if dups[0]["word"] != "token" || len(dups[0]["records"].([]any)) != 2 {
		t.Errorf("duplicate_words = %v", dups)
	}
	// What the re names: the bug (with its record), and nothing (no re_record).
	if field(t, strays[0], "record", "id") != idA1 || field(t, strays[0], "re_record", "id") != idBug1 || field(t, strays[0], "re_record", "kind") != "bug" {
		t.Errorf("unattached_replies[0] = %v", strays[0])
	}
	if _, has := strays[1]["re_record"]; has || field(t, strays[1], "record", "re") != idGone {
		t.Errorf("unattached_replies[1] = %v", strays[1])
	}

	// Nothing to review: three empty lists.
	h = initialized(t)
	_, out, _ = h.run("--json", "review")
	for _, key := range []string{"concurrent_status_changes", "duplicate_words", "unattached_replies"} {
		if !strings.Contains(out, `"`+key+`": []`) {
			t.Errorf("no empty %s in:\n%s", key, out)
		}
	}
}

func TestStatusShowsConflictsOnlyWhenThereAreSome(t *testing.T) {
	h := reviewFixture(t)
	_, out, _ := h.run("status")
	if !strings.Contains(out, "\nConflicts           1  (concurrent changes; see mtqg review)\n\n") {
		t.Errorf("stdout:\n%s", out)
	}
	_, out, _ = h.run("--json", "status")
	if obj := jsonObject(t, out); obj["concurrent_status_changes"] != float64(1) {
		t.Errorf("object = %v", obj)
	}

	h = initialized(t)
	h.setJournal(record(idA, "todo", "a todo", "yamada", "2026-09-17T09:00:00Z"))
	_, out, _ = h.run("status")
	if strings.Contains(out, "Conflicts") {
		t.Errorf("stdout:\n%s", out)
	}
	// The count is there in JSON even when it is 0: a count is never left out.
	_, out, _ = h.run("--json", "status")
	if !strings.Contains(out, `"concurrent_status_changes": 0`) {
		t.Errorf("stdout:\n%s", out)
	}
}

func TestContextNamesTheConflicts(t *testing.T) {
	h := reviewFixture(t)
	_, out, _ := h.run("context")
	want := "## Attention\n" +
		"- Glossary term \"token\" has conflicting definitions (see mtqg glossary list)\n" +
		"- todo " + idA[:10] + " \"Skip block comments\" has concurrent changes (see mtqg review)\n"
	if !strings.Contains(out, want) {
		t.Errorf("stdout:\n%s\nwant it to contain:\n%s", out, want)
	}

	_, out, _ = h.run("--json", "context")
	var conflict map[string]any
	for _, a := range records(t, jsonObject(t, out), "attention") {
		if a["kind"] == "concurrent_status_change" {
			conflict = a
		}
	}
	if conflict == nil || conflict["id"] != idA || conflict["text"] != "Skip block comments" {
		t.Errorf("attention = %v", conflict)
	}

	// Attention is never left out, whatever the budget.
	_, out, _ = h.run("context", "--max-tokens", "1")
	if !strings.Contains(out, "has concurrent changes (see mtqg review)") {
		t.Errorf("with a budget of 1:\n%s", out)
	}
}

func TestContextSaysHowManyMoreRecordsHaveConflicts(t *testing.T) {
	h := initialized(t)
	var lines []string
	for i := 0; i < 8; i++ {
		id := "c0ffee0" + string(rune('0'+i)) + "0000000000000000000000000"
		lines = append(lines,
			record(id, "todo", "todo "+string(rune('a'+i)), "yamada", "2026-09-17T09:0"+string(rune('0'+i))+":00Z"),
			change(id, "open", "done", nameC, "2026-09-17T10:00:00Z"),
			change(id, "open", "done", "yamada", "2026-09-17T10:01:00Z"),
		)
	}
	h.setJournal(lines...)
	_, out, _ := h.run("context")
	if strings.Count(out, "has concurrent changes") != 5 || !strings.Contains(out, "- (3 more records have concurrent changes; see mtqg review)\n") {
		t.Errorf("stdout:\n%s", out)
	}
	_, out, _ = h.run("--json", "context")
	n := 0
	for _, a := range records(t, jsonObject(t, out), "attention") {
		if a["kind"] == "concurrent_status_change" {
			n++
		}
	}
	if n != 8 {
		t.Errorf("JSON names %d records, want all 8", n)
	}
}
