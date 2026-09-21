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
	c.println(msgOpenFooter(open, len(state.Todos(true))-open, all))
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
	c.printLines(formatList(rows, c.listWidth(), c.st))
}

// listWidth is the width to cut the text of a list to: the window, when the
// output is a terminal, and otherwise 0, which means never.
func (c *ctx) listWidth() int {
	if c.env.StdoutIsTerminal {
		return c.env.StdoutWidth
	}
	return 0
}

func (c *ctx) printLines(lines []string) {
	for _, line := range lines {
		c.println(line)
	}
}
