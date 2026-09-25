package cli

import (
	"errors"
	"slices"

	"github.com/amisonnet8/mtqg"
	"github.com/amisonnet8/mtqg/internal/journal"
)

// runInit makes .mtqg/, and with --agent also wires up that agent's hooks
// (§11.3 "mtqg init --agent <agent>"). It does not commit, and does not change
// the configuration of git: it only says that the records are to be committed.
//
// With --agent, an existing .mtqg/ is not an error (unlike plain "mtqg init"):
// the agent's configuration is still wired up, so the command can be run again
// safely once .mtqg/ already exists.
func runInit(c *ctx) int {
	agent := c.inv.values["--agent"]
	if agent != "" && !slices.Contains(initAgents, agent) {
		return c.usageFailure(msgUnknownAgent(agent, initAgents))
	}

	start, err := c.startDir()
	if err != nil {
		return c.fail(err)
	}
	loc, created, err := ensureMtqg(start, agent != "", c.inv.dryRun)
	if err != nil {
		return c.fail(err)
	}

	var files []agentFileResult
	if agent == "claude-code" {
		files, err = wireClaudeCode(loc.Root, c.inv.dryRun)
		if err != nil {
			return c.fail(err)
		}
	}

	if c.inv.json {
		return c.emit(jsonInit{Command: c.inv.cmd.label(), Root: loc.Root, Agent: agent, Files: agentFilesJSON(files)})
	}
	switch {
	case created && c.inv.dryRun:
		c.println(msgWouldCreateMtqg(loc.Root))
	case created:
		for _, line := range msgInitialized(loc.Root) {
			c.println(line)
		}
	case agent != "":
		c.println(msgAgentExisting(loc.Dir))
	}
	for _, f := range files {
		c.println(msgAgentFileResult(f.status, f.path, c.inv.dryRun))
	}
	return exitOK
}

// ensureMtqg makes .mtqg/ the way journal.Init does, or, with dryRun, only
// looks at whether it is there already: nothing is written either way when
// dryRun is true.
//
// When tolerateExisting is true, a .mtqg/ that is already there is not an
// error: ensureMtqg finds it instead (created is false), so that
// "mtqg init --agent" can be run again on a repository that already uses
// mtqg. Without it (plain "mtqg init", with or without -n), an existing
// .mtqg/ is the same *AlreadyInitializedError either way.
func ensureMtqg(start string, tolerateExisting, dryRun bool) (loc journal.Location, created bool, err error) {
	if dryRun {
		found, ferr := journal.Find(start)
		if ferr == nil {
			if !tolerateExisting {
				return journal.Location{}, false, &journal.AlreadyInitializedError{Path: found.Dir}
			}
			return found, false, nil
		}
		var notInit *journal.NotInitializedError
		if errors.As(ferr, &notInit) {
			// Nothing is there yet: this is where journal.Init would make it, but
			// dryRun means it does not.
			return journal.Location{Root: notInit.Root}, true, nil
		}
		return journal.Location{}, false, ferr
	}

	loc, err = journal.Init(start, mtqg.Schema)
	if err == nil {
		return loc, true, nil
	}
	var already *journal.AlreadyInitializedError
	if tolerateExisting && errors.As(err, &already) {
		loc, err = journal.Find(start)
		return loc, false, err
	}
	return journal.Location{}, false, err
}
