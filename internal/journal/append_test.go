package journal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// newJournal creates a repository with .mtqg/ (format 0) and opens it. The
// journal.jsonl is created with content, or not at all when content is nil.
func newJournal(t *testing.T, content *string) *Journal {
	t.Helper()
	root := newRepo(t)
	newMtqg(t, root, "0\n", content)
	j, err := Open(root, Options{Author: yamada, TTY: "3e9a0b12", LockTimeout: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func rawJournal(t *testing.T, j *Journal) string {
	t.Helper()
	data, err := os.ReadFile(j.journalPath())
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func fixedClock(j *Journal, ts string) {
	at, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		panic(err)
	}
	j.now = func() time.Time { return at }
}

func TestAppendFillsInWhatTheCallerDoesNotOwn(t *testing.T) {
	j := newJournal(t, nil)
	fixedClock(j, "2026-09-17T10:32:00+09:00") // written as UTC

	got, err := j.Append(Event{
		Op: OpCreate, Type: TypeMemo, Text: "hello",
		// The journal layer owns these and overwrites what the caller put.
		V: 9, TS: "yesterday", Author: Author{Kind: AuthorAI, Name: "someone"}, TTY: "zz",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !isID(got.ID) || got.V != 0 || got.TS != "2026-09-17T01:32:00Z" || got.Author != yamada || got.TTY != "3e9a0b12" {
		t.Errorf("got %+v", got)
	}

	want, err := encodeLine(got)
	if err != nil {
		t.Fatal(err)
	}
	if rawJournal(t, j) != string(want) {
		t.Errorf("the file holds %q, want %q", rawJournal(t, j), want)
	}
	read, err := j.Read()
	if err != nil || len(read.Events) != 1 || read.Events[0] != got {
		t.Errorf("Read = %+v, %v; want the appended event", read, err)
	}
}

func TestAppendGolden(t *testing.T) {
	j := newJournal(t, nil)
	fixedClock(j, "2026-09-17T01:30:00Z")
	j.opts.Author = Author{Kind: AuthorAI, Name: "claude-code"}
	j.opts.TTY = ""

	if _, err := j.Append(Event{ID: idA, Op: OpStatus, From: StatusOpen, Status: StatusDone}); err != nil {
		t.Fatal(err)
	}
	fixedClock(j, "2026-09-17T01:31:00Z")
	if _, err := j.Append(Event{ID: idA, Op: OpEdit, Text: "a<b> & 日本語\n2行目"}); err != nil {
		t.Fatal(err)
	}
	want := `{"id":"6b0d549b6f03475a8600a35a099950d8","op":"status","from":"open","status":"done","v":0,"ts":"2026-09-17T01:30:00Z","author":{"kind":"ai","name":"claude-code"}}` + "\n" +
		`{"id":"6b0d549b6f03475a8600a35a099950d8","op":"edit","text":"a<b> & 日本語\n2行目","v":0,"ts":"2026-09-17T01:31:00Z","author":{"kind":"ai","name":"claude-code"}}` + "\n"
	if got := rawJournal(t, j); got != want {
		t.Errorf("the file holds\n%q\nwant\n%q", got, want)
	}
}

func TestAppendValidation(t *testing.T) {
	tests := []struct {
		name string
		ev   Event
	}{
		{"no op", Event{ID: idA}},
		{"unknown op", Event{ID: idA, Op: "rename"}},
		{"create with an id", Event{ID: idA, Op: OpCreate, Type: TypeMemo}},
		{"create without a type", Event{Op: OpCreate}},
		{"create with an unknown type", Event{Op: OpCreate, Type: "note"}},
		{"create with a short re", Event{Op: OpCreate, Type: TypeQA, Re: "1012f037b6"}},
		{"status without an id", Event{Op: OpStatus, Status: StatusDone}},
		{"status with a short id", Event{ID: "6b0d549b6f", Op: OpStatus, Status: StatusDone}},
		{"edit with an uppercase id", Event{ID: strings.ToUpper(idA), Op: OpEdit, Text: "x"}},
		{"edit with re", Event{ID: idA, Op: OpEdit, Re: idB, Text: "x"}},
		{"unknown status", Event{ID: idA, Op: OpStatus, Status: "closed"}},
		{"unknown from", Event{ID: idA, Op: OpStatus, From: "closed", Status: StatusDone}},
		{"negative basis", Event{ID: idA, Op: OpEdit, Text: "x", Basis: -1}},
		{"text that is not UTF-8", Event{Op: OpCreate, Type: TypeMemo, Text: "caf\xe9"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := newJournal(t, nil)
			if _, err := j.Append(tt.ev); !errors.Is(err, ErrInvalidEvent) {
				t.Fatalf("err = %v, want ErrInvalidEvent", err)
			}
			if _, err := os.Stat(j.journalPath()); err == nil {
				t.Errorf("the journal was written: %q", rawJournal(t, j))
			}
		})
	}

	t.Run("the author must be known", func(t *testing.T) {
		for _, author := range []Author{{}, {Kind: AuthorHuman}, {Kind: "robot", Name: "x"}} {
			j := newJournal(t, nil)
			j.opts.Author = author
			if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "x"}); !errors.Is(err, ErrInvalidEvent) {
				t.Errorf("author %+v: err = %v, want ErrInvalidEvent", author, err)
			}
		}
	})
}

func TestAppendWritesEveryKind(t *testing.T) {
	j := newJournal(t, nil)
	for _, typ := range []string{TypeMemo, TypeTodo, TypeQA, TypeBug, TypeGlossary} {
		ev, err := j.Append(Event{Op: OpCreate, Type: typ, Text: "x"})
		if err != nil {
			t.Fatalf("type %s: %v", typ, err)
		}
		// The journal layer checks the shape of a line, not what it means: re is
		// accepted with any type (the model decides what can be replied to).
		if _, err := j.Append(Event{Op: OpCreate, Type: typ, Re: ev.ID, Text: "y"}); err != nil {
			t.Errorf("type %s with re: %v", typ, err)
		}
	}
	read, err := j.Read()
	if err != nil || len(read.Events) != 10 {
		t.Fatalf("Read = %d events, %v; want 10", len(read.Events), err)
	}
}

func TestAppendCreatesTheJournalWhenItIsMissing(t *testing.T) {
	j := newJournal(t, nil)
	if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "x"}); err != nil {
		t.Fatal(err)
	}
	if read, err := j.Read(); err != nil || len(read.Events) != 1 {
		t.Fatalf("Read = %+v, %v", read, err)
	}
}

func TestAppendAfterALineWithoutALineFeed(t *testing.T) {
	complete := lineOf(t, memo(idA, "complete"))

	t.Run("a cut-off line is kept apart from the new one", func(t *testing.T) {
		cutOff := `{"id":"` + idB + `","op":"cre`
		j := newJournal(t, str(complete+"\n"+cutOff))
		got, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "new"})
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimSuffix(rawJournal(t, j), "\n"), "\n")
		if len(lines) != 3 || lines[1] != cutOff {
			t.Fatalf("lines = %q: the cut-off line must stay alone", lines)
		}
		read, err := j.Read()
		if err != nil {
			t.Fatal(err)
		}
		if len(read.Events) != 2 || read.Events[1].ID != got.ID {
			t.Errorf("events = %+v, want the complete line and the new one", read.Events)
		}
		if len(read.Warnings) != 1 || read.Warnings[0] != (Warning{Kind: WarnInvalidJSON, Line: 2}) {
			t.Errorf("warnings = %+v, want one for the cut-off line", read.Warnings)
		}
	})

	t.Run("a complete last line without a line feed", func(t *testing.T) {
		j := newJournal(t, str(complete))
		if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "new"}); err != nil {
			t.Fatal(err)
		}
		read, err := j.Read()
		if err != nil || len(read.Events) != 2 || len(read.Warnings) != 0 {
			t.Fatalf("Read = %+v, %v; want both lines and no warning", read, err)
		}
	})
}

