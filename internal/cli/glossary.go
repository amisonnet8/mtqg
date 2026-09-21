package cli

import (
	"github.com/amisonnet8/mtqg/internal/model"
)

// runAddGlossary defines a word. The first word of the command is the term, so a
// term of several words needs quotes; the rest is the definition. There are at
// least two of them (the table of commands sees to that), and the editor is never
// opened: it is only for a text that was left out altogether.
func runAddGlossary(c *ctx) int {
	words := c.inv.words
	j, err := c.writer()
	if err != nil {
		return c.fail(err)
	}
	text, err := c.inputText(words[1:])
	if err != nil {
		return c.fail(err)
	}
	ev, err := model.GlossaryCreate(words[0], text)
	if err != nil {
		return c.fail(err)
	}
	written, err := j.Append(ev)
	if err != nil {
		return c.fail(err)
	}
	c.println(shortID(written.ID, c.inv.fullID))
	return exitOK
}

// runListGlossary lists every entry, in the order they were written. Two entries
// for one word are two lines: mtqg does not choose between them.
func runListGlossary(c *ctx) int {
	j, err := c.reader()
	if err != nil {
		return c.fail(err)
	}
	state, err := c.load(j)
	if err != nil {
		return c.fail(err)
	}
	now := c.env.Now()
	entries := state.Glossary()
	rows := make([]listRow, len(entries))
	for i, r := range entries {
		rows[i] = listRow{
			ID:     shortID(r.ID, c.inv.fullID),
			Word:   oneLine(r.Word),
			Text:   oneLine(r.Text),
			Author: oneLine(r.Author.Name),
			When:   formatTime(r.Created, now, c.env.Location),
		}
	}
	c.printLines(formatList(rows, c.listWidth(), c.st))
	c.println(msgGlossaryFooter(len(entries), len(state.DuplicateWords())))
	return exitOK
}
