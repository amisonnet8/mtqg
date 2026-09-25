package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// candidatesFixture is a repository with every kind of record, at times that put
// them in a known order (newest first: bug 3, todo C, entry, bug 2, answer, todo A,
// question, memo). Todo B is deleted.
func candidatesFixture(t *testing.T) *harness {
	t.Helper()
	h := initialized(t)
	h.setJournal(
		record(idM, "memo", "Policy: use English for all error messages", "yamada", "2026-09-17T09:00:00Z"),
		question(idQ, "Should nested block comments be supported?", "yamada", "2026-09-17T09:30:00Z"),
		record(idA, "todo", "Skip block comments", "yamada", "2026-09-17T10:00:00Z"),
		answerLine(idA1, idQ, "Not in the first version", "yamada", "human", "2026-09-17T10:10:00Z"),
		bugLine(idBug2, "Parser crashes on empty input", "yamada", "2026-09-17T10:20:00Z"),
		entryLine(idW1, "token", "The smallest unit of the source", "yamada", "2026-09-17T10:30:00Z"),
		record(idC, "todo", "Show error positions", "yamada", "2026-09-17T10:40:00Z"),
		change(idC, "open", "done", "yamada", "2026-09-17T10:50:00Z"),
		bugLine(idBug3, "Empty file gives no first token", "yamada", "2026-09-17T11:00:00Z"),
		change(idBug3, "open", "done", "yamada", "2026-09-17T11:10:00Z"),
		record(idB, "todo", "A todo that was deleted", "yamada", "2026-09-17T11:20:00Z"),
		deleteLine(idB, "yamada", "2026-09-17T11:30:00Z"),
	)
	return h
}

func short(id string) string { return id[:10] }

// offered runs candidates for a line: the words typed so far and the word being typed.
func (h *harness) offered(partial string, words ...string) []string {
	h.t.Helper()
	code, out, errOut := h.run(append([]string{"candidates", "--word=" + partial, "--"}, words...)...)
	wantExit(h.t, code, 0, out, errOut)
	if errOut != "" {
		h.t.Errorf("candidates %q %q wrote to standard error: %q", words, partial, errOut)
	}
	return strings.Split(strings.TrimSuffix(out, "\n"), "\n")
}

// values are the candidates without their descriptions.
func (h *harness) values(partial string, words ...string) []string {
	h.t.Helper()
	var values []string
	for _, line := range h.offered(partial, words...) {
		if line != "" {
			values = append(values, strings.SplitN(line, "\t", 2)[0])
		}
	}
	return values
}

