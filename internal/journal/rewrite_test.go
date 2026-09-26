package journal

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func keepAll(lines []Line) ([]Line, error) { return lines, nil }

func TestRewriteKeepsTheOriginalBytes(t *testing.T) {
	// Formatting mtqg would never write, an unknown field, a line that cannot be
	// read, invalid UTF-8, and two lines with the same content: none of them may
	// change. Blank lines and CRLF are the only things that are normalized.
	handMade := `{ "op" : "create", "id":"` + idA + `", "priority":"high" }`
	broken := "not json at all"
	bad := "{\"id\":\"" + idB + "\",\"op\":\"create\",\"text\":\"caf\xe9\"}"
	dup := lineOf(t, memo(idC, "dup"))
	input := handMade + "\r\n\n" + broken + "\n" + bad + "\n" + dup + "\n" + dup + "\n  \n"

	j := newJournal(t, str(input))
	if err := j.Rewrite(keepAll); err != nil {
		t.Fatal(err)
	}
	want := handMade + "\n" + broken + "\n" + bad + "\n" + dup + "\n" + dup + "\n"
	if got := rawJournal(t, j); got != want {
		t.Errorf("the file holds\n%q\nwant\n%q", got, want)
	}
}

func TestRewriteRemovesOnlyWhatFnLeavesOut(t *testing.T) {
	keep1 := lineOf(t, memo(idA, "keep 1"))
	drop := lineOf(t, memo(idB, "drop"))
	broken := "not json"
	keep2 := lineOf(t, memo(idC, "keep 2"))
	j := newJournal(t, str(keep1+"\n"+drop+"\n"+broken+"\n"+keep2+"\n"))

	err := j.Rewrite(func(lines []Line) ([]Line, error) {
		var kept []Line
		for _, line := range lines {
			if line.Event != nil && line.Event.Text == "drop" {
				continue
			}
			kept = append(kept, line)
		}
		return kept, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := rawJournal(t, j), keep1+"\n"+broken+"\n"+keep2+"\n"; got != want {
		t.Errorf("the file holds\n%q\nwant\n%q", got, want)
	}
}

func TestRewriteWritesALineFromAnEventByTheWritingRules(t *testing.T) {
	first := lineOf(t, memo(idA, "first"))
	j := newJournal(t, str(first+"\n"))
	added := Event{ID: idA, Op: OpDelete, V: 0, TS: "2026-09-17T02:00:00Z", Author: yamada}

	err := j.Rewrite(func(lines []Line) ([]Line, error) {
		return append(lines, Line{Event: &added}), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := first + "\n" +
		`{"id":"6b0d549b6f03475a8600a35a099950d8","op":"delete","v":0,"ts":"2026-09-17T02:00:00Z","author":{"kind":"human","name":"yamada"}}` + "\n"
	if got := rawJournal(t, j); got != want {
		t.Errorf("the file holds\n%q\nwant\n%q", got, want)
	}
}

func TestRewriteLeavesTheFileAloneWhenItFails(t *testing.T) {
	original := lineOf(t, memo(idA, "first")) + "\n"
	boom := errors.New("boom")

	tests := []struct {
		name    string
		fn      func([]Line) ([]Line, error)
		wantErr func(error) bool
	}{
		{
			name:    "fn fails",
			fn:      func([]Line) ([]Line, error) { return nil, boom },
			wantErr: func(err error) bool { return errors.Is(err, boom) },
		},
		{
			name:    "a line with a line ending inside",
			fn:      func([]Line) ([]Line, error) { return []Line{{Raw: []byte("a\nb")}}, nil },
			wantErr: func(err error) bool { return errors.Is(err, ErrInvalidEvent) },
		},
		{
			name:    "a line with neither raw bytes nor an event",
			fn:      func([]Line) ([]Line, error) { return []Line{{}}, nil },
			wantErr: func(err error) bool { return errors.Is(err, ErrInvalidEvent) },
		},
		{
			name: "an event that is not valid UTF-8",
			fn: func([]Line) ([]Line, error) {
				ev := memo(idB, "caf\xe9")
				return []Line{{Event: &ev}}, nil
			},
			wantErr: func(err error) bool { return errors.Is(err, ErrInvalidEvent) },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := newJournal(t, str(original))
			if err := j.Rewrite(tt.fn); err == nil || !tt.wantErr(err) {
				t.Fatalf("err = %v", err)
			}
			if got := rawJournal(t, j); got != original {
				t.Errorf("the file was changed: %q", got)
			}
			// The lock was given back.
			if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "after"}); err != nil {
				t.Errorf("Append after a failed rewrite: %v", err)
			}
		})
	}
}

func TestRewriteRefusesUnresolvedConflictMarkers(t *testing.T) {
	conflicted := "<<<<<<< HEAD\n" + lineOf(t, memo(idA, "a")) + "\n=======\n" + lineOf(t, memo(idB, "b")) + "\n>>>>>>> other\n"
	j := newJournal(t, str(conflicted))
	called := false
	err := j.Rewrite(func(lines []Line) ([]Line, error) { called = true; return lines, nil })
	if !errors.Is(err, ErrConflictMarkers) {
		t.Fatalf("err = %v, want ErrConflictMarkers", err)
	}
	if called {
		t.Error("fn was called although the rewrite is refused")
	}
	if got := rawJournal(t, j); got != conflicted {
		t.Errorf("the file was changed: %q", got)
	}
}

func TestRewriteWithNothingLeft(t *testing.T) {
	t.Run("an existing file becomes empty", func(t *testing.T) {
		j := newJournal(t, str(lineOf(t, memo(idA, "only"))+"\n"))
		if err := j.Rewrite(func([]Line) ([]Line, error) { return nil, nil }); err != nil {
			t.Fatal(err)
		}
		if got := rawJournal(t, j); got != "" {
			t.Errorf("the file holds %q, want it empty", got)
		}
	})
	t.Run("a missing file is not created for nothing", func(t *testing.T) {
		j := newJournal(t, nil)
		if err := j.Rewrite(func([]Line) ([]Line, error) { return nil, nil }); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(j.journalPath()); err == nil {
			t.Error("journal.jsonl was created")
		}
	})
	t.Run("a missing file is created when there is something to write", func(t *testing.T) {
		j := newJournal(t, nil)
		ev := memo(idA, "new")
		if err := j.Rewrite(func([]Line) ([]Line, error) { return []Line{{Event: &ev}}, nil }); err != nil {
			t.Fatal(err)
		}
		if read, err := j.Read(); err != nil || len(read.Events) != 1 {
			t.Fatalf("Read = %+v, %v", read, err)
		}
	})
}

func TestRewriteKeepsThePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no permission bits to keep")
	}
	j := newJournal(t, str(lineOf(t, memo(idA, "x"))+"\n"))
	if err := os.Chmod(j.journalPath(), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := j.Rewrite(keepAll); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(j.journalPath())
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Errorf("mode = %o, want 640", got)
	}
}

