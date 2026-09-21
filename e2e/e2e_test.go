//go:build e2e

// Package e2e runs the real mtqg binary against real git repositories. It is
// built with the tag e2e and run by make test; it is not part of make check.
//
// The tests in internal/cli call Run directly and check the details. These
// check that the pieces are joined: the binary that is built, the git command,
// the files on disk, standard input and the terminal that is not one.
package e2e

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// binary is the mtqg that TestMain built.
var binary string

func TestMain(m *testing.M) {
	// The test binary is also used as an editor: a test sets EDITOR to it, and
	// this writes the text into the file that the editor is given.
	if text := os.Getenv("MTQG_E2E_EDITOR_TEXT"); text != "" {
		if err := os.WriteFile(os.Args[len(os.Args)-1], []byte(text), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	dir, err := os.MkdirTemp("", "mtqg-e2e-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	binary = filepath.Join(dir, "mtqg")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, "github.com/amisonnet8/mtqg/cmd/mtqg")
	build.Stdout, build.Stderr = os.Stderr, os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "cannot build mtqg:", err)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// repo is a git repository in a temporary directory, with its own git
// configuration, so that the tests do not depend on whoever runs them.
type repo struct {
	t    *testing.T
	dir  string
	home string
	env  []string
}

func newRepo(t *testing.T) *repo {
	t.Helper()
	home := t.TempDir()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r := &repo{t: t, dir: dir, home: home}
	r.env = append(cleanEnv(),
		"HOME="+home, "USERPROFILE="+home,
		"GIT_CONFIG_GLOBAL="+filepath.Join(home, "gitconfig"), "GIT_CONFIG_NOSYSTEM=1",
		"TZ=UTC",
	)
	r.git("init", "-q", "-b", "main")
	r.git("config", "user.name", "yamada")
	r.git("config", "user.email", "yamada@example.com")
	return r
}

// cleanEnv is the environment of the test without what would change mtqg.
func cleanEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(key, "MTQG_") || key == "EDITOR" || key == "NO_COLOR" || key == "VISUAL" {
			continue
		}
		env = append(env, kv)
	}
	return env
}

func (r *repo) git(args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir
	cmd.Env = r.env
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// result is what a run of mtqg did.
type result struct {
	stdout, stderr string
	code           int
}

// run runs mtqg in the repository with extra environment variables and the
// given standard input.
func (r *repo) run(extraEnv []string, stdin string, args ...string) result {
	return r.runIn(r.dir, extraEnv, stdin, args...)
}

func (r *repo) runIn(dir string, extraEnv []string, stdin string, args ...string) result {
	r.t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	cmd.Env = append(append([]string{}, r.env...), extraEnv...)
	cmd.Stdin = strings.NewReader(stdin)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	code := 0
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			r.t.Fatalf("cannot run mtqg: %v", err)
		}
		code = exit.ExitCode()
	}
	return result{stdout: out.String(), stderr: errOut.String(), code: code}
}

// mtqg runs mtqg with no extra environment and no input, and requires success.
func (r *repo) mtqg(args ...string) string {
	r.t.Helper()
	res := r.run(nil, "", args...)
	if res.code != 0 {
		r.t.Fatalf("mtqg %s: exit code %d\nstdout: %s\nstderr: %s", strings.Join(args, " "), res.code, res.stdout, res.stderr)
	}
	return res.stdout
}

var idPattern = regexp.MustCompile(`^[0-9a-f]{10}\n$`)

