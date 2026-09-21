package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// ttyOfLastLine is the tty of the last line of the journal, or "" if it has none.
func ttyOfLastLine(t *testing.T, h *harness) string {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(h.readJournal(), "\n"), "\n")
	var ev struct {
		TTY string `json:"tty"`
	}
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &ev); err != nil {
		t.Fatal(err)
	}
	return ev.TTY
}

func TestTTYFromMTQGTTY(t *testing.T) {
	h := initialized(t)
	h.vars["MTQG_TTY"] = "terminal-a"
	h.run("m", "add", "one")
	// sha256("env:terminal-a"), first 4 bytes (computed apart from the code).
	if got := ttyOfLastLine(t, h); got != "62eb74c8" {
		t.Errorf("tty = %q, want 62eb74c8", got)
	}

	h.vars["MTQG_TTY"] = "terminal-b"
	h.run("m", "add", "two")
	if got := ttyOfLastLine(t, h); got != "b01fca5a" {
		t.Errorf("another value gave tty %q, want b01fca5a", got)
	}
}

func TestTTYFromTheTerminal(t *testing.T) {
	h := initialized(t)
	h.env.TTY = "dev:34817"
	h.run("m", "add", "at a terminal")
	if got := ttyOfLastLine(t, h); got != "3e4ca368" {
		t.Errorf("tty = %q, want 3e4ca368", got)
	}
	// The device number is not written, only the hash.
	if strings.Contains(h.readJournal(), "34817") {
		t.Errorf("the device number is in the journal:\n%s", h.readJournal())
	}
}

func TestMTQGTTYComesBeforeTheTerminal(t *testing.T) {
	h := initialized(t)
	h.env.TTY = "dev:34817"
	h.vars["MTQG_TTY"] = "terminal-a"
	h.run("m", "add", "both")
	if got := ttyOfLastLine(t, h); got != "62eb74c8" {
		t.Errorf("tty = %q: MTQG_TTY should win over the terminal", got)
	}
}

func TestNoTTYWhenThereIsNoTerminal(t *testing.T) {
	h := initialized(t)
	h.run("m", "add", "through a pipe")
	if got := ttyOfLastLine(t, h); got != "" {
		t.Errorf("tty = %q, want none", got)
	}
	if strings.Contains(h.readJournal(), `"tty"`) {
		t.Errorf("a line without a terminal must not have a tty field:\n%s", h.readJournal())
	}

	// An empty or blank MTQG_TTY is as if it were not set.
	h.vars["MTQG_TTY"] = "  "
	h.run("m", "add", "blank")
	if got := ttyOfLastLine(t, h); got != "" {
		t.Errorf("a blank MTQG_TTY gave tty %q, want none", got)
	}
}
