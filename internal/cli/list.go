package cli

import (
	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

func runListTodos(c *ctx) int {
	j, err := c.reader()
	if err != nil {
		return c.fail(err)
	}
	state, err := c.load(j)
	if err != nil {
		return c.fail(err)
	}
	all := c.inv.all
	todos := state.Todos(all)
	c.printRows(todos)
	open := len(state.Todos(false))
	c.println(msgTodoFooter(open, len(state.Todos(true))-open, all))
	return exitOK
}

func runListMemos(c *ctx) int {
	j, err := c.reader()
	if err != nil {
		return c.fail(err)
	}
	state, err := c.load(j)
	if err != nil {
		return c.fail(err)
	}
	memos := state.Memos()
	c.printRows(memos)
	c.println(msgMemoFooter(len(memos)))
	return exitOK
}

// printRows shows records as a list: ID, text, author, time. Only the first
// line of a text is shown, made safe to show, and cut to the window when the
// output is a terminal.
func (c *ctx) printRows(records []*model.Record) {
	now := c.env.Now()
	rows := make([]listRow, len(records))
	for i, r := range records {
		rows[i] = listRow{
			ID:     shortID(r.ID, c.inv.fullID),
			Text:   oneLine(r.Text),
			Author: oneLine(r.Author.Name),
			When:   formatTime(r.Created, now, c.env.Location),
			Done:   r.Status == journal.StatusDone,
		}
	}
	width := 0
	if c.env.StdoutIsTerminal {
		width = c.env.StdoutWidth
	}
	for _, line := range formatList(rows, width, c.st) {
		c.println(line)
	}
}
