package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// harness runs mtqg in a real git repository in a temporary directory, with a
// fixed clock and a terminal that the test decides on.
type harness struct {
	t    *testing.T
	root string
	vars map[string]string
	env  Env
	// The time that "now" is, and where times are shown.
	now time.Time
	// stdin is what standard input holds.
	stdin string
}

func isolateGit(t testing.TB) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

func git(t testing.TB, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// newHarness makes a git repository with a user.name of "tester". It does not
// run mtqg init.
func newHarness(t *testing.T) *harness {
	t.Helper()
	isolateGit(t)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "config", "user.name", "tester")
	git(t, root, "config", "user.email", "tester@example.com")
	return &harness{
		t:    t,
		root: root,
		vars: map[string]string{},
		now:  time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC),
	}
}

// initialized is a harness whose repository has had mtqg init.
func initialized(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t)
	if code, _, errOut := h.run("init"); code != 0 {
		t.Fatalf("init failed: %d %s", code, errOut)
	}
	return h
}

// run runs mtqg from the root of the repository and returns the exit code and
// what was written to standard output and standard error.
func (h *harness) run(args ...string) (code int, stdout, stderr string) {
	return h.runIn(h.root, args...)
}

func (h *harness) runIn(dir string, args ...string) (code int, stdout, stderr string) {
	h.t.Helper()
	var out, errOut bytes.Buffer
	env := h.env
	env.Stdin = strings.NewReader(h.stdin)
	env.Stdout, env.Stderr = &out, &errOut
	env.Getenv = func(k string) string { return h.vars[k] }
	env.Getwd = func() (string, error) { return dir, nil }
	env.Now = func() time.Time { return h.now }
	env.Location = time.UTC
	if env.RunEditor == nil {
		env.RunEditor = func([]string) error { return io.EOF }
	}
	code = Run(env, args)
	return code, out.String(), errOut.String()
}

// journalPath is the journal of the repository.
func (h *harness) journalPath() string { return filepath.Join(h.root, ".mtqg", "journal.jsonl") }

func (h *harness) readJournal() string {
	h.t.Helper()
	data, err := os.ReadFile(h.journalPath())
	if err != nil {
		h.t.Fatal(err)
	}
	return string(data)
}

// setJournal replaces the journal with the given lines, as a fixture.
func (h *harness) setJournal(lines ...string) {
	h.t.Helper()
	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n"
	}
	if err := os.WriteFile(h.journalPath(), []byte(content), 0o600); err != nil {
		h.t.Fatal(err)
	}
}

// record makes the line of a record that was created at ts, for setJournal.
func record(id, typ, text, author, ts string) string {
	ev := map[string]any{
		"id": id, "op": "create", "type": typ, "text": text,
		"v": 0, "ts": ts, "author": map[string]string{"kind": "human", "name": author},
	}
	if typ == "todo" || typ == "qa" || typ == "bug" {
		ev["status"] = "open"
	}
	return marshal(ev)
}

// change makes the line of a change of state, for setJournal.
func change(id, from, to, author, ts string) string {
	return marshal(map[string]any{
		"id": id, "op": "status", "from": from, "status": to,
		"v": 0, "ts": ts, "author": map[string]string{"kind": "human", "name": author},
	})
}

func marshal(v any) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		panic(err)
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// The IDs of the fixtures. The first eight digits of the first two are the same.
const (
	idA = "6cad4a268d0f4e2f8c1b7a3d5e9f0a11"
	idB = "6cad4a26ffff4e2f8c1b7a3d5e9f0a22"
	idC = "1e27a1c08a3b4c5d8e9f0a1b2c3d4e33"
	idM = "81e74ef5e8e24d949ed904759531985d"
	idQ = "2217beaddb1f4b6e9c0d1e2f3a4b5c66"
)
