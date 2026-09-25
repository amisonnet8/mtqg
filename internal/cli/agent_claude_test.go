package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitAgent(t *testing.T) {
	t.Run("unknown agent is a usage error", func(t *testing.T) {
		h := newHarness(t)
		code, out, errOut := h.run("init", "--agent", "codex")
		if code != exitUsage || out != "" || !strings.Contains(errOut, "codex") {
			t.Fatalf("code = %d, out = %q, stderr = %q", code, out, errOut)
		}
		if _, err := os.Stat(filepath.Join(h.root, ".mtqg")); err == nil {
			t.Error(".mtqg/ should not have been made for an unknown agent")
		}
	})

	t.Run("creates .mtqg/, settings.json and CLAUDE.md", func(t *testing.T) {
		h := newHarness(t)
		code, out, errOut := h.run("init", "--agent", "claude-code")
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(out, "Created: .claude/settings.json") || !strings.Contains(out, "Created: CLAUDE.md") {
			t.Fatalf("stdout = %q", out)
		}
		if _, err := os.Stat(filepath.Join(h.root, ".mtqg", "journal.jsonl")); err != nil {
			t.Errorf(".mtqg/ was not made: %v", err)
		}

		settings := readAgentFile(t, h, ".claude/settings.json")
		for _, want := range []string{
			`"mtqg hook claude-code session-start"`,
			`"mtqg hook claude-code stop"`,
			`"MTQG_AUTHOR_KIND": "ai"`,
			`"MTQG_AUTHOR_NAME": "claude-code"`,
		} {
			if !strings.Contains(settings, want) {
				t.Errorf("settings.json missing %s:\n%s", want, settings)
			}
		}

		memory := readAgentFile(t, h, "CLAUDE.md")
		if !strings.Contains(memory, "mtqg context") {
			t.Errorf("CLAUDE.md missing the mtqg line:\n%s", memory)
		}
	})

	t.Run("-n writes nothing", func(t *testing.T) {
		h := newHarness(t)
		code, out, errOut := h.run("init", "--agent", "claude-code", "-n")
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(out, "(dry run)") {
			t.Fatalf("stdout = %q, want a dry run marker", out)
		}
		if _, err := os.Stat(filepath.Join(h.root, ".claude", "settings.json")); err == nil {
			t.Error("settings.json should not have been written with -n")
		}
		if _, err := os.Stat(filepath.Join(h.root, ".mtqg")); err == nil {
			t.Error(".mtqg/ should not have been made with -n either")
		}
	})

	t.Run("an existing .mtqg/ is not an error with --agent", func(t *testing.T) {
		h := initialized(t)
		code, out, errOut := h.run("init", "--agent", "claude-code")
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(out, ".mtqg/ already exists") {
			t.Errorf("stdout = %q", out)
		}
		if !strings.Contains(out, "Created: .claude/settings.json") {
			t.Errorf("stdout = %q, want settings.json to still be wired up", out)
		}
	})

	t.Run("running it twice changes nothing the second time", func(t *testing.T) {
		h := newHarness(t)
		h.run("init", "--agent", "claude-code")
		before := readAgentFile(t, h, ".claude/settings.json")

		code, out, errOut := h.run("init", "--agent", "claude-code")
		wantExit(t, code, 0, out, errOut)
		if !strings.Contains(out, "Unchanged: .claude/settings.json") || !strings.Contains(out, "Unchanged: CLAUDE.md") {
			t.Fatalf("stdout = %q", out)
		}
		if after := readAgentFile(t, h, ".claude/settings.json"); after != before {
			t.Errorf("settings.json changed on a second run:\nbefore: %s\nafter:  %s", before, after)
		}
	})

	t.Run("keeps what was already there, and does not overwrite an existing env value", func(t *testing.T) {
		h := newHarness(t)
		if err := os.MkdirAll(filepath.Join(h.root, ".claude"), 0o750); err != nil {
			t.Fatal(err)
		}
		existing := `{"permissions":{"deny":["Bash(git push*)"]},"env":{"MTQG_AUTHOR_NAME":"someone-else"}}`
		if err := os.WriteFile(filepath.Join(h.root, ".claude", "settings.json"), []byte(existing), 0o600); err != nil {
			t.Fatal(err)
		}
		code, out, errOut := h.run("init", "--agent", "claude-code")
		wantExit(t, code, 0, out, errOut)

		settings := readAgentFile(t, h, ".claude/settings.json")
		for _, want := range []string{`"Bash(git push*)"`, `"MTQG_AUTHOR_NAME": "someone-else"`, `"mtqg hook claude-code stop"`} {
			if !strings.Contains(settings, want) {
				t.Errorf("settings.json missing %s:\n%s", want, settings)
			}
		}
	})

	t.Run("invalid JSON in settings.json is refused, not overwritten", func(t *testing.T) {
		h := newHarness(t)
		if err := os.MkdirAll(filepath.Join(h.root, ".claude"), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(h.root, ".claude", "settings.json"), []byte("not json"), 0o600); err != nil {
			t.Fatal(err)
		}
		code, out, errOut := h.run("init", "--agent", "claude-code")
		if code != exitError || out != "" || !strings.Contains(errOut, "settings.json") {
			t.Fatalf("code = %d, out = %q, stderr = %q", code, out, errOut)
		}
		if got := readAgentFile(t, h, ".claude/settings.json"); got != "not json" {
			t.Errorf("settings.json was changed: %q", got)
		}
	})

	t.Run("--json reports each file", func(t *testing.T) {
		h := newHarness(t)
		code, out, errOut := h.run("init", "--agent", "claude-code", "--json")
		wantExit(t, code, 0, out, errOut)
		for _, want := range []string{`"agent": "claude-code"`, `"path": ".claude/settings.json"`, `"result": "created"`, `"path": "CLAUDE.md"`} {
			if !strings.Contains(out, want) {
				t.Errorf("--json output missing %s:\n%s", want, out)
			}
		}
	})
}

func readAgentFile(t *testing.T, h *harness, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(h.root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
