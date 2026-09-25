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
	open := len(state.Todos(false))
	done := len(state.Todos(true)) - open
	if c.inv.json {
		return c.emit(jsonTodoList{Command: c.inv.cmd.label(), Records: recordsJSON(todos), Open: open, Done: done})
	}
	c.printRows(todos)
	c.println(msgOpenFooter(open, done, all))
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
	if c.inv.json {
		return c.emit(jsonMemoList{Command: c.inv.cmd.label(), Records: recordsJSON(memos), Count: len(memos)})
	}
	c.printRows(memos)
	c.println(msgMemoFooter(len(memos)))
	return exitOK
}

func runListRules(c *ctx) int {
	j, err := c.reader()
	if err != nil {
		return c.fail(err)
	}
	state, err := c.load(j)
	if err != nil {
		return c.fail(err)
	}
	rules := state.Rules()
	if c.inv.json {
		return c.emit(jsonMemoList{Command: c.inv.cmd.label(), Records: recordsJSON(rules), Count: len(rules)})
	}
	c.printRows(rules)
	c.println(msgRuleFooter(len(rules)))
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
