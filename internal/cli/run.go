package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

// ctx is one run of a command: what it was given and what it may use.
type ctx struct {
	env Env
	inv *invocation
	st  style
}

// failure is a complaint for people that needs no further work: its message is
// printed as it is, and the exit code is 1.
type failure struct{ msg string }

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
		var usage *usageError
		if errors.As(err, &usage) {
			line(env.Stderr, usage.msg)
			return exitUsage
		}
		line(env.Stderr, err)
		return exitError
	}

	c := &ctx{env: env, inv: inv, st: style{on: colorOn(env, inv)}}
	switch {
	case inv.help && inv.cmd.name != "help":
		return c.printCommandHelp()
	case inv.cmd.run == nil:
		line(env.Stderr, msgNotYet(inv.cmd.full()))
		return exitError
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
// say it is.
func (c *ctx) writer() (*journal.Journal, error) {
	start, err := c.startDir()
	if err != nil {
		return nil, err
	}
	loc, err := journal.Find(start)
	if err != nil {
		return nil, err
	}
	author, err := c.author(loc.Root)
	if err != nil {
		return nil, err
	}
	return journal.Open(loc.Root, journal.Options{Author: author})
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
			c.eprintln(msgMoreWarnings(len(warnings) - maxWarnings))
			return
		}
		c.eprintln(msgWarning(w))
	}
}

// fail prints what went wrong, in words, and returns the exit code.
func (c *ctx) fail(err error) int {
	for _, l := range c.describe(err) {
		c.eprintln(l)
	}
	return exitError
}

// describe turns an error of the core into the lines to show.
func (c *ctx) describe(err error) []string {
	var (
		failed     *failure
		notInit    *journal.NotInitializedError
		already    *journal.AlreadyInitializedError
		tooNew     *journal.FormatTooNewError
		lock       *journal.LockTimeoutError
		invalid    *journal.InvalidEventError
		ambiguous  *model.AmbiguousError
		wrongKind  *model.WrongKindError
		noState    *model.NoStateError
		notFound   *model.NotFoundError
		gitMissing *journal.GitUnavailableError
	)
	verb := ""
	if c.inv != nil && c.inv.cmd != nil {
		verb = c.inv.cmd.name
	}
	switch {
	case errors.As(err, &failed):
		return []string{failed.msg}
	case errors.As(err, &notInit):
		return []string{msgNotInitialized(notInit.Root)}
	case errors.Is(err, journal.ErrNotInRepository):
		return []string{msgNotInRepository()}
	case errors.As(err, &already):
		return []string{msgAlreadyInitialized(already.Path)}
	case errors.As(err, &tooNew):
		return []string{msgFormatTooNew(tooNew.Found, tooNew.Supported)}
	case errors.Is(err, journal.ErrConflictMarkers):
		return []string{msgConflictMarkers()}
	case errors.As(err, &lock):
		return []string{msgLockTimeout(lock.Waited.Round(100_000_000).String())}
	case errors.As(err, &invalid) && invalid.Field == "text":
		return []string{msgTextNotUTF8()}
	case errors.As(err, &ambiguous):
		return c.describeAmbiguous(ambiguous)
	case errors.As(err, &wrongKind):
		return []string{msgWrongKind(wrongKind, verb)}
	case errors.As(err, &noState):
		return []string{msgNoState(noState, verb)}
	case errors.As(err, &notFound):
		return []string{msgNotFound(notFound.Prefix)}
	case errors.As(err, &gitMissing):
		return []string{msgGitUnavailable(gitMissing.Err)}
	default:
		return []string{err.Error()}
	}
}

// describeAmbiguous lists the records an ID could mean, with their full IDs so
// that any length of them can be typed again.
func (c *ctx) describeAmbiguous(e *model.AmbiguousError) []string {
	lines := []string{msgAmbiguousHeader(e.Prefix, len(e.Candidates))}
	var idW, kindW, textW, authorW int
	for _, r := range e.Candidates {
		idW = max(idW, displayWidth(r.ID))
		kindW = max(kindW, displayWidth(r.Kind()))
		textW = max(textW, displayWidth(oneLine(r.Text)))
		authorW = max(authorW, displayWidth(oneLine(r.Author.Name)))
	}
	if w := c.env.StdoutWidth; w > 0 {
		fixed := 2 + idW + len(gap) + kindW + len(gap) + authorW + len(gap) + len("2006-01-02")
		textW = min(textW, max(w-1-fixed, minTextWidth))
	}
	now := c.env.Now()
	for _, r := range e.Candidates {
		lines = append(lines, "  "+
			padRight(r.ID, idW)+gap+padRight(r.Kind(), kindW)+gap+
			padRight(truncate(oneLine(r.Text), textW), textW)+gap+
			padRight(oneLine(r.Author.Name), authorW)+gap+
			formatTime(r.Created, now, c.env.Location))
	}
	return lines
}
