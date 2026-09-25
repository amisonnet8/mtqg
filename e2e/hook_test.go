//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHookClaudeCode runs the real binary through a whole SessionStart/Stop
// cycle, with git and .mtqg/ real: session-start prints the same context an
// agent would read, a change without a record makes stop block once, and a
// second stop for the same session stays quiet.
func TestHookClaudeCode(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")

	start := r.run(nil, `{"session_id":"e2e-1"}`, "hook", "claude-code", "session-start")
	if start.code != 0 || start.stderr != "" {
		t.Fatalf("session-start: code=%d stderr=%q", start.code, start.stderr)
	}
	wantContext := r.mtqg("context")
	if start.stdout != wantContext {
		t.Fatalf("session-start printed a different document than context:\ngot:  %q\nwant: %q", start.stdout, wantContext)
	}
	if _, err := os.Stat(filepath.Join(r.dir, ".mtqg", ".local", "sessions")); err != nil {
		t.Fatalf(".local/sessions/ was not made: %v", err)
	}

	// A change with nothing recorded: stop should block, once.
	if err := os.WriteFile(filepath.Join(r.dir, "code.go"), []byte("package p\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	first := r.run(nil, `{"session_id":"e2e-1"}`, "hook", "claude-code", "stop")
	if first.code != 0 || !strings.Contains(first.stdout, `"decision":"block"`) {
		t.Fatalf("first stop: code=%d stdout=%q", first.code, first.stdout)
	}
	second := r.run(nil, `{"session_id":"e2e-1"}`, "hook", "claude-code", "stop")
	if second.code != 0 || second.stdout != "" {
		t.Fatalf("second stop should stay quiet: code=%d stdout=%q", second.code, second.stdout)
	}
}

// TestHookClaudeCodeSilentWhenRecorded covers the case that matters most: a
// session that did record something is never asked again, even with an
// unrelated working-tree change.
func TestHookClaudeCodeSilentWhenRecorded(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")

	r.run(nil, `{"session_id":"e2e-2"}`, "hook", "claude-code", "session-start")
	r.mtqg("m", "add", "noted something during the session")
	if err := os.WriteFile(filepath.Join(r.dir, "code.go"), []byte("package p\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stop := r.run(nil, `{"session_id":"e2e-2"}`, "hook", "claude-code", "stop")
	if stop.code != 0 || stop.stdout != "" {
		t.Fatalf("stop after a record was written should stay quiet: code=%d stdout=%q", stop.code, stop.stdout)
	}
}

// TestHookClaudeCodeNoMtqg makes sure the hook never breaks an agent's turn
// just because there is nothing here for it to read.
func TestHookClaudeCodeNoMtqg(t *testing.T) {
	r := newRepo(t) // no mtqg init

	start := r.run(nil, `{"session_id":"e2e-3"}`, "hook", "claude-code", "session-start")
	if start.code != 0 || start.stdout != "" || start.stderr != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q", start.code, start.stdout, start.stderr)
	}
}

// TestHookClaudeCodeUnknown checks that only the agent/event pair, not
// anything read at run time, is a usage error.
func TestHookClaudeCodeUnknown(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")

	res := r.run(nil, "", "hook", "codex", "stop")
	if res.code != 2 {
		t.Fatalf("unknown agent: code=%d, want 2", res.code)
	}
	res = r.run(nil, "", "hook", "claude-code", "prompt-submit")
	if res.code != 2 {
		t.Fatalf("unknown event: code=%d, want 2", res.code)
	}
}

// TestInitAgentClaudeCode wires up .claude/settings.json and CLAUDE.md with
// the real binary, and checks the wiring is actually idempotent on disk.
func TestInitAgentClaudeCode(t *testing.T) {
	r := newRepo(t)

	out := r.mtqg("init", "--agent", "claude-code")
	if !strings.Contains(out, "Created: .claude/settings.json") || !strings.Contains(out, "Created: CLAUDE.md") {
		t.Fatalf("init --agent output: %q", out)
	}

	settingsPath := filepath.Join(r.dir, ".claude", "settings.json")
	settings := readFile(t, settingsPath)
	for _, want := range []string{"mtqg hook claude-code session-start", "mtqg hook claude-code stop", "MTQG_AUTHOR_KIND", "MTQG_AUTHOR_NAME"} {
		if !strings.Contains(settings, want) {
			t.Errorf("settings.json missing %q:\n%s", want, settings)
		}
	}

	before := settings
	out = r.mtqg("init", "--agent", "claude-code")
	if !strings.Contains(out, "Unchanged: .claude/settings.json") {
		t.Fatalf("second run should be unchanged: %q", out)
	}
	if after := readFile(t, settingsPath); after != before {
		t.Errorf("settings.json changed on an unchanged run:\nbefore: %s\nafter:  %s", before, after)
	}

	// The wired-up hooks actually work end to end: session-start followed by a
	// real mtqg command run the way settings.json's env says to run it.
	r.env = append(r.env, "MTQG_AUTHOR_KIND=ai", "MTQG_AUTHOR_NAME=claude-code")
	r.mtqg("m", "add", "author from settings.json's env")
}
