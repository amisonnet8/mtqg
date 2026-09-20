package journal

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// isolateGit keeps the developer's git configuration out of the tests: HOME and
// GIT_CONFIG_GLOBAL point into a temporary directory.
func isolateGit(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

// git runs the real git command in dir and fails the test if it fails.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// newRepo creates a git repository in a temporary directory and returns its
// physical path (so that comparisons hold where the temporary directory is
// reached through a symbolic link, as on macOS).
func newRepo(t *testing.T) string {
	t.Helper()
	isolateGit(t)
	dir := realPath(t, t.TempDir())
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "config", "user.name", "tester")
	git(t, dir, "config", "user.email", "tester@example.com")
	return dir
}

func realPath(t *testing.T, path string) string {
	t.Helper()
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return real
}

// newMtqg creates .mtqg/ inside root with the given format version and journal
// content and returns the .mtqg/ directory. A nil journal creates no
// journal.jsonl.
func newMtqg(t *testing.T, root string, version string, journal *string) string {
	t.Helper()
	dir := filepath.Join(root, mtqgDirName)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, versionName), []byte(version), 0o600); err != nil {
		t.Fatal(err)
	}
	if journal != nil {
		if err := os.WriteFile(filepath.Join(dir, journalName), []byte(*journal), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func str(s string) *string { return &s }

func mkdir(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(path, 0o750); err != nil {
		t.Fatal(err)
	}
	return path
}
