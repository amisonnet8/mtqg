package journal

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Waiting for the lock: the first retry comes after a millisecond, and the wait
// doubles up to a limit. A lock is held for one append, so the first retries
// usually succeed.
const (
	lockFirstWait = time.Millisecond
	lockMaxWait   = 50 * time.Millisecond
)

// lockPath is the lock file. It sits in .mtqg/.local/ and not in a temporary
// directory: however .mtqg/ is reached (a symbolic link, another spelling of the
// path), this is the same file, so the lock excludes the same writers. It is not
// journal.jsonl either, because a rewrite replaces that file, and a process that
// waited on the old one would lock a file nobody sees any more.
func (j *Journal) lockPath() string {
	return filepath.Join(j.loc.Dir, localName, lockName)
}

// lockOpenAttempts bounds how often opening the lock file is tried again when
// .local/ is not there.
const lockOpenAttempts = 20

// openLockFile opens the lock file, creating it and .local/ when needed.
//
// .local/ is not committed, so a fresh clone does not have it, and it may be
// deleted at any time. The file is opened first and .local/ is only made when it
// is missing: the common case then costs one call, and several processes that
// start together in a fresh clone (agents running commands in parallel) do not
// all make the directory at once. A missing .local/ is tried again after making
// it, a few times, because opening the file can still fail with "no such file"
// while another process is creating the directory (seen on macOS in CI).
func (j *Journal) openLockFile() (*os.File, error) {
	root, err := j.openRoot()
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()

	name := filepath.Join(localName, lockName)
	var missing error
	for attempt := range lockOpenAttempts {
		f, err := root.OpenFile(name, os.O_RDWR|os.O_CREATE, 0o600)
		if err == nil {
			return f, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("journal: %w", err)
		}
		missing = err
		if err := root.MkdirAll(localName, 0o750); err != nil {
			return nil, fmt.Errorf("journal: %w", err)
		}
		time.Sleep(time.Duration(attempt) * time.Millisecond)
	}
	return nil, fmt.Errorf("journal: %w", missing)
}

// lock takes the write lock and returns the function that gives it back.
//
// It is an operating system lock (flock, LockFileEx) on a file that is never
// removed. If the process dies, even by being killed, the system releases the
// lock. Nothing is left behind that has to be cleaned up by hand.
//
// It waits up to the lock timeout. The lock is not re-entrant: a second call
// from the code that holds it waits until the timeout.
//
// The lock is not fair. A waiter gets it when it happens to try while it is
// free, so a stream of writers that take it again the moment they let go can
// keep a waiter out until the timeout. That does not happen with the way mtqg is
// used (short commands, with pauses between them); the tests that write in a
// tight loop pause between writes for the same reason.
func (j *Journal) lock() (release func(), err error) {
	path := j.lockPath()
	f, err := j.openLockFile()
	if err != nil {
		return nil, err
	}

	start := time.Now()
	wait := lockFirstWait
	for {
		ok, err := tryLock(f)
		if err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("journal: lock %s: %w", path, err)
		}
		if ok {
			return func() {
				unlock(f)
				_ = f.Close()
			}, nil
		}
		waited := time.Since(start)
		if waited >= j.opts.LockTimeout {
			_ = f.Close()
			return nil, &LockTimeoutError{Path: path, Waited: waited}
		}
		time.Sleep(min(wait, j.opts.LockTimeout-waited))
		wait = min(wait*2, lockMaxWait)
	}
}
