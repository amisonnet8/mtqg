package cli

import "slices"

// runHook runs the adapter for one agent's hook event ("mtqg hook <agent>
// <event>", called from the agent's own configuration - "mtqg init --agent",
// docs/design/07-integrations.md §11.3). The adapter only translates input and
// output; the judgment ("is there something to say") lives in
// model.ShouldPrompt and the session bookkeeping in journal.Session, neither of
// which knows what agent is asking.
//
// An unknown agent or event is a mistake in the command line (exit code 2), the
// same as any other command's usage error: it does not depend on anything a
// hook reads at run time. Once the agent and event are known, hook never fails
// the agent's turn: a journal that cannot be opened, input mtqg cannot parse,
// or git being unavailable are all quietly nothing to report, with exit code 0
// (see hookClaudeCode). Unlike every other command, hook does not use exit
// code 1.
func runHook(c *ctx) int {
	words := c.inv.words
	switch {
	case len(words) < 2:
		return c.usageFailure(msgMissingArgument(c.inv.cmd.usage))
	case len(words) > 2:
		return c.usageFailure(msgTooManyArguments(c.inv.cmd.usage))
	}
	agent, event := words[0], words[1]
	events, ok := hookEvents[agent]
	if !ok {
		return c.usageFailure(msgUnknownAgent(agent, initAgents))
	}
	if !slices.Contains(events, event) {
		return c.usageFailure(msgUnknownHookEvent(agent, event, events))
	}
	switch agent {
	case "claude-code":
		return c.hookClaudeCode(event)
	default:
		return c.usageFailure(msgUnknownAgent(agent, initAgents))
	}
}
