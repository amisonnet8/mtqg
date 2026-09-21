package cli

import (
	"errors"
	"strconv"

	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

// runStatus shows what is open and how many records are not committed. The line
// for conflicts comes with the command that finds them (review).
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

	// How many records are not committed, or -1 when git cannot say.
	uncommitted := -1
	events, err := j.UncommittedEvents()
	switch {
	case err == nil:
		uncommitted = model.UncommittedRecords(events)
	case errors.Is(err, journal.ErrGitUnavailable):
		c.warning(warningReport{kind: kindGitUnavailable, message: "warning: " + msgGitUnavailable(errors.Unwrap(err))})
	default:
		return c.fail(err)
	}

	if c.inv.json {
		out := jsonStatus{
			Command:                       c.inv.cmd.label(),
			OpenTodos:                     summary.OpenTodos,
			OpenQuestions:                 summary.OpenQuestions,
			QuestionsAwaitingConfirmation: summary.QuestionsAwaitingConfirmation,
			OpenBugs:                      summary.OpenBugs,
			BugsAwaitingConfirmation:      summary.BugsAwaitingConfirmation,
			GlossaryEntries:               summary.GlossaryEntries,
			DuplicateWords:                summary.DuplicateWords,
		}
		if uncommitted >= 0 {
			out.UncommittedRecords = &uncommitted
		}
		return c.emit(out)
	}

	c.println(msgStatusLine("Open todos", strconv.Itoa(summary.OpenTodos)))
	c.println(msgStatusLine("Open questions", msgOpenValue(summary.OpenQuestions, summary.QuestionsAwaitingConfirmation)))
	c.println(msgStatusLine("Open bugs", msgOpenValue(summary.OpenBugs, summary.BugsAwaitingConfirmation)))
	c.println(msgStatusLine("Glossary", msgGlossaryValue(summary.GlossaryEntries, summary.DuplicateWords)))
	c.println()

	shown := "unknown"
	if uncommitted >= 0 {
		shown = strconv.Itoa(uncommitted)
	}
	c.println(msgStatusLine("Uncommitted records", shown))
	return exitOK
}
