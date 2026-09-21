//go:build windows

package cli

// terminalDevice is empty on Windows: a console has no device number that stays
// the same for one window, so lines written there have no tty, and count as one
// terminal (docs/reference/cli.md, undo). MTQG_TTY still works.
func terminalDevice() string { return "" }