func TestCandidatesOfTheCommandLine(t *testing.T) {
	h := candidatesFixture(t)
	withUnbuiltCommand(t)

	tests := []struct {
		name    string
		partial string
		words   []string
		want    []string
	}{
		{name: "the first word starts a kind", partial: "t", want: []string{"todo"}},
		{name: "a kind is not offered as its letter", partial: "m", want: []string{"memo"}},
		{name: "a command that has no kind", partial: "sta", want: []string{"status"}},
		{name: "a kind and a command can share a prefix", partial: "r", want: []string{"rule", "review"}},
		{name: "a command that is not built is not offered", partial: "unb"},
		{name: "the verbs of a kind", words: []string{"t"}, want: []string{"add", "list", "done", "reopen"}},
		{name: "the verbs of a kind spelled out", words: []string{"todo"}, want: []string{"add", "list", "done", "reopen"}},
		{name: "a kind that has few verbs", words: []string{"m"}, want: []string{"add", "list"}},
		{name: "a verb that is being typed", partial: "re", words: []string{"q"}, want: []string{"reopen"}},
		{name: "an unknown kind or command gets nothing", partial: "", words: []string{"frobnicate"}},
		{name: "an unknown verb gets nothing", partial: "", words: []string{"t", "frobnicate"}},

		{name: "the options of a command", partial: "--", words: []string{"log"}, want: []string{"--limit", "--kind", "--full-id", "--no-color", "--json", "--help"}},
		{name: "an option on the line is not offered again", partial: "--", words: []string{"log", "--limit", "5"}, want: []string{"--kind", "--full-id", "--no-color", "--json", "--help"}},
		{name: "the form with = counts too", partial: "--", words: []string{"log", "--limit=5"}, want: []string{"--kind", "--full-id", "--no-color", "--json", "--help"}},
		{name: "-n and --dry-run are one option", partial: "-", words: []string{"archive", "-n"}, want: []string{"-C", "--full-id", "--no-color", "--json", "-h", "--help"}},
		{name: "-h and --help are one option", partial: "-", words: []string{"t", "list", "--help"}, want: []string{"--all", "-C", "--full-id", "--no-color", "--json"}},
		{name: "-C on the line is not offered again", partial: "-", words: []string{"format", "-C", "x"}, want: []string{"--mark", "--full-id", "--no-color", "--json", "-h", "--help"}},
		{name: "-C with its path attached counts too", partial: "-", words: []string{"format", "-Cx"}, want: []string{"--mark", "--full-id", "--no-color", "--json", "-h", "--help"}},
		{name: "before the command, only the options of every command", partial: "-", want: []string{"-C", "--full-id", "--no-color", "--json", "-h", "--help"}},
		{name: "before the verb, too", partial: "-", words: []string{"t"}, want: []string{"-C", "--full-id", "--no-color", "--json", "-h", "--help"}},
		{name: "options are offered only when a - has been typed", partial: "", words: []string{"log"}},
		{name: "an option that was typed in part", partial: "--k", words: []string{"log"}, want: []string{"--kind"}},
		{name: "the options of candidates itself", partial: "--w", words: []string{"candidates"}, want: []string{"--word"}},
		{name: "in a text, a word that starts with - is text", partial: "-x", words: []string{"t", "add", "fix", "the"}},
		{name: "before a text, it is an option", partial: "--f", words: []string{"t", "add"}, want: []string{"--full-id"}},
		{name: "after --, it is text", partial: "-", words: []string{"t", "add", "--"}},
		{name: "after the ID of edit, it is text", partial: "-", words: []string{"edit", idA}},
		{name: "a command that takes an ID takes options anywhere", partial: "--f", words: []string{"t", "done", idA}, want: []string{"--full-id"}},

		{name: "the values of --kind", words: []string{"log", "--kind"}, want: []string{"memo", "todo", "qa", "bug", "glossary", "rule"}},
		{name: "a value of --kind that was typed in part", partial: "b", words: []string{"log", "--kind"}, want: []string{"bug"}},
		{name: "the value of --limit is not completed", words: []string{"log", "--limit"}},
		{name: "a path is left to the shell", words: []string{"-C"}},
		{name: "a path after -C in the middle of a line", words: []string{"t", "done", "-C"}},
		{name: "the form --kind=todo is not completed", partial: "--kind=t", words: []string{"log"}},

		{name: "the shells", words: []string{"completion"}, want: []string{"bash", "zsh", "fish", "powershell"}},
		{name: "a shell that was typed in part", partial: "p", words: []string{"completion"}, want: []string{"powershell"}},
		{name: "nothing after the shell", words: []string{"completion", "bash"}},

		{name: "the text of a memo is not completed", words: []string{"m", "add"}},
		{name: "nor the words of a search", words: []string{"search"}},
		{name: "nor the range of archive", words: []string{"archive"}},
		{name: "a command that takes no argument", words: []string{"status"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := h.values(tc.partial, tc.words...); !slices.Equal(got, tc.want) {
				t.Errorf("candidates %q after %q\n got  %q\n want %q", tc.partial, tc.words, got, tc.want)
			}
		})
	}

	t.Run("the first word is every kind and every command that has no kind and is built", func(t *testing.T) {
		got := h.values("")
		var want []string
		for _, k := range kinds {
			want = append(want, k.name)
		}
		for _, cmd := range commands {
			if cmd.kind == "" && cmd.run != nil {
				want = append(want, cmd.name)
			}
		}
		if !slices.Equal(got, want) {
			t.Errorf("got %q\nwant %q", got, want)
		}
		for _, name := range []string{"completion", "candidates", "init", "help"} {
			if !slices.Contains(got, name) {
				t.Errorf("%q is not offered", name)
			}
		}
		for _, letter := range []string{"m", "t", "q", "b", "g", "r", "unbuilt"} {
			if slices.Contains(got, letter) {
				t.Errorf("%q is offered", letter)
			}
		}
	})

	t.Run("the verbs of every kind are the verbs of the table", func(t *testing.T) {
		for _, k := range kinds {
			if got, want := h.values("", k.name), verbsOf(k.name); !slices.Equal(got, want) {
				t.Errorf("verbs of %s: got %q, want %q", k.name, got, want)
			}
		}
	})
}

