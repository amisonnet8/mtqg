package journal

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitUserName(t *testing.T) {
	t.Run("the name of the repository", func(t *testing.T) {
		root := newRepo(t) // sets user.name to tester
		got, err := GitUserName(root)
		if err != nil || got != "tester" {
			t.Fatalf("got %q, %v; want tester", got, err)
		}
	})

	t.Run("a name with Japanese and spaces", func(t *testing.T) {
		root := newRepo(t)
		git(t, root, "config", "user.name", "山田 太郎")
		if got, err := GitUserName(root); err != nil || got != "山田 太郎" {
			t.Fatalf("got %q, %v", got, err)
		}
	})

	t.Run("no name", func(t *testing.T) {
		root := newRepo(t)
		git(t, root, "config", "--unset", "user.name")
		if _, err := GitUserName(root); !errors.Is(err, ErrNoUserName) {
			t.Fatalf("err = %v, want ErrNoUserName", err)
		}
	})

	t.Run("an empty name is no name", func(t *testing.T) {
		root := newRepo(t)
		git(t, root, "config", "user.name", "")
		if _, err := GitUserName(root); !errors.Is(err, ErrNoUserName) {
			t.Fatalf("err = %v, want ErrNoUserName", err)
		}
	})

	t.Run("git cannot be run", func(t *testing.T) {
		root := newRepo(t)
		t.Setenv("PATH", t.TempDir()) // a directory with no git in it
		_, err := GitUserName(root)
		if !errors.Is(err, ErrGitUnavailable) {
			t.Fatalf("err = %v, want ErrGitUnavailable", err)
		}
	})
}

func TestGitBranch(t *testing.T) {
	t.Run("the branch that is checked out, before the first commit too", func(t *testing.T) {
		root := newRepo(t) // init -b main, and no commit
		if got, err := GitBranch(root); err != nil || got != "main" {
			t.Fatalf("got %q, %v; want main", got, err)
		}
		git(t, root, "commit", "-q", "--allow-empty", "-m", "first")
		git(t, root, "checkout", "-q", "-b", "feature/日本語")
		if got, err := GitBranch(root); err != nil || got != "feature/日本語" {
			t.Fatalf("got %q, %v; want feature/日本語", got, err)
		}
	})

	t.Run("a detached HEAD has no branch", func(t *testing.T) {
		root := newRepo(t)
		git(t, root, "commit", "-q", "--allow-empty", "-m", "first")
		git(t, root, "checkout", "-q", "--detach")
		if got, err := GitBranch(root); err != nil || got != "" {
			t.Fatalf("got %q, %v; want no branch", got, err)
		}
	})

	t.Run("git cannot be run", func(t *testing.T) {
		root := newRepo(t)
		t.Setenv("PATH", t.TempDir())
		if _, err := GitBranch(root); !errors.Is(err, ErrGitUnavailable) {
			t.Fatalf("err = %v, want ErrGitUnavailable", err)
		}
	})
}

// journalWith opens a repository with a .mtqg/ and appends n memos to it.
func journalWith(t *testing.T, n int) *Journal {
	t.Helper()
	root := newRepo(t)
	newMtqg(t, root, "0\n", nil)
	j, err := Open(root, Options{Author: yamada, LockTimeout: 30_000_000_000})
	if err != nil {
		t.Fatal(err)
	}
	appendMemos(t, j, n, "first")
	return j
}

func appendMemos(t *testing.T, j *Journal, n int, text string) []Event {
	t.Helper()
	var events []Event
	for range n {
		ev, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: text})
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, ev)
	}
	return events
}

func commitAll(t *testing.T, j *Journal) {
	t.Helper()
	git(t, j.loc.Root, "add", ".mtqg")
	git(t, j.loc.Root, "commit", "-q", "-m", "records")
}

func uncommitted(t *testing.T, j *Journal) []Event {
	t.Helper()
	events, err := j.UncommittedEvents()
	if err != nil {
		t.Fatal(err)
	}
	return events
}

