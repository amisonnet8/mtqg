package journal

import (
	"fmt"
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

// openLockFile opens the lock file, creating it and .local/ when needed. .local/
// is not committed, so a fresh clone does not have it.
func (j *Journal) openLockFile() (*os.File, error) {
	root, err := j.openRoot()
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	if err := root.MkdirAll(localName, 0o750); err != nil {
		return nil, fmt.Errorf("journal: %w", err)
	}
	f, err := root.OpenFile(filepath.Join(localName, lockName), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("journal: %w", err)
	}
	return f, nil
}

// lock takes the write lock and returns the function that gives it back.
//
// It is an operating system lock (flock, LockFileEx) on a file that is never
// removed. If the process dies, even by being killed, the system releases the
// lock. Nothing is left behind that has to be cleaned up by hand.
//
// It waits up to the lock timeout. The lock is not re-entrant: a second call
// from the code that holds it waits until the timeout.
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
