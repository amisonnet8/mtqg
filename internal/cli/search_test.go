package cli

import (
	"strings"
	"testing"
)

func TestSearch(t *testing.T) {
	h := editFixture(t)
	h.run("delete", idBug1[:6]) // "Parser crashes on empty input" and its reply are hidden

	t.Run("newest first, one line each, and a count that says what it was for", func(t *testing.T) {
		code, out, errOut := h.run("search", "block")
		wantExit(t, code, 0, out, errOut)
		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		if len(lines) != 3 || lines[2] != `2 records contain "block"` {
			t.Fatalf("stdout =\n%s", out)
		}
		// The question is newer than the todo, so it comes first.
		if !strings.Contains(lines[0], "question  "+idQ2[:10]) || !strings.Contains(lines[1], "todo      "+idA[:10]+"  Skip block comments") {
			t.Errorf("stdout =\n%s", out)
		}
	})

	t.Run("the lines are those of log", func(t *testing.T) {
		_, logOut, _ := h.run("log", "--limit", "0")
		_, out, _ := h.run("search", "block")
		for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n")[:2] {
			// A line of log has the same columns, but they are laid out for all of the
			// records, so compare what is in them.
			if !strings.Contains(strings.Join(strings.Fields(logOut), " "), strings.Join(strings.Fields(line), " ")) {
				t.Errorf("the line %q of search is not in log:\n%s", line, logOut)
			}
		}
	})

	t.Run("case is ignored and a glossary entry is found by its word", func(t *testing.T) {
		code, out, _ := h.run("search", "TOKEN")
		wantExit(t, code, 0, out, "")
		if !strings.Contains(out, "glossary  "+idW1[:10]+"  token: The smallest unit produced by lexing") || !strings.HasSuffix(out, "1 record contains \"TOKEN\"\n") {
			t.Errorf("stdout =\n%s", out)
		}
	})

	t.Run("the words are joined with spaces", func(t *testing.T) {
		_, out, _ := h.run("search", "nested", "block")
		if strings.Contains(out, "No records") {
			t.Errorf("stdout =\n%s", out)
		}
		_, out, _ = h.run("search", "block", "nested")
		if !strings.HasPrefix(out, "No records contain \"block nested\"") {
			t.Errorf("stdout =\n%s", out)
		}
	})

	t.Run("a record that was deleted is not found", func(t *testing.T) {
		code, out, errOut := h.run("search", "crashes")
		wantExit(t, code, 0, out, errOut)
		if out != "No records contain \"crashes\"\n" {
			t.Errorf("stdout %q", out)
		}
		// Its reply went with it.
		_, out, _ = h.run("search", "macOS")
		if out != "No records contain \"macOS\"\n" {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("nothing found is not a failure", func(t *testing.T) {
		code, out, errOut := h.run("search", "zebra")
		wantExit(t, code, 0, out, errOut)
		if out != "No records contain \"zebra\"\n" || errOut != "" {
			t.Errorf("stdout %q, stderr %q", out, errOut)
		}
	})

	t.Run("one match", func(t *testing.T) {
		_, out, _ := h.run("search", "Not in the first")
		if !strings.HasSuffix(out, "1 record contains \"Not in the first\"\n") || !strings.Contains(out, "(to "+idQ2[:10]+")") {
			t.Errorf("stdout =\n%s", out)
		}
	})

	t.Run("no text, or a blank one, is a mistake in the command line", func(t *testing.T) {
		for _, args := range [][]string{{"search"}, {"search", " "}, {"search", ""}} {
			code, out, errOut := h.run(args...)
			wantExit(t, code, 2, out, errOut)
			if !strings.Contains(errOut, "Missing argument. Usage: mtqg search <text>") {
				t.Errorf("%q: stderr %q", args, errOut)
			}
		}
	})

	t.Run("control characters are replaced when shown", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(record(idM, "memo", "escape \x1b[31m here", "yamada", "2026-09-17T09:00:00Z"))
		_, out, _ := h.run("search", "escape")
		if strings.Contains(out, "\x1b") || !strings.Contains(out, "\ufffd") {
			t.Errorf("stdout %q", out)
		}
	})
}

func TestSearchJSON(t *testing.T) {
	h := editFixture(t)
	code, out, errOut := h.run("--json", "search", "block")
	wantExit(t, code, 0, out, errOut)
	obj := jsonObject(t, out)
	recs := records(t, obj, "records")
	if obj["command"] != "search" || obj["query"] != "block" || obj["count"] != float64(2) || len(recs) != 2 ||
		recs[0]["id"] != idQ2 || recs[1]["id"] != idA {
		t.Errorf("object = %v", obj)
	}

	// No match: an empty list, not a missing one.
	_, out, _ = h.run("--json", "search", "zebra")
	if !strings.Contains(out, `"records": []`) || !strings.Contains(out, `"count": 0`) {
		t.Errorf("stdout %s", out)
	}
}