func TestRewriteClearsWhatAnEarlierRewriteLeftBehind(t *testing.T) {
	j := newJournal(t, str(lineOf(t, memo(idA, "x"))+"\n"))
	tmp := filepath.Join(j.loc.Dir, localName, tmpName)
	if err := os.MkdirAll(tmp, 0o750); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"journal.jsonl.tmp", "other"} {
		if err := os.WriteFile(filepath.Join(tmp, name), []byte("unfinished"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := j.Rewrite(keepAll); err != nil {
		t.Fatalf("a leftover must not stop a rewrite: %v", err)
	}
	if entries, err := os.ReadDir(tmp); err != nil || len(entries) != 0 {
		t.Errorf("tmp holds %v (%v), want nothing", entries, err)
	}
}

func TestRewriteWaitsForTheLockAndTimesOut(t *testing.T) {
	original := lineOf(t, memo(idA, "x")) + "\n"
	holder := newJournal(t, str(original))
	release, err := holder.lock()
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	waiter, err := Open(holder.loc.Root, Options{Author: yamada, LockTimeout: 100 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	called := false
	err = waiter.Rewrite(func(lines []Line) ([]Line, error) { called = true; return lines, nil })
	if !errors.Is(err, ErrLockTimeout) || called {
		t.Fatalf("err = %v, fn called = %v; want ErrLockTimeout and no call", err, called)
	}
}

func TestRewriteDoesNotFollowALinkOutsideMtqg(t *testing.T) {
	j := newJournal(t, nil)
	outside := filepath.Join(realPath(t, t.TempDir()), "victim")
	if err := os.WriteFile(outside, []byte("keep me\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	symlinkOrSkip(t, outside, j.journalPath())

	if err := j.Rewrite(keepAll); err == nil {
		t.Fatal("Rewrite followed a link that leads outside .mtqg/")
	}
	if data, err := os.ReadFile(outside); err != nil || string(data) != "keep me\n" {
		t.Errorf("the file outside .mtqg/ was changed: %q, %v", data, err)
	}
}

// A reader has the file open only for a moment. On Windows that stops a
// replacement, so the rewrite has to try again; elsewhere it goes through at
// once. Either way it must succeed.
func TestRewriteWhileAnotherHandleIsOpen(t *testing.T) {
	j := newJournal(t, str(lineOf(t, memo(idA, "x"))+"\n"))
	f, err := os.Open(j.journalPath())
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(200 * time.Millisecond)
		_ = f.Close()
	}()
	if err := j.Rewrite(keepAll); err != nil {
		t.Fatalf("Rewrite while a reader had the file open: %v", err)
	}
}

// A line whose Append returned must still be there after any number of
// rewrites: this is the reason the rewrite takes the same lock as an append.
//
// The appenders go on until the rewrites have run wantRewrites times, so every
// rewrite runs while lines are being appended, however the lock (which is not
// fair, see lock) is shared out. A fixed count on both sides flaked in CI
// (windows-latest, macos-latest race: "only 1 rewrites ran while appending",
// 2026-09-25) because the appenders could finish before the rewriter got its
// share of the lock.
func TestRewriteWhileGoroutinesAppend(t *testing.T) {
	const writers, wantRewrites = 4, 3
	j := newJournal(t, nil)
	seedDroppable(t, j, 10)

	var stop atomic.Bool
	written := make([]int, writers) // how many lines each writer got in; read after they stop
	var appenders sync.WaitGroup
	errs := make(chan error, writers+1)
	for w := range writers {
		appenders.Add(1)
		go func() {
			defer appenders.Done()
			for i := 0; !stop.Load(); i++ {
				if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: fmt.Sprintf("w%d-%d", w, i)}); err != nil {
					errs <- err
					return
				}
				written[w] = i + 1
				// The lock is not fair (see lock): without a pause the appenders
				// could keep the rewriter out for the whole test.
				time.Sleep(time.Millisecond)
			}
		}()
	}

	for n := 0; n < wantRewrites; n++ {
		if err := j.Rewrite(dropDroppable); err != nil {
			errs <- err
			break
		}
		time.Sleep(2 * time.Millisecond) // let a waiting append take its turn
	}
	stop.Store(true)
	appenders.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}

	read, err := j.Read()
	if err != nil || len(read.Warnings) != 0 {
		t.Fatalf("reading: %v, warnings %+v", err, read.Warnings)
	}
	total := 0
	for _, n := range written {
		total += n
	}
	if len(read.Events) != total {
		t.Fatalf("journal.jsonl holds %d events, want the %d that were appended: lines were lost (or a drop- line survived the last rewrite)", len(read.Events), total)
	}
	next := make([]int, writers)
	for _, ev := range read.Events {
		var w, i int
		if _, err := fmt.Sscanf(ev.Text, "w%d-%d", &w, &i); err != nil || w >= writers {
			t.Fatalf("unexpected text %q", ev.Text)
		}
		if i != next[w] {
			t.Fatalf("writer %d: got number %d, want %d: its lines are out of order", w, i, next[w])
		}
		next[w]++
	}
}

// The same across processes, which is how mtqg is used: several agents append
// while a person runs undo or archive.
//
// The processes append until they are killed, so every rewrite runs while
// lines are being appended, however the lock (which is not fair, see lock) is
// shared out. A fixed count on both sides (a process exiting on its own after
// a set number of lines, the rewriter stopping once it had counted 3) was the
// same design flaw already fixed in TestRewriteWhileGoroutinesAppend: how many
// lines each process got in is read back from journal.jsonl afterwards,
// rather than reported by a process that was killed mid-loop (mtqg's bug
// 5e7e4a8492, 2026-09-26).
func TestRewriteWhileProcessesAppend(t *testing.T) {
	const processes, wantRewrites = 3, 3
	j := newJournal(t, nil)
	seedDroppable(t, j, 10)

	cmds := make([]*exec.Cmd, processes)
	outputs := make([]*syncBuffer, processes)
	waits := make(chan error, processes)
	for p := range processes {
		cmds[p] = helper(t, j.loc.Root, "append", fmt.Sprintf("MTQG_TAG=p%d", p), "MTQG_PAUSE_MS=1")
		outputs[p] = &syncBuffer{}
		cmds[p].Stdout, cmds[p].Stderr = outputs[p], outputs[p]
		if err := cmds[p].Start(); err != nil {
			t.Fatal(err)
		}
	}
	for p, cmd := range cmds {
		go func() {
			err := cmd.Wait()
			if err != nil {
				err = fmt.Errorf("process %d: %w\n%s", p, err, outputs[p].String())
			}
			waits <- err
		}()
	}

	killed := false
	kill := func() {
		killed = true
		for p, cmd := range cmds {
			if err := cmd.Process.Kill(); err != nil {
				t.Errorf("process %d: killing: %v", p, err)
			}
		}
		for range cmds {
			<-waits // each reports that it was killed
		}
	}
	defer func() {
		if !killed {
			kill()
		}
	}()

	appended := func() int {
		read, err := j.Read()
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, ev := range read.Events {
			if !strings.HasPrefix(ev.Text, "drop-") {
				n++
			}
		}
		return n
	}

	// wantRewrites is reached quickly; going past it only happens if nothing
	// has been appended yet, which a healthy run does not need (the cap is a
	// safety net against a genuine hang, not a timing budget).
	for rewrites := 0; rewrites < wantRewrites || appended() == 0; rewrites++ {
		select {
		case err := <-waits:
			t.Fatalf("a process exited before being told to stop: %v", err)
		default:
		}
		if err := j.Rewrite(dropDroppable); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Millisecond) // let a waiting append take its turn
		if rewrites > 500 {
			t.Fatalf("still nothing appended after %d rewrites: the test proves nothing", rewrites)
		}
	}
	kill()

	read, err := j.Read()
	if err != nil || len(read.Warnings) != 0 {
		t.Fatalf("reading: %v, warnings %+v", err, read.Warnings)
	}
	next := make([]int, processes)
	for _, ev := range read.Events {
		var p, i int
		if _, err := fmt.Sscanf(ev.Text, "p%d-%d", &p, &i); err != nil || p >= processes {
			t.Fatalf("unexpected text %q", ev.Text)
		}
		if i != next[p] {
			t.Fatalf("process %d: got number %d, want %d: its lines are out of order", p, i, next[p])
		}
		next[p]++
	}
}

// seedDroppable puts lines into the journal that dropDroppable removes.
func seedDroppable(t *testing.T, j *Journal, n int) {
	t.Helper()
	for i := range n {
		if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: fmt.Sprintf("drop-%d", i)}); err != nil {
			t.Fatal(err)
		}
	}
}

func dropDroppable(lines []Line) ([]Line, error) {
	var kept []Line
	for _, line := range lines {
		if line.Event != nil && strings.HasPrefix(line.Event.Text, "drop-") {
			continue
		}
		kept = append(kept, line)
	}
	return kept, nil
}
