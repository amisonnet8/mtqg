package journal

import (
	"os"
	"path/filepath"
	"testing"
)

// mtqg runs inside repositories that were just cloned, and whoever made a
// repository can commit .mtqg/journal.jsonl or .mtqg/.local as a symbolic link.
// The journal layer must not write through a link that leads outside .mtqg/.

func symlinkOrSkip(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("cannot create a symbolic link here: %v", err)
	}
}

func TestAppendDoesNotWriteThroughALinkOutsideMtqg(t *testing.T) {
	for _, target := range []string{"absolute", "relative"} {
		t.Run(target, func(t *testing.T) {
			j := newJournal(t, nil)
			outside := filepath.Join(realPath(t, t.TempDir()), "victim")
			if err := os.WriteFile(outside, []byte("keep me\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			linkTo := outside
			if target == "relative" {
				rel, err := filepath.Rel(j.loc.Dir, outside)
				if err != nil {
					t.Skipf("no relative path to the file: %v", err)
				}
				linkTo = rel
			}
			symlinkOrSkip(t, linkTo, j.journalPath())

			if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "x"}); err == nil {
				t.Fatal("Append wrote through a link that leads outside .mtqg/")
			}
			if data, err := os.ReadFile(outside); err != nil || string(data) != "keep me\n" {
				t.Errorf("the file outside .mtqg/ was changed: %q, %v", data, err)
			}
			if _, err := j.Read(); err == nil {
				t.Error("Read followed a link that leads outside .mtqg/")
			}
		})
	}
}

func TestLockDoesNotUseALocalDirectoryLinkedOutsideMtqg(t *testing.T) {
	j := newJournal(t, nil)
	outside := realPath(t, t.TempDir())
	symlinkOrSkip(t, outside, filepath.Join(j.loc.Dir, localName))

	if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "x"}); err == nil {
		t.Fatal("Append used a .local that leads outside .mtqg/")
	}
	if entries, err := os.ReadDir(outside); err != nil || len(entries) != 0 {
		t.Errorf("files were created outside .mtqg/: %v, %v", entries, err)
	}
}

func TestAppendFollowsALinkThatStaysInsideMtqg(t *testing.T) {
	j := newJournal(t, nil)
	real := filepath.Join(j.loc.Dir, "real.jsonl")
	if err := os.WriteFile(real, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	symlinkOrSkip(t, "real.jsonl", j.journalPath())

	if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "x"}); err != nil {
		t.Fatalf("a link inside .mtqg/ is fine: %v", err)
	}
	if data, err := os.ReadFile(real); err != nil || len(data) == 0 {
		t.Errorf("the line should be in the file the link points at: %q, %v", data, err)
	}
}
