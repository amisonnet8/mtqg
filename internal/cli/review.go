package cli

import (
	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

// runReview shows the places where the journal holds facts that do not agree:
// status changes that were made without knowing of each other, words that are
// defined more than once, and answers or replies that belong to no question or bug.
// It only shows them. The exit code is 0 whatever it finds.
func runReview(c *ctx) int {
	j, err := c.reader()
	if err != nil {
		return c.fail(err)
	}
	state, err := c.load(j)
	if err != nil {
		return c.fail(err)
	}
	conflicts := state.ConcurrentStatusChanges()
	duplicates := state.DuplicateWords()
	unattached := state.UnattachedReplies()

	if c.inv.json {
		out := jsonReview{
			Command:                 c.inv.cmd.label(),
			ConcurrentStatusChanges: make([]jsonStatusConflict, len(conflicts)),
			DuplicateWords:          make([]jsonDuplicateWord, len(duplicates)),
			UnattachedReplies:       make([]jsonUnattached, len(unattached)),
		}
		for i, k := range conflicts {
			events := make([]journal.Event, len(k.Changes))
			for n, e := range k.Changes {
				events[n] = e.Event
			}
			out.ConcurrentStatusChanges[i] = jsonStatusConflict{Record: recordJSON(k.Record), Changes: events}
		}
		for i, group := range duplicates {
			out.DuplicateWords[i] = jsonDuplicateWord{Word: group[0].Word, Records: recordsJSON(group)}
		}
		for i, r := range unattached {
			out.UnattachedReplies[i] = jsonUnattached{Record: recordJSON(r)}
			if target := state.Record(r.Re); target != nil {
				t := recordJSON(target)
				out.UnattachedReplies[i].ReRecord = &t
			}
		}
		return c.emit(out)
	}

	if len(conflicts) == 0 && len(duplicates) == 0 && len(unattached) == 0 {
		c.println(msgReviewNothing())
		return exitOK
	}
	first := true
	section := func(heading string) {
		if !first {
			c.println()
		}
		first = false
		c.println(heading)
	}
	if len(conflicts) > 0 {
		section(msgReviewConcurrent(len(conflicts)))
		c.printConcurrent(conflicts)
	}
	if len(duplicates) > 0 {
		section(msgReviewDuplicates(len(duplicates)))
		c.printDuplicates(duplicates)
	}
	if len(unattached) > 0 {
		section(msgReviewUnattached(len(unattached)))
		c.printUnattached(state, unattached)
	}
	return exitOK
}

// indented is the width that a table has when it starts n columns in from the left
// of a terminal, or 0 for a pipe or a file, where nothing is cut.
func (c *ctx) indented(n int) int {
	if w := c.listWidth(); w > 0 {
		return max(w-n, 1)
	}
	return 0
}

// printIndented prints lines with n spaces in front of each.
func (c *ctx) printIndented(n int, lines []string) {
	pad := make([]byte, n)
	for i := range pad {
		pad[i] = ' '
	}
	for _, l := range lines {
		c.println(string(pad) + l)
	}
}

// printConcurrent names each record and lists all of its status changes and
// edits, oldest first, with the time, the author and the change.
func (c *ctx) printConcurrent(conflicts []model.StatusConflict) {
	for _, k := range conflicts {
		r := k.Record
		c.println(msgReviewRecord(r.Kind(), shortID(r.ID, c.inv.fullID), oneLine(r.Text)))
		rows := make([]tableRow, len(k.Changes))
		for i, e := range k.Changes {
			change := "edited"
			if e.Event.Op == journal.OpStatus {
				from := oneLine(e.Event.From)
				if from == "" {
					from = "?"
				}
				change = from + " -> " + oneLine(e.Event.Status)
			}
			rows[i] = tableRow{cells: []string{
				formatFull(e.At, c.env.Location),
				oneLine(e.Event.Author.Name),
				change,
			}}
		}
		c.printIndented(4, formatTable(rows, 2, -1, 0, c.st))
	}
}

// printDuplicates lists each word that is defined more than once, with all of its
// entries in the order they were written.
func (c *ctx) printDuplicates(groups [][]*model.Record) {
	for _, group := range groups {
		c.println("  " + oneLine(group[0].Word))
		rows := make([]tableRow, len(group))
		for i, r := range group {
			rows[i] = tableRow{cells: []string{shortID(r.ID, c.inv.fullID), oneLine(r.Author.Name), oneLine(r.Text)}}
		}
		c.printIndented(4, formatTable(rows, 2, 0, c.indented(4), c.st))
	}
}

// printUnattached lists the answers and replies that belong to no question or bug,
// each with what its re names.
func (c *ctx) printUnattached(state *model.State, records []*model.Record) {
	rows := make([]tableRow, len(records))
	for i, r := range records {
		rows[i] = tableRow{cells: []string{shortID(r.ID, c.inv.fullID), showKind(r), oneLine(r.Author.Name), oneLine(r.Text)}}
	}
	lines := formatTable(rows, 3, 0, c.indented(2), c.st)
	for i, r := range records {
		c.println("  " + lines[i])
		what := "not in the journal"
		if target := state.Record(r.Re); target != nil {
			what = article(target.Kind()) + ", not " + article(model.ParentKind(r.Type))
		}
		c.println(msgReviewRe(shortID(r.Re, c.inv.fullID), what))
	}
}
