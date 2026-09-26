package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseAt(t *testing.T) {
	tests := []struct {
		value    string
		wantPath string
		wantLine int
		wantErr  bool
	}{
		{value: "a.go:12", wantPath: "a.go", wantLine: 12},
		{value: "a.go", wantPath: "a.go"},
		{value: "a:b:12", wantPath: "a:b", wantLine: 12},
		{value: "a:b", wantPath: "a:b"},
		{value: "file:", wantPath: "file:"}, // an empty tail after the last colon is not a line
		{value: ":12", wantErr: true},       // a line number needs a path
		{value: "", wantErr: true},
		{value: "a.go:0", wantErr: true},
		{value: "a.go:99999999999999999999", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			at, err := parseAt(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseAt(%q) = %+v, want an error", tt.value, at)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseAt(%q): %v", tt.value, err)
			}
			if at.Path != tt.wantPath || at.Line != tt.wantLine {
				t.Errorf("parseAt(%q) = {Path: %q, Line: %d}, want {%q, %d}", tt.value, at.Path, at.Line, tt.wantPath, tt.wantLine)
			}
		})
	}
}

// atOfLastLine is the "at" of the last line of the journal, or nil if it has
// none.
func atOfLastLine(t *testing.T, h *harness) *struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Head string `json:"head"`
} {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(h.readJournal(), "\n"), "\n")
	var ev struct {
		At *struct {
			Path string `json:"path"`
			Line int    `json:"line"`
			Head string `json:"head"`
		} `json:"at"`
	}
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &ev); err != nil {
		t.Fatal(err)
	}
	return ev.At
}

func TestAtIsRecordedByEveryCreatingCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"memo", []string{"m", "add", "--at", "x.go:3", "hi"}},
		{"todo", []string{"t", "add", "--at", "x.go:3", "hi"}},
		{"rule", []string{"r", "add", "--at", "x.go:3", "hi"}},
		{"glossary", []string{"g", "add", "--at", "x.go:3", "word", "def"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := initialized(t)
			git(h.t, h.root, "commit", "-q", "--allow-empty", "-m", "first")
			code, out, errOut := h.run(tt.args...)
			wantExit(t, code, 0, out, errOut)
			at := atOfLastLine(t, h)
			if at == nil {
				t.Fatal("no at recorded")
			}
			if at.Path != "x.go" || at.Line != 3 {
				t.Errorf("at = %+v, want path x.go, line 3", at)
			}
			if at.Head == "" {
				t.Errorf("at = %+v, want a head", at)
			}
		})
	}

	t.Run("a new question and its answer", func(t *testing.T) {
		h := initialized(t)
		_, out, errOut := h.run("q", "add", "--at", "q.go:1", "a question")
		wantExit(t, 0, 0, out, errOut)
		id := strings.TrimSpace(out)
		code, out, errOut := h.run("q", "add", "--at", "a.go:2", id, "an answer")
		wantExit(t, code, 0, out, errOut)
		lines := strings.Split(strings.TrimSuffix(h.readJournal(), "\n"), "\n")
		if len(lines) != 2 {
			t.Fatalf("got %d lines, want 2", len(lines))
		}
		if !strings.Contains(lines[0], `"path":"q.go"`) {
			t.Errorf("question line: %s", lines[0])
		}
		if !strings.Contains(lines[1], `"path":"a.go"`) {
			t.Errorf("answer line: %s", lines[1])
		}
	})

	t.Run("a new bug and its reply", func(t *testing.T) {
		h := initialized(t)
		_, out, errOut := h.run("b", "add", "--at", "b.go:1", "a bug")
		wantExit(t, 0, 0, out, errOut)
		id := strings.TrimSpace(out)
		code, out, errOut := h.run("b", "add", "--at", "r.go:2", id, "a reply")
		wantExit(t, code, 0, out, errOut)
		lines := strings.Split(strings.TrimSuffix(h.readJournal(), "\n"), "\n")
		if !strings.Contains(lines[1], `"path":"r.go"`) {
			t.Errorf("reply line: %s", lines[1])
		}
	})

	t.Run("head matches git rev-parse --short HEAD", func(t *testing.T) {
		h := initialized(t)
		git(h.t, h.root, "commit", "-q", "--allow-empty", "-m", "first")
		want := strings.TrimSpace(git(h.t, h.root, "rev-parse", "--short", "HEAD"))
		code, out, errOut := h.run("m", "add", "--at", "x.go:1", "hi")
		wantExit(t, code, 0, out, errOut)
		if at := atOfLastLine(t, h); at.Head != want {
			t.Errorf("head = %q, want %q", at.Head, want)
		}
	})

	t.Run("no commit yet: at is written without a head", func(t *testing.T) {
		h := initialized(t)
		code, out, errOut := h.run("m", "add", "--at", "x.go:1", "hi")
		wantExit(t, code, 0, out, errOut)
		at := atOfLastLine(t, h)
		if at.Path != "x.go" || at.Head != "" {
			t.Errorf("at = %+v, want no head", at)
		}
	})

	// withHead swallowing a git error (git unavailable) is covered at the
	// journal layer (TestGitShortHead): at the CLI layer, an empty PATH also
	// breaks author resolution (GitUserName), which fails the write before
	// --at is even reached, so it cannot be exercised here in isolation.

	t.Run("no --at: no at field at all", func(t *testing.T) {
		h := initialized(t)
		code, out, errOut := h.run("m", "add", "hi")
		wantExit(t, code, 0, out, errOut)
		if at := atOfLastLine(t, h); at != nil {
			t.Errorf("at = %+v, want none", at)
		}
	})

	t.Run("a bad --at writes nothing", func(t *testing.T) {
		h := initialized(t)
		code, out, errOut := h.run("m", "add", "--at", ":12", "hi")
		wantExit(t, code, 2, out, errOut)
		if h.readJournal() != "" {
			t.Errorf("journal was written: %q", h.readJournal())
		}
	})
}
