package cli

import (
	"fmt"
	"strings"
	"testing"
)

func logFixture(h *harness) {
	h.setJournal(
		question(idQ2, "Should nested block comments be supported?", nameC, "2026-09-17T09:10:00Z"),
		answerLine(idA2, idQ2, "Not in the first version\nRevisit if there is demand", "yamada", "human", "2026-09-17T09:41:00Z"),
		record(idM, "memo", "Policy: use English for all error messages", "yamada", "2026-09-17T10:32:00Z"),
		record(idA, "todo", "Skip block comments", "yamada", "2026-09-16T10:18:00Z"),
		change(idA, "open", "done", "yamada", "2026-09-17T10:40:00Z"),
		entryLine(idW1, "lexing", "Reading source and turning it into tokens", "yamada", "2026-09-17T11:24:00Z"),
	)
}

func TestLog(t *testing.T) {
	t.Run("every kind, newest first, and a footer", func(t *testing.T) {
		h := initialized(t)
		logFixture(h)
		code, out, errOut := h.run("log")
		wantExit(t, code, 0, out, errOut)
		want := "11:24       glossary  f28c105d1f  lexing: Reading source and turning it into tokens  yamada\n" +
			"10:32       memo      81e74ef5e8  Policy: use English for all error messages         yamada\n" +
			"09:41       answer    a1a1a1a1a1  (to 1012f037b6) Not in the first version           yamada\n" +
			"09:10       question  1012f037b6  Should nested block comments be supported?         claude-code\n" +
			"2026-09-16  todo      6cad4a268d  Skip block comments                                yamada       done\n" +
			"5 records\n"
		if out != want || errOut != "" {
			t.Errorf("stdout:\n%s\nwant:\n%s\nstderr %q", out, want, errOut)
		}
	})

	t.Run("--limit shows the newest few and says how many are left out", func(t *testing.T) {
		h := initialized(t)
		logFixture(h)
		_, out, _ := h.run("log", "--limit", "2")
		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		if len(lines) != 3 || !strings.Contains(lines[0], "glossary") || !strings.Contains(lines[1], "memo") ||
			lines[2] != "2 of 5 records (--limit 0 for all)" {
			t.Errorf("stdout %q", out)
		}
		for _, arg := range []string{"--limit=0", "--limit=5", "--limit=100"} {
			_, out, _ = h.run("log", arg)
			if !strings.HasSuffix(out, "\n5 records\n") || strings.Count(out, "\n") != 6 {
				t.Errorf("%s: stdout %q", arg, out)
			}
		}
	})

	t.Run("the default is the newest 20", func(t *testing.T) {
		h := initialized(t)
		var lines []string
		for i := 0; i < 25; i++ {
			lines = append(lines, record(fmt.Sprintf("%032x", i+0x1000), "memo", fmt.Sprintf("note %02d", i), "yamada", fmt.Sprintf("2026-09-17T09:%02d:00Z", i)))
		}
		h.setJournal(lines...)
		_, out, _ := h.run("log")
		got := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		if len(got) != 21 || !strings.Contains(got[0], "note 24") || !strings.Contains(got[19], "note 05") || got[20] != "20 of 25 records (--limit 0 for all)" {
			t.Errorf("stdout %q", out)
		}
		_, out, _ = h.run("log", "--limit", "0")
		if got := strings.Split(strings.TrimSuffix(out, "\n"), "\n"); len(got) != 26 || got[25] != "25 records" {
			t.Errorf("--limit 0: %d lines", len(got))
		}
	})

	t.Run("--kind picks one kind, by name or letter, and qa is questions and answers", func(t *testing.T) {
		h := initialized(t)
		logFixture(h)
		for _, tt := range []struct {
			kind  string
			lines int
			want  []string
		}{
			{"qa", 2, []string{"answer", "question"}},
			{"q", 2, []string{"answer", "question"}},
			{"memo", 1, []string{"memo"}},
			{"t", 1, []string{"todo"}},
			{"glossary", 1, []string{"glossary"}},
			{"g", 1, []string{"glossary"}},
		} {
			_, out, _ := h.run("log", "--kind", tt.kind)
			lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
			if len(lines) != tt.lines+1 {
				t.Errorf("--kind %s: stdout %q", tt.kind, out)
				continue
			}
			for i, w := range tt.want {
				if !strings.Contains(lines[i], w) {
					t.Errorf("--kind %s: line %d = %q, want %q", tt.kind, i, lines[i], w)
				}
			}
		}
		// The count of the footer is of the kind that was asked for.
		_, out, _ := h.run("log", "--kind", "qa", "--limit", "1")
		if !strings.HasSuffix(out, "1 of 2 records (--limit 0 for all)\n") {
			t.Errorf("stdout %q", out)
		}
		_, out, _ = h.run("log", "--kind=memo")
		if !strings.HasSuffix(out, "\n1 record\n") {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("hidden records are left out", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			question(idQ2, "Nested?", nameC, "2026-09-17T09:10:00Z"),
			answerLine(idA1, idQ2, "An answer of a deleted question", "yamada", "human", "2026-09-17T09:15:00Z"),
			record(idM, "memo", "a memo", "yamada", "2026-09-17T09:20:00Z"),
			deleteLine(idQ2, "yamada", "2026-09-17T09:30:00Z"),
		)
		_, out, _ := h.run("log")
		if strings.Contains(out, "Nested?") || strings.Contains(out, "An answer") || !strings.HasSuffix(out, "\n1 record\n") {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("mistakes in the options are mistakes in the command line", func(t *testing.T) {
		h := initialized(t)
		logFixture(h)
		before := h.readJournal()
		for _, tt := range []struct {
			args []string
			want string
		}{
			{[]string{"log", "--limit", "-1"}, "Option --limit needs a whole number of 0 or more"},
			{[]string{"log", "--limit", "many"}, `not "many"`},
			{[]string{"log", "--limit="}, "Option --limit needs a whole number"},
			{[]string{"log", "--kind", "task"}, `Option --kind needs one of memo, todo, qa, bug, glossary, rule (or its letter), not "task".`},
			{[]string{"log", "--limit"}, "Option --limit needs a value"},
		} {
			code, out, errOut := h.run(tt.args...)
			wantExit(t, code, 2, out, errOut)
			if !strings.Contains(errOut, tt.want) || out != "" {
				t.Errorf("%v: stdout %q, stderr %q", tt.args, out, errOut)
			}
		}
		if h.readJournal() != before {
			t.Error("log wrote to the journal")
		}
	})

	t.Run("an empty journal", func(t *testing.T) {
		h := initialized(t)
		_, out, _ := h.run("log")
		if out != "0 records\n" {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("a closed question dims its answer too", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			question(idQ3, "Nested block comments?", nameC, "2026-09-17T09:00:00Z"),
			answerLine(idA3, idQ3, "Not yet", "yamada", "human", "2026-09-17T09:05:00Z"),
			change(idQ3, "open", "done", "yamada", "2026-09-17T09:10:00Z"),
		)
		h.env.StdoutIsTerminal, h.env.ANSI = true, true
		_, out, _ := h.run("log")
		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		if len(lines) != 3 { // the answer, the question, the footer
			t.Fatalf("stdout %q", out)
		}
		for _, l := range lines[:2] {
			if !strings.Contains(l, "\x1b[2m") {
				t.Errorf("not dimmed: %q", l)
			}
		}
	})

	t.Run("on a terminal the text is cut and the other columns are kept; a pipe is not cut", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(record(idM, "memo", strings.Repeat("あ", 60), "yamada", "2026-09-17T09:00:00Z"))
		h.env.StdoutIsTerminal, h.env.StdoutWidth = true, 60
		_, out, _ := h.run("log")
		line := strings.Split(out, "\n")[0]
		if displayWidth(line) > 59 || !strings.Contains(line, "...") || !strings.HasSuffix(line, "yamada") {
			t.Errorf("line %q is %d columns", line, displayWidth(line))
		}
		h.env.StdoutIsTerminal = false
		_, out, _ = h.run("log")
		if !strings.Contains(out, strings.Repeat("あ", 60)) {
			t.Errorf("stdout %q", out)
		}
	})
}
