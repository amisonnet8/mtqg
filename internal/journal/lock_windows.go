//go:build windows

package journal

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// tryLock takes an exclusive lock on the first byte of f without waiting. ok is
// false when another holder has it.
func tryLock(f *os.File) (ok bool, err error) {
	rc, err := f.SyscallConn()
	if err != nil {
		return false, err
	}
	var lockErr error
	if err := rc.Control(func(fd uintptr) {
		lockErr = windows.LockFileEx(
			windows.Handle(fd),
			windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
			0, 1, 0, new(windows.Overlapped),
		)
	}); err != nil {
		return false, err
	}
	switch {
	case lockErr == nil:
		return true, nil
	case errors.Is(lockErr, windows.ERROR_LOCK_VIOLATION):
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
		_ = windows.UnlockFileEx(windows.Handle(fd), 0, 1, 0, new(windows.Overlapped))
	})
}
