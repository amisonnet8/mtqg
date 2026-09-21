package journal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// moveMatching moves the lines whose event text starts with prefix; the rest,
// including lines that could not be read, are kept.
func moveMatching(prefix string) func([]Line) ([]Line, []Line, error) {
	return func(lines []Line) ([]Line, []Line, error) {
		var move, keep []Line
		for _, line := range lines {
			if line.Event != nil && strings.HasPrefix(line.Event.Text, prefix) {
				move = append(move, line)
			} else {
				keep = append(keep, line)
			}
		}
		return move, keep, nil
	}
}

func archiveFile(j *Journal, name string) string {
	return filepath.Join(j.loc.Dir, archiveDirName, name)
}

func readArchive(t *testing.T, j *Journal, name string) string {
	t.Helper()
	data, err := os.ReadFile(archiveFile(j, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestArchivePath(t *testing.T) {
	if got := ArchivePath("2021-01-01..2024-09-18.jsonl"); got != ".mtqg/archive/2021-01-01..2024-09-18.jsonl" {
		t.Errorf("ArchivePath = %q", got)
	}
}

func TestArchiveMovesLinesByteForByte(t *testing.T) {
	// The formatting mtqg never writes, an unknown field, a line that cannot be
	// read, and two lines with the same content: on either side, none may change.
	handMade := `{ "op" : "create", "id":"` + idA + `", "type":"memo", "text":"old hand-made", "priority":"high" }`
	broken := "not json at all"
	dup := lineOf(t, memo(idC, "old dup"))
	stays := lineOf(t, memo(idB, "new"))
	j := newJournal(t, str(handMade+"\n"+stays+"\n"+broken+"\n"+dup+"\n"+dup+"\n"))

	if err := j.Archive("a.jsonl", moveMatching("old")); err != nil {
		t.Fatal(err)
	}
	if got, want := rawJournal(t, j), stays+"\n"+broken+"\n"; got != want {
		t.Errorf("journal.jsonl holds\n%q\nwant\n%q", got, want)
	}
	if got, want := readArchive(t, j, "a.jsonl"), handMade+"\n"+dup+"\n"+dup+"\n"; got != want {
		t.Errorf("the archive holds\n%q\nwant\n%q", got, want)
	}
}

func TestArchiveThenAppendingItBackRestoresTheEvents(t *testing.T) {
	a, b, c := lineOf(t, memo(idA, "old a")), lineOf(t, memo(idB, "new b")), lineOf(t, memo(idC, "old c"))
	j := newJournal(t, str(a+"\n"+b+"\n"+c+"\n"))
	before, err := j.Read()
	if err != nil {
		t.Fatal(err)
	}

	if err := j.Archive("a.jsonl", moveMatching("old")); err != nil {
		t.Fatal(err)
	}
	mid, err := j.Read()
	if err != nil || len(mid.Events) != 1 {
		t.Fatalf("after archiving: %d events, %v", len(mid.Events), err)
	}

	// The way to restore, as the spec gives it: append the file and delete it.
	f, err := os.OpenFile(j.journalPath(), os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(readArchive(t, j, "a.jsonl")); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	after, err := j.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Warnings) != 0 || len(after.Events) != len(before.Events) {
		t.Fatalf("restored %d events with %d warnings, want %d", len(after.Events), len(after.Warnings), len(before.Events))
	}
	seen := map[string]bool{}
	for _, ev := range after.Events {
		seen[ev.ID] = true
	}
	for _, ev := range before.Events {
		if !seen[ev.ID] {
			t.Errorf("%s did not come back", ev.ID)
		}
	}
}

func TestArchiveAppendsToAnExistingFile(t *testing.T) {
	first, second := lineOf(t, memo(idA, "old 1")), lineOf(t, memo(idB, "old 2"))
	j := newJournal(t, str(first+"\n"+lineOf(t, memo(idC, "new"))+"\n"))
	if err := j.Archive("a.jsonl", moveMatching("old")); err != nil {
		t.Fatal(err)
	}
	// A second line to move, added by hand.
	f, err := os.OpenFile(j.journalPath(), os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(second + "\n"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	if err := j.Archive("a.jsonl", moveMatching("old")); err != nil {
		t.Fatal(err)
	}
	if got, want := readArchive(t, j, "a.jsonl"), first+"\n"+second+"\n"; got != want {
		t.Errorf("the archive holds\n%q\nwant\n%q", got, want)
	}
}

func TestArchiveAddsALineEndingToAFileThatHasNone(t *testing.T) {
	j := newJournal(t, str(lineOf(t, memo(idA, "old"))+"\n"))
	if err := os.MkdirAll(filepath.Join(j.loc.Dir, archiveDirName), 0o750); err != nil {
		t.Fatal(err)
	}
	partial := `{"id":"` + idB + `","op":"crea` // a write that stopped half way
	if err := os.WriteFile(archiveFile(j, "a.jsonl"), []byte(partial), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := j.Archive("a.jsonl", moveMatching("old")); err != nil {
		t.Fatal(err)
	}
	if got, want := readArchive(t, j, "a.jsonl"), partial+"\n"+lineOf(t, memo(idA, "old"))+"\n"; got != want {
		t.Errorf("the archive holds\n%q\nwant\n%q", got, want)
	}
}

func TestArchiveMovesNothing(t *testing.T) {
	original := lineOf(t, memo(idA, "new")) + "\n"
	j := newJournal(t, str(original))
	if err := j.Archive("a.jsonl", moveMatching("old")); err != nil {
		t.Fatal(err)
	}
	if got := rawJournal(t, j); got != original {
		t.Errorf("journal.jsonl was changed: %q", got)
	}
	if _, err := os.Stat(filepath.Join(j.loc.Dir, archiveDirName)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("archive/ was made although nothing moved: %v", err)
	}
}

func TestArchiveLeavesEverythingAloneWhenItFails(t *testing.T) {
	a, b := lineOf(t, memo(idA, "old")), lineOf(t, memo(idB, "new"))
	original := a + "\n" + b + "\n"
	boom := errors.New("boom")

	tests := []struct {
		name    string
		fn      func([]Line) ([]Line, []Line, error)
		wantErr func(error) bool
	}{
		{"fn fails", func([]Line) ([]Line, []Line, error) { return nil, nil, boom },
			func(err error) bool { return errors.Is(err, boom) }},
		{"a line that is in neither list", func(lines []Line) ([]Line, []Line, error) { return lines[:1], nil, nil },
			func(err error) bool { return err != nil && strings.Contains(err.Error(), "not the 2 lines") }},
		{"a line that is in both lists", func(lines []Line) ([]Line, []Line, error) { return lines, lines[:1], nil },
			func(err error) bool { return err != nil && strings.Contains(err.Error(), "not the 2 lines") }},
		{"a line with a line ending inside", func(lines []Line) ([]Line, []Line, error) {
			return []Line{{Raw: []byte("a\nb")}}, lines[1:], nil
		}, func(err error) bool { return errors.Is(err, ErrInvalidEvent) }},
		{"a line to keep with a line ending inside", func(lines []Line) ([]Line, []Line, error) {
			return lines[:1], []Line{{Raw: []byte("a\nb")}}, nil
		}, func(err error) bool { return errors.Is(err, ErrInvalidEvent) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := newJournal(t, str(original))
			if err := j.Archive("a.jsonl", tt.fn); err == nil || !tt.wantErr(err) {
				t.Fatalf("err = %v", err)
			}
			if got := rawJournal(t, j); got != original {
				t.Errorf("journal.jsonl was changed: %q", got)
			}
			if _, err := os.Stat(filepath.Join(j.loc.Dir, archiveDirName)); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("archive/ was made: %v", err)
			}
			// The lock was given back.
			if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "after"}); err != nil {
				t.Errorf("Append after a failed archive: %v", err)
			}
		})
	}
}

func TestArchiveRefusesUnresolvedConflictMarkers(t *testing.T) {
	conflicted := "<<<<<<< HEAD\n" + lineOf(t, memo(idA, "old a")) + "\n=======\n" + lineOf(t, memo(idB, "old b")) + "\n>>>>>>> other\n"
	j := newJournal(t, str(conflicted))
	called := false
	err := j.Archive("a.jsonl", func(lines []Line) ([]Line, []Line, error) {
		called = true
		return lines, nil, nil
	})
	if !errors.Is(err, ErrConflictMarkers) || called {
		t.Fatalf("err = %v, fn called = %v", err, called)
	}
	if got := rawJournal(t, j); got != conflicted {
		t.Errorf("journal.jsonl was changed: %q", got)
	}
}

func TestArchiveNameMustBeASimpleFileName(t *testing.T) {
	original := lineOf(t, memo(idA, "old")) + "\n"
	j := newJournal(t, str(original))
	for _, name := range []string{"", ".", "..", "sub/a.jsonl", `sub\a.jsonl`, "../a.jsonl", "/abs.jsonl"} {
		called := false
		err := j.Archive(name, func(lines []Line) ([]Line, []Line, error) {
			called = true
			return lines, nil, nil
		})
		if err == nil || called {
			t.Errorf("name %q: err = %v, fn called = %v", name, err, called)
		}
	}
	if got := rawJournal(t, j); got != original {
		t.Errorf("journal.jsonl was changed: %q", got)
	}
}

func TestArchiveKeepsThePermissionsOfTheJournal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no permission bits to keep")
	}
	j := newJournal(t, str(lineOf(t, memo(idA, "old"))+"\n"+lineOf(t, memo(idB, "new"))+"\n"))
	if err := os.Chmod(j.journalPath(), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := j.Archive("a.jsonl", moveMatching("old")); err != nil {
		t.Fatal(err)
	}
	for name, path := range map[string]string{"journal.jsonl": j.journalPath(), "the archive": archiveFile(j, "a.jsonl")} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o640 {
			t.Errorf("%s: mode = %o, want 640", name, got)
		}
	}
}

// A person may read the archive of a repository they just cloned, so an archive
// file that is a link must not be written through, nor archive/ itself.
func TestArchiveDoesNotWriteThroughALinkOutsideMtqg(t *testing.T) {
	for _, what := range []string{"the file", "the directory"} {
		t.Run(what, func(t *testing.T) {
			original := lineOf(t, memo(idA, "old")) + "\n"
			j := newJournal(t, str(original))
			outside := filepath.Join(realPath(t, t.TempDir()), "victim")
			if what == "the file" {
				if err := os.WriteFile(outside, []byte("keep me\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				mkdir(t, filepath.Join(j.loc.Dir, archiveDirName))
				symlinkOrSkip(t, outside, archiveFile(j, "a.jsonl"))
			} else {
				mkdir(t, outside)
				symlinkOrSkip(t, outside, filepath.Join(j.loc.Dir, archiveDirName))
			}

			if err := j.Archive("a.jsonl", moveMatching("old")); err == nil {
				t.Fatal("Archive followed a link that leads outside .mtqg/")
			}
			if got := rawJournal(t, j); got != original {
				t.Errorf("journal.jsonl was changed: %q", got)
			}
			if what == "the file" {
				if data, err := os.ReadFile(outside); err != nil || string(data) != "keep me\n" {
					t.Errorf("the file outside .mtqg/ was changed: %q, %v", data, err)
				}
			} else if entries, err := os.ReadDir(outside); err != nil || len(entries) != 0 {
				t.Errorf("something was written outside .mtqg/: %v, %v", entries, err)
			}
		})
	}
}

// The order is the point of Archive: the lines reach the archive before they
// leave journal.jsonl, so a failure at any step never loses one.
func TestArchiveNeverLosesALineWhenAWriteFails(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs a directory that cannot be written, which this system does not give")
	}
	old, other := lineOf(t, memo(idA, "old")), lineOf(t, memo(idB, "new"))
	original := old + "\n" + other + "\n"

	t.Run("archive/ cannot be written: journal.jsonl still holds every line", func(t *testing.T) {
		j := newJournal(t, str(original))
		dir := mkdir(t, filepath.Join(j.loc.Dir, archiveDirName))
		if err := os.Chmod(dir, 0o500); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0o750) })

		if err := j.Archive("a.jsonl", moveMatching("old")); err == nil {
			t.Fatal("Archive succeeded although archive/ cannot be written")
		}
		if got := rawJournal(t, j); got != original {
			t.Errorf("journal.jsonl lost lines: %q", got)
		}
	})

	t.Run("journal.jsonl cannot be replaced: the lines are in the archive too", func(t *testing.T) {
		j := newJournal(t, str(original))
		// One append makes .local/ and its lock; then .local/ is closed to new files,
		// which stops the temporary file that replaces journal.jsonl.
		if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "new 2"}); err != nil {
			t.Fatal(err)
		}
		local := filepath.Join(j.loc.Dir, localName)
		if err := os.Chmod(local, 0o500); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(local, 0o750) })
		before := rawJournal(t, j)

		if err := j.Archive("a.jsonl", moveMatching("old")); err == nil {
			t.Fatal("Archive succeeded although journal.jsonl cannot be replaced")
		}
		if got := rawJournal(t, j); got != before {
			t.Errorf("journal.jsonl was changed: %q", got)
		}
		if got := readArchive(t, j, "a.jsonl"); got != old+"\n" {
			t.Errorf("the archive holds %q, want the moved line", got)
		}
	})
}

