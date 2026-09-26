package cli

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestSplitCommand(t *testing.T) {
	tests := []struct {
		in      string
		want    []string
		wantErr bool
	}{
		{"vim", []string{"vim"}, false},
		{"code --wait", []string{"code", "--wait"}, false},
		{"  emacs   -nw  ", []string{"emacs", "-nw"}, false},
		{`"C:\Program Files\Editor\ed.exe" -w`, []string{`C:\Program Files\Editor\ed.exe`, "-w"}, false},
		{`'/opt/my editor/ed' --flag`, []string{"/opt/my editor/ed", "--flag"}, false},
		{`ed --name="two words"`, []string{"ed", "--name=two words"}, false},
		{`ed ""`, []string{"ed", ""}, false},
		{`"unclosed`, nil, true},
		{"   ", nil, true},
		{"", nil, true},
	}
	for _, tt := range tests {
		got, err := splitCommand(tt.in)
		if (err != nil) != tt.wantErr || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("splitCommand(%q) = %q, %v; want %q (error %v)", tt.in, got, err, tt.want, tt.wantErr)
		}
	}
}

func newCtx(h *harness, editor func([]string) error) *ctx {
	env := Env{
		Stdin:     strings.NewReader(h.stdin),
		Getenv:    func(k string) string { return h.vars[k] },
		RunEditor: editor,
	}
	return &ctx{env: env, inv: &invocation{}}
}

func TestInputText(t *testing.T) {
	failsWith := func(t *testing.T, err error, want string) {
		t.Helper()
		var f *failure
		if !errors.As(err, &f) || f.msg != want {
			t.Errorf("err = %v, want the failure %q", err, want)
		}
	}

	t.Run("the words are joined with spaces", func(t *testing.T) {
		h := newHarness(t)
		got, err := newCtx(h, nil).inputText([]string{"Skip", "block", "comments"})
		if err != nil || got != "Skip block comments" {
			t.Errorf("got %q, %v", got, err)
		}
	})

	t.Run("- reads standard input to its end and drops the line breaks at the end", func(t *testing.T) {
		h := newHarness(t)
		h.stdin = "first line\nsecond line\r\n\n"
		got, err := newCtx(h, nil).inputText([]string{"-"})
		if err != nil || got != "first line\nsecond line" {
			t.Errorf("got %q, %v", got, err)
		}
	})

	t.Run("- with other words is text", func(t *testing.T) {
		h := newHarness(t)
		h.stdin = "not read"
		got, err := newCtx(h, nil).inputText([]string{"-", "5", "degrees"})
		if err != nil || got != "- 5 degrees" {
			t.Errorf("got %q, %v", got, err)
		}
	})

	t.Run("empty and blank texts are refused", func(t *testing.T) {
		for _, stdin := range []string{"", "\n", "  \t\n"} {
			h := newHarness(t)
			h.stdin = stdin
			_, err := newCtx(h, nil).inputText([]string{"-"})
			failsWith(t, err, msgEmptyText())
		}
		h := newHarness(t)
		_, err := newCtx(h, nil).inputText([]string{"   "})
		failsWith(t, err, msgEmptyText())
	})

	t.Run("no words open the editor on an empty file", func(t *testing.T) {
		h := newHarness(t)
		h.vars["EDITOR"] = `myeditor --wait "a b"`
		var argv []string
		var before string
		editor := func(a []string) error {
			argv = a
			data, err := os.ReadFile(a[len(a)-1])
			if err != nil {
				return err
			}
			before = string(data)
			return os.WriteFile(a[len(a)-1], []byte("from the editor\nline two\n\n"), 0o600)
		}
		got, err := newCtx(h, editor).inputText(nil)
		if err != nil || got != "from the editor\nline two" {
			t.Errorf("got %q, %v", got, err)
		}
		if before != "" {
			t.Errorf("the editor was given a file with %q in it", before)
		}
		if len(argv) != 4 || argv[0] != "myeditor" || argv[1] != "--wait" || argv[2] != "a b" {
			t.Errorf("argv = %q", argv)
		}
		if _, err := os.Stat(argv[len(argv)-1]); err == nil {
			t.Error("the temporary file was left behind")
		}
	})

	t.Run("an empty file from the editor aborts", func(t *testing.T) {
		h := newHarness(t)
		h.vars["EDITOR"] = "ed"
		_, err := newCtx(h, func([]string) error { return nil }).inputText(nil)
		failsWith(t, err, msgEmptyText())
	})

	t.Run("no $EDITOR falls back to nano", func(t *testing.T) {
		h := newHarness(t)
		var argv []string
		editor := func(a []string) error {
			argv = a
			return os.WriteFile(a[len(a)-1], []byte("from nano\n"), 0o600)
		}
		got, err := newCtx(h, editor).inputText(nil)
		if err != nil || got != "from nano" {
			t.Errorf("got %q, %v", got, err)
		}
		if len(argv) != 2 || argv[0] != "nano" {
			t.Errorf("argv = %q, want nano as the editor", argv)
		}
	})

	t.Run("a blank $EDITOR also falls back to nano", func(t *testing.T) {
		h := newHarness(t)
		h.vars["EDITOR"] = "   "
		var argv []string
		editor := func(a []string) error {
			argv = a
			return os.WriteFile(a[len(a)-1], []byte("from nano\n"), 0o600)
		}
		if _, err := newCtx(h, editor).inputText(nil); err != nil {
			t.Fatal(err)
		}
		if len(argv) != 2 || argv[0] != "nano" {
			t.Errorf("argv = %q, want nano as the editor", argv)
		}
	})

	t.Run("an editor that fails", func(t *testing.T) {
		h := newHarness(t)
		h.vars["EDITOR"] = "ed"
		boom := errors.New("exit status 1")
		_, err := newCtx(h, func([]string) error { return boom }).inputText(nil)
		failsWith(t, err, msgEditorFailed(boom))
	})

	t.Run("an $EDITOR that cannot be read", func(t *testing.T) {
		h := newHarness(t)
		h.vars["EDITOR"] = `"unclosed`
		_, err := newCtx(h, nil).inputText(nil)
		var f *failure
		if !errors.As(err, &f) || !strings.Contains(f.msg, "Cannot read $EDITOR") {
			t.Errorf("err = %v", err)
		}
	})
}

