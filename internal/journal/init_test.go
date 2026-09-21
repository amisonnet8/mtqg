package journal

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testSchema = "# mtqg journal format\n\nThe test schema.\n"

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestInitMakesTheFiles(t *testing.T) {
	root := newRepo(t)
	loc, err := Init(root, testSchema)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, mtqgDirName)
	if loc.Root != root || loc.Dir != dir {
		t.Errorf("got %+v, want root %s dir %s", loc, root, dir)
	}

	want := map[string]string{
		"journal.jsonl":  "",
		".gitattributes": "*.jsonl text eol=lf merge=union\n",
		".gitignore":     ".local/\n",
		"version":        "0\n",
		"SCHEMA.md":      testSchema,
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(want) {
		t.Errorf("got %d entries, want %d", len(entries), len(want))
	}
	for name, content := range want {
		if got := readFile(t, filepath.Join(dir, name)); got != content {
			t.Errorf("%s holds %q, want %q", name, got, content)
		}
	}

	// What Init made can be opened and read at once.
	j, err := Open(root, Options{})
	if err != nil {
		t.Fatalf("Open after Init: %v", err)
	}
	if result, err := j.Read(); err != nil || len(result.Events) != 0 || len(result.Warnings) != 0 {
		t.Errorf("Read = %+v, %v; want an empty journal", result, err)
	}
}

func TestInitFromASubdirectoryMakesItAtTheRoot(t *testing.T) {
	root := newRepo(t)
	sub := mkdir(t, filepath.Join(root, "src", "deep"))
	loc, err := Init(sub, testSchema)
	if err != nil {
		t.Fatal(err)
	}
	if loc.Root != root {
		t.Errorf("made it at %s, want the root %s", loc.Root, root)
	}
	if _, err := os.Stat(filepath.Join(sub, mtqgDirName)); err == nil {
		t.Error("a .mtqg/ was made in the subdirectory")
	}
}

func TestInitRefusesWhereThereIsAMtqgAlready(t *testing.T) {
	root := newRepo(t)
	dir := newMtqg(t, root, "0\n", str("keep me\n"))
	sub := mkdir(t, filepath.Join(root, "sub"))

	for _, start := range []string{root, sub} {
		_, err := Init(start, testSchema)
		var already *AlreadyInitializedError
		if !errors.As(err, &already) || already.Path != dir || !errors.Is(err, ErrAlreadyInitialized) {
			t.Fatalf("Init(%s) = %v, want an AlreadyInitializedError for %s", start, err, dir)
		}
	}
	if got := readFile(t, filepath.Join(dir, journalName)); got != "keep me\n" {
		t.Errorf("an existing .mtqg/ was changed: %q", got)
	}
}

func TestInitOutsideARepository(t *testing.T) {
	isolateGit(t)
	dir := realPath(t, t.TempDir())
	if _, err := Init(dir, testSchema); !errors.Is(err, ErrNotInRepository) {
		t.Fatalf("err = %v, want ErrNotInRepository", err)
	}
	if _, err := os.Stat(filepath.Join(dir, mtqgDirName)); err == nil {
		t.Error("a .mtqg/ was made outside a repository")
	}
}

func TestInitInAWorktreeAndInANestedRepository(t *testing.T) {
	root := newRepo(t)
	newMtqg(t, root, "0\n", nil)

	// A nested repository is its own boundary: Init makes a .mtqg/ there, and
	// leaves the outer one alone.
	inner := mkdir(t, filepath.Join(root, "inner"))
	git(t, inner, "init", "-q", "-b", "main")
	loc, err := Init(inner, testSchema)
	if err != nil || loc.Root != inner {
		t.Fatalf("Init in the nested repository = %+v, %v", loc, err)
	}

	git(t, root, "commit", "-q", "--allow-empty", "-m", "first")
	worktree := filepath.Join(realPath(t, t.TempDir()), "wt")
	git(t, root, "worktree", "add", "-q", "-b", "feature", worktree)
	if loc, err := Init(worktree, testSchema); err != nil || loc.Root != worktree {
		t.Fatalf("Init in a worktree = %+v, %v", loc, err)
	}
}

func TestInitRemovesWhatItMadeWhenAFileFails(t *testing.T) {
	root := newRepo(t)
	// The second file cannot be made: its directory does not exist.
	_, err := createMtqg(root, []initFile{
		{journalName, ""},
		{"missing-dir/file", "x"},
	})
	if err == nil {
		t.Fatal("expected an error")
	}
	if _, err := os.Stat(filepath.Join(root, mtqgDirName)); err == nil {
		t.Error("the half-made .mtqg/ was left behind")
	}
	// The next Init works.
	if _, err := Init(root, testSchema); err != nil {
		t.Fatalf("Init after a failed one: %v", err)
	}
}

// Init must not touch git: no commit, no change of the configuration. And what
// it makes must behave as the format says under the real git.
func TestInitAndTheRealGit(t *testing.T) {
	root := newRepo(t)
	before := git(t, root, "config", "--list", "--local")
	if _, err := Init(root, testSchema); err != nil {
		t.Fatal(err)
	}
	if after := git(t, root, "config", "--list", "--local"); after != before {
		t.Errorf("git's configuration changed:\n%s\n->\n%s", before, after)
	}
	if out := git(t, root, "status", "--porcelain", "--untracked-files=all"); strings.Contains(out, "??") == false {
		t.Errorf("status = %q, want the new files as untracked", out)
	}
	cmdOut := func(args ...string) string {
		t.Helper()
		return git(t, root, args...)
	}
	if out := cmdOut("check-attr", "merge", "eol", "--", ".mtqg/journal.jsonl"); !strings.Contains(out, "merge: union") || !strings.Contains(out, "eol: lf") {
		t.Errorf("check-attr = %q, want merge=union and eol=lf for journal.jsonl", out)
	}

	// Writing creates .local/, which git must ignore.
	j, err := Open(root, Options{Author: yamada})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "x"}); err != nil {
		t.Fatal(err)
	}
	status := cmdOut("status", "--porcelain", "--untracked-files=all")
	if strings.Contains(status, ".local") {
		t.Errorf("git does not ignore .local/:\n%s", status)
	}
	if !strings.Contains(status, ".mtqg/journal.jsonl") {
		t.Errorf("the journal should show as untracked:\n%s", status)
	}
}
