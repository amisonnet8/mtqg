package cli

import (
	"github.com/amisonnet8/mtqg"
	"github.com/amisonnet8/mtqg/internal/journal"
)

// runInit makes .mtqg/. It does not commit, and does not change the
// configuration of git: it only says that the records are to be committed.
func runInit(c *ctx) int {
	start, err := c.startDir()
	if err != nil {
		return c.fail(err)
	}
	loc, err := journal.Init(start, mtqg.Schema)
	if err != nil {
		return c.fail(err)
	}
	if c.inv.json {
		return c.emit(jsonInit{Command: c.inv.cmd.label(), Root: loc.Root})
	}
	for _, line := range msgInitialized(loc.Root) {
		c.println(line)
	}
	return exitOK
}