func TestCandidatesOfIDs(t *testing.T) {
	h := candidatesFixture(t)

	tests := []struct {
		name    string
		partial string
		words   []string
		want    []string
	}{
		{name: "todo done: the open todos, newest first", words: []string{"t", "done"}, want: []string{short(idA)}},
		{name: "todo reopen: the done ones", words: []string{"t", "reopen"}, want: []string{short(idC)}},
		{name: "qa done: the open questions", words: []string{"q", "done"}, want: []string{short(idQ)}},
		{name: "qa reopen: none is done", words: []string{"q", "reopen"}},
		{name: "bug done: the open bugs", words: []string{"b", "done"}, want: []string{short(idBug2)}},
		{name: "bug reopen: the done bugs", words: []string{"b", "reopen"}, want: []string{short(idBug3)}},
		{name: "the kind by its name", words: []string{"bug", "done"}, want: []string{short(idBug2)}},
		{name: "qa add: the questions, open or done", words: []string{"q", "add"}, want: []string{short(idQ)}},
		{name: "bug add: the bugs, open or done", words: []string{"b", "add"}, want: []string{short(idBug3), short(idBug2)}},
		{name: "todo add is a text: no ID", words: []string{"t", "add"}},
		{
			name: "show: every record in view, newest first", words: []string{"show"},
			want: []string{short(idBug3), short(idC), short(idW1), short(idBug2), short(idA1), short(idA), short(idQ), short(idM)},
		},
		{
			name: "edit: the same", words: []string{"edit"},
			want: []string{short(idBug3), short(idC), short(idW1), short(idBug2), short(idA1), short(idA), short(idQ), short(idM)},
		},
		{
			name: "delete: the same", words: []string{"delete"},
			want: []string{short(idBug3), short(idC), short(idW1), short(idBug2), short(idA1), short(idA), short(idQ), short(idM)},
		},
		{name: "only the IDs that start with what was typed", partial: "6c", words: []string{"show"}, want: []string{short(idA)}},
		{name: "a prefix that fits nothing", partial: "0000", words: []string{"show"}},
		{name: "a word that is not hexadecimal fits nothing", partial: "Should", words: []string{"q", "add"}},
		{name: "more than 10 digits typed: the full ID", partial: idA[:12], words: []string{"show"}, want: []string{idA}},
		{name: "the ID of a done todo is not offered to done", partial: short(idC), words: []string{"t", "done"}},
		{name: "after the ID, the text of an answer", words: []string{"q", "add", idQ}},
		{name: "after the ID, the text of edit", words: []string{"edit", idA}},
		{name: "after the ID of show, nothing", words: []string{"show", idA}},
		{name: "an ID after an option", words: []string{"--json", "show"}, want: []string{
			short(idBug3), short(idC), short(idW1), short(idBug2), short(idA1), short(idA), short(idQ), short(idM),
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := h.values(tc.partial, tc.words...); !slices.Equal(got, tc.want) {
				t.Errorf("candidates %q after %q\n got  %q\n want %q", tc.partial, tc.words, got, tc.want)
			}
		})
	}

	t.Run("a deleted record is not offered anywhere", func(t *testing.T) {
		for _, words := range [][]string{{"show"}, {"edit"}, {"delete"}, {"t", "done"}, {"t", "reopen"}} {
			if slices.Contains(h.values("", words...), short(idB)) {
				t.Errorf("%q offers the deleted todo", words)
			}
		}
	})

	t.Run("an ID that starts the same as another for 10 digits is given in full", func(t *testing.T) {
		const (
			one = "70430f77ff91c2e04a8b33f1d7e6a025"
			two = "70430f77ff4b475185d5cae12dff1a17"
		)
		h := initialized(t)
		h.setJournal(
			record(one, "memo", "Parser now skips // at line end", "yamada", "2026-09-17T09:00:00Z"),
			record(two, "todo", "Skip block comments", "yamada", "2026-09-17T10:00:00Z"),
			record(idM, "memo", "Policy: use English", "yamada", "2026-09-17T08:00:00Z"),
		)
		if got, want := h.values("7043", "show"), []string{two, one}; !slices.Equal(got, want) {
			t.Errorf("got %q, want %q", got, want)
		}
		// The others are still short.
		if got, want := h.values("", "show"), []string{two, one, short(idM)}; !slices.Equal(got, want) {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("-C among the words says where to look", func(t *testing.T) {
		elsewhere := t.TempDir()
		for _, words := range [][]string{
			{"-C", h.root, "t", "done"},
			{"t", "done", "-C", h.root},
			{"-C" + h.root, "t", "done"},
		} {
			code, out, errOut := h.runIn(elsewhere, append([]string{"candidates", "--word=", "--"}, words...)...)
			wantExit(t, code, 0, out, errOut)
			if want := short(idA) + "\tSkip block comments\n"; out != want || errOut != "" {
				t.Errorf("%q: stdout %q, stderr %q", words, out, errOut)
			}
		}
		// Without it, the repository is the one the command runs in.
		code, out, errOut := h.runIn(elsewhere, "candidates", "--word=", "--", "t", "done")
		wantExit(t, code, 0, out, errOut)
		if out != "" || errOut != "" {
			t.Errorf("outside a repository: stdout %q, stderr %q", out, errOut)
		}
		// -C of candidates itself is the same as one among the words.
		code, out, errOut = h.runIn(elsewhere, "-C", h.root, "candidates", "--word=", "--", "t", "done")
		wantExit(t, code, 0, out, errOut)
		if !strings.HasPrefix(out, short(idA)) {
			t.Errorf("-C before the command: stdout %q, stderr %q", out, errOut)
		}
	})
}

func TestCandidatesHaveDescriptions(t *testing.T) {
	h := initialized(t)
	long := strings.Repeat("abcdefghij", 8)
	h.setJournal(
		record(idA, "todo", "Skip block comments", "yamada", "2026-09-17T10:00:00Z"),
		record(idB, "todo", "First line\nSecond line", "yamada", "2026-09-17T10:01:00Z"),
		record(idC, "todo", "Escape \x1b[31m red", "yamada", "2026-09-17T10:02:00Z"),
		record(idM, "memo", long, "yamada", "2026-09-17T10:03:00Z"),
		record(idQ, "memo", strings.Repeat("あ", 40), "yamada", "2026-09-17T10:04:00Z"),
		entryLine(idW1, "token", "The smallest unit", "yamada", "2026-09-17T10:05:00Z"),
	)
	got := map[string]string{}
	for _, line := range h.offered("", "show") {
		value, description, _ := strings.Cut(line, "\t")
		got[value] = description
	}

	if got[short(idA)] != "Skip block comments" {
		t.Errorf("a text: %q", got[short(idA)])
	}
	if got[short(idB)] != "First line" {
		t.Errorf("a text of two lines is its first line: %q", got[short(idB)])
	}
	if got[short(idC)] != "Escape �[31m red" {
		t.Errorf("a control character is replaced: %q", got[short(idC)])
	}
	if d := got[short(idM)]; displayWidth(d) != candidateWidth || !strings.HasSuffix(d, "...") || !strings.HasPrefix(d, long[:candidateWidth-3]) {
		t.Errorf("a long text is cut to %d columns: %q", candidateWidth, d)
	}
	if d := got[short(idQ)]; displayWidth(d) > candidateWidth || !strings.HasSuffix(d, "...") {
		t.Errorf("a wide text is cut by columns, not characters: %q (%d columns)", d, displayWidth(d))
	}
	if got[short(idW1)] != "token: The smallest unit" {
		t.Errorf("an entry of the glossary says its word: %q", got[short(idW1)])
	}

	// A verb or a command says what it does; a kind and an option say nothing.
	for line, want := range map[string]string{
		"reopen": "reopen\tMark a todo as open again",
		"done":   "done\tMark a todo as done",
	} {
		if out := h.offered(line, "t"); len(out) != 1 || out[0] != want {
			t.Errorf("%q: %q", line, out)
		}
	}
	if out := h.offered("st"); len(out) != 1 || !strings.HasPrefix(out[0], "status\tShow what is open") {
		t.Errorf("a command: %q", out)
	}
	if out := h.offered("to"); len(out) != 1 || out[0] != "todo" {
		t.Errorf("a kind: %q", out)
	}
	if out := h.offered("--js", "log"); len(out) != 1 || out[0] != "--json" {
		t.Errorf("an option: %q", out)
	}
}

func TestCandidatesJSON(t *testing.T) {
	h := candidatesFixture(t)
	code, out, errOut := h.run("candidates", "--json", "--word=re", "--", "t")
	wantExit(t, code, 0, out, errOut)
	var got struct {
		Command    string
		Candidates []map[string]string
		Count      int
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if got.Command != "candidates" || got.Count != 1 || len(got.Candidates) != 1 ||
		got.Candidates[0]["value"] != "reopen" || got.Candidates[0]["description"] != "Mark a todo as open again" {
		t.Errorf("got %+v", got)
	}
	if !strings.HasPrefix(out, "{\n  \"command\": \"candidates\",") {
		t.Errorf("the first field is command:\n%s", out)
	}

	// A candidate with nothing to say has no description, and none is an empty list.
	code, out, errOut = h.run("candidates", "--json", "--word=me", "--")
	wantExit(t, code, 0, out, errOut)
	if strings.Contains(out, "description") || !strings.Contains(out, `"value": "memo"`) {
		t.Errorf("out = %s", out)
	}
	code, out, errOut = h.run("candidates", "--json", "--word=zzz", "--")
	wantExit(t, code, 0, out, errOut)
	if !strings.Contains(out, `"candidates": []`) || !strings.Contains(out, `"count": 0`) {
		t.Errorf("out = %s", out)
	}

	// A description is cut to the menu even for a program.
	h.setJournal(record(idA, "todo", strings.Repeat("x", 100), "yamada", "2026-09-17T10:00:00Z"))
	code, out, errOut = h.run("candidates", "--json", "--word=", "--", "show")
	wantExit(t, code, 0, out, errOut)
	if strings.Contains(out, strings.Repeat("x", 100)) || !strings.Contains(out, "...") {
		t.Errorf("the description is not cut:\n%s", out)
	}
}

// A completion is asked for at every TAB, so it does not complain about anything: not
// about the repository, and not about the words.
func TestCandidatesAreQuietWhateverIsWrong(t *testing.T) {
	// The commands and the options need no repository at all.
	staticGot := func(t *testing.T, h *harness, dir string) {
		t.Helper()
		code, out, errOut := h.runIn(dir, "candidates", "--word=sta", "--")
		wantExit(t, code, 0, out, errOut)
		if !strings.HasPrefix(out, "status\t") || errOut != "" {
			t.Errorf("stdout %q, stderr %q", out, errOut)
		}
	}
	silent := func(t *testing.T, h *harness, dir string) {
		t.Helper()
		code, out, errOut := h.runIn(dir, "candidates", "--word=", "--", "t", "done")
		wantExit(t, code, 0, out, errOut)
		if errOut != "" {
			t.Errorf("standard error is not empty: %q", errOut)
		}
		if out != "" && !strings.HasPrefix(out, short(idA)) {
			t.Errorf("stdout %q", out)
		}
	}

	t.Run("no .mtqg/", func(t *testing.T) {
		h := newHarness(t)
		silent(t, h, h.root)
		staticGot(t, h, h.root)
	})
	t.Run("not in a repository", func(t *testing.T) {
		h := newHarness(t)
		dir := t.TempDir()
		silent(t, h, dir)
		staticGot(t, h, dir)
	})
	t.Run("-C names a place that does not exist", func(t *testing.T) {
		h := newHarness(t)
		code, out, errOut := h.run("candidates", "--word=", "--", "-C", filepath.Join(h.root, "nowhere"), "t", "done")
		wantExit(t, code, 0, out, errOut)
		if out != "" || errOut != "" {
			t.Errorf("stdout %q, stderr %q", out, errOut)
		}
	})
	t.Run("a format from the future", func(t *testing.T) {
		h := candidatesFixture(t)
		if err := os.WriteFile(filepath.Join(h.root, ".mtqg", "version"), []byte("1\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		code, out, errOut := h.run("candidates", "--word=", "--", "t", "done")
		wantExit(t, code, 0, out, errOut)
		if out != "" || errOut != "" {
			t.Errorf("stdout %q, stderr %q", out, errOut)
		}
		staticGot(t, h, h.root)
	})
	t.Run("lines that cannot be read and conflict markers are skipped without a word", func(t *testing.T) {
		h := candidatesFixture(t)
		journal := h.readJournal()
		h.setJournal(strings.Split(strings.TrimSuffix("<<<<<<< HEAD\nnot json at all\n"+journal+"=======\n>>>>>>> feature\n{\"id\":\"", "\n"), "\n")...)
		silent(t, h, h.root)
		if got := h.values("", "t", "done"); !slices.Equal(got, []string{short(idA)}) {
			t.Errorf("the records that can be read are still offered: %q", got)
		}
	})
	t.Run("a mistake in the options of candidates itself is a usage error, as anywhere", func(t *testing.T) {
		h := candidatesFixture(t)
		code, out, errOut := h.run("candidates", "--bogus", "--", "t")
		wantExit(t, code, 2, out, errOut)
		if out != "" || !strings.HasPrefix(errOut, "Unknown option --bogus") {
			t.Errorf("stdout %q, stderr %q", out, errOut)
		}
	})
	t.Run("no --word is an empty word, and --word may be given with a space", func(t *testing.T) {
		h := candidatesFixture(t)
		code, out, errOut := h.run("candidates", "--", "t")
		wantExit(t, code, 0, out, errOut)
		if !strings.HasPrefix(out, "add\t") {
			t.Errorf("stdout %q", out)
		}
		code, out, errOut = h.run("candidates", "--word", "re", "--", "t")
		wantExit(t, code, 0, out, errOut)
		if out != "reopen\tMark a todo as open again\n" {
			t.Errorf("stdout %q", out)
		}
	})
	t.Run("what was typed is data: words that look like options of mtqg do not act", func(t *testing.T) {
		h := candidatesFixture(t)
		// --json, --all and -n are words of the line, not options of candidates.
		code, out, errOut := h.run("candidates", "--word=", "--", "t", "list", "--json", "--all")
		wantExit(t, code, 0, out, errOut)
		if out != "" || errOut != "" {
			t.Errorf("stdout %q, stderr %q", out, errOut)
		}
	})
}

func TestCompletionScripts(t *testing.T) {
	h := newHarness(t)
	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			code, out, errOut := h.run("completion", shell)
			wantExit(t, code, 0, out, errOut)
			if errOut != "" || !strings.Contains(out, "mtqg candidates") || !strings.HasSuffix(out, "\n") {
				t.Errorf("stdout %q, stderr %q", out, errOut)
			}
			// The scripts are fixed text that asks mtqg: they do not list a command.
			for _, cmd := range []string{"reopen", "archive", "glossary"} {
				if strings.Contains(out, cmd) {
					t.Errorf("the script lists %q", cmd)
				}
			}

			code, jsonOut, errOut := h.run("completion", shell, "--json")
			wantExit(t, code, 0, jsonOut, errOut)
			var got struct{ Command, Shell, Script string }
			if err := json.Unmarshal([]byte(jsonOut), &got); err != nil || got.Command != "completion" || got.Shell != shell || got.Script != out {
				t.Errorf("--json: %v %+v", err, got)
			}
		})
	}

	for _, tc := range []struct {
		args []string
		want string
	}{
		{nil, "Missing shell: `mtqg completion` needs one of bash, zsh, fish, powershell.\n"},
		{[]string{"nu"}, "Unknown shell \"nu\": `mtqg completion` needs one of bash, zsh, fish, powershell.\n"},
		{[]string{"bash", "zsh"}, "Too many arguments. Usage: mtqg completion <shell>\n"},
	} {
		code, out, errOut := h.run(append([]string{"completion"}, tc.args...)...)
		wantExit(t, code, 2, out, errOut)
		if out != "" || errOut != tc.want {
			t.Errorf("completion %q: stdout %q, stderr %q", tc.args, out, errOut)
		}
	}
	code, out, errOut := h.run("completion", "nu", "--json")
	wantExit(t, code, 2, out, errOut)
	if out != "" || !strings.HasPrefix(errOut, `{"error":{"kind":"usage"`) {
		t.Errorf("--json: stdout %q, stderr %q", out, errOut)
	}
}

// What is completed comes from the tables that parseArgs reads. These tests are what
// keeps the two from drifting apart.

func TestCandidateOptionsAreTheOnesParseArgsTakes(t *testing.T) {
	spellings := append(slices.Clone(globalOptions), "--all", "--mark", "-n", "--dry-run", "--limit", "--kind", "--max-tokens", "--word")
	for _, cmd := range commands {
		offered := map[string]bool{}
		for _, cand := range optionsAt(position{cmd: cmd, seen: map[string]bool{}}) {
			offered[cand.value] = true
		}
		var words []string
		if cmd.kind != "" {
			words = append(words, cmd.kind)
		}
		words = append(words, cmd.name)
		for _, opt := range spellings {
			arg := opt
			switch {
			case opt == "-C":
				arg = "-Cx"
			case strings.HasPrefix(opt, "--") && slices.Contains([]string{"--limit", "--kind", "--max-tokens", "--word"}, opt):
				arg = opt + "=1"
			}
			// A word at the end keeps the arity of the command from being what fails.
			_, err := parseArgs(append(slices.Clone(words), arg, "x"))
			taken := err == nil || !strings.HasPrefix(err.Error(), "Unknown option")
			if taken != offered[opt] {
				t.Errorf("%s: parseArgs takes %s = %v, but candidates offers it = %v", cmd.full(), opt, taken, offered[opt])
			}
		}
	}
}

func TestEveryCommandThatTakesAnIDCompletesIt(t *testing.T) {
	for _, cmd := range commands {
		takesID := cmd.args == argsID || cmd.args == argsIDText
		if takesID && cmd.ids == idsNone {
			t.Errorf("%s takes an ID and does not complete it", cmd.full())
		}
		if cmd.ids != idsNone && !takesID && cmd.name != "add" {
			t.Errorf("%s completes IDs and takes none", cmd.full())
		}
		if cmd.ids != idsNone && cmd.choices != nil {
			t.Errorf("%s completes both IDs and words", cmd.full())
		}
	}
}
