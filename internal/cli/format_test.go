package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// diff makes the text of a commit as git show prints it: a header, the header of a
// file, and the lines of the diff, each with the mark it has.
func diff(marked ...string) string {
	return "commit 3f9a1c0e7d\nAuthor: yamada <yamada@example.com>\nDate:   Thu Sep 17 12:00:00 2026 +0000\n\n" +
		"    Record what happened\n\n" +
		"diff --git a/.mtqg/journal.jsonl b/.mtqg/journal.jsonl\n" +
		"index 1a2b3c4..5d6e7f8 100644\n" +
		"--- a/.mtqg/journal.jsonl\n" +
		"+++ b/.mtqg/journal.jsonl\n" +
		"@@ -1,2 +1,4 @@\n" +
		strings.Join(marked, "\n") + "\n" +
		" context that is not json\n"
}

func TestFormat(t *testing.T) {
	todo := record(idA, "todo", "Skip block comments", "yamada", "2026-09-17T09:00:00Z")
	done := change(idA, "open", "done", nameC, "2026-09-17T09:30:00Z")

	t.Run("the events of a diff, in time order, whatever order they were in", func(t *testing.T) {
		h := newHarness(t)
		h.stdin = diff("+"+done, "+"+todo)
		code, out, errOut := h.run("format")
		wantExit(t, code, 0, out, errOut)
		want := "2026-09-17 09:00  todo  " + idA[:10] + "  Skip block comments  yamada\n" +
			"2026-09-17 09:30  done  " + idA[:10] + "  Skip block comments  claude-code\n"
		if out != want || errOut != "" {
			t.Errorf("stdout\n%s\nwant\n%s\nstderr %q", out, want, errOut)
		}
	})

	t.Run("it needs no repository", func(t *testing.T) {
		// A directory that is not in a git repository at all.
		h := newHarness(t)
		dir := t.TempDir()
		h.stdin = todo + "\n"
		code, out, errOut := h.runIn(dir, "format")
		wantExit(t, code, 0, out, errOut)
		if !strings.HasPrefix(out, "2026-09-17 09:00  todo  "+idA[:10]) {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("a removed line is always marked, the others only with --mark", func(t *testing.T) {
		h := newHarness(t)
		h.stdin = diff("-"+todo, "+"+done)
		_, out, _ := h.run("format")
		want := "-  2026-09-17 09:00  todo  " + idA[:10] + "  Skip block comments  yamada\n" +
			"   2026-09-17 09:30  done  " + idA[:10] + "  Skip block comments  claude-code\n"
		if out != want {
			t.Errorf("stdout\n%s\nwant\n%s", out, want)
		}

		_, out, _ = h.run("format", "--mark")
		want = "-  2026-09-17 09:00  todo  " + idA[:10] + "  Skip block comments  yamada\n" +
			"+  2026-09-17 09:30  done  " + idA[:10] + "  Skip block comments  claude-code\n"
		if out != want {
			t.Errorf("--mark: stdout\n%s\nwant\n%s", out, want)
		}
	})

	t.Run("without any mark there is no column for it, even with --mark", func(t *testing.T) {
		h := newHarness(t)
		h.stdin = todo + "\n" + done + "\n"
		for _, args := range [][]string{{"format"}, {"format", "--mark"}} {
			_, out, _ := h.run(args...)
			if strings.HasPrefix(out, " ") || !strings.HasPrefix(out, "2026-09-17 09:00") {
				t.Errorf("%v: stdout\n%s", args, out)
			}
		}
	})

	t.Run("a merge shows the marks of several diffs", func(t *testing.T) {
		h := newHarness(t)
		h.stdin = "++" + todo + "\n -" + done + "\n"
		_, out, _ := h.run("format", "--mark")
		if !strings.HasPrefix(out, "+  2026-09-17 09:00  todo") || !strings.Contains(out, "\n-  2026-09-17 09:30  done") {
			t.Errorf("stdout\n%s", out)
		}
	})

	t.Run("what a line did: the kind for a creation, the verb for the rest", func(t *testing.T) {
		h := newHarness(t)
		h.stdin = strings.Join([]string{
			record(idM, "memo", "a memo", "yamada", "2026-09-17T09:00:00Z"),
			question(idQ2, "a question", "yamada", "2026-09-17T09:01:00Z"),
			answerLine(idA1, idQ2, "an answer", nameC, "ai", "2026-09-17T09:02:00Z"),
			bugLine(idBug1, "a bug", "yamada", "2026-09-17T09:03:00Z"),
			replyLine(idRep1, idBug1, "a reply", nameC, "ai", "2026-09-17T09:04:00Z"),
			entryLine(idW1, "token", "a definition", "yamada", "2026-09-17T09:05:00Z"),
			todo,
			change(idA, "open", "done", nameC, "2026-09-17T09:31:00Z"),
			change(idA, "done", "open", nameC, "2026-09-17T09:32:00Z"),
			lineBy(map[string]any{"id": idA, "op": "edit", "text": "Skip all comments", "ts": "2026-09-17T09:33:00Z"}, "human", "yamada"),
			deleteLine(idA, "yamada", "2026-09-17T09:34:00Z"),
		}, "\n") + "\n"
		_, out, _ := h.run("format")
		var kinds []string
		for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
			kinds = append(kinds, strings.Fields(line)[2])
		}
		// The todo and the memo are of the same minute, and the todo has the smaller ID.
		want := "todo memo question answer bug reply glossary done reopen edit delete"
		if strings.Join(kinds, " ") != want {
			t.Errorf("kinds %q, want %q\n%s", strings.Join(kinds, " "), want, out)
		}
		for _, want := range []string{
			"answer    " + idA1[:10] + "  (to " + idQ2[:10] + ") an answer",
			"reply     " + idRep1[:10] + "  (to " + idBug1[:10] + ") a reply",
			"glossary  " + idW1[:10] + "  token: a definition",
			"edit      " + idA[:10] + "  Skip all comments",
			"delete    " + idA[:10] + "  Skip block comments",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("stdout lacks %q:\n%s", want, out)
			}
		}
	})

	t.Run("a change of state shows its record's text only if the creation is there", func(t *testing.T) {
		h := newHarness(t)
		h.stdin = done + "\n"
		_, out, _ := h.run("format")
		want := "2026-09-17 09:30  done  " + idA[:10] + "  " + "" + "claude-code"
		if strings.Join(strings.Fields(out), " ") != strings.Join(strings.Fields(want), " ") {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("a line the diff shows unchanged is not shown, and lends its text to a change", func(t *testing.T) {
		h := newHarness(t)
		other := record(idM, "memo", "an unchanged memo", "yamada", "2026-09-17T08:00:00Z")
		h.stdin = diff(" "+todo, " "+other, "+"+done)
		_, out, _ := h.run("format")
		want := "2026-09-17 09:30  done  " + idA[:10] + "  Skip block comments  claude-code\n"
		if out != want {
			t.Errorf("stdout\n%s\nwant\n%s", out, want)
		}
		h.stdin = diff(" "+todo, "+"+done)
		_, out, _ = h.run("--json", "format")
		if obj := jsonObject(t, out); obj["count"] != float64(1) {
			t.Errorf("--json counts the unchanged line: %v", obj)
		}
	})

	t.Run("what is not an event is skipped without a word", func(t *testing.T) {
		h := newHarness(t)
		h.stdin = "not json at all\n{\"a\":1}\n{broken\n+{\"id\":\"x\"}\n" + diff("+"+todo) + "\n"
		code, out, errOut := h.run("format")
		wantExit(t, code, 0, out, errOut)
		if strings.Count(out, "\n") != 1 || errOut != "" {
			t.Errorf("stdout\n%s\nstderr %q", out, errOut)
		}
	})

	t.Run("no events is an empty output", func(t *testing.T) {
		h := newHarness(t)
		h.stdin = "nothing here\n"
		code, out, errOut := h.run("format")
		wantExit(t, code, 0, out, errOut)
		if out != "" || errOut != "" {
			t.Errorf("stdout %q, stderr %q", out, errOut)
		}
	})

	t.Run("a file, and - for standard input", func(t *testing.T) {
		h := newHarness(t)
		path := filepath.Join(t.TempDir(), "2021-01-01..2024-09-18.jsonl")
		if err := os.WriteFile(path, []byte(todo+"\n"+done+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		code, out, errOut := h.run("format", path)
		wantExit(t, code, 0, out, errOut)
		if strings.Count(out, "\n") != 2 {
			t.Errorf("stdout\n%s", out)
		}
		h.stdin = todo + "\n"
		_, out, _ = h.run("format", "-")
		if strings.Count(out, "\n") != 1 {
			t.Errorf("stdout\n%s", out)
		}
	})

	t.Run("a file that cannot be read", func(t *testing.T) {
		h := newHarness(t)
		missing := filepath.Join(t.TempDir(), "missing.jsonl")
		code, out, errOut := h.run("format", missing)
		wantExit(t, code, 1, out, errOut)
		if out != "" || !strings.HasPrefix(errOut, "Cannot read "+missing+": ") {
			t.Errorf("stdout %q, stderr %q", out, errOut)
		}
	})

	t.Run("two files are a mistake in the command line", func(t *testing.T) {
		h := newHarness(t)
		code, out, errOut := h.run("format", "a", "b")
		wantExit(t, code, 2, out, errOut)
	})

	t.Run("--mark is for format only", func(t *testing.T) {
		h := initialized(t)
		code, out, errOut := h.run("log", "--mark")
		wantExit(t, code, 2, out, errOut)
		if !strings.Contains(errOut, "Unknown option --mark") {
			t.Errorf("stderr %q", errOut)
		}
	})

	t.Run("--full-id, and control characters replaced", func(t *testing.T) {
		h := newHarness(t)
		h.stdin = record(idM, "memo", "an \x1b[31m escape", "yamada", "2026-09-17T09:00:00Z") + "\n"
		_, out, _ := h.run("format", "--full-id")
		if !strings.Contains(out, idM+"  an \ufffd[31m escape") || strings.Contains(out, "\x1b") {
			t.Errorf("stdout %q", out)
		}
	})
}

func TestFormatJSON(t *testing.T) {
	h := newHarness(t)
	todo := record(idA, "todo", "Skip block comments", "yamada", "2026-09-17T09:00:00Z")
	done := change(idA, "open", "done", nameC, "2026-09-17T09:30:00Z")
	h.stdin = diff("+"+done, "-"+todo)
	code, out, errOut := h.run("--json", "format")
	wantExit(t, code, 0, out, errOut)
	obj := jsonObject(t, out)
	events := records(t, obj, "events")
	if obj["command"] != "format" || obj["count"] != float64(2) || len(events) != 2 {
		t.Fatalf("object = %v", obj)
	}
	// In time order, as lines of the journal, and the mark that the line had.
	if events[0]["op"] != "create" || events[0]["mark"] != "-" || events[0]["id"] != idA || events[0]["text"] != "Skip block comments" ||
		events[1]["op"] != "status" || events[1]["mark"] != "+" || events[1]["from"] != "open" || events[1]["ts"] != "2026-09-17T09:30:00Z" {
		t.Errorf("events = %v", events)
	}

	h.stdin = todo + "\n"
	_, out, _ = h.run("--json", "format")
	if strings.Contains(out, `"mark"`) {
		t.Errorf("a line without a mark has none: %s", out)
	}
	h.stdin = "nothing\n"
	_, out, _ = h.run("--json", "format")
	if !strings.Contains(out, `"events": []`) || !strings.Contains(out, `"count": 0`) {
		t.Errorf("stdout %s", out)
	}
}
