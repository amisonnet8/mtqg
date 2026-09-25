package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHookUsage(t *testing.T) {
	h := initialized(t)

	t.Run("no arguments", func(t *testing.T) {
		if code, _, _ := h.run("hook"); code != exitUsage {
			t.Fatalf("code = %d, want %d", code, exitUsage)
		}
	})
	t.Run("one argument", func(t *testing.T) {
		if code, _, _ := h.run("hook", "claude-code"); code != exitUsage {
			t.Fatalf("code = %d, want %d", code, exitUsage)
		}
	})
	t.Run("too many arguments", func(t *testing.T) {
		if code, _, _ := h.run("hook", "claude-code", "stop", "extra"); code != exitUsage {
			t.Fatalf("code = %d, want %d", code, exitUsage)
		}
	})
	t.Run("unknown agent", func(t *testing.T) {
		code, _, errOut := h.run("hook", "codex", "stop")
		if code != exitUsage || !strings.Contains(errOut, "codex") {
			t.Fatalf("code = %d, stderr = %q", code, errOut)
		}
	})
	t.Run("unknown event", func(t *testing.T) {
		code, _, errOut := h.run("hook", "claude-code", "prompt")
		if code != exitUsage || !strings.Contains(errOut, "prompt") {
			t.Fatalf("code = %d, stderr = %q", code, errOut)
		}
	})
}

func TestHookClaudeCodeSessionStart(t *testing.T) {
	t.Run("prints what context prints", func(t *testing.T) {
		h := initialized(t)
		h.stdin = `{"session_id":"s1"}`
		code, out, errOut := h.run("hook", "claude-code", "session-start")
		if code != exitOK || errOut != "" {
			t.Fatalf("code = %d, stderr = %q", code, errOut)
		}
		_, wantOut, _ := h.run("context")
		if out != wantOut {
			t.Fatalf("session-start output differs from context:\ngot:  %q\nwant: %q", out, wantOut)
		}
	})

	t.Run("without .mtqg/, nothing is printed and it still exits 0", func(t *testing.T) {
		h := newHarness(t) // no init
		h.stdin = `{"session_id":"s1"}`
		code, out, errOut := h.run("hook", "claude-code", "session-start")
		if code != exitOK || out != "" || errOut != "" {
			t.Fatalf("code = %d, out = %q, stderr = %q", code, out, errOut)
		}
	})

	t.Run("bad JSON input is a warning, not a failure", func(t *testing.T) {
		h := initialized(t)
		h.stdin = `not json`
		code, out, errOut := h.run("hook", "claude-code", "session-start")
		if code != exitOK || out != "" || errOut == "" {
			t.Fatalf("code = %d, out = %q, stderr = %q", code, out, errOut)
		}
	})

	t.Run("a session file is written under .local/sessions/", func(t *testing.T) {
		h := initialized(t)
		h.stdin = `{"session_id":"s1"}`
		if code, _, _ := h.run("hook", "claude-code", "session-start"); code != exitOK {
			t.Fatal("session-start failed")
		}
		entries, err := os.ReadDir(filepath.Join(h.root, ".mtqg", ".local", "sessions"))
		if err != nil || len(entries) != 1 {
			t.Fatalf("entries = %v, err = %v", entries, err)
		}
	})

	t.Run("a second session-start for the same session does not move the baseline", func(t *testing.T) {
		// A change with no record between the two session-starts (a compact, say)
		// must still be seen by stop: session-start only remembers the state of a
		// session it has not seen before.
		h := initialized(t)
		h.stdin = `{"session_id":"s1"}`
		h.run("hook", "claude-code", "session-start")

		if err := os.WriteFile(filepath.Join(h.root, "changed.txt"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}

		h.stdin = `{"session_id":"s1","source":"compact"}`
		if code, _, _ := h.run("hook", "claude-code", "session-start"); code != exitOK {
			t.Fatal("second session-start failed")
		}

		h.stdin = `{"session_id":"s1"}`
		code, out, _ := h.run("hook", "claude-code", "stop")
		if code != exitOK || !strings.Contains(out, `"decision":"block"`) {
			t.Fatalf("code = %d, out = %q, want a block: the change before the second session-start was never recorded", code, out)
		}
	})
}

func TestHookClaudeCodeStop(t *testing.T) {
	change := func(h *harness) {
		if err := os.WriteFile(filepath.Join(h.root, "changed.txt"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("blocks once when there was work and nothing was recorded", func(t *testing.T) {
		h := initialized(t)
		h.stdin = `{"session_id":"s1"}`
		h.run("hook", "claude-code", "session-start")
		change(h)

		h.stdin = `{"session_id":"s1"}`
		code, out, errOut := h.run("hook", "claude-code", "stop")
		if code != exitOK || errOut != "" {
			t.Fatalf("code = %d, stderr = %q", code, errOut)
		}
		if !strings.Contains(out, `"decision":"block"`) {
			t.Fatalf("out = %q, want a block decision", out)
		}

		// A second Stop for the same session does not ask again.
		h.stdin = `{"session_id":"s1"}`
		code, out, _ = h.run("hook", "claude-code", "stop")
		if code != exitOK || out != "" {
			t.Fatalf("second stop: code = %d, out = %q, want nothing", code, out)
		}
	})

	t.Run("says nothing when nothing changed", func(t *testing.T) {
		h := initialized(t)
		h.stdin = `{"session_id":"s1"}`
		h.run("hook", "claude-code", "session-start")

		h.stdin = `{"session_id":"s1"}`
		code, out, _ := h.run("hook", "claude-code", "stop")
		if code != exitOK || out != "" {
			t.Fatalf("code = %d, out = %q, want nothing", code, out)
		}
	})

	t.Run("says nothing when a record was written meanwhile", func(t *testing.T) {
		h := initialized(t)
		h.stdin = `{"session_id":"s1"}`
		h.run("hook", "claude-code", "session-start")
		if code, _, _ := h.run("m", "add", "noted something"); code != exitOK {
			t.Fatal("m add failed")
		}
		change(h)

		h.stdin = `{"session_id":"s1"}`
		code, out, _ := h.run("hook", "claude-code", "stop")
		if code != exitOK || out != "" {
			t.Fatalf("code = %d, out = %q, want nothing", code, out)
		}
	})

	t.Run("stop_hook_active stays quiet even with work and nothing recorded", func(t *testing.T) {
		h := initialized(t)
		h.stdin = `{"session_id":"s1"}`
		h.run("hook", "claude-code", "session-start")
		change(h)

		h.stdin = `{"session_id":"s1","stop_hook_active":true}`
		code, out, _ := h.run("hook", "claude-code", "stop")
		if code != exitOK || out != "" {
			t.Fatalf("code = %d, out = %q, want nothing", code, out)
		}
	})

	t.Run("a session that never called session-start says nothing", func(t *testing.T) {
		h := initialized(t)
		change(h)
		h.stdin = `{"session_id":"never-started"}`
		code, out, _ := h.run("hook", "claude-code", "stop")
		if code != exitOK || out != "" {
			t.Fatalf("code = %d, out = %q, want nothing", code, out)
		}
	})
}
