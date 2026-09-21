//go:build windows

package cli

import (
	"os"

	"golang.org/x/sys/windows"
)

// enableANSI asks the console to understand color codes. Windows terminals
// before Windows 10 cannot, and then there is no color.
func enableANSI() bool {
	handle := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return false
	}
	if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 {
		return true
	}
	return windows.SetConsoleMode(handle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) == nil
}