func TestAppendRefusesUnresolvedConflictMarkers(t *testing.T) {
	a, b := lineOf(t, memo(idA, "first")), lineOf(t, memo(idB, "second"))
	conflicted := "<<<<<<< HEAD\n" + a + "\n=======\n" + b + "\n>>>>>>> other\n"
	j := newJournal(t, str(conflicted))

	_, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "x"})
	if !errors.Is(err, ErrConflictMarkers) {
		t.Fatalf("err = %v, want ErrConflictMarkers", err)
	}
	if got := rawJournal(t, j); got != conflicted {
		t.Errorf("the journal was changed: %q", got)
	}

	// Removing only the marker lines keeps both sides and lets writing resume.
	if err := os.WriteFile(j.journalPath(), []byte(a+"\n"+b+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "x"}); err != nil {
		t.Fatalf("after the conflict was resolved: %v", err)
	}
}

func TestAppendIsNotStoppedByMarkerLookalikesInText(t *testing.T) {
	j := newJournal(t, nil)
	for _, text := range []string{"=======", "<<<<<<< HEAD", ">>>>>>> other", "a\n=======\nb"} {
		if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: text}); err != nil {
			t.Fatalf("text %q: %v", text, err)
		}
	}
	// The next write scans a file in which the strings are present.
	if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "last"}); err != nil {
		t.Fatalf("after the lookalikes: %v", err)
	}
	if read, err := j.Read(); err != nil || len(read.Events) != 5 || len(read.Warnings) != 0 {
		t.Fatalf("Read = %+v, %v", read, err)
	}
}