func TestAuthor(t *testing.T) {
	root := func(t *testing.T) (string, *harness) {
		h := newHarness(t)
		return h.root, h
	}

	t.Run("a human, named by git", func(t *testing.T) {
		dir, h := root(t)
		got, err := newCtx(h, nil).author(dir)
		if err != nil || got.Kind != "human" || got.Name != "tester" {
			t.Errorf("got %+v, %v", got, err)
		}
	})

	t.Run("a name from the environment beats git", func(t *testing.T) {
		dir, h := root(t)
		h.vars["MTQG_AUTHOR_NAME"] = "  山田 太郎  "
		got, err := newCtx(h, nil).author(dir)
		if err != nil || got.Kind != "human" || got.Name != "山田 太郎" {
			t.Errorf("got %+v, %v", got, err)
		}
	})

	t.Run("an AI with a name", func(t *testing.T) {
		dir, h := root(t)
		h.vars["MTQG_AUTHOR_KIND"] = "AI"
		h.vars["MTQG_AUTHOR_NAME"] = "claude-code"
		got, err := newCtx(h, nil).author(dir)
		if err != nil || got.Kind != "ai" || got.Name != "claude-code" {
			t.Errorf("got %+v, %v", got, err)
		}
	})

	t.Run("an AI without a name is never recorded under the name from git", func(t *testing.T) {
		dir, h := root(t)
		h.vars["MTQG_AUTHOR_KIND"] = "ai"
		_, err := newCtx(h, nil).author(dir)
		var f *failure
		if !errors.As(err, &f) || f.msg != msgAIneedsName() {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("a kind that is neither", func(t *testing.T) {
		dir, h := root(t)
		h.vars["MTQG_AUTHOR_KIND"] = "robot"
		_, err := newCtx(h, nil).author(dir)
		var f *failure
		if !errors.As(err, &f) || f.msg != msgBadAuthorKind("robot") {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("no name anywhere", func(t *testing.T) {
		dir, h := root(t)
		git(t, dir, "config", "--unset", "user.name")
		_, err := newCtx(h, nil).author(dir)
		var f *failure
		if !errors.As(err, &f) || f.msg != msgNoUserName() {
			t.Errorf("err = %v", err)
		}
	})
}
