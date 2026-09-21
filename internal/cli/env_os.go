package cli

import (
	"os"
	"os/exec"
	"strconv"
	"time"

	"golang.org/x/term"
)

// OSEnv is the Env of the running program: the real streams, environment,
// clock and terminal.
func OSEnv() Env {
	fd := int(os.Stdout.Fd())
	isTerminal := term.IsTerminal(fd)
	cols := 0
	if isTerminal {
		if w, _, err := term.GetSize(fd); err == nil {
			cols = w
		} else if n, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil {
			cols = n
		}
	}
	return Env{
		Stdin:            os.Stdin,
		Stdout:           os.Stdout,
		Stderr:           os.Stderr,
		Getenv:           os.Getenv,
		Getwd:            os.Getwd,
		Now:              time.Now,
		Location:         time.Local,
		StdoutIsTerminal: isTerminal,
		StdoutWidth:      cols,
		ANSI:             isTerminal && enableANSI(),
		TTY:              terminalDevice(),
		ReadFile:         os.ReadFile,
		RunEditor:        runEditor,
	}
}

// runEditor runs the editor with the terminal of mtqg, and waits for it.
func runEditor(argv []string) error {
	cmd := exec.Command(argv[0], argv[1:]...) //nolint:gosec // G204: the editor is what the user set in $EDITOR; running it is the feature. It is not run through a shell.
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
