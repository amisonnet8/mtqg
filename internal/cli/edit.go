package cli

import (
	"errors"
	"slices"

	"github.com/amisonnet8/mtqg/internal/model"
)

// runEdit replaces the text of a record. The words after the ID are the new text;
// with none, $EDITOR opens on the text the record has now. A text that is the
// same as it was is not written.
func runEdit(c *ctx) int {
	// Opening first means that a missing .mtqg/ or author is reported before the
	// editor is opened.
	j, err := c.writer()
	if err != nil {
		return c.fail(err)
	}
	state, err := c.load(j)
	if err != nil {
		return c.fail(err)
	}
	rec, err := state.Resolve(c.inv.words[0])
	if err != nil {
		return c.fail(err)
	}
	text, err := c.inputTextFrom(c.inv.words[1:], rec.Text)
	if err != nil {
		return c.fail(err)
	}
	id := shortID(rec.ID, c.inv.fullID)

	ev, err := model.EditText(rec, text)
	if errors.Is(err, model.ErrNoChange) {
		if c.inv.json {
			return c.emit(jsonChangeResult{Command: c.inv.cmd.label(), Record: recordJSON(rec), Changed: false})
		}
		c.println(msgUnchanged(id, oneLine(rec.Text)))
		return exitOK
	}
	if err != nil {
		return c.fail(err)
	}
	written, err := j.Append(ev)
	if err != nil {
		return c.fail(err)
	}
	if c.inv.json {
		// The record with the edit in it: its own events and the one just written.
		current := model.Build(append(slices.Clone(rec.Events), written)).Record(rec.ID)
		return c.emit(jsonChangeResult{Command: c.inv.cmd.label(), Record: recordJSON(current), Changed: true})
	}
	c.println(msgEdited(id, oneLine(text)))
	return exitOK
}

// runDelete hides a record, and says what else went with it: a question or a bug
// takes its answers or replies along.
func runDelete(c *ctx) int {
	j, err := c.writer()
	if err != nil {
		return c.fail(err)
	}
	state, err := c.load(j)
	if err != nil {
		return c.fail(err)
	}
	rec, err := state.Resolve(c.inv.words[0])
	if err != nil {
		return c.fail(err)
	}
	ev, err := model.Delete(rec)
	if err != nil {
		return c.fail(err)
	}
	hidden := state.Replies(rec.ID) // in view now, and gone after the delete
	if _, err := j.Append(ev); err != nil {
		return c.fail(err)
	}
	if c.inv.json {
		return c.emit(jsonDelete{Command: c.inv.cmd.label(), Record: recordJSON(rec), HiddenReplies: recordsJSON(hidden)})
	}
	c.println(msgDeleted(shortID(rec.ID, c.inv.fullID), oneLine(rec.Text)))
	if len(hidden) > 0 {
		c.println(msgAlsoHidden(rec.Type, len(hidden), authorsOf(hidden)))
	}
	c.println(msgDeleteNote())
	return exitOK
}

// authorsOf is the names of the authors of the records, each once, in the order
// they first appear, made safe to show.
func authorsOf(records []*model.Record) []string {
	var names []string
	for _, r := range records {
		if name := oneLine(r.Author.Name); !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	return names
}
