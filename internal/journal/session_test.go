package journal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// openTestJournal makes .mtqg/ in a fresh repository and opens it, with no
// records: what the session tests need to work in.
func openTestJournal(t *testing.T) *Journal {
	t.Helper()
	root := newRepo(t)
	newMtqg(t, root, "0\n", nil)
	j, err := Open(root, Options{Author: yamada})
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func TestSessionReadWrite(t *testing.T) {
	t.Run("nothing yet is not found, not an error", func(t *testing.T) {
		j := openTestJournal(t)
		if _, ok := j.ReadSession("s1"); ok {
			t.Fatal("found a session that was never written")
		}
	})

	t.Run("round-trips what was written", func(t *testing.T) {
		j := openTestJournal(t)
		want := Session{StartedAt: time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC), Head: "abc123", StatusDigest: "def456"}
		if err := j.WriteSession("s1", want); err != nil {
			t.Fatal(err)
		}
		got, ok := j.ReadSession("s1")
		if !ok {
			t.Fatal("not found after writing")
		}
		if !got.StartedAt.Equal(want.StartedAt) || got.Head != want.Head || got.StatusDigest != want.StatusDigest || got.Prompted != want.Prompted {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})

	t.Run("a second write replaces the first", func(t *testing.T) {
		j := openTestJournal(t)
		if err := j.WriteSession("s1", Session{Head: "first"}); err != nil {
			t.Fatal(err)
		}
		if err := j.WriteSession("s1", Session{Head: "second", Prompted: true}); err != nil {
			t.Fatal(err)
		}
		got, ok := j.ReadSession("s1")
		if !ok || got.Head != "second" || !got.Prompted {
			t.Fatalf("got %+v, ok=%v; want Head=second, Prompted=true", got, ok)
		}
	})

	t.Run("two different session IDs do not collide", func(t *testing.T) {
		j := openTestJournal(t)
		if err := j.WriteSession("s1", Session{Head: "one"}); err != nil {
			t.Fatal(err)
		}
		if err := j.WriteSession("s2", Session{Head: "two"}); err != nil {
			t.Fatal(err)
		}
		one, _ := j.ReadSession("s1")
		two, _ := j.ReadSession("s2")
		if one.Head != "one" || two.Head != "two" {
			t.Fatalf("got %q, %q; want one, two", one.Head, two.Head)
		}
	})

	t.Run("a session ID with path separators does not escape .local/sessions/", func(t *testing.T) {
		j := openTestJournal(t)
		if err := j.WriteSession("../../etc/passwd", Session{Head: "x"}); err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(j.loc.Dir, localName, sessionsDirName)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 {
			t.Fatalf("got %d files in %s, want 1", len(entries), dir)
		}
		if strings.ContainsAny(entries[0].Name(), "/\\") {
			t.Fatalf("file name %q escapes the directory", entries[0].Name())
		}
		// It must still be readable back by the same (unsafe-looking) ID.
		if _, ok := j.ReadSession("../../etc/passwd"); !ok {
			t.Fatal("could not read back the session written with a path-like ID")
		}
	})

	t.Run("a corrupt session file reads as nothing, not an error", func(t *testing.T) {
		j := openTestJournal(t)
		dir := filepath.Join(j.loc.Dir, localName, sessionsDirName)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, sessionFileName("s1")), []byte("not json"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, ok := j.ReadSession("s1"); ok {
			t.Fatal("a corrupt session file should read as \"not found\"")
		}
	})
}

func TestPruneSessions(t *testing.T) {
	j := openTestJournal(t)
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	if err := j.WriteSession("old", Session{Head: "old"}); err != nil {
		t.Fatal(err)
	}
	if err := j.WriteSession("fresh", Session{Head: "fresh"}); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(j.loc.Dir, localName, sessionsDirName)
	old := now.Add(-SessionMaxAge - time.Hour)
	if err := os.Chtimes(filepath.Join(dir, sessionFileName("old")), old, old); err != nil {
		t.Fatal(err)
	}

	j.PruneSessions(now)

	if _, ok := j.ReadSession("old"); ok {
		t.Fatal("an old session file should have been pruned")
	}
	if _, ok := j.ReadSession("fresh"); !ok {
		t.Fatal("a fresh session file should not have been pruned")
	}
}