func TestAppendWithoutTheLocalDirectory(t *testing.T) {
	j := newJournal(t, nil)
	local := filepath.Join(j.loc.Dir, localName)

	// A fresh clone has no .local/ (it is not committed), and it may be deleted
	// at any time.
	for i := range 2 {
		if _, err := os.Stat(local); err == nil && i == 0 {
			t.Fatal("the test expects a repository without .local/")
		}
		if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "x"}); err != nil {
			t.Fatalf("round %d: %v", i, err)
		}
		if _, err := os.Stat(filepath.Join(local, "lock")); err != nil {
			t.Fatalf("round %d: the lock file was not created: %v", i, err)
		}
		if err := os.RemoveAll(local); err != nil {
			t.Fatal(err)
		}
	}
}

// Every append lands as a whole line and none is lost, when goroutines write
// through one Journal and through separate ones.
func TestAppendConcurrentGoroutines(t *testing.T) {
	const writers, perWriter = 8, 40

	for _, shared := range []bool{true, false} {
		t.Run(fmt.Sprintf("shared journal %v", shared), func(t *testing.T) {
			j := newJournal(t, nil)
			var wg sync.WaitGroup
			errs := make(chan error, writers*perWriter)
			for w := range writers {
				wg.Add(1)
				go func() {
					defer wg.Done()
					writer := j
					if !shared {
						var err error
						if writer, err = Open(j.loc.Root, j.opts); err != nil {
							errs <- err
							return
						}
					}
					for i := range perWriter {
						if _, err := writer.Append(Event{Op: OpCreate, Type: TypeMemo, Text: fmt.Sprintf("w%d-%d", w, i)}); err != nil {
							errs <- err
							return
						}
					}
				}()
			}
			wg.Wait()
			close(errs)
			for err := range errs {
				t.Fatal(err)
			}
			checkAllWritten(t, j, writers, perWriter, "w")
		})
	}
}

// checkAllWritten verifies a journal written by writers that each wrote count
// events with the text "<prefix><writer>-<number>".
func checkAllWritten(t *testing.T, j *Journal, writers, count int, prefix string) {
	t.Helper()
	read, err := j.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Warnings) != 0 {
		t.Fatalf("warnings: %+v: a line is broken or mixed with another", read.Warnings)
	}
	if len(read.Events) != writers*count {
		t.Fatalf("got %d events, want %d: lines were lost", len(read.Events), writers*count)
	}
	ids := make(map[string]bool)
	next := make([]int, writers) // the number each writer must write next
	for _, ev := range read.Events {
		if ids[ev.ID] {
			t.Fatalf("the ID %s appears twice", ev.ID)
		}
		ids[ev.ID] = true
		var w, i int
		if _, err := fmt.Sscanf(ev.Text, prefix+"%d-%d", &w, &i); err != nil || w >= writers {
			t.Fatalf("unexpected text %q", ev.Text)
		}
		if i != next[w] {
			t.Fatalf("writer %d: got number %d, want %d: its lines are out of order", w, i, next[w])
		}
		next[w]++
	}
	// Each line must also be well formed as a raw line of the file.
	for n, line := range strings.Split(strings.TrimSuffix(rawJournal(t, j), "\n"), "\n") {
		if strings.Count(line, `"op":"create"`) != 1 || !strings.HasPrefix(line, `{"id":"`) || !strings.HasSuffix(line, "}") {
			t.Fatalf("line %d is not a single event: %q", n+1, line)
		}
	}
}