func TestKeepInTheRepository(t *testing.T) {
	r := newRepo(t)

	out := r.mtqg("init")
	if want := "Created .mtqg/ in " + r.dir + "\nCommit it to share the records.\n"; out != want {
		t.Errorf("init printed %q, want %q", out, want)
	}

	first := r.mtqg("t", "add", "ブロックコメントの読み飛ばし")
	second := r.mtqg("t", "add", "Ignore", "//", "inside", "string", "literals")
	memo := r.mtqg("m", "add", "エラーメッセージは英語で統一する")
	for _, id := range []string{first, second, memo} {
		if !idPattern.MatchString(id) {
			t.Fatalf("add printed %q, want only a short ID", id)
		}
	}

	list := r.mtqg("t", "list")
	if !strings.Contains(list, strings.TrimSpace(first)) || !strings.Contains(list, "ブロックコメントの読み飛ばし") ||
		!strings.Contains(list, "Ignore // inside string literals") || !strings.HasSuffix(list, "2 open (show done: --all)\n") {
		t.Errorf("list =\n%s", list)
	}
	if strings.Contains(list, strings.TrimSpace(memo)) {
		t.Error("a memo is in the list of todos")
	}

	if got := r.mtqg("t", "done", strings.TrimSpace(first)); !strings.HasPrefix(got, "Done: "+strings.TrimSpace(first)+"  ") {
		t.Errorf("done printed %q", got)
	}
	all := r.mtqg("t", "list", "--all")
	if !strings.HasSuffix(all, "1 open, 1 done\n") || strings.Count(all, "done") != 2 { // one line ends with done; the footer says done
		t.Errorf("list --all =\n%s", all)
	}

	status := r.mtqg("status")
	if status != "Open todos          1\nOpen questions      0\nGlossary            0\n\nUncommitted records 3\n" {
		t.Errorf("status = %q", status)
	}

	// Committing takes the records off the count of what is not committed.
	r.git("add", ".mtqg")
	r.git("commit", "-q", "-m", "records")
	if status := r.mtqg("status"); !strings.HasSuffix(status, "Uncommitted records 0\n") {
		t.Errorf("status after the commit = %q", status)
	}
	r.mtqg("t", "add", "one more")
	if status := r.mtqg("status"); !strings.HasSuffix(status, "Uncommitted records 1\n") {
		t.Errorf("status after one more = %q", status)
	}

	// What .mtqg/ holds behaves as the format says under git.
	if out := r.git("status", "--porcelain", "--untracked-files=all"); strings.Contains(out, ".local") {
		t.Errorf("git does not ignore .mtqg/.local/:\n%s", out)
	}
}

func TestFromASubdirectoryAndWithC(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")
	sub := filepath.Join(r.dir, "src", "deep")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}

	res := r.runIn(sub, nil, "", "t", "add", "from a subdirectory")
	if res.code != 0 || !idPattern.MatchString(res.stdout) {
		t.Fatalf("add from a subdirectory: %+v", res)
	}
	elsewhere := t.TempDir()
	res = r.runIn(elsewhere, nil, "", "-C", r.dir, "t", "list")
	if res.code != 0 || !strings.Contains(res.stdout, "from a subdirectory") {
		t.Errorf("-C: %+v", res)
	}

	// Outside a repository the message says where .mtqg/ would be.
	bare := newRepo(t)
	res = bare.run(nil, "", "t", "list")
	if res.code != 1 || res.stderr != "No .mtqg/ found. Run `mtqg init` (will be created at "+bare.dir+")\n" {
		t.Errorf("before init: %+v", res)
	}
}

func TestTextFromStandardInputAndTheEditor(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")

	res := r.run(nil, "first line\nsecond line\n", "m", "add", "-")
	if res.code != 0 {
		t.Fatalf("stdin: %+v", res)
	}

	// The test binary is the editor: it writes the text into the file it is given.
	editor := `"` + os.Args[0] + `"`
	res = r.run([]string{"EDITOR=" + editor, "MTQG_E2E_EDITOR_TEXT=written in the editor\n"}, "", "m", "add")
	if res.code != 0 {
		t.Fatalf("editor: %+v", res)
	}

	memos := r.mtqg("m", "list")
	if !strings.Contains(memos, "first line") || strings.Contains(memos, "second line") ||
		!strings.Contains(memos, "written in the editor") || !strings.HasSuffix(memos, "2 memos\n") {
		t.Errorf("memos =\n%s", memos)
	}

	res = r.run(nil, "\n", "m", "add", "-")
	if res.code != 1 || res.stderr != "Aborting: the text is empty\n" {
		t.Errorf("empty text: %+v", res)
	}
}

func TestExitCodesAndStreams(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")

	tests := []struct {
		name   string
		args   []string
		code   int
		stderr string // a part of it
	}{
		{"an unknown command", []string{"frobnicate"}, 2, `Unknown command "frobnicate"`},
		{"an unknown option", []string{"t", "add", "-x"}, 2, "Unknown option -x"},
		{"no command", nil, 2, "No command given"},
		{"an ID that matches nothing", []string{"t", "done", "ffff"}, 1, `No record matches "ffff"`},
		{"a command that is not built yet", []string{"undo"}, 1, "not available yet"},
		{"an ID that is too short", []string{"t", "done", "ffa"}, 1, "too short"},
		{"an option that takes no value", []string{"log", "--limit"}, 2, "needs a value"},
	}
	for _, tt := range tests {
		res := r.run(nil, "", tt.args...)
		if res.code != tt.code || !strings.Contains(res.stderr, tt.stderr) || res.stdout != "" {
			t.Errorf("%s: %+v, want code %d and %q on standard error only", tt.name, res, tt.code, tt.stderr)
		}
	}

	res := r.run(nil, "", "help")
	if res.code != 0 || !strings.Contains(res.stdout, "mtqg todo done <id>") || res.stderr != "" {
		t.Errorf("help: %+v", res)
	}
	res = r.run(nil, "", "version")
	if res.code != 0 || !strings.HasPrefix(res.stdout, "mtqg ") || !strings.HasSuffix(res.stdout, "Repository format version: 0 (this mtqg supports up to 0)\n") {
		t.Errorf("version: %+v", res)
	}
}

