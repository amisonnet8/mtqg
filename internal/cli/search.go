package cli

import (
	"slices"
	"strings"
)

// runSearch lists the records whose text contains what was asked for, newest first,
// the way log lists them. A search that finds nothing has still worked.
func runSearch(c *ctx) int {
	query := strings.Join(c.inv.words, " ")
	if strings.TrimSpace(query) == "" {
		// A blank text would match everything; that is not a search.
		return c.usageFailure(msgMissingArgument(c.inv.cmd.usage))
	}
	j, err := c.reader()
	if err != nil {
		return c.fail(err)
	}
	state, err := c.load(j)
	if err != nil {
		return c.fail(err)
	}
	found := state.Search(query)
	slices.Reverse(found) // newest first

	if c.inv.json {
		return c.emit(jsonSearch{Command: c.inv.cmd.label(), Query: query, Records: recordsJSON(found), Count: len(found)})
	}
	c.printRecordLines(state, found)
	c.println(msgSearchFooter(len(found), oneLine(query)))
	return exitOK
}
