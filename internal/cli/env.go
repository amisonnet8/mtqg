package cli

import (
	"io"
	"time"
)

// Env is everything Run touches outside the arguments. cmd/mtqg fills it from
// the operating system (OSEnv); tests fill it with buffers and fixed values.
type Env struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	// Getenv returns an environment variable, empty when it is not set.
	Getenv func(string) string

	// Getwd returns the current directory.
	Getwd func() (string, error)

	// Now returns the current time. Location is where times are shown.
	Now      func() time.Time
	Location *time.Location

	// StdoutIsTerminal says that standard output is a terminal, not a pipe or a
	// file. Only then is text cut to the width of the window and color used.
	StdoutIsTerminal bool

	// StdoutWidth is the width of the terminal window in columns, or 0 when it is
	// not known.
	StdoutWidth int

	// ANSI says that the terminal understands color codes.
	ANSI bool

	// TTY names the terminal that the command was typed at, for undo to tell one
	// terminal from another: "dev:" and the device number of the first of standard
	// input, output and error that is a terminal. Empty when there is none, and
	// always on Windows. It is never written as it is: terminalID hashes it.
	TTY string

	// ReadFile reads a file. It is what format is given as an argument.
	ReadFile func(name string) ([]byte, error)

	// RunEditor runs an editor, given as the command and its arguments, with the
	// terminal attached, and returns when the editor has exited.
	RunEditor func(argv []string) error
}
