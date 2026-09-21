package cli

import (
	"errors"

	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

// A question and its answers, and a bug and its replies, have one shape: a record
// that can be replied to and is open until it is closed, and the records that
// reply to it. The commands of the kinds qa and bug are the same code, which finds
// out what it works on from the kind of its command (kindSpec).

// runAddThread adds a question or a bug, or an answer or a reply to one. The first
// word decides, and only the first word: a reply starts with the ID of its parent,
// which is 4 or more hex digits and nothing else. Nothing else about the words can
// tell the two apart, because the text is whatever is left, joined with spaces.
//
// A first word that looks like an ID and names no record is an error and not a new
// record: a mistyped ID must not turn into a new question without a word, and a
// text that really starts with such a word (Face detection is slow. Why?) is
// written with quotes, which make the first word contain a space.
func runAddThread(c *ctx) int {
	typ := c.inv.cmd.spec().typ
	words := c.inv.words
	if len(words) == 0 || !model.IsIDLike(words[0]) {
		return c.add(func(text string) (journal.Event, error) { return model.ParentCreate(typ, text) })
	}
	if len(words) < 2 {
		// An ID and no reply. Nothing to write, and the editor is not opened for an
		// answer: only "mtqg qa add" alone opens it, for a question.
		return c.usageFailure(msgMissingArgument(c.inv.cmd.usage))
	}

	// Opening first means that a missing .mtqg/ or author is reported before the
	// journal is read.
	j, err := c.writer()
	if err != nil {
		return c.fail(err)
	}
	state, err := c.load(j)
	if err != nil {
		return c.fail(err)
	}
	parent, err := state.ResolveKind(words[0], model.ParentKind(typ))
	var notFound *model.NotFoundError
	if errors.As(err, &notFound) {
		for _, l := range msgNoParentToReplyTo(typ, c.inv.cmd.spec().name, words[0]) {
			c.eprintln(l)
		}
		return exitError
	}
	if err != nil {
		return c.fail(err)
	}
	text, err := c.inputText(words[1:])
	if err != nil {
		return c.fail(err)
	}
	ev, err := model.ReplyCreate(parent, text)
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

// runListThread lists the questions or the bugs that are open, or all of them with
// --all: each with its state, and under it the latest answer or reply.
func runListThread(c *ctx) int {
	typ := c.inv.cmd.spec().typ
	j, err := c.reader()
	if err != nil {
		return c.fail(err)
	}
	state, err := c.load(j)
	if err != nil {
		return c.fail(err)
	}
	now := c.env.Now()
	var rows []listRow
	for _, p := range state.Parents(typ, c.inv.all) {
		replies := state.Replies(p.ID)
		done := p.Status == journal.StatusDone
		rows = append(rows, listRow{
			ID:     shortID(p.ID, c.inv.fullID),
			Text:   oneLine(p.Text),
			Author: oneLine(p.Author.Name),
			When:   formatTime(p.Created, now, c.env.Location),
			Tail:   msgThreadState(typ, len(replies), done),
			Done:   done,
		})
		if len(replies) > 0 {
			latest := replies[len(replies)-1]
			rows = append(rows, listRow{
				Text:   oneLine(latest.Text),
				Author: oneLine(latest.Author.Name),
				When:   formatTime(latest.Created, now, c.env.Location),
				Done:   done,
				Reply:  true,
			})
		}
	}
	c.printLines(formatList(rows, c.listWidth(), c.st))
	open := len(state.Parents(typ, false))
	c.println(msgOpenFooter(open, len(state.Parents(typ, true))-open, c.inv.all))
	return exitOK
}
