package cli

import (
	jsonv2 "encoding/json/v2"

	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

// claudeCodeInput is read from what Claude Code gives a hook on standard input
// (code.claude.com/docs/en/hooks, checked 2026-09-25). mtqg reads only the
// fields it needs; a field Claude Code adds later, or one this event does not
// send, is simply not there (docs/design/07-integrations.md §11.3 "仕様の変
// 化").
type claudeCodeInput struct {
	SessionID      string `json:"session_id"`
	Cwd            string `json:"cwd"`
	StopHookActive bool   `json:"stop_hook_active"`
}

// claudeStopBlock is how a Stop hook keeps Claude Code from ending the turn
// (code.claude.com/docs/en/hooks, "Stop decision control").
type claudeStopBlock struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

// hookClaudeCode runs one Claude Code hook event. Nothing past reading the
// event name (runHook already checked that) makes this fail the agent's turn:
// bad input, no .mtqg/ here, or git being unavailable are all quietly nothing
// to report, with exit code 0 (see runHook).
func (c *ctx) hookClaudeCode(event string) int {
	var in claudeCodeInput
	if err := jsonv2.UnmarshalRead(c.env.Stdin, &in); err != nil {
		c.warning(warningReport{kind: kindHookInput, message: "warning: " + msgHookBadInput(err)})
		return exitOK
	}
	if in.Cwd != "" {
		c.inv.dir = in.Cwd
	}
	j, err := c.reader()
	if err != nil {
		// No .mtqg/ here, or it cannot be read: a hook says nothing, the same way
		// candidates does (cli-output.md).
		return exitOK
	}
	switch event {
	case "session-start":
		return c.hookSessionStart(j, in)
	case "stop":
		return c.hookStop(j, in)
	default:
		return exitOK // unreachable: runHook already checked the event
	}
}

// hookSessionStart records that a session began here (so hookStop can compare
// against it later) and prints the same document "mtqg context" would, for
// Claude Code to add to the agent's context.
func (c *ctx) hookSessionStart(j *journal.Journal, in claudeCodeInput) int {
	now := c.env.Now()
	j.PruneSessions(now)

	if in.SessionID != "" {
		if _, already := j.ReadSession(in.SessionID); !already {
			root := j.Location().Root
			head, _ := journal.GitHead(root)
			digest, _ := journal.GitStatusDigest(root)
			_ = j.WriteSession(in.SessionID, journal.Session{StartedAt: now, Head: head, StatusDigest: digest})
		}
	}

	text, _, _, _, err := c.contextDocument(defaultContextTokens)
	if err != nil {
		return exitOK
	}
	_, _ = c.env.Stdout.Write([]byte(text))
	return exitOK
}

// hookStop asks model.ShouldPrompt whether to keep Claude Code from ending its
// turn, and if so, prints the block that does that.
func (c *ctx) hookStop(j *journal.Journal, in claudeCodeInput) int {
	if in.StopHookActive || in.SessionID == "" {
		return exitOK
	}
	session, ok := j.ReadSession(in.SessionID)
	if !ok || session.Prompted {
		return exitOK
	}

	root := j.Location().Root
	head, errHead := journal.GitHead(root)
	digest, errDigest := journal.GitStatusDigest(root)
	if errHead != nil || errDigest != nil {
		return exitOK // git unavailable: say nothing rather than guess
	}
	result, err := j.Read()
	if err != nil {
		return exitOK
	}
	c.warn(result.Warnings)

	start := model.RepoState{Head: session.Head, StatusDigest: session.StatusDigest}
	now := model.RepoState{Head: head, StatusDigest: digest}
	if !model.ShouldPrompt(session.Prompted, session.StartedAt, start, now, result.Events) {
		return exitOK
	}

	session.Prompted = true
	if err := j.WriteSession(in.SessionID, session); err != nil {
		return exitOK // could not remember that it asked: better to say nothing than to ask every turn
	}
	b, err := jsonv2.Marshal(claudeStopBlock{Decision: "block", Reason: msgHookStopReason()})
	if err != nil {
		return exitOK
	}
	_, _ = c.env.Stdout.Write(append(b, '\n'))
	return exitOK
}
