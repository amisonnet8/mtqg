package cli

import (
	"slices"
	"strconv"

	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

// defaultLogLimit is how many records log shows when it is not told: enough to
// see what has been going on, and few enough to read.
const defaultLogLimit = 20

// runLog shows the newest records of every kind, one line each, newest first.
func runLog(c *ctx) int {
	if c.inv.events && !c.inv.json {
		return c.usageFailure(msgEventsNeedsJSON())
	}
	limit := defaultLogLimit
	if v, ok := c.inv.values["--limit"]; ok {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return c.usageFailure(msgBadLimit(v))
		}
		limit = n
	}
	typ := "" // every type
	if v, ok := c.inv.values["--kind"]; ok {
		k, found := findKind(v)
		if !found {
			return c.usageFailure(msgBadKind(v))
		}
		typ = k.typ
	}

	j, err := c.reader()
	if err != nil {
		return c.fail(err)
	}
	state, err := c.load(j)
	if err != nil {
		return c.fail(err)
	}

	source := state.All()
	before := "" // the full ID --before resolved to, if given
	if v, ok := c.inv.values["--before"]; ok {
		rec, err := state.Resolve(v)
		if err != nil {
			return c.fail(err)
		}
		source = state.Before(rec)
		before = rec.ID
	}

	var records []*model.Record
	for _, r := range source {
		if typ == "" || r.Type == typ {
			records = append(records, r)
		}
	}
	slices.Reverse(records) // newest first
	total := len(records)
	if limit > 0 && total > limit {
		records = records[:limit]
	}

	if c.inv.json {
		recs := recordsJSON(records)
		if c.inv.events {
			for i, r := range records {
				recs[i].Events = r.Events
			}
		}
		return c.emit(jsonLog{Command: c.inv.cmd.label(), Records: recs, Shown: len(records), Total: total, Before: before})
	}
	c.printRecordLines(state, records)
	beforeShown := ""
	if before != "" {
		beforeShown = shortID(before, c.inv.fullID)
	}
	c.println(msgLogFooter(len(records), total, beforeShown))
	return exitOK
}

// printRecordLines prints records of every kind one to a line: the time, the kind,
// the ID, the text, the author, and `done` for a finished one. log and search show
// their records the same way.
func (c *ctx) printRecordLines(state *model.State, records []*model.Record) {
	now := c.env.Now()
	rows := make([]tableRow, len(records))
	for i, r := range records {
		done := r.Status == journal.StatusDone
		tail := ""
		if done {
			tail = "done"
		}
		rows[i] = tableRow{
			cells: []string{
				formatTime(r.Created, now, c.env.Location),
				showKind(r),
				shortID(r.ID, c.inv.fullID),
				c.logText(r),
				oneLine(r.Author.Name),
				tail,
			},
			dim: recordDim(state, r),
		}
	}
	c.printLines(formatTable(rows, 3, 2, c.listWidth(), c.st))
}

// recordDim reports whether a record's line should be dimmed: a todo, a
// question or a bug by its own status, an answer or a reply by the status of
// the question or bug it belongs to (qa list and bug list dim the two
// together the same way; log and search read every kind on one line, so they
// need to look the status up instead of already having it at hand).
func recordDim(state *model.State, r *model.Record) bool {
	if r.HasState() {
		return r.Status == journal.StatusDone
	}
	if p := state.Parent(r); p != nil {
		return p.Status == journal.StatusDone
	}
	return false
}

// logText is the text of a record for the line of log: an answer or a reply says
// which question or bug it is for, and a glossary entry gives the word before its
// definition.
func (c *ctx) logText(r *model.Record) string {
	switch {
	case r.IsReply():
		return "(to " + shortID(r.Re, c.inv.fullID) + ") " + oneLine(r.Text)
	case r.Kind() == model.KindGlossary:
		return oneLine(r.Word) + ": " + oneLine(r.Text)
	default:
		return oneLine(r.Text)
	}
}
