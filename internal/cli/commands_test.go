package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func wantExit(t *testing.T, code, want int, stdout, stderr string) {
	t.Helper()
	if code != want {
		t.Fatalf("exit code %d, want %d\nstdout: %s\nstderr: %s", code, want, stdout, stderr)
	}
}

var shortIDPattern = regexp.MustCompile(`^[0-9a-f]{10}\n$`)

func TestInit(t *testing.T) {
	t.Run("makes .mtqg/ and says to commit it", func(t *testing.T) {
		h := newHarness(t)
		code, out, errOut := h.run("init")
		wantExit(t, code, 0, out, errOut)
		want := "Created .mtqg/ in " + h.root + "\nCommit it to share the records.\n"
		if out != want || errOut != "" {
			t.Errorf("stdout %q, stderr %q; want %q", out, errOut, want)
		}
		for _, name := range []string{"journal.jsonl", ".gitattributes", ".gitignore", "version", "SCHEMA.md"} {
			if _, err := os.Stat(filepath.Join(h.root, ".mtqg", name)); err != nil {
				t.Errorf("%s was not made: %v", name, err)
			}
		}
		// It does not commit.
		if out := git(t, h.root, "status", "--porcelain"); !strings.Contains(out, ".mtqg") {
			t.Errorf("the files should be untracked, status: %q", out)
		}
	})

	t.Run("writes the schema for readers of the project", func(t *testing.T) {
		h := initialized(t)
		data, err := os.ReadFile(filepath.Join(h.root, ".mtqg", "SCHEMA.md"))
		if err != nil || !strings.HasPrefix(string(data), "# mtqg journal format") {
			t.Errorf("SCHEMA.md = %.40q, %v", data, err)
		}
	})

	t.Run("from a subdirectory it still makes it at the root", func(t *testing.T) {
		h := newHarness(t)
		sub := filepath.Join(h.root, "src", "deep")
		if err := os.MkdirAll(sub, 0o750); err != nil {
			t.Fatal(err)
		}
		code, out, errOut := h.runIn(sub, "init")
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(out, "in "+h.root+"\n") {
			t.Errorf("stdout %q should name the root %s", out, h.root)
		}
	})

	t.Run("a second init is refused and changes nothing", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(record(idA, "todo", "keep me", "yamada", "2026-09-17T10:18:00Z"))
		before := h.readJournal()
		code, out, errOut := h.run("init")
		wantExit(t, code, 1, out, errOut)
		if !strings.Contains(errOut, ".mtqg/ already exists: "+filepath.Join(h.root, ".mtqg")) || out != "" {
			t.Errorf("stdout %q, stderr %q", out, errOut)
		}
		if h.readJournal() != before {
			t.Error("the journal was changed")
		}
	})

	t.Run("outside a git repository", func(t *testing.T) {
		isolateGit(t)
		h := &harness{t: t, vars: map[string]string{}}
		dir, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		code, out, errOut := h.runIn(dir, "init")
		wantExit(t, code, 1, out, errOut)
		if !strings.Contains(errOut, "Not inside a git repository") {
			t.Errorf("stderr %q", errOut)
		}
	})

	t.Run("-C points somewhere else", func(t *testing.T) {
		h := newHarness(t)
		elsewhere := t.TempDir()
		code, out, errOut := h.runIn(elsewhere, "-C", h.root, "init")
		wantExit(t, code, 0, out, errOut)
		if _, err := os.Stat(filepath.Join(h.root, ".mtqg")); err != nil {
			t.Error(err)
		}
	})
}

func TestNoMtqgYet(t *testing.T) {
	h := newHarness(t)
	for _, args := range [][]string{{"t", "add", "x"}, {"t", "list"}, {"m", "list"}, {"t", "done", "abcd"}, {"status"}} {
		code, out, errOut := h.run(args...)
		wantExit(t, code, 1, out, errOut)
		want := "No .mtqg/ found. Run `mtqg init` (will be created at " + h.root + ")\n"
		if errOut != want || out != "" {
			t.Errorf("%v: stdout %q, stderr %q; want the message %q", args, out, errOut, want)
		}
	}
}

