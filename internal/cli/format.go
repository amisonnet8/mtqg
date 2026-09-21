package cli

import (
	"io"
	"strings"

	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

// runFormat picks event lines out of any text, from a file or standard input, and
// prints them as one table in time order. It reads no .mtqg/: the lines can come
// from a diff, from the output of git show, from an archive.
func runFormat(c *ctx) int {
	if len(c.inv.words) > 1 {
		return c.usageFailure(msgTooManyArguments(c.inv.cmd.usage))
	}
	var data []byte
	var err error
	if len(c.inv.words) == 1 && c.inv.words[0] != "-" {
		data, err = c.env.ReadFile(c.inv.words[0])
		if err != nil {
			return c.fail(&failure{kindInput, msgCannotReadFile(c.inv.words[0], err)})
		}
	} else if data, err = io.ReadAll(c.env.Stdin); err != nil {
		return c.fail(&failure{kindInput, msgStdinFailed(err)})
	}

	all := eventsIn(string(data))
	events := make([]journal.Event, len(all))
	for i, f := range all {
		events[i] = f.event
	}
	order := model.EventOrder(events)

	// What is shown is what the text is about: the lines of a diff that it did not
	// leave unchanged. The others are only looked at for the text of a record.
	var shown []int
	for _, at := range order {
		if !all[at].context {
			shown = append(shown, at)
		}
	}

	if c.inv.json {
		out := jsonFormat{Command: c.inv.cmd.label(), Events: make([]jsonFormatEvent, len(shown)), Count: len(shown)}
		for i, at := range shown {
			out.Events[i] = jsonFormatEvent{Mark: all[at].mark, Event: all[at].event}
		}
		return c.emit(out)
	}

	// What the lines that create a record say, so that a change of state or a delete
	// can show what it was about, if its record is in the same text.
	created := make(map[string]journal.Event)
	for _, at := range order {
		if ev := events[at]; ev.Op == journal.OpCreate {
			if _, ok := created[ev.ID]; !ok {
				created[ev.ID] = ev
			}
		}
	}
	rows := make([]tableRow, len(shown))
	for i, at := range shown {
		ev := events[at]
		mark := ""
		if all[at].mark == "-" || (c.inv.mark && all[at].mark != "") {
			mark = all[at].mark
		}
		what, text := c.formatEvent(ev, created)
		rows[i] = tableRow{cells: []string{
			mark,
			formatFull(model.EventTime(ev), c.env.Location),
			what,
			shortID(ev.ID, c.inv.fullID),
			text,
			oneLine(ev.Author.Name),
		}}
	}
	c.printLines(formatTable(rows, 4, 3, c.listWidth(), c.st))
	return exitOK
}

// formatEvent says what an event did and shows its text. A line that creates a
// record says its kind; the others say done, reopen, edit or delete.
func (c *ctx) formatEvent(ev journal.Event, created map[string]journal.Event) (what, text string) {
	switch ev.Op {
	case journal.OpCreate:
		switch {
		case ev.Type == journal.TypeGlossary:
			return "glossary", oneLine(ev.Word) + ": " + oneLine(ev.Text)
		case ev.Re != "" && model.ReplyKind(ev.Type) != model.ParentKind(ev.Type):
			return model.ReplyKind(ev.Type), "(to " + shortID(ev.Re, c.inv.fullID) + ") " + oneLine(ev.Text)
		case ev.Type == journal.TypeQA || ev.Type == journal.TypeBug || ev.Type == journal.TypeMemo || ev.Type == journal.TypeTodo:
			return model.ParentKind(ev.Type), oneLine(ev.Text)
		default:
			return oneLine(ev.Type), oneLine(ev.Text)
		}
	case journal.OpStatus:
		what = "reopen"
		if ev.Status == journal.StatusDone {
			what = "done"
		}
		return what, oneLine(created[ev.ID].Text)
	case journal.OpEdit:
		return "edit", oneLine(ev.Text)
	default:
		return oneLine(ev.Op), oneLine(created[ev.ID].Text)
	}
}

// foundEvent is an event that was found in a text, with the mark of its line.
type foundEvent struct {
	event journal.Event
	mark  string // "+", "-" or ""

	// context is a line that a diff shows unchanged: not part of what the diff is
	// about. It is not shown, and only lends its text to the changes of its record.
	context bool
}

// eventsIn picks the event lines out of a text, in the order they are in it. A line
// counts if, after any leading +, - and spaces (the marks of a diff, of several
// diffs for a merge, and the space of a line that did not change), what follows is
// a JSON object that is an event. Anything else is skipped without a word: the
// header of a commit, the code of a diff.
func eventsIn(text string) []foundEvent {
	var (
		candidates strings.Builder
		found      []foundEvent // the marks; the events are filled in below
	)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSuffix(line, "\r")
		rest := strings.TrimLeft(line, "+- ")
		if !strings.HasPrefix(rest, "{") {
			continue
		}
		prefix := line[:len(line)-len(rest)]
		f := foundEvent{}
		switch {
		case strings.Contains(prefix, "-"):
			f.mark = "-"
		case strings.Contains(prefix, "+"):
			f.mark = "+"
		case prefix != "":
			f.context = true
		}
		candidates.WriteString(rest)
		candidates.WriteByte('\n')
		found = append(found, f)
	}

	// The lines are read by the same code as journal.jsonl. Every candidate is a
	// line of its own, so the number of a line says which it was.
	lines, _, _ := journal.Scan(strings.NewReader(candidates.String()))
	var events []foundEvent
	for _, l := range lines {
		if l.Event != nil {
			f := found[l.Number-1]
			f.event = *l.Event
			events = append(events, f)
		}
	}
	return events
}
