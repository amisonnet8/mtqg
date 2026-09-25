package model

import (
	"time"

	"github.com/amisonnet8/mtqg/internal/journal"
)

// RepoState is the repository facts a Stop hook compares against what a
// SessionStart hook saw: the commit HEAD points to, and a digest of the
// working tree other than .mtqg/ itself (journal.GitHead,
// journal.GitStatusDigest). It knows nothing about any agent.
type RepoState struct {
	Head         string
	StatusDigest string
}

// ShouldPrompt says whether a Stop hook should ask the agent to record
// something with mtqg before ending its turn (docs/design/07-integrations.md
// §11.3 "セッションの記録と判定"). It knows nothing about any agent: the
// adapter (internal/cli/hook_claude.go) is the only place that does.
//
// prompted is whether this session already asked once: it is never asked
// twice. start is what session-start saw, now is the same facts read again.
// events is every event of journal.jsonl (any author, any kind - create,
// status, edit and delete all count): if one of them happened at or after
// startedAt, something was already recorded and there is nothing to ask for.
//
// Otherwise, the condition is that something changed in the repository since
// the session started: the commit HEAD points to, or anything in the working
// tree (GitStatusDigest already leaves .mtqg/ out). A session in which nothing
// happened is not asked to record anything.
//
// startedAt is truncated to the second before it is compared with an event's
// ts: ts is written with second precision (journal-format.md "書き出し規則"),
// but startedAt comes from a hook's own clock and normally carries a fraction
// of a second. Without truncating, a record written in the same second the
// session started could read as "before" startedAt purely from that fraction,
// and be missed.
func ShouldPrompt(prompted bool, startedAt time.Time, start, now RepoState, events []journal.Event) bool {
	if prompted {
		return false
	}
	if start == now {
		return false
	}
	startedAt = startedAt.Truncate(time.Second)
	for _, e := range events {
		if !parseTime(e.TS).Before(startedAt) {
			return false
		}
	}
	return true
}