func TestAdd(t *testing.T) {
	t.Run("prints only the ID", func(t *testing.T) {
		h := initialized(t)
		for _, args := range [][]string{{"t", "add", "Skip", "block", "comments"}, {"m", "add", "エラーメッセージは英語で統一する"}} {
			code, out, errOut := h.run(args...)
			wantExit(t, code, 0, out, errOut)
			if !shortIDPattern.MatchString(out) || errOut != "" {
				t.Errorf("%v: stdout %q, stderr %q; want a short ID and nothing else", args, out, errOut)
			}
		}
		journal := h.readJournal()
		if !strings.Contains(journal, `"type":"todo","status":"open","text":"Skip block comments"`) ||
			!strings.Contains(journal, `"type":"memo","text":"エラーメッセージは英語で統一する"`) {
			t.Errorf("journal = %s", journal)
		}
		if !strings.Contains(journal, `"author":{"kind":"human","name":"tester"}`) {
			t.Errorf("the author should be the user.name of git:\n%s", journal)
		}
	})

	t.Run("--full-id prints the whole ID", func(t *testing.T) {
		h := initialized(t)
		code, out, errOut := h.run("t", "add", "--full-id", "x")
		wantExit(t, code, 0, out, errOut)
		if !regexp.MustCompile(`^[0-9a-f]{32}\n$`).MatchString(out) {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("reads the text from standard input", func(t *testing.T) {
		h := initialized(t)
		h.stdin = "first line\nsecond line\n"
		code, out, errOut := h.run("m", "add", "-")
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(h.readJournal(), `"text":"first line\nsecond line"`) {
			t.Errorf("journal = %s", h.readJournal())
		}
	})

	t.Run("opens the editor when there is no text", func(t *testing.T) {
		h := initialized(t)
		h.vars["EDITOR"] = "myeditor"
		h.env.RunEditor = func(argv []string) error {
			return os.WriteFile(argv[len(argv)-1], []byte("a long note\nover two lines\n"), 0o600)
		}
		code, out, errOut := h.run("m", "add")
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(h.readJournal(), `"text":"a long note\nover two lines"`) {
			t.Errorf("journal = %s", h.readJournal())
		}
	})

	t.Run("an empty text writes nothing", func(t *testing.T) {
		h := initialized(t)
		h.stdin = "\n"
		code, out, errOut := h.run("m", "add", "-")
		wantExit(t, code, 1, out, errOut)
		if errOut != "Aborting: the text is empty\n" || out != "" {
			t.Errorf("stdout %q, stderr %q", out, errOut)
		}
		if h.readJournal() != "" {
			t.Errorf("journal = %q, want it empty", h.readJournal())
		}
	})

	t.Run("text that is not UTF-8 is refused, not altered", func(t *testing.T) {
		h := initialized(t)
		h.stdin = "caf\xe9\n"
		code, out, errOut := h.run("m", "add", "-")
		wantExit(t, code, 1, out, errOut)
		if !strings.Contains(errOut, "The text is not valid UTF-8.") || h.readJournal() != "" {
			t.Errorf("stderr %q, journal %q", errOut, h.readJournal())
		}
	})

	t.Run("the text can start with - after --", func(t *testing.T) {
		h := initialized(t)
		code, out, errOut := h.run("t", "add", "--", "-1", "is", "not", "allowed")
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(h.readJournal(), `"text":"-1 is not allowed"`) {
			t.Errorf("journal = %s", h.readJournal())
		}
	})

	t.Run("an AI says so with the environment", func(t *testing.T) {
		h := initialized(t)
		h.vars["MTQG_AUTHOR_KIND"] = "ai"
		h.vars["MTQG_AUTHOR_NAME"] = "claude-code"
		code, out, errOut := h.run("t", "add", "written by an agent")
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(h.readJournal(), `"author":{"kind":"ai","name":"claude-code"}`) {
			t.Errorf("journal = %s", h.readJournal())
		}
	})

	t.Run("an AI without a name is refused before anything is written", func(t *testing.T) {
		h := initialized(t)
		h.vars["MTQG_AUTHOR_KIND"] = "ai"
		code, out, errOut := h.run("t", "add", "x")
		wantExit(t, code, 1, out, errOut)
		if !strings.Contains(errOut, "MTQG_AUTHOR_KIND=ai needs MTQG_AUTHOR_NAME") || h.readJournal() != "" {
			t.Errorf("stderr %q, journal %q", errOut, h.readJournal())
		}
	})

	t.Run("no author name", func(t *testing.T) {
		h := initialized(t)
		git(t, h.root, "config", "--unset", "user.name")
		code, out, errOut := h.run("t", "add", "x")
		wantExit(t, code, 1, out, errOut)
		if !strings.Contains(errOut, "No author name") {
			t.Errorf("stderr %q", errOut)
		}
	})

	t.Run("refused while conflict markers are in the journal", func(t *testing.T) {
		h := initialized(t)
		h.setJournal("<<<<<<< HEAD", record(idA, "todo", "a", "yamada", "2026-09-17T10:18:00Z"), "=======", record(idB, "todo", "b", "yamada", "2026-09-17T10:19:00Z"), ">>>>>>> other")
		before := h.readJournal()
		code, out, errOut := h.run("t", "add", "x")
		wantExit(t, code, 1, out, errOut)
		if !strings.Contains(errOut, "unresolved merge conflict markers") || h.readJournal() != before {
			t.Errorf("stderr %q", errOut)
		}
	})

	t.Run("from -C", func(t *testing.T) {
		h := initialized(t)
		code, out, errOut := h.runIn(t.TempDir(), "-C", h.root, "t", "add", "from elsewhere")
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(h.readJournal(), "from elsewhere") {
			t.Error("the record is not there")
		}
	})
}

func TestListTodos(t *testing.T) {
	fixture := func(h *harness) {
		h.setJournal(
			record(idA, "todo", "Skip block comments /* */", "claude-code", "2026-09-17T10:18:00Z"),
			record(idB, "todo", "ブロックコメントの読み飛ばし", "yamada", "2026-09-16T09:00:00Z"),
			record(idC, "todo", "Show error positions\nsecond line of the note", "yamada", "2026-09-17T11:06:00Z"),
			change(idC, "open", "done", "yamada", "2026-09-17T11:30:00Z"),
			record(idM, "memo", "not a todo", "yamada", "2026-09-17T09:00:00Z"),
		)
	}

	t.Run("the open todos, oldest first, and a footer", func(t *testing.T) {
		h := initialized(t)
		fixture(h)
		code, out, errOut := h.run("t", "list")
		wantExit(t, code, 0, out, errOut)
		lines := strings.Split(out, "\n")
		if len(lines) != 4 {
			t.Fatalf("stdout %q", out)
		}
		// The oldest is first: the one from the day before.
		if !strings.HasPrefix(lines[0], idB[:10]) || !strings.HasPrefix(lines[1], idA[:10]) {
			t.Errorf("order: %q", out)
		}
		if !strings.HasSuffix(lines[0], "2026-09-16") || !strings.HasSuffix(lines[1], "10:18") {
			t.Errorf("times: %q", out)
		}
		if lines[2] != "2 open (show done: --all)" {
			t.Errorf("footer = %q", lines[2])
		}
		if errOut != "" {
			t.Errorf("stderr %q", errOut)
		}
	})

	t.Run("--all shows the finished, with done, and a different footer", func(t *testing.T) {
		h := initialized(t)
		fixture(h)
		code, out, errOut := h.run("t", "list", "--all")
		wantExit(t, code, 0, out, errOut)
		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		if len(lines) != 4 {
			t.Fatalf("stdout %q", out)
		}
		var doneLines int
		for _, l := range lines[:3] {
			if strings.HasSuffix(l, "done") {
				doneLines++
				if !strings.HasPrefix(l, idC[:10]) {
					t.Errorf("done on the wrong line: %q", l)
				}
				// Only the first line of a text is shown.
				if !strings.Contains(l, "Show error positions") || strings.Contains(l, "second line") {
					t.Errorf("text: %q", l)
				}
			}
		}
		if doneLines != 1 || lines[3] != "2 open, 1 done" {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("nothing to list", func(t *testing.T) {
		h := initialized(t)
		code, out, errOut := h.run("t", "list")
		wantExit(t, code, 0, out, errOut)
		if out != "0 open (show done: --all)\n" {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("--full-id", func(t *testing.T) {
		h := initialized(t)
		fixture(h)
		_, out, _ := h.run("t", "list", "--full-id")
		if !strings.Contains(out, idA) || !strings.Contains(out, idB) {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("text is cut to the window on a terminal, and never to a pipe", func(t *testing.T) {
		h := initialized(t)
		long := strings.Repeat("あ", 80)
		h.setJournal(record(idA, "todo", long, "yamada", "2026-09-17T10:18:00Z"))

		h.env.StdoutIsTerminal, h.env.StdoutWidth = true, 60
		_, out, _ := h.run("t", "list")
		first := strings.Split(out, "\n")[0]
		if displayWidth(first) > 59 || !strings.Contains(first, "...") {
			t.Errorf("on a terminal: %q (%d columns)", first, displayWidth(first))
		}

		h.env.StdoutIsTerminal, h.env.StdoutWidth = false, 60
		_, out, _ = h.run("t", "list")
		if !strings.Contains(out, long) {
			t.Errorf("to a pipe the whole text should be there: %q", out)
		}
	})

	t.Run("color only on a terminal that understands it, unless it is turned off", func(t *testing.T) {
		h := initialized(t)
		fixture(h)
		colored := func() bool {
			_, out, _ := h.run("t", "list")
			return strings.Contains(out, "\x1b[")
		}
		if colored() {
			t.Error("color to a pipe")
		}
		h.env.StdoutIsTerminal, h.env.ANSI = true, true
		if !colored() {
			t.Error("no color on a terminal")
		}
		if _, out, _ := h.run("--no-color", "t", "list"); strings.Contains(out, "\x1b[") {
			t.Error("color with --no-color")
		}
		h.vars["NO_COLOR"] = "1"
		if colored() {
			t.Error("color with NO_COLOR")
		}
		delete(h.vars, "NO_COLOR")
		h.vars["TERM"] = "dumb"
		if colored() {
			t.Error("color on a dumb terminal")
		}
		delete(h.vars, "TERM")
		h.env.ANSI = false
		if colored() {
			t.Error("color on a terminal that cannot show it")
		}
	})

	t.Run("a record cannot change what the terminal does", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(record(idA, "todo", "clear\x1b[2Jscreen\rfake", "evil\x1b]0;title\x07", "2026-09-17T10:18:00Z"))
		code, out, errOut := h.run("t", "list")
		wantExit(t, code, 0, out, errOut)
		if strings.ContainsAny(out, "\x1b\r\x07") {
			t.Errorf("stdout has a control character: %q", out)
		}
		// A carriage return ends the first line, so what follows it is not shown.
		if !strings.Contains(out, "clear\uFFFD[2Jscreen") || strings.Contains(out, "fake") {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("lines that cannot be read are warnings on standard error", func(t *testing.T) {
		h := initialized(t)
		lines := []string{record(idA, "todo", "fine", "yamada", "2026-09-17T10:18:00Z")}
		for range 7 {
			lines = append(lines, "not json")
		}
		h.setJournal(lines...)
		code, out, errOut := h.run("t", "list")
		wantExit(t, code, 0, out, errOut)
		if !strings.HasPrefix(out, idA[:10]) || strings.Contains(out, "warning") {
			t.Errorf("stdout %q should hold the list only", out)
		}
		warnings := strings.Split(strings.TrimSuffix(errOut, "\n"), "\n")
		if len(warnings) != maxWarnings+1 || !strings.Contains(warnings[0], "line 2 is not a valid JSON object") || !strings.Contains(warnings[maxWarnings], "(2 more lines could not be read)") {
			t.Errorf("stderr %q", errOut)
		}
	})

	t.Run("records that other people wrote and merged in any order", func(t *testing.T) {
		h := initialized(t)
		// The lines are in the reverse of their times, as a union merge may leave them.
		h.setJournal(
			change(idA, "open", "done", "yamada", "2026-09-17T11:00:00Z"),
			record(idB, "todo", "second", "yamada", "2026-09-17T10:30:00Z"),
			record(idA, "todo", "first", "claude-code", "2026-09-17T10:00:00Z"),
		)
		_, out, _ := h.run("t", "list")
		if strings.Count(out, "\n") != 2 || !strings.HasPrefix(out, idB[:10]) {
			t.Errorf("stdout %q: idA is done, so only idB is left", out)
		}
	})
}

func TestListMemos(t *testing.T) {
	h := initialized(t)
	h.setJournal(
		record(idM, "memo", "エラーメッセージは英語で統一する", "yamada", "2026-09-17T09:00:00Z"),
		record(idA, "todo", "not a memo", "yamada", "2026-09-17T09:30:00Z"),
	)
	code, out, errOut := h.run("m", "list")
	wantExit(t, code, 0, out, errOut)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], idM[:10]) || lines[1] != "1 memo" {
		t.Errorf("stdout %q", out)
	}

	h.setJournal(record(idM, "memo", "a", "yamada", "2026-09-17T09:00:00Z"), record(idQ, "memo", "b", "yamada", "2026-09-17T09:01:00Z"))
	_, out, _ = h.run("memo", "list")
	if !strings.HasSuffix(out, "2 memos\n") {
		t.Errorf("stdout %q", out)
	}
}

func TestDoneAndReopen(t *testing.T) {
	setup := func(t *testing.T) *harness {
		h := initialized(t)
		h.setJournal(
			record(idA, "todo", "Skip block comments /* */", "claude-code", "2026-09-17T10:18:00Z"),
			record(idB, "todo", "ブロックコメントの読み飛ばし", "yamada", "2026-09-17T10:52:00Z"),
			record(idM, "memo", "a memo", "yamada", "2026-09-17T09:00:00Z"),
			record(idQ, "qa", "Nested block comments?", "claude-code", "2026-09-17T11:05:00Z"),
		)
		return h
	}

	t.Run("done says what it did and writes the state it saw", func(t *testing.T) {
		h := setup(t)
		code, out, errOut := h.run("t", "done", idA[:10])
		wantExit(t, code, 0, out, errOut)
		if out != "Done: 6cad4a268d  Skip block comments /* */\n" || errOut != "" {
			t.Errorf("stdout %q, stderr %q", out, errOut)
		}
		lines := strings.Split(strings.TrimSuffix(h.readJournal(), "\n"), "\n")
		last := lines[len(lines)-1]
		if !strings.Contains(last, `"op":"status","from":"open","status":"done"`) || !strings.Contains(last, `"id":"`+idA+`"`) {
			t.Errorf("last line: %s", last)
		}
		_, list, _ := h.run("t", "list")
		if strings.Contains(list, idA[:10]) {
			t.Errorf("the todo should be gone from the open list: %q", list)
		}
	})

	t.Run("an ID can be as short as it is unique", func(t *testing.T) {
		h := setup(t)
		// idA and idB both start with 6cad4a26; nine digits tell them apart.
		code, out, errOut := h.run("t", "done", "6cad4a268")
		wantExit(t, code, 0, out, errOut)
		if !strings.HasPrefix(out, "Done: 6cad4a268d") {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("done twice writes nothing the second time and is not an error", func(t *testing.T) {
		h := setup(t)
		h.run("t", "done", idA[:10])
		before := h.readJournal()
		code, out, errOut := h.run("t", "done", idA[:10])
		wantExit(t, code, 0, out, errOut)
		if out != "Already done: 6cad4a268d  Skip block comments /* */\n" || h.readJournal() != before {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("reopen", func(t *testing.T) {
		h := setup(t)
		h.run("t", "done", idB[:10])
		code, out, errOut := h.run("t", "reopen", idB[:10])
		wantExit(t, code, 0, out, errOut)
		if out != "Reopened: 6cad4a26ff  ブロックコメントの読み飛ばし\n" {
			t.Errorf("stdout %q", out)
		}
		if !strings.Contains(h.readJournal(), `"from":"done","status":"open"`) {
			t.Errorf("journal = %s", h.readJournal())
		}
		code, out, _ = h.run("t", "reopen", idB[:10])
		if code != 0 || !strings.HasPrefix(out, "Already open: ") {
			t.Errorf("code %d stdout %q", code, out)
		}
	})

	t.Run("an ID that matches two records lists them with their full IDs", func(t *testing.T) {
		h := setup(t)
		// idA and idB both start with 6cad4a26.
		code, out, errOut := h.run("t", "done", "6cad4a26")
		wantExit(t, code, 1, out, errOut)
		if !strings.HasPrefix(errOut, `Ambiguous ID "6cad4a26" matches 2 records:`+"\n") {
			t.Fatalf("stderr %q", errOut)
		}
		if !strings.Contains(errOut, "  "+idA+"  todo  ") || !strings.Contains(errOut, "  "+idB+"  todo  ") {
			t.Errorf("stderr %q should list the full IDs", errOut)
		}
		if !strings.Contains(errOut, "claude-code") || !strings.Contains(errOut, "yamada") {
			t.Errorf("stderr %q should show who wrote them", errOut)
		}
	})

	t.Run("nothing matches", func(t *testing.T) {
		h := setup(t)
		code, out, errOut := h.run("t", "done", "ffff")
		wantExit(t, code, 1, out, errOut)
		if errOut != "No record matches \"ffff\"\n" {
			t.Errorf("stderr %q", errOut)
		}
	})

	t.Run("a record of another kind says what it is", func(t *testing.T) {
		h := setup(t)
		code, out, errOut := h.run("t", "done", idM[:10])
		wantExit(t, code, 1, out, errOut)
		if errOut != "81e74ef5e8 is a memo, not a todo\n" {
			t.Errorf("stderr %q", errOut)
		}
		code, out, errOut = h.run("t", "done", idQ[:10])
		wantExit(t, code, 1, out, errOut)
		if errOut != "2217beaddb is a question, not a todo; use `mtqg qa done 2217beaddb`\n" {
			t.Errorf("stderr %q", errOut)
		}
	})

	t.Run("nothing is written when the ID is wrong", func(t *testing.T) {
		h := setup(t)
		before := h.readJournal()
		h.run("t", "done", "ffff")
		h.run("t", "done", idM[:10])
		if h.readJournal() != before {
			t.Error("the journal was changed")
		}
	})

	t.Run("the state changes as the writer saw it, even by another author", func(t *testing.T) {
		h := setup(t)
		h.vars["MTQG_AUTHOR_KIND"] = "ai"
		h.vars["MTQG_AUTHOR_NAME"] = "claude-code"
		code, out, errOut := h.run("t", "done", idB[:10])
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(h.readJournal(), `"author":{"kind":"ai","name":"claude-code"}`) {
			t.Errorf("journal = %s", h.readJournal())
		}
	})
}

func TestStatus(t *testing.T) {
	t.Run("counts the open todos and the uncommitted records", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(
			record(idA, "todo", "a", "yamada", "2026-09-17T10:18:00Z"),
			record(idB, "todo", "b", "yamada", "2026-09-17T10:19:00Z"),
			change(idA, "open", "done", "yamada", "2026-09-17T10:20:00Z"),
			record(idM, "memo", "m", "yamada", "2026-09-17T10:21:00Z"),
		)
		code, out, errOut := h.run("status")
		wantExit(t, code, 0, out, errOut)
		want := "Open todos          1\n\nUncommitted records 3\n"
		if out != want {
			t.Errorf("stdout %q, want %q", out, want)
		}
	})

	t.Run("committing brings the count to zero, and a change to an old record counts once", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(record(idA, "todo", "a", "yamada", "2026-09-17T10:18:00Z"), record(idB, "todo", "b", "yamada", "2026-09-17T10:19:00Z"))
		git(t, h.root, "add", ".mtqg")
		git(t, h.root, "commit", "-q", "-m", "records")

		_, out, _ := h.run("status")
		if !strings.HasSuffix(out, "Uncommitted records 0\n") {
			t.Errorf("stdout %q", out)
		}
		h.run("t", "done", idA[:10])
		h.run("t", "reopen", idA[:10])
		h.run("t", "add", "a new one")
		_, out, _ = h.run("status")
		if !strings.HasSuffix(out, "Uncommitted records 2\n") { // the old todo, changed twice, and the new one
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("git that cannot be run leaves the count unknown, with a warning", func(t *testing.T) {
		h := initialized(t)
		h.run("t", "add", "x")
		t.Setenv("PATH", t.TempDir())
		code, out, errOut := h.run("status")
		wantExit(t, code, 0, out, errOut)
		if !strings.HasSuffix(out, "Uncommitted records unknown\n") || !strings.HasPrefix(out, "Open todos          1\n") {
			t.Errorf("stdout %q", out)
		}
		if !strings.Contains(errOut, "warning: git could not be run") {
			t.Errorf("stderr %q", errOut)
		}
	})
}

func TestVersion(t *testing.T) {
	t.Run("in a repository", func(t *testing.T) {
		h := initialized(t)
		code, out, errOut := h.run("version")
		wantExit(t, code, 0, out, errOut)
		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		if len(lines) != 2 || !strings.HasPrefix(lines[0], "mtqg ") || lines[1] != "Repository format version: 0 (this mtqg supports up to 0)" {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("outside a repository it is not an error", func(t *testing.T) {
		isolateGit(t)
		h := &harness{t: t, vars: map[string]string{}}
		dir, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		code, out, errOut := h.runIn(dir, "version")
		wantExit(t, code, 0, out, errOut)
		if !strings.HasSuffix(out, "Repository format version: unknown (no .mtqg/ found)\n") {
			t.Errorf("stdout %q", out)
		}
	})

	t.Run("a format that is too new is shown, and other commands refuse it", func(t *testing.T) {
		h := initialized(t)
		if err := os.WriteFile(filepath.Join(h.root, ".mtqg", "version"), []byte("3\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, out, _ := h.run("version")
		if !strings.HasSuffix(out, "Repository format version: 3 (this mtqg supports up to 0)\n") {
			t.Errorf("stdout %q", out)
		}
		code, out, errOut := h.run("t", "list")
		wantExit(t, code, 1, out, errOut)
		if !strings.Contains(errOut, "format version 3, but this mtqg understands up to version 0") {
			t.Errorf("stderr %q", errOut)
		}
		code, out, errOut = h.run("t", "add", "x")
		wantExit(t, code, 1, out, errOut)
	})
}

func TestHelpAndMistakes(t *testing.T) {
	h := newHarness(t)

	code, out, errOut := h.run("help")
	wantExit(t, code, 0, out, errOut)
	for _, want := range []string{"mtqg todo done <id>", "mtqg init", "mtqg status", "--full-id", "memo (m), todo (t), qa (q), glossary (g)"} {
		if !strings.Contains(out, want) {
			t.Errorf("help does not mention %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "mtqg archive") {
		t.Error("help lists a command that is not available yet")
	}

	code, out, errOut = h.run("t", "done", "--help")
	wantExit(t, code, 0, out, errOut)
	if !strings.HasPrefix(out, "usage: mtqg todo done <id>\n") {
		t.Errorf("stdout %q", out)
	}

	code, out, errOut = h.run()
	wantExit(t, code, 2, out, errOut)
	if !strings.Contains(errOut, "No command given") || out != "" {
		t.Errorf("stdout %q, stderr %q", out, errOut)
	}

	code, out, errOut = h.run("frobnicate")
	wantExit(t, code, 2, out, errOut)
	if !strings.Contains(errOut, `Unknown command "frobnicate"`) {
		t.Errorf("stderr %q", errOut)
	}

	code, out, errOut = h.run("t", "add", "-x")
	wantExit(t, code, 2, out, errOut)
	if !strings.Contains(errOut, "Unknown option -x") {
		t.Errorf("stderr %q", errOut)
	}

	// A command that exists but is not built yet says so.
	code, out, errOut = h.run("q", "add", "why?")
	wantExit(t, code, 1, out, errOut)
	if !strings.Contains(errOut, "`mtqg qa add` is not available yet") {
		t.Errorf("stderr %q", errOut)
	}
}
