package journal

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

// TestHelperProcess is not a test. Other tests run this test binary again with
// MTQG_HELPER set, to have code run in a separate process. Without it, it does
// nothing.
func TestHelperProcess(t *testing.T) {
	mode := os.Getenv("MTQG_HELPER")
	if mode == "" {
		t.Skip("helper for the tests that start other processes")
	}
	fail := func(err error) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	j, err := Open(os.Getenv("MTQG_ROOT"), Options{Author: yamada, LockTimeout: 60 * time.Second})
	if err != nil {
		fail(err)
	}
	switch mode {
	case "append":
		count, _ := strconv.Atoi(os.Getenv("MTQG_COUNT"))
		for i := range count {
			text := fmt.Sprintf("%s-%d", os.Getenv("MTQG_TAG"), i)
			if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: text}); err != nil {
				fail(err)
			}
		}
	case "hold":
		release, err := j.lock()
		if err != nil {
			fail(err)
		}
		defer release()
		if _, err := os.Stdout.WriteString("locked\n"); err != nil {
			fail(err)
		}
		time.Sleep(time.Hour) // until the test kills this process
	default:
		fail(fmt.Errorf("unknown helper mode %q", mode))
	}
	os.Exit(0)
}

func helper(t *testing.T, root, mode string, extra ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
	cmd.Env = append(os.Environ(), "MTQG_HELPER="+mode, "MTQG_ROOT="+root)
	cmd.Env = append(cmd.Env, extra...)
	return cmd
}

func TestAppendConcurrentProcesses(t *testing.T) {
	const processes, perProcess = 4, 40
	j := newJournal(t, nil)

	cmds := make([]*exec.Cmd, processes)
	outputs := make([]*syncBuffer, processes)
	for p := range processes {
		cmds[p] = helper(t, j.loc.Root, "append", fmt.Sprintf("MTQG_TAG=p%d", p), fmt.Sprintf("MTQG_COUNT=%d", perProcess))
		outputs[p] = &syncBuffer{}
		cmds[p].Stdout, cmds[p].Stderr = outputs[p], outputs[p]
	}
	for p, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			t.Fatalf("process %d: %v", p, err)
		}
	}
	for p, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			t.Fatalf("process %d: %v\n%s", p, err, outputs[p].String())
		}
	}
	checkAllWritten(t, j, processes, perProcess, "p")
}

type syncBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

func TestLockTimeout(t *testing.T) {
	holder := newJournal(t, nil)
	release, err := holder.lock()
	if err != nil {
		t.Fatal(err)
	}

	waiter, err := Open(holder.loc.Root, Options{Author: yamada, LockTimeout: 150 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_, err = waiter.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "x"})
	elapsed := time.Since(start)

	var timeout *LockTimeoutError
	if !errors.Is(err, ErrLockTimeout) || !errors.As(err, &timeout) {
		t.Fatalf("err = %v, want a LockTimeoutError", err)
	}
	if elapsed < 150*time.Millisecond || elapsed > 5*time.Second {
		t.Errorf("waited %s, want about the 150ms timeout", elapsed)
	}
	if _, err := os.Stat(waiter.journalPath()); err == nil {
		t.Error("a write that timed out changed the journal")
	}

	release()
	if _, err := waiter.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "x"}); err != nil {
		t.Fatalf("after the lock was released: %v", err)
	}
}

func TestLockIsNotReentrant(t *testing.T) {
	j := newJournal(t, nil)
	j.opts.LockTimeout = 50 * time.Millisecond
	release, err := j.lock()
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	// Code that holds the lock must not append: it would wait for itself.
	if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "x"}); !errors.Is(err, ErrLockTimeout) {
		t.Fatalf("err = %v, want ErrLockTimeout", err)
	}
}

func TestLockCanBeTakenAgainAndTheFileStays(t *testing.T) {
	j := newJournal(t, nil)
	for range 3 {
		release, err := j.lock()
		if err != nil {
			t.Fatal(err)
		}
		release()
	}
	if _, err := os.Stat(j.lockPath()); err != nil {
		t.Errorf("the lock file must stay in place: %v", err)
	}
}

// The system releases the lock when the holder dies, even when it is killed, so
// no stale lock can stop mtqg from writing.
func TestLockIsReleasedWhenTheHolderIsKilled(t *testing.T) {
	j := newJournal(t, nil)

	cmd := helper(t, j.loc.Root, "hold")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stderr := &syncBuffer{}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	killed := false
	defer func() {
		if !killed {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()

	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || line != "locked\n" {
		t.Fatalf("the helper did not report the lock: %q, %v\n%s", line, err, stderr.String())
	}

	blocked, err := Open(j.loc.Root, Options{Author: yamada, LockTimeout: 100 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := blocked.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "x"}); !errors.Is(err, ErrLockTimeout) {
		t.Fatalf("while the other process holds the lock: err = %v, want ErrLockTimeout", err)
	}

	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	killed = true
	_ = cmd.Wait() // it reports that it was killed

	j.opts.LockTimeout = 10 * time.Second
	if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "after the kill"}); err != nil {
		t.Fatalf("after the holder was killed: %v", err)
	}
}

// The lock is on the file, not on its name, so the same .mtqg/ reached by
// another path is excluded by the same lock.
func TestLockIsTheSameThroughASymbolicLink(t *testing.T) {
	first := newJournal(t, nil)
	link := filepath.Join(realPath(t, t.TempDir()), "link")
	if err := os.Symlink(first.loc.Root, link); err != nil {
		t.Skipf("cannot create a symbolic link here: %v", err)
	}
	// Built by hand: Open would resolve the link and hide the difference.
	second := &Journal{
		loc:  Location{Root: link, Dir: filepath.Join(link, mtqgDirName)},
		opts: Options{Author: yamada, LockTimeout: 100 * time.Millisecond},
		now:  time.Now,
	}
	if first.lockPath() == second.lockPath() {
		t.Fatal("the test needs two spellings of the path")
	}

	release, err := first.lock()
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if _, err := second.lock(); !errors.Is(err, ErrLockTimeout) {
		t.Fatalf("err = %v, want ErrLockTimeout: the link led to a different lock", err)
	}
}