// A line whose Append returned must still be there after any number of archives,
// in journal.jsonl or in the archive: an archive takes the lock of an append.
func TestArchiveWhileGoroutinesAppend(t *testing.T) {
	const writers, perWriter = 3, 15
	j := newJournal(t, nil)

	var appenders sync.WaitGroup
	errs := make(chan error, writers*perWriter+1)
	for w := range writers {
		appenders.Add(1)
		go func() {
			defer appenders.Done()
			for i := range perWriter {
				if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: fmt.Sprintf("w%d-%d", w, i)}); err != nil {
					errs <- err
					return
				}
				time.Sleep(time.Millisecond) // the lock is not fair; see TestRewriteWhileGoroutinesAppend
			}
		}()
	}

	var done atomic.Bool
	result := make(chan [2]int, 1) // archives run, lines that were sent to be archived
	go func() {
		runs, seeded := 0, 0
		for !done.Load() {
			if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: fmt.Sprintf("drop-%d", seeded)}); err != nil {
				errs <- err
				break
			}
			seeded++
			if err := j.Archive("a.jsonl", moveMatching("drop-")); err != nil {
				errs <- err
				break
			}
			runs++
			time.Sleep(2 * time.Millisecond)
		}
		result <- [2]int{runs, seeded}
	}()

	appenders.Wait()
	done.Store(true)
	got := <-result
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if got[0] < 3 {
		t.Fatalf("only %d archives ran while appending: the test proves nothing", got[0])
	}
	checkAllWritten(t, j, writers, perWriter, "w")
	if n := strings.Count(readArchive(t, j, "a.jsonl"), "\n"); n != got[1] {
		t.Errorf("the archive holds %d lines, want the %d that were sent to it", n, got[1])
	}
}
