//go:build windows

package journal

import (
	"errors"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

// On Windows a file that another process has open cannot be replaced. A reader
// has it open only while it reads, so a replacement that is refused is tried
// again for a short while.
const (
	replaceWindow   = 3 * time.Second
	replaceFirstGap = 5 * time.Millisecond
	replaceMaxGap   = 100 * time.Millisecond
)

// replaceFile puts oldname in place of newname. It is not atomic on Windows in
// the way a rename is elsewhere, but the old file is replaced by a single call.
func replaceFile(root *os.Root, oldname, newname string) error {
	start := time.Now()
	gap := replaceFirstGap
	for {
		err := root.Rename(oldname, newname)
		if err == nil || !isBusy(err) || time.Since(start) > replaceWindow {
			return err
		}
		time.Sleep(gap)
		gap = min(gap*2, replaceMaxGap)
	}
}

// isBusy reports whether err means that the file is open somewhere else.
func isBusy(err error) bool {
	return errors.Is(err, windows.ERROR_ACCESS_DENIED) || errors.Is(err, windows.ERROR_SHARING_VIOLATION)
}
