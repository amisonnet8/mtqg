package cli

import (
	"errors"
	"strconv"

	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

// runStatus shows what is open and how many records are not committed. The lines
// for questions, glossary entries and conflicts come with the commands that make
// them.
func runStatus(c *ctx) int {
	j, err := c.reader()
	if err != nil {
		return c.fail(err)
	}
	state, err := c.load(j)
	if err != nil {
		return c.fail(err)
	}
	summary := state.Summary()

	c.println(msgStatusLine("Open todos", strconv.Itoa(summary.OpenTodos)))
	c.println()

	uncommitted := "unknown"
	events, err := j.UncommittedEvents()
	switch {
	case err == nil:
		uncommitted = strconv.Itoa(model.UncommittedRecords(events))
	case errors.Is(err, journal.ErrGitUnavailable):
		c.eprintln("warning: " + msgGitUnavailable(errors.Unwrap(err)))
	default:
		return c.fail(err)
	}
	c.println(msgStatusLine("Uncommitted records", uncommitted))
	return exitOK
}
