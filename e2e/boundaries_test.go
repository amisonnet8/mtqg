//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// gitResult runs git and returns its output and its error, for a command that is
// expected to fail (a merge with a conflict).
func (r *repo) gitResult(args ...string) (string, error) {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir
	cmd.Env = r.env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// trimmed is the ID that a command printed, without the line feed.
func trimmed(s string) string { return strings.TrimSpace(s) }

// Two branches that each add a line, merged with no union merge to keep them, leave
// conflict markers in journal.jsonl. Reading goes on and says so; writing stops, since
// a line added under a marker would be lost or misplaced when the conflict is resolved.
func TestConflictMarkersWarnOnReadAndRefuseToWrite(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")
	// init makes .mtqg/.gitattributes say merge=union. Without it, git conflicts.
	if err := os.Remove(filepath.Join(r.dir, ".mtqg", ".gitattributes")); err != nil {
		t.Fatal(err)
	}
	shared := trimmed(r.mtqg("t", "add", "shared starting point"))
	r.commitAll("start")
	r.git("checkout", "-q", "-b", "feature")
	r.mtqg("t", "add", "added on the feature branch")
	r.commitAll("feature")
	r.git("checkout", "-q", "main")
	r.mtqg("t", "add", "added on main")
	r.commitAll("main")
	if out, err := r.gitResult("merge", "-q", "--no-edit", "feature"); err == nil {
		t.Fatalf("the merge did not conflict:\n%s", out)
	}
	conflicted := r.journal()
	if strings.Count(conflicted, "<<<<<<<") != 1 || strings.Count(conflicted, "=======") != 1 || strings.Count(conflicted, ">>>>>>>") != 1 {
		t.Fatalf("the journal does not have one conflict:\n%s", conflicted)
	}

	// Reading: a warning for each marker on standard error, and every record.
	res := r.run(nil, "", "t", "list")
	if res.code != 0 || strings.Count(res.stderr, "merge conflict marker") != 3 {
		t.Errorf("t list: %+v", res)
	}
	for _, want := range []string{"shared starting point", "added on the feature branch", "added on main", "3 open"} {
		if !strings.Contains(res.stdout, want) {
			t.Errorf("t list lacks %q:\n%s", want, res.stdout)
		}
	}
	// With --json, standard output is one object and the warnings are JSON lines.
	res = r.run(nil, "", "t", "list", "--json")
	var listed struct{ Open int }
	if res.code != 0 || json.Unmarshal([]byte(res.stdout), &listed) != nil || listed.Open != 3 || strings.Count(res.stderr, `{"warning":{"kind":"conflict_marker"`) != 3 {
		t.Errorf("t list --json: %+v", res)
	}

	// Writing: refused, with exit code 1, whatever the command, and not one byte changes.
	for _, args := range [][]string{
		{"t", "add", "x"}, {"m", "add", "x"}, {"q", "add", "x"}, {"t", "done", shared},
		{"edit", shared, "x"}, {"delete", shared}, {"undo"}, {"archive", "2020..2020"},
	} {
		res := r.run(nil, "", args...)
		if res.code != 1 || !strings.Contains(res.stderr, "unresolved merge conflict markers") {
			t.Errorf("mtqg %s: %+v", strings.Join(args, " "), res)
		}
		if got := r.journal(); got != conflicted {
			t.Fatalf("mtqg %s changed the journal:\n%s", strings.Join(args, " "), got)
		}
	}

	// Removing only the marker lines keeps both sides, and everything works again.
	var kept []string
	for _, l := range strings.Split(conflicted, "\n") {
		if !strings.HasPrefix(l, "<<<<<<<") && !strings.HasPrefix(l, "=======") && !strings.HasPrefix(l, ">>>>>>>") {
			kept = append(kept, l)
		}
	}
	if err := os.WriteFile(filepath.Join(r.dir, ".mtqg", "journal.jsonl"), []byte(strings.Join(kept, "\n")), 0o600); err != nil {
		t.Fatal(err)
	}
	res = r.run(nil, "", "t", "list")
	if res.code != 0 || res.stderr != "" || !strings.HasSuffix(res.stdout, "3 open (show done: --all)\n") {
		t.Errorf("t list after the resolution: %+v", res)
	}
	r.mtqg("t", "add", "written after the resolution")
	if got := r.mtqg("t", "list"); !strings.HasSuffix(got, "4 open (show done: --all)\n") {
		t.Errorf("t list after a write:\n%s", got)
	}
}

// A repository from the future is read by nobody who does not know its format, and
// written by nobody: what an old mtqg would write might not mean what it thinks.
func TestAFormatVersionFromTheFutureStops(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")
	r.mtqg("t", "add", "written by this version")
	before := r.journal()
	versionFile := filepath.Join(r.dir, ".mtqg", "version")
	if err := os.WriteFile(versionFile, []byte("1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	const message = "This repository uses format version 1, but this mtqg understands up to version 0. Update mtqg.\n"
	for _, args := range [][]string{
		{"t", "list"}, {"status"}, {"context"}, {"log"}, {"t", "add", "x"}, {"undo"}, {"archive", "2020..2020"},
	} {
		res := r.run(nil, "", args...)
		if res.code != 1 || res.stdout != "" || res.stderr != message {
			t.Errorf("mtqg %s: %+v", strings.Join(args, " "), res)
		}
	}
	if got := r.journal(); got != before {
		t.Errorf("the journal changed:\n%s", got)
	}
	res := r.run(nil, "", "t", "list", "--json")
	if res.code != 1 || res.stdout != "" || !strings.HasPrefix(res.stderr, `{"error":{"kind":"format_too_new"`) {
		t.Errorf("--json: %+v", res)
	}

	// What does not read the repository still works, and says what is wrong.
	res = r.run(nil, "", "version")
	if res.code != 0 || !strings.HasSuffix(res.stdout, "Repository format version: 1 (this mtqg supports up to 0)\n") {
		t.Errorf("version: %+v", res)
	}
	res = r.run(nil, before, "format")
	if res.code != 0 || !strings.Contains(res.stdout, "written by this version") {
		t.Errorf("format: %+v", res)
	}

	if err := os.WriteFile(versionFile, []byte("0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := r.mtqg("t", "list"); !strings.Contains(got, "written by this version") {
		t.Errorf("after the version is back:\n%s", got)
	}
}

// .mtqg/.local/ holds what only this machine needs, and is never committed, so a fresh
// clone does not have it, and neither does a repository where someone has cleaned up.
func TestTheLocalDirectoryComesBack(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")
	r.mtqg("t", "add", "first")
	local := filepath.Join(r.dir, ".mtqg", ".local")
	if _, err := os.Stat(filepath.Join(local, "lock")); err != nil {
		t.Fatalf("a write leaves no lock file: %v", err)
	}

	if err := os.RemoveAll(local); err != nil {
		t.Fatal(err)
	}
	r.mtqg("t", "add", "second")
	if _, err := os.Stat(filepath.Join(local, "lock")); err != nil {
		t.Errorf("the lock file did not come back: %v", err)
	}

	// A rewrite needs the directory for its temporary file.
	if err := os.RemoveAll(local); err != nil {
		t.Fatal(err)
	}
	if out := r.mtqg("undo"); !strings.HasPrefix(out, "Undone: ") {
		t.Errorf("undo: %q", out)
	}
	if got := r.mtqg("t", "list"); !strings.HasSuffix(got, "1 open (show done: --all)\n") {
		t.Errorf("list after the undo:\n%s", got)
	}
	if out := r.git("status", "--porcelain", "--untracked-files=all"); strings.Contains(out, ".local") {
		t.Errorf("git sees .mtqg/.local/:\n%s", out)
	}
}

// There is one .mtqg/ for each git repository, and a submodule is a repository.
func TestEachRepositoryHasItsOwnMtqgDirectory(t *testing.T) {
	outer := newRepo(t)
	outer.mtqg("init")
	outer.mtqg("t", "add", "a todo of the outer repository")
	outer.commitAll("outer")
	outerJournal := outer.journal()

	inner := newRepo(t)
	inner.git("commit", "-q", "--allow-empty", "-m", "inner")
	// Git does not clone a local path into a submodule unless it is told to.
	outer.git("-c", "protocol.file.allow=always", "submodule", "add", "-q", inner.dir, "sub")
	sub := filepath.Join(outer.dir, "sub")

	// The submodule has .git, so it is where the search stops: the .mtqg/ of the
	// repository around it is not used.
	res := outer.runIn(sub, nil, "", "t", "list")
	if res.code != 1 || res.stderr != "No .mtqg/ found. Run `mtqg init` (will be created at "+sub+")\n" {
		t.Errorf("t list in the submodule: %+v", res)
	}
	res = outer.runIn(sub, nil, "", "init")
	if res.code != 0 || !strings.HasPrefix(res.stdout, "Created .mtqg/ in "+sub+"\n") {
		t.Fatalf("init in the submodule: %+v", res)
	}
	// The submodule has no user.name of its own here, so it is named.
	subAuthor := []string{"MTQG_AUTHOR_NAME=sub-author"}
	if res := outer.runIn(sub, subAuthor, "", "t", "add", "a todo of the submodule"); res.code != 0 {
		t.Fatalf("add in the submodule: %+v", res)
	}

	if got := outer.journal(); got != outerJournal {
		t.Errorf("the outer journal changed:\n%s", got)
	}
	outerList := outer.mtqg("t", "list")
	if !strings.Contains(outerList, "a todo of the outer repository") || strings.Contains(outerList, "submodule") {
		t.Errorf("outer list:\n%s", outerList)
	}
	subList := outer.runIn(sub, nil, "", "t", "list")
	if !strings.Contains(subList.stdout, "a todo of the submodule") || strings.Contains(subList.stdout, "outer") {
		t.Errorf("submodule list: %+v", subList)
	}
}

func TestInitRefusesWhereThereIsAlreadyAMtqgDirectory(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")
	r.mtqg("t", "add", "kept")
	before := r.journal()
	deep := filepath.Join(r.dir, "src", "deep")
	if err := os.MkdirAll(deep, 0o750); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{r.dir, deep} {
		res := r.runIn(dir, nil, "", "init")
		if res.code != 1 || !strings.HasPrefix(res.stderr, ".mtqg/ already exists: ") {
			t.Errorf("init in %s: %+v", dir, res)
		}
	}
	if got := r.journal(); got != before {
		t.Errorf("the journal changed:\n%s", got)
	}

	// Outside a git repository, there is nowhere to put it.
	outside := t.TempDir()
	res := r.runIn(outside, nil, "", "init")
	if res.code != 1 || res.stderr == "" || res.stdout != "" {
		t.Errorf("init outside a repository: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(outside, ".mtqg")); err == nil {
		t.Error("init made .mtqg/ outside a repository")
	}
}
