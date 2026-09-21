package cli

import (
	"errors"

	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

// runAddQA adds a question, or an answer to one. The first word decides, and only
// the first word: an answer starts with the ID of its question, which is 4 or more
// hex digits and nothing else. Nothing else about the words can tell the two
// apart, because the text is whatever is left, joined with spaces.
//
// A first word that looks like an ID and names no record is an error and not a
// question: a mistyped ID must not turn into a new question without a word, and a
// question that really starts with such a word (Face detection is slow. Why?) is
// written with quotes, which make the first word contain a space.
func runAddQA(c *ctx) int {
	words := c.inv.words
	if len(words) == 0 || !model.IsIDLike(words[0]) {
		return c.add(model.QuestionCreate)
	}
	if len(words) < 2 {
		// An ID and no answer. Nothing to write, and the editor is not opened for an
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
	question, err := state.ResolveKind(words[0], model.KindQuestion)
	var notFound *model.NotFoundError
	if errors.As(err, &notFound) {
		for _, l := range msgNoQuestionToAnswer(words[0]) {
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
	ev, err := model.AnswerCreate(question, text)
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

// runListQA lists the questions that are open, or all of them with --all: each
// with its state, and under it the latest answer.
func runListQA(c *ctx) int {
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
	for _, q := range state.Questions(c.inv.all) {
		answers := state.Answers(q.ID)
		done := q.Status == journal.StatusDone
		rows = append(rows, listRow{
			ID:     shortID(q.ID, c.inv.fullID),
			Text:   oneLine(q.Text),
			Author: oneLine(q.Author.Name),
			When:   formatTime(q.Created, now, c.env.Location),
			Tail:   msgQuestionState(len(answers), done),
			Done:   done,
		})
		if len(answers) > 0 {
			latest := answers[len(answers)-1]
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
	open := len(state.Questions(false))
	c.println(msgOpenFooter(open, len(state.Questions(true))-open, c.inv.all))
	return exitOK
}