func TestAnAgentSaysWhoItIs(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")

	agent := []string{"MTQG_AUTHOR_KIND=ai", "MTQG_AUTHOR_NAME=claude-code"}
	if res := r.run(agent, "", "t", "add", "written by an agent"); res.code != 0 {
		t.Fatalf("%+v", res)
	}
	r.mtqg("t", "add", "written by a person")

	list := r.mtqg("t", "list")
	if !strings.Contains(list, "claude-code") || !strings.Contains(list, "yamada") {
		t.Errorf("list should show both authors:\n%s", list)
	}
	data, err := os.ReadFile(filepath.Join(r.dir, ".mtqg", "journal.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"author":{"kind":"ai","name":"claude-code"}`) || !strings.Contains(string(data), `"author":{"kind":"human","name":"yamada"}`) {
		t.Errorf("journal:\n%s", data)
	}

	// An AI is never recorded under the name from git.
	res := r.run([]string{"MTQG_AUTHOR_KIND=ai"}, "", "t", "add", "x")
	if res.code != 1 || !strings.Contains(res.stderr, "needs MTQG_AUTHOR_NAME") {
		t.Errorf("an AI without a name: %+v", res)
	}
}

// Two branches that both add records, and are merged: git keeps the lines of
// both sides, because .mtqg/.gitattributes says merge=union, and the records
// are all there. This is the reason the records are kept the way they are.
func TestTwoBranchesMergeAndKeepEveryRecord(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")
	r.mtqg("t", "add", "shared starting point")
	r.git("add", ".mtqg")
	r.git("commit", "-q", "-m", "start")

	r.git("checkout", "-q", "-b", "feature")
	r.mtqg("t", "add", "added on the feature branch")
	r.git("add", ".mtqg")
	r.git("commit", "-q", "-m", "feature")

	r.git("checkout", "-q", "main")
	r.mtqg("t", "add", "added on main")
	r.git("add", ".mtqg")
	r.git("commit", "-q", "-m", "main")

	r.git("merge", "-q", "--no-edit", "feature")

	list := r.mtqg("t", "list")
	for _, want := range []string{"shared starting point", "added on the feature branch", "added on main"} {
		if !strings.Contains(list, want) {
			t.Errorf("the merged list lacks %q:\n%s", want, list)
		}
	}
	if !strings.HasSuffix(list, "3 open (show done: --all)\n") {
		t.Errorf("list =\n%s", list)
	}
	if strings.Contains(r.git("status", "--porcelain"), "<<<<<<<") || strings.Contains(readFile(t, filepath.Join(r.dir, ".mtqg", "journal.jsonl")), "<<<<<<<") {
		t.Error("the merge left conflict markers")
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

var fullIDPattern = regexp.MustCompile(`^[0-9a-f]{32}\n$`)

// The four kinds, end to end: a question is answered and closed, terms are
// defined (one of them twice), and show, log and status read them back. The
// answer holds the full ID of its question in the journal, however short the
// ID that was typed.
func TestQuestionsAnswersAndTheGlossary(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")

	full := r.mtqg("q", "add", "--full-id", "Should nested block comments be supported?")
	if !fullIDPattern.MatchString(full) {
		t.Fatalf("q add --full-id printed %q", full)
	}
	question := strings.TrimSpace(full)

	// Four digits of the ID are enough, and the journal keeps all 32.
	answer := r.mtqg("q", "add", question[:4], "Not", "in", "the", "first", "version")
	if !idPattern.MatchString(answer) {
		t.Fatalf("an answer printed %q", answer)
	}
	agent := []string{"MTQG_AUTHOR_KIND=ai", "MTQG_AUTHOR_NAME=claude-code"}
	if res := r.run(agent, "", "q", "add", question[:10], "Supporting them is generally preferable"); res.code != 0 {
		t.Fatalf("%+v", res)
	}
	journal := readFile(t, filepath.Join(r.dir, ".mtqg", "journal.jsonl"))
	if strings.Count(journal, `"re":"`+question+`"`) != 2 {
		t.Errorf("both answers should hold the full ID of the question:\n%s", journal)
	}

	list := r.mtqg("q", "list")
	if !strings.Contains(list, "2 answers, awaiting confirmation") || !strings.Contains(list, "\u2514 ") || !strings.HasSuffix(list, "1 open (show done: --all)\n") {
		t.Errorf("q list =\n%s", list)
	}

	// A mistyped ID does not turn into a new question.
	res := r.run(nil, "", "q", "add", "a8ec0000", "What", "does", "this", "mean?")
	if res.code != 1 || res.stdout != "" || !strings.Contains(res.stderr, `No record matches "a8ec0000"`) || !strings.Contains(res.stderr, "put the whole text in quotes") {
		t.Errorf("a mistyped ID: %+v", res)
	}
	if res := r.run(nil, "", "q", "add", question[:10]); res.code != 2 || !strings.Contains(res.stderr, "Missing argument") {
		t.Errorf("an ID and no answer: %+v", res)
	}

	if got := r.mtqg("q", "done", question[:10]); !strings.HasPrefix(got, "Done: "+question[:10]+"  ") {
		t.Errorf("q done printed %q", got)
	}
	if got := r.mtqg("q", "list"); got != "0 open (show done: --all)\n" {
		t.Errorf("q list after done =\n%s", got)
	}
	if got := r.mtqg("q", "list", "--all"); !strings.Contains(got, "2 answers, done") || !strings.HasSuffix(got, "0 open, 1 done\n") {
		t.Errorf("q list --all =\n%s", got)
	}

	// The glossary, with a word that is defined twice.
	r.mtqg("g", "add", "token", "The", "smallest", "unit", "produced", "by", "lexing")
	r.mtqg("g", "add", "block comment", "A comment enclosed in /* and */")
	if res := r.run(agent, "", "g", "add", "block comment", "A comment that can span lines"); res.code != 0 {
		t.Fatalf("%+v", res)
	}
	glossary := r.mtqg("g", "list")
	if !strings.HasSuffix(glossary, "3 terms (1 with duplicate definitions)\n") || strings.Count(glossary, "block comment") != 2 {
		t.Errorf("g list =\n%s", glossary)
	}

	if status := r.mtqg("status"); !strings.HasPrefix(status, "Open todos          0\nOpen questions      0\nGlossary            3  (1 with duplicate definitions)\n") {
		t.Errorf("status = %q", status)
	}

	show := r.mtqg("show", question[:6])
	for _, want := range []string{"question  " + question[:10] + "  done", "Should nested block comments be supported?", "Answers (2)", "Not in the first version", "claude-code (ai)", "-> done"} {
		if !strings.Contains(show, want) {
			t.Errorf("show lacks %q:\n%s", want, show)
		}
	}

	log := r.mtqg("log")
	if !strings.HasSuffix(log, "\n6 records\n") || !strings.Contains(log, "(to "+question[:10]+")") {
		t.Errorf("log =\n%s", log)
	}
	if short := r.mtqg("log", "--limit", "2"); !strings.HasSuffix(short, "\n2 of 6 records (--limit 0 for all)\n") {
		t.Errorf("log --limit 2 =\n%s", short)
	}
	if only := r.mtqg("log", "--kind", "g"); !strings.HasSuffix(only, "\n3 records\n") || strings.Contains(only, "question") {
		t.Errorf("log --kind g =\n%s", only)
	}
}

// Two branches answer the same question and are merged: the union merge keeps
// both answers, and both belong to the question, because each holds its full ID.
func TestTwoBranchesAnswerTheSameQuestion(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")
	question := r.mtqg("q", "add", "Should nested block comments be supported?")
	question = strings.TrimSpace(question)
	r.git("add", ".mtqg")
	r.git("commit", "-q", "-m", "question")

	r.git("checkout", "-q", "-b", "feature")
	r.mtqg("q", "add", question, "Yes, they are common")
	r.git("add", ".mtqg")
	r.git("commit", "-q", "-m", "feature answer")

	r.git("checkout", "-q", "main")
	r.mtqg("q", "add", question, "No, not in the first version")
	r.git("add", ".mtqg")
	r.git("commit", "-q", "-m", "main answer")

	r.git("merge", "-q", "--no-edit", "feature")

	list := r.mtqg("q", "list")
	if !strings.Contains(list, "2 answers, awaiting confirmation") {
		t.Errorf("both answers should be there:\n%s", list)
	}
	show := r.mtqg("show", question)
	for _, want := range []string{"Answers (2)", "Yes, they are common", "No, not in the first version"} {
		if !strings.Contains(show, want) {
			t.Errorf("show lacks %q:\n%s", want, show)
		}
	}
}
