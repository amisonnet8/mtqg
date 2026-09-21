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
	if status != "Open todos          1\n\nUncommitted records 3\n" {
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
		{"a command that is not built yet", []string{"q", "add", "why?"}, 1, "not available yet"},
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
