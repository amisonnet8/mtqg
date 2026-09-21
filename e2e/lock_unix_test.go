//go:build e2e && unix

package e2e

// These tests hold the lock file of a repository from the test, the way another mtqg
// would, and see which commands wait for it. The lock is flock, so the test can take
// it itself on Linux and macOS. (Windows has LockFileEx, and the journal layer's tests
// take it from another process.)

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// holdLock takes the lock of the .mtqg/ of dir, and returns what gives it back.
func holdLock(t *testing.T, dir string) (release func()) {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(dir, ".mtqg", ".local", "lock"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatalf("the lock is taken already: %v", err)
	}
	released := false
	release = func() {
		if !released {
			released = true
			_ = unix.Flock(int(f.Fd()), unix.LOCK_UN)
			_ = f.Close()
		}
	}
	t.Cleanup(release)
	return release
}

// started is a run of mtqg that has not finished yet.
type started struct {
	cmd    *exec.Cmd
	stderr *strings.Builder
	done   chan error
}

// start begins mtqg in dir and returns at once.
func (r *repo) start(dir string, args ...string) *started {
	r.t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	cmd.Env = r.env
	s := &started{cmd: cmd, stderr: &strings.Builder{}, done: make(chan error, 1)}
	cmd.Stderr = s.stderr
	if err := cmd.Start(); err != nil {
		r.t.Fatal(err)
	}
	go func() { s.done <- cmd.Wait() }()
	return s
}

// waits says whether the run is still waiting after a while.
func (s *started) waits() bool {
	select {
	case err := <-s.done:
		s.done <- err
		return false
	case <-time.After(700 * time.Millisecond):
		return true
	}
}

// finishes waits for the run to end, and returns how.
func (s *started) finishes(t *testing.T) error {
	t.Helper()
	select {
	case err := <-s.done:
		return err
	case <-time.After(10 * time.Second):
		_ = s.cmd.Process.Kill()
		t.Fatal("mtqg did not finish after the lock was given back")
		return nil
	}
}

// Every worktree has its own .mtqg/ (it is in the files that git checks out), and so its
// own lock: a writer in one does not wait for a writer in another, and a record made in
// one is not in the other until the branches are merged.
func TestAWorktreeHasItsOwnLock(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")
	r.mtqg("t", "add", "from the main worktree")
	r.commitAll("start")
	wt := filepath.Join(t.TempDir(), "wt")
	r.git("worktree", "add", "-q", "-b", "feature", wt)

	if res := r.runIn(wt, nil, "", "t", "add", "from the other worktree"); res.code != 0 {
		t.Fatalf("add in the worktree: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(wt, ".mtqg", ".local", "lock")); err != nil {
		t.Errorf("the worktree has no lock file of its own: %v", err)
	}
	mainList, wtList := r.mtqg("t", "list"), r.runIn(wt, nil, "", "t", "list").stdout
	if strings.Contains(mainList, "other worktree") || !strings.Contains(wtList, "main worktree") || !strings.Contains(wtList, "other worktree") {
		t.Errorf("main:\n%s\nworktree:\n%s", mainList, wtList)
	}

	release := holdLock(t, r.dir)
	defer release()

	// The lock of the main worktree does not stop a write in the other.
	begin := time.Now()
	if res := r.runIn(wt, nil, "", "t", "add", "written while the main worktree is locked"); res.code != 0 || time.Since(begin) > 3*time.Second {
		t.Errorf("a write in the other worktree: %+v after %s", res, time.Since(begin))
	}
	// It does stop a write in the main worktree, which goes on when it is given back.
	blocked := r.start(r.dir, "t", "add", "written when the lock is given back")
	if !blocked.waits() {
		t.Fatalf("a write in the main worktree did not wait for its lock: %s", blocked.stderr)
	}
	release()
	if err := blocked.finishes(t); err != nil {
		t.Errorf("the write that waited: %v\n%s", err, blocked.stderr)
	}
	if got := r.mtqg("t", "list"); !strings.Contains(got, "written when the lock is given back") || strings.Contains(got, "while the main worktree is locked") {
		t.Errorf("main after the writes:\n%s", got)
	}
}

// Reached through a link, it is the same .mtqg/ and so the same lock. (A lock file kept
// in a temporary directory would be a different one for each path.)
func TestTheSameMtqgDirectoryThroughALinkHasTheSameLock(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")
	r.mtqg("t", "add", "first")
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(r.dir, link); err != nil {
		t.Skipf("cannot make a symbolic link: %v", err)
	}

	release := holdLock(t, r.dir)
	defer release()
	blocked := r.start(t.TempDir(), "-C", link, "t", "add", "written through the link")
	if !blocked.waits() {
		t.Fatalf("a write through the link did not wait for the lock: %s", blocked.stderr)
	}
	release()
	if err := blocked.finishes(t); err != nil {
		t.Errorf("the write that waited: %v\n%s", err, blocked.stderr)
	}
	if got := r.mtqg("t", "list"); !strings.Contains(got, "written through the link") || !strings.HasSuffix(got, "2 open (show done: --all)\n") {
		t.Errorf("list:\n%s", got)
	}
}
