package cli

import "github.com/amisonnet8/mtqg"

// runUpgrade raises the repository's format version to the one this build of
// mtqg supports (journal.SupportedVersion), rewriting only .mtqg/version and
// .mtqg/SCHEMA.md. Existing lines of journal.jsonl are never touched: a line's
// own v keeps meaning what it meant when it was written
// (docs/reference/schema.md "Versioning"). It needs no author, the same as
// archive: it writes no event.
func runUpgrade(c *ctx) int {
	j, err := c.reader()
	if err != nil {
		return c.fail(err)
	}
	from, to, err := j.Upgrade(mtqg.Schema, c.inv.dryRun)
	if err != nil {
		return c.fail(err)
	}

	if c.inv.json {
		return c.emit(jsonUpgrade{Command: c.inv.cmd.label(), From: from, To: to, DryRun: c.inv.dryRun})
	}
	if from == to {
		c.println(msgUpgradeUnchanged(to))
		return exitOK
	}
	c.println(msgUpgraded(from, to, c.inv.dryRun))
	return exitOK
}