func TestUncommittedEvents(t *testing.T) {
	t.Run("no commit yet: everything is uncommitted", func(t *testing.T) {
		j := journalWith(t, 3)
		if got := len(uncommitted(t, j)); got != 3 {
			t.Errorf("got %d, want 3", got)
		}
	})

	t.Run("nothing after a commit", func(t *testing.T) {
		j := journalWith(t, 3)
		commitAll(t, j)
		if got := len(uncommitted(t, j)); got != 0 {
			t.Errorf("got %d, want 0", got)
		}
	})

	t.Run("lines added after a commit", func(t *testing.T) {
		j := journalWith(t, 3)
		commitAll(t, j)
		added := appendMemos(t, j, 2, "later")
		got := uncommitted(t, j)
		if len(got) != 2 || got[0].ID != added[0].ID || got[1].ID != added[1].ID {
			t.Errorf("got %+v, want the two added events", got)
		}
	})

	t.Run("staged but not committed counts as uncommitted", func(t *testing.T) {
		j := journalWith(t, 1)
		commitAll(t, j)
		appendMemos(t, j, 1, "staged")
		git(t, j.loc.Root, "add", ".mtqg")
		if got := len(uncommitted(t, j)); got != 1 {
			t.Errorf("got %d, want 1", got)
		}
	})

	t.Run("a line that only the commit has is not an event of the journal", func(t *testing.T) {
		j := journalWith(t, 3)
		commitAll(t, j)
		// Drop one line, as undo would.
		if err := j.Rewrite(func(lines []Line) ([]Line, error) { return lines[:2], nil }); err != nil {
			t.Fatal(err)
		}
		if got := len(uncommitted(t, j)); got != 0 {
			t.Errorf("got %d, want 0: a removed line is not an uncommitted event", got)
		}
	})

	t.Run("the commit has no journal.jsonl", func(t *testing.T) {
		j := journalWith(t, 2)
		// A commit that has other files only.
		if err := os.WriteFile(j.loc.Root+"/other.txt", []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		git(t, j.loc.Root, "add", "other.txt")
		git(t, j.loc.Root, "commit", "-q", "-m", "other")
		if got := len(uncommitted(t, j)); got != 2 {
			t.Errorf("got %d, want 2", got)
		}
	})

	t.Run("identical lines count once", func(t *testing.T) {
		j := journalWith(t, 1)
		commitAll(t, j)
		line := lineOf(t, memo(idA, "twice"))
		f, err := os.OpenFile(j.journalPath(), os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.WriteString(line + "\n" + line + "\n"); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		if got := len(uncommitted(t, j)); got != 1 {
			t.Errorf("got %d, want 1", got)
		}
	})

	t.Run("git cannot be run", func(t *testing.T) {
		j := journalWith(t, 1)
		t.Setenv("PATH", t.TempDir())
		if _, err := j.UncommittedEvents(); !errors.Is(err, ErrGitUnavailable) {
			t.Fatalf("err = %v, want ErrGitUnavailable", err)
		}
	})
}

func TestGitHead(t *testing.T) {
	t.Run("no commit yet", func(t *testing.T) {
		root := newRepo(t)
		if got, err := GitHead(root); err != nil || got != "" {
			t.Fatalf("got %q, %v; want \"\"", got, err)
		}
	})

	t.Run("the commit HEAD points to", func(t *testing.T) {
		root := newRepo(t)
		git(t, root, "commit", "-q", "--allow-empty", "-m", "first")
		want := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
		if got, err := GitHead(root); err != nil || got != want {
			t.Fatalf("got %q, %v; want %q", got, err, want)
		}
		git(t, root, "commit", "-q", "--allow-empty", "-m", "second")
		want = strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
		if got, err := GitHead(root); err != nil || got != want {
			t.Fatalf("after a second commit: got %q, %v; want %q", got, err, want)
		}
	})

	t.Run("git cannot be run", func(t *testing.T) {
		root := newRepo(t)
		t.Setenv("PATH", t.TempDir())
		if _, err := GitHead(root); !errors.Is(err, ErrGitUnavailable) {
			t.Fatalf("err = %v, want ErrGitUnavailable", err)
		}
	})
}

func TestGitStatusDigest(t *testing.T) {
	t.Run("a clean tree digests the same twice", func(t *testing.T) {
		root := newRepo(t)
		a, err := GitStatusDigest(root)
		if err != nil {
			t.Fatal(err)
		}
		b, err := GitStatusDigest(root)
		if err != nil {
			t.Fatal(err)
		}
		if a != b {
			t.Fatalf("digest changed with nothing done: %q vs %q", a, b)
		}
	})

	t.Run("a new untracked file changes the digest", func(t *testing.T) {
		root := newRepo(t)
		before, err := GitStatusDigest(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("hi"), 0o600); err != nil {
			t.Fatal(err)
		}
		after, err := GitStatusDigest(root)
		if err != nil {
			t.Fatal(err)
		}
		if before == after {
			t.Fatalf("digest did not change: %q", before)
		}
	})

	t.Run("a change under .mtqg/ does not change the digest", func(t *testing.T) {
		root := newRepo(t)
		if err := os.MkdirAll(filepath.Join(root, mtqgDirName), 0o750); err != nil {
			t.Fatal(err)
		}
		before, err := GitStatusDigest(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, mtqgDirName, journalName), []byte(`{"id":"a"}`+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		after, err := GitStatusDigest(root)
		if err != nil {
			t.Fatal(err)
		}
		if before != after {
			t.Fatalf("digest changed from a .mtqg/ file: %q vs %q", before, after)
		}
	})

	t.Run("git cannot be run", func(t *testing.T) {
		root := newRepo(t)
		t.Setenv("PATH", t.TempDir())
		if _, err := GitStatusDigest(root); !errors.Is(err, ErrGitUnavailable) {
			t.Fatalf("err = %v, want ErrGitUnavailable", err)
		}
	})
}
