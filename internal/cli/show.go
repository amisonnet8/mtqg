package cli

import (
	"strings"
	"time"

	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

// runShow shows one record in full: its text, whatever the width of the window,
// its answers if it is a question, and what happened to it.
func runShow(c *ctx) int {
	j, err := c.reader()
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
	if c.inv.json {
		history := state.History(rec)
		events := make([]journal.Event, len(history))
		for i, e := range history {
			events[i] = e.Event
		}
		record := recordJSON(rec)
		if rec.CanHaveReplies() {
			record = threadJSON(state, rec)
		}
		return c.emit(jsonShow{Command: c.inv.cmd.label(), Record: record, Events: events})
	}
	c.printRecord(state, rec)
	return exitOK
}

func (c *ctx) printRecord(state *model.State, rec *model.Record) {
	loc := c.env.Location
	id := func(full string) string { return shortID(full, c.inv.fullID) }
	who := func(a journal.Author) string { return oneLine(a.Name) + " (" + oneLine(a.Kind) + ")" }

	header := c.st.id(id(rec.ID))
	head := showKind(rec) + "  " + header
	if rec.HasState() {
		head += "  " + rec.Status
	}
	c.println(head)
	c.println(msgShowBy(who(rec.Author), formatFull(rec.Created, loc)))
	if rec.IsReply() {
		// What it is for is named only if the record it points at is its parent: a
		// re that names a record of another type is not a reply to it.
		parent := ""
		if state.HasParent(rec) {
			parent = oneLine(state.Record(rec.Re).Text)
		}
		c.println(msgShowToParent(model.ParentKind(rec.Type), id(rec.Re), parent))
	}
	c.println()

	if rec.Kind() == model.KindGlossary {
		c.println(msgShowWord(oneLine(rec.Word)))
		c.println()
	}
	c.printBody("  ", rec.Text)

	if rec.CanHaveReplies() {
		if replies := state.Replies(rec.ID); len(replies) > 0 {
			c.println()
			c.println(msgShowReplyCount(rec.Type, len(replies)))
			var idW, whoW int
			for _, a := range replies {
				idW = max(idW, displayWidth(id(a.ID)))
				whoW = max(whoW, displayWidth(who(a.Author)))
			}
			for _, a := range replies {
				c.println("  " + c.st.id(padRight(id(a.ID), idW)) + gap + padRight(who(a.Author), whoW) + gap + formatFull(a.Created, loc))
				c.printBody("    ", a.Text)
			}
		}
	}

	history := state.History(rec)
	c.println()
	c.println(msgShowEvents)
	var opW, whoW int
	for _, e := range history {
		opW = max(opW, displayWidth(oneLine(e.Event.Op)))
		whoW = max(whoW, displayWidth(who(e.Event.Author)))
	}
	for _, e := range history {
		line := "  " + formatFull(e.At, loc) + gap + padRight(oneLine(e.Event.Op), opW) + gap + padRight(who(e.Event.Author), whoW)
		if detail := eventDetail(e, id); detail != "" {
			line += gap + detail
		}
		c.println(strings.TrimRight(line, " "))
	}
}

// showKind is the kind of a record as a header names it.
func showKind(r *model.Record) string {
	if r.Kind() == model.KindGlossary {
		return journal.TypeGlossary
	}
	return r.Kind()
}

// eventDetail says what an event did beyond its name: the change of state, or
// the answer that a question's history is about.
func eventDetail(e model.Entry, id func(string) string) string {
	switch {
	case e.Answer != nil:
		return msgShowReplyEvent(e.Answer.Kind(), id(e.Answer.ID))
	case e.Event.Op == journal.OpStatus:
		from := oneLine(e.Event.From)
		if from == "" {
			from = "?"
		}
		return from + " -> " + oneLine(e.Event.Status)
	default:
		return ""
	}
}

// printBody prints a text in full, every line of it, each indented and made safe
// to show.
func (c *ctx) printBody(indent, text string) {
	for _, l := range bodyLines(text) {
		if l == "" {
			c.println()
			continue
		}
		c.println(indent + l)
	}
}

// formatFull shows a time as a date and a time, in the given place: the history
// of a record spans days, and a bare time would not say which one. A time that
// could not be read is "-".
func formatFull(t time.Time, loc *time.Location) string {
	if t.IsZero() {
		return "-"
	}
	return t.In(loc).Format("2006-01-02 15:04")
}

// bodyLines splits a text into its lines, made safe to show. A line ending of a
// file from Windows (CR LF) is one line ending, not a control character.
func bodyLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.Split(sanitize(text), "\n")
}
