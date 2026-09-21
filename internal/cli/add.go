package cli

import (
	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

func runAddMemo(c *ctx) int { return c.add(model.MemoCreate) }

func runAddTodo(c *ctx) int { return c.add(model.TodoCreate) }

// add records a memo or a todo and prints its ID, and nothing else: the speed of
// writing a note down comes first. It does not read the journal.
func (c *ctx) add(create func(text string) (journal.Event, error)) int {
	// Opening first means that a missing .mtqg/ or author is reported before an
	// editor is opened.
	j, err := c.writer()
	if err != nil {
		return c.fail(err)
	}
	text, err := c.inputText(c.inv.words)
	if err != nil {
		return c.fail(err)
	}
	ev, err := create(text)
	if err != nil {
		return c.fail(err)
	}
	written, err := j.Append(ev)
	if err != nil {
		return c.fail(err)
	}
	return c.printWritten(written)
}

// printWritten says what was written: the ID of the new record and nothing else,
// so that writing stays quick; with --json, the record. It does not read the
// journal: the record is what the one event that was written makes.
func (c *ctx) printWritten(written journal.Event) int {
	if !c.inv.json {
		c.println(shortID(written.ID, c.inv.fullID))
		return exitOK
	}
	rec := model.Build([]journal.Event{written}).Record(written.ID)
	return c.emit(jsonRecordResult{Command: c.inv.cmd.label(), Record: recordJSON(rec)})
}
