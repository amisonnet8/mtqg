package cli

import (
	"fmt"
	"strings"
)

// runHelp lists the commands that are available, from the same table that reads
// the command line.
func runHelp(c *ctx) int {
	c.println("usage: mtqg [-C <path>] [--no-color] <kind> <verb> [<args>]")
	c.println("       mtqg [-C <path>] [--no-color] <command> [<args>]")
	c.println()

	names := make([]string, len(kinds))
	for i, k := range kinds {
		names[i] = fmt.Sprintf("%s (%s)", k.name, k.short)
	}
	c.println("Kinds (one letter is enough): " + strings.Join(names, ", "))
	c.println()

	c.println("Commands:")
	var usageW int
	for _, cmd := range commands {
		if cmd.run != nil {
			usageW = max(usageW, len(cmd.usage))
		}
	}
	for _, cmd := range commands {
		if cmd.run != nil {
			c.println("  " + padRight(cmd.usage, usageW) + "  " + cmd.summary)
		}
	}

	c.println()
	c.println("Options:")
	c.println("  -C <path>    Look for .mtqg/ from <path> instead of the current directory")
	c.println("  --all        In a list, include finished items")
	c.println("  --full-id    Show full 32-digit IDs instead of the first 10 digits")
	c.println("  --no-color   Do not use color (NO_COLOR is also honored)")
	c.println("  -h, --help   Show this help, or the usage of a command")
	return exitOK
}

// printCommandHelp shows what one command is and takes, for "mtqg <command> -h".
func (c *ctx) printCommandHelp() int {
	c.println("usage: " + c.inv.cmd.usage)
	c.println(c.inv.cmd.summary)
	return exitOK
}
