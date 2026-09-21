package cli

import (
	"errors"

	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

func runDone(c *ctx) int { return c.changeStatus(journal.StatusDone) }

func runReopen(c *ctx) int { return c.changeStatus(journal.StatusOpen) }

// changeStatus marks a todo as done or open again and prints one line saying
// what it did, so that the ID typed can be seen to be the one meant. If the todo
// is in that state already, nothing is written: repeating a change is not an
// error.
func (c *ctx) changeStatus(status string) int {
	verb := c.inv.cmd.name
	j, err := c.reader()
	if err != nil {
		return c.fail(err)
	}
	state, err := c.load(j)
	if err != nil {
		return c.fail(err)
	}
	rec, err := state.ResolveKind(c.inv.words[0], model.KindTodo)
	if err != nil {
		return c.fail(err)
	}
	id, text := shortID(rec.ID, c.inv.fullID), oneLine(rec.Text)

	ev, err := model.SetStatus(rec, status)
	if errors.Is(err, model.ErrNoChange) {
		c.println(msgAlreadyInState(verb, id, text))
		return exitOK
	}
	if err != nil {
		return c.fail(err)
	}

	w, err := c.writer()
	if err != nil {
		return c.fail(err)
	}
	if _, err := w.Append(ev); err != nil {
		return c.fail(err)
	}
	c.println(msgStatusChanged(verb, id, text))
	return exitOK
}
