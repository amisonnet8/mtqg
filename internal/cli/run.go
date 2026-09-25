package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

// ctx is one run of a command: what it was given and what it may use.
type ctx struct {
	env Env
	inv *invocation
	st  style

	// authorAs, when set, is who writerAs writes as, instead of working it out
	// from MTQG_AUTHOR_* and git (author.go). Only the MCP server sets this
	// (mcp.go): its author always comes from the connecting client, never from
	// the shell's environment or from git.
	authorAs *journal.Author
}

// failure is a complaint for people that needs no further work: its message is
// printed as it is, and the exit code is 1. kind is what --json says it is.
type failure struct{ kind, msg string }

func (f *failure) Error() string { return f.msg }

// println writes a line to standard output, and eprintln to standard error. A
// write that fails, as when the reader of a pipe has gone (mtqg t list | head),
// is not a failure of mtqg: there is nobody left to tell.
func (c *ctx) println(a ...any)  { line(c.env.Stdout, a...) }
func (c *ctx) eprintln(a ...any) { line(c.env.Stderr, a...) }

func line(w io.Writer, a ...any) { _, _ = fmt.Fprintln(w, a...) }

// The exit codes (docs/reference/cli.md).
const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

// Run runs mtqg with the given arguments (without the name of the program) and
// returns the exit code.
func Run(env Env, args []string) int {
	inv, err := parseArgs(args)
	if err != nil {
		// The command line could not be read, so whether --json was asked for is
		// found by looking for it.
		c := &ctx{env: env, inv: &invocation{json: wantsJSON(args)}}
		var usage *usageError
		if errors.As(err, &usage) {
			return c.usageFailure(usage.msg)
		}
		return c.fail(err)
	}

	c := &ctx{env: env, inv: inv, st: style{on: colorOn(env, inv)}}
	switch {
	case inv.help && inv.kindHelp != "":
		return c.printKindHelp()
	case inv.help && inv.cmd.name != "help":
		return c.printCommandHelp()
	case inv.cmd.run == nil:
		return c.failWith(errorReport{kind: kindNotAvailable, lines: []string{msgNotYet(inv.cmd.full())}})
	}
	return inv.cmd.run(c)
}

// colorOn decides whether to color: on a terminal that understands it, unless
// NO_COLOR is set (and not empty) or --no-color was given.
func colorOn(env Env, inv *invocation) bool {
	return env.StdoutIsTerminal && env.ANSI && !inv.noColor &&
		env.Getenv("NO_COLOR") == "" && env.Getenv("TERM") != "dumb"
}

// startDir is where to look for .mtqg/ from: -C, or the current directory.
func (c *ctx) startDir() (string, error) {
	if c.inv.dir != "" {
		return c.inv.dir, nil
	}
	return c.env.Getwd()
}

// reader opens the journal for reading. No author is needed.
func (c *ctx) reader() (*journal.Journal, error) {
	start, err := c.startDir()
	if err != nil {
		return nil, err
	}
	return journal.Open(start, journal.Options{})
}

// writer opens the journal for writing, as the author that MTQG_AUTHOR_* and git
// say it is, from the terminal that the command was typed at.
func (c *ctx) writer() (*journal.Journal, error) {
	j, _, _, err := c.writerAs()
	return j, err
}

// writerAs is writer, and says who and where it writes as: undo looks for the
// lines of that author and that terminal.
func (c *ctx) writerAs() (j *journal.Journal, author journal.Author, tty string, err error) {
	start, err := c.startDir()
	if err != nil {
		return nil, journal.Author{}, "", err
	}
	loc, err := journal.Find(start)
	if err != nil {
		return nil, journal.Author{}, "", err
	}
	if c.authorAs != nil {
		author = *c.authorAs
	} else {
		author, err = c.author(loc.Root)
		if err != nil {
			return nil, journal.Author{}, "", err
		}
	}
	tty = c.terminalID()
	j, err = journal.Open(loc.Root, journal.Options{Author: author, TTY: tty})
	return j, author, tty, err
}

// terminalID is the tty of the lines this run writes: 8 hex digits of a hash of
// MTQG_TTY if that is set, else of the terminal the command was typed at, and
// empty if there is none. Only the hash is written, so that the device number and
// the text of the variable stay on this machine, and a record that is committed
// says no more than that two lines came from one terminal.
func (c *ctx) terminalID() string {
	raw := ""
	if v := strings.TrimSpace(c.env.Getenv("MTQG_TTY")); v != "" {
		raw = "env:" + v
	} else {
		raw = c.env.TTY
	}
	if raw == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:4])
}

// load reads the journal and builds the state of the records. Lines that could
// not be read are reported as warnings on standard error and skipped.
func (c *ctx) load(j *journal.Journal) (*model.State, error) {
	result, err := j.Read()
	if err != nil {
		return nil, err
	}
	c.warn(result.Warnings)
	return model.Build(result.Events), nil
}

// warn prints what was skipped while reading, on standard error so that standard
// output stays what another tool expects. The first few are named, and the rest
// counted.
func (c *ctx) warn(warnings []journal.Warning) {
	for i, w := range warnings {
		if i == maxWarnings {
			c.warning(warningReport{kind: "more", count: len(warnings) - maxWarnings, message: msgMoreWarnings(len(warnings) - maxWarnings)})
			return
		}
		c.warning(warningReport{kind: warningKind(w.Kind), line: w.Line, message: msgWarning(w)})
	}
}

// usageFailure prints a mistake in the command line that only the command itself
// could see (it depends on what the words mean), and returns the exit code for
// one.
func (c *ctx) usageFailure(msg string) int {
	c.printError(errorReport{kind: kindUsage, lines: []string{msg}})
	return exitUsage
}
