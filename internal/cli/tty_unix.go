//go:build !windows

package cli

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// terminalDevice names the terminal that mtqg was started at: the device number
// of the first of standard input, output and error that is a terminal. Output
// may go to a pipe (mtqg t list | less) while the terminal is still the one that
// was typed at, so all three are tried. Empty when none of them is a terminal.
func terminalDevice() string {
	for _, f := range []*os.File{os.Stdin, os.Stdout, os.Stderr} {
		fd := int(f.Fd())
		if !term.IsTerminal(fd) {
			continue
		}
		var st unix.Stat_t
		if err := unix.Fstat(fd, &st); err != nil {
			continue
		}
		// The device number is a different type on Linux and macOS.
		return fmt.Sprintf("dev:%d", st.Rdev)
	}
	return ""
}
