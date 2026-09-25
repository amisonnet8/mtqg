package cli

import (
	"errors"
	"strings"

	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

// What went wrong, and what a warning is. The core returns kinds of error; this
// file turns them into words for people (lines) and into a kind and a few facts
// for programs (--json). The words are all in messages.go.

// The kinds of error that --json names (docs/reference/cli.md "JSON output").
const (
	kindUsage             = "usage"
	kindNotAvailable      = "not_available"
	kindNotInRepository   = "not_in_repository"
	kindNotInitialized    = "not_initialized"
	kindAlreadyInit       = "already_initialized"
	kindFormatTooNew      = "format_too_new"
	kindConflictMarkers   = "conflict_markers"
	kindLockTimeout       = "lock_timeout"
	kindNoAuthor          = "no_author"
	kindBadAuthorKind     = "bad_author_kind"
	kindEmptyText         = "empty_text"
	kindEmptyWord         = "empty_word"
	kindInvalidText       = "invalid_text"
	kindInput             = "input"
	kindEditor            = "editor"
	kindGitUnavailable    = "git_unavailable"
	kindNotFound          = "not_found"
	kindIDTooShort        = "id_too_short"
	kindAmbiguous         = "ambiguous"
	kindWrongKind         = "wrong_kind"
	kindNoState           = "no_state"
	kindNoReplies         = "no_replies"
	kindNothingToUndo     = "nothing_to_undo"
	kindHasLaterEvents    = "has_later_events"
	kindHookInput         = "hook_input"
	kindHookConfigInvalid = "hook_config_invalid"
	kindUnknown           = "unknown"
)

// errorReport is one error, ready to be shown.
type errorReport struct {
	kind  string
	lines []string // what a person reads

	// What only some kinds have, for --json.
	prefix     string
	candidates []*model.Record
	record     *model.Record
	wanted     string
}

// reportOf turns an error of the core into a report.
func (c *ctx) reportOf(err error) errorReport {
	var (
		failed     *failure
		notInit    *journal.NotInitializedError
		already    *journal.AlreadyInitializedError
		tooNew     *journal.FormatTooNewError
		lock       *journal.LockTimeoutError
		invalid    *journal.InvalidEventError
		ambiguous  *model.AmbiguousError
		wrongKind  *model.WrongKindError
		noState    *model.NoStateError
		noReplies  *model.NoRepliesError
		laterOnes  *model.HasLaterEventsError
		notFound   *model.NotFoundError
		tooShort   *model.TooShortError
		gitMissing *journal.GitUnavailableError
	)
	verb := ""
	if c.inv != nil && c.inv.cmd != nil {
		verb = c.inv.cmd.name
	}
	one := func(kind, line string) errorReport { return errorReport{kind: kind, lines: []string{line}} }
	switch {
	case errors.As(err, &failed):
		return one(failed.kind, failed.msg)
	case errors.As(err, &notInit):
		return one(kindNotInitialized, msgNotInitialized(notInit.Root))
	case errors.Is(err, journal.ErrNotInRepository):
		return one(kindNotInRepository, msgNotInRepository())
	case errors.As(err, &already):
		return one(kindAlreadyInit, msgAlreadyInitialized(already.Path))
	case errors.As(err, &tooNew):
		return one(kindFormatTooNew, msgFormatTooNew(tooNew.Found, tooNew.Supported))
	case errors.Is(err, journal.ErrConflictMarkers):
		return one(kindConflictMarkers, msgConflictMarkers())
	case errors.As(err, &lock):
		return one(kindLockTimeout, msgLockTimeout(lock.Waited.Round(100_000_000).String()))
	case errors.As(err, &invalid) && invalid.Field == "text":
		return one(kindInvalidText, msgTextNotUTF8())
	case errors.As(err, &ambiguous):
		return errorReport{kind: kindAmbiguous, lines: c.describeAmbiguous(ambiguous), prefix: ambiguous.Prefix, candidates: ambiguous.Candidates}
	case errors.As(err, &wrongKind):
		return errorReport{kind: kindWrongKind, lines: []string{msgWrongKind(wrongKind, verb)}, record: wrongKind.Record, wanted: wrongKind.Want}
	case errors.As(err, &noState):
		return errorReport{kind: kindNoState, lines: []string{msgNoState(noState, verb)}, record: noState.Record}
	case errors.As(err, &noReplies):
		return errorReport{kind: kindNoReplies, lines: []string{msgNoReplies(noReplies)}, record: noReplies.Record}
	case errors.As(err, &laterOnes):
		return errorReport{kind: kindHasLaterEvents, lines: msgHasLaterEvents(laterOnes.Record, len(laterOnes.Events)), record: laterOnes.Record}
	case errors.As(err, &notFound):
		return errorReport{kind: kindNotFound, lines: []string{msgNotFound(notFound.Prefix)}, prefix: notFound.Prefix}
	case errors.As(err, &tooShort):
		return errorReport{kind: kindIDTooShort, lines: []string{msgIDTooShort(tooShort.Prefix)}, prefix: tooShort.Prefix}
	case errors.Is(err, model.ErrEmptyWord):
		return one(kindEmptyWord, msgEmptyWord())
	case errors.Is(err, model.ErrEmptyText):
		return one(kindEmptyText, msgEmptyText())
	case errors.As(err, &gitMissing):
		return one(kindGitUnavailable, msgGitUnavailable(gitMissing.Err))
	default:
		return one(kindUnknown, err.Error())
	}
}

// printError shows an error on standard error: in words, or as one line of JSON.
func (c *ctx) printError(rep errorReport) {
	if !c.inv.json {
		for _, l := range rep.lines {
			c.eprintln(l)
		}
		return
	}
	c.emitLine(jsonError{Error: rep.json()})
}

// json is the same shape --json writes to standard error for this error, used
// by printError and, as a tool result instead of a line on standard error, by
// the MCP server (mcp.go).
func (rep errorReport) json() jsonErrorBody {
	body := jsonErrorBody{
		Kind:    rep.kind,
		Message: strings.Join(rep.lines, "\n"),
		Prefix:  rep.prefix,
		Wanted:  rep.wanted,
	}
	if rep.record != nil {
		r := recordJSON(rep.record)
		body.Record = &r
	}
	if len(rep.candidates) > 0 {
		body.Candidates = recordsJSON(rep.candidates)
	}
	return body
}

// fail prints what went wrong, in words, and returns the exit code.
func (c *ctx) fail(err error) int { return c.failWith(c.reportOf(err)) }

// failWith prints a report and returns the exit code for an error (1).
func (c *ctx) failWith(rep errorReport) int {
	c.printError(rep)
	return exitError
}

type jsonError struct {
	Error jsonErrorBody `json:"error"`
}

type jsonErrorBody struct {
	Kind       string       `json:"kind"`
	Message    string       `json:"message"`
	Prefix     string       `json:"prefix,omitempty"`
	Candidates []jsonRecord `json:"candidates,omitzero"`
	Record     *jsonRecord  `json:"record,omitempty"`
	Wanted     string       `json:"wanted,omitempty"`
}

// A warning: something that was skipped or is suspect. It never changes the exit
// code.
type warningReport struct {
	kind    string
	line    int // the line of journal.jsonl, or 0
	count   int // for "more": how many
	message string
}

type jsonWarning struct {
	Warning jsonWarningBody `json:"warning"`
}

type jsonWarningBody struct {
	Kind    string `json:"kind"`
	Line    int    `json:"line,omitzero"`
	Count   int    `json:"count,omitzero"`
	Message string `json:"message"`
}

// warning shows a warning on standard error: in words, or as one line of JSON.
func (c *ctx) warning(w warningReport) {
	if c.inv.json {
		c.emitLine(jsonWarning{Warning: jsonWarningBody{Kind: w.kind, Line: w.line, Count: w.count, Message: w.message}})
		return
	}
	c.eprintln(w.message)
}

// warningKind is the name that --json gives a kind of warning of the journal.
func warningKind(k journal.WarningKind) string {
	switch k {
	case journal.WarnInvalidJSON:
		return "invalid_json"
	case journal.WarnInvalidUTF8:
		return "invalid_utf8"
	case journal.WarnMissingField:
		return "missing_field"
	case journal.WarnConflictMarker:
		return "conflict_marker"
	case journal.WarnNoTrailingNewline:
		return "no_trailing_newline"
	default:
		return "unreadable"
	}
}

// describeAmbiguous lists the records an ID could mean, with their full IDs so
// that any length of them can be typed again.
func (c *ctx) describeAmbiguous(e *model.AmbiguousError) []string {
	lines := []string{msgAmbiguousHeader(e.Prefix, len(e.Candidates))}
	var idW, kindW, textW, authorW int
	for _, r := range e.Candidates {
		idW = max(idW, displayWidth(r.ID))
		kindW = max(kindW, displayWidth(r.Kind()))
		textW = max(textW, displayWidth(oneLine(r.Text)))
		authorW = max(authorW, displayWidth(oneLine(r.Author.Name)))
	}
	if w := c.env.StdoutWidth; w > 0 {
		fixed := 2 + idW + len(gap) + kindW + len(gap) + authorW + len(gap) + len("2006-01-02")
		textW = min(textW, max(w-1-fixed, minTextWidth))
	}
	now := c.env.Now()
	for _, r := range e.Candidates {
		lines = append(lines, "  "+
			padRight(r.ID, idW)+gap+padRight(r.Kind(), kindW)+gap+
			padRight(truncate(oneLine(r.Text), textW), textW)+gap+
			padRight(oneLine(r.Author.Name), authorW)+gap+
			formatTime(r.Created, now, c.env.Location))
	}
	return lines
}
