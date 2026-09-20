//go:build unix

package journal

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// tryLock takes an exclusive lock on f without waiting. ok is false when
// another holder has it. flock cannot be interrupted while it waits, so waiting
// is done by the caller, by trying again.
func tryLock(f *os.File) (ok bool, err error) {
	rc, err := f.SyscallConn()
	if err != nil {
		return false, err
	}
	var lockErr error
	if err := rc.Control(func(fd uintptr) {
		for {
			lockErr = unix.Flock(int(fd), unix.LOCK_EX|unix.LOCK_NB)
			if !errors.Is(lockErr, unix.EINTR) {
				return
			}
		}
	}); err != nil {
		return false, err
	}
	switch {
	case lockErr == nil:
		return true, nil
	case errors.Is(lockErr, unix.EWOULDBLOCK):
		return false, nil
	default:
		return false, lockErr
	}
}

// unlock gives the lock back. Closing f would do the same; a failure here has
// no consequence.
func unlock(f *os.File) {
	rc, err := f.SyscallConn()
	if err != nil {
		return
	}
	_ = rc.Control(func(fd uintptr) {
		_ = unix.Flock(int(fd), unix.LOCK_UN)
	})
}
