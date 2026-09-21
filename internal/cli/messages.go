package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

// Every sentence mtqg prints is here. The core returns kinds of error and
// structured results, and this is where they become English
// (.claude/rules/cli-output.md); a record's own words are shown as written.

const hintHelp = "Try `mtqg help`."

// Mistakes in the command line (exit code 2).

func msgNoCommand() string { return "No command given. " + hintHelp }

func msgUnknownCommand(word string) string {
	return fmt.Sprintf("Unknown command %q. %s", word, hintHelp)
}

func msgMissingVerb(kind string, verbs []string) string {
	return fmt.Sprintf("`mtqg %s` needs a verb: %s. %s", kind, strings.Join(verbs, ", "), hintHelp)
}

func msgUnknownVerb(kind, word string, verbs []string) string {
	return fmt.Sprintf("Unknown verb %q for `mtqg %s` (%s). %s", word, kind, strings.Join(verbs, ", "), hintHelp)
}

func msgUnknownOption(opt, usage string) string {
	if usage == "" {
		return fmt.Sprintf("Unknown option %s. A text that starts with - needs -- before it. %s", opt, hintHelp)
	}
	return fmt.Sprintf("Unknown option %s for `%s`. A text that starts with - needs -- before it. %s", opt, usage, hintHelp)
}

func msgOptionNeedsValue(opt string) string { return fmt.Sprintf("Option %s needs a value.", opt) }

func msgMissingArgument(usage string) string { return "Missing argument. Usage: " + usage }

func msgTooManyArguments(usage string) string { return "Too many arguments. Usage: " + usage }

func msgNotYet(usage string) string {
	return fmt.Sprintf("`%s` is not available yet in this build of mtqg.", usage)
}

// Where the records are.

func msgNotInRepository() string {
	return "Not inside a git repository. mtqg keeps its records in the repository of a project."
}

func msgNotInitialized(root string) string {
	return fmt.Sprintf("No .mtqg/ found. Run `mtqg init` (will be created at %s)", root)
}

func msgAlreadyInitialized(path string) string {
	return fmt.Sprintf(".mtqg/ already exists: %s", path)
}

func msgInitialized(root string) []string {
	return []string{fmt.Sprintf("Created .mtqg/ in %s", root), "Commit it to share the records."}
}

func msgFormatTooNew(found, supported int) string {
	return fmt.Sprintf("This repository uses format version %d, but this mtqg understands up to version %d. Update mtqg.", found, supported)
}

func msgConflictMarkers() string {
	return "journal.jsonl has unresolved merge conflict markers, so mtqg will not write to it. " +
		"Remove only the marker lines (<<<<<<<, =======, >>>>>>>) and keep the lines of both sides."
}

func msgLockTimeout(waited string) string {
	return fmt.Sprintf("Another mtqg is writing to the journal (waited %s). Try again.", waited)
}

// Who is writing.

func msgNoUserName() string {
	return "No author name. Set one with `git config user.name \"Your Name\"`, or with MTQG_AUTHOR_NAME."
}

func msgGitUnavailable(err error) string {
	return fmt.Sprintf("git could not be run: %v", err)
}

func msgBadAuthorKind(value string) string {
	return fmt.Sprintf("MTQG_AUTHOR_KIND must be human or ai, not %q.", value)
}

func msgAIneedsName() string {
	return "MTQG_AUTHOR_KIND=ai needs MTQG_AUTHOR_NAME: an AI is not recorded under the name in `git config`."
}

// The text of a record.

func msgEmptyText() string { return "Aborting: the text is empty" }

func msgNoEditor() string { return "No text given, and $EDITOR is not set." }

func msgEditorFailed(err error) string { return fmt.Sprintf("The editor failed: %v", err) }

func msgBadEditorCommand(reason string) string { return fmt.Sprintf("Cannot read $EDITOR: %s", reason) }

func msgTextNotUTF8() string { return "The text is not valid UTF-8." }

func msgStdinFailed(err error) string { return fmt.Sprintf("Cannot read standard input: %v", err) }

// Finding a record.

func msgNotFound(prefix string) string { return fmt.Sprintf("No record matches %q", prefix) }

func msgIDTooShort(prefix string) string {
	return fmt.Sprintf("The ID %q is too short: give at least %d digits", prefix, model.MinIDDigits)
}

// msgNoQuestionToAnswer is what `qa add` says when its first word looks like an
// ID and names no record: a mistyped ID must not turn into a new question, and the
// way to ask a question that really starts with such a word is to quote it.
func msgNoQuestionToAnswer(word string) []string {
	return []string{
		fmt.Sprintf("No record matches %q. A first word of %d or more hex digits is read as the ID of the question to answer.", word, model.MinIDDigits),
		fmt.Sprintf("To ask a question that starts with it, put the whole text in quotes: mtqg qa add \"%s ...\"", word),
	}
}

func msgEmptyWord() string { return "Aborting: the word is empty" }

func msgAmbiguousHeader(prefix string, n int) string {
	return fmt.Sprintf("Ambiguous ID %q matches %d records:", prefix, n)
}

// article gives "a memo", "an answer".
func article(kind string) string {
	if strings.ContainsRune("aeiou", rune(kind[0])) {
		return "an " + kind
	}
	return "a " + kind
}

// msgWrongKind says what a record is when a command was for another kind. If
// another command is the right one, it is named.
func msgWrongKind(e *model.WrongKindError, verb string) string {
	id := shortID(e.Record.ID, false)
	msg := fmt.Sprintf("%s is %s, not %s", id, article(e.Record.Kind()), article(e.Want))
	if e.Record.Kind() == model.KindQuestion && e.Want == model.KindTodo && (verb == "done" || verb == "reopen") {
		return fmt.Sprintf("%s; use `mtqg qa %s %s`", msg, verb, id)
	}
	if e.Record.Kind() == model.KindTodo && e.Want == model.KindQuestion && (verb == "done" || verb == "reopen") {
		return fmt.Sprintf("%s; use `mtqg todo %s %s`", msg, verb, id)
	}
	return msg
}

func msgNoState(e *model.NoStateError, verb string) string {
	return fmt.Sprintf("%s is %s; %s has no state to change", shortID(e.Record.ID, false), article(e.Record.Kind()), article(e.Record.Kind()))
}

// The result of a change to a state.

func msgStatusChanged(verb, id, text string) string {
	switch verb {
	case "done":
		return fmt.Sprintf("Done: %s  %s", id, text)
	default:
		return fmt.Sprintf("Reopened: %s  %s", id, text)
	}
}

func msgAlreadyInState(verb, id, text string) string {
	if verb == "done" {
		return fmt.Sprintf("Already done: %s  %s", id, text)
	}
	return fmt.Sprintf("Already open: %s  %s", id, text)
}

// Reading the journal.

// maxWarnings is how many skipped lines are named before the rest is counted.
const maxWarnings = 5

func msgWarning(w journal.Warning) string {
	const file = ".mtqg/journal.jsonl"
	switch w.Kind {
	case journal.WarnInvalidJSON:
		return fmt.Sprintf("warning: %s line %d is not a valid JSON object; skipped it", file, w.Line)
	case journal.WarnInvalidUTF8:
		return fmt.Sprintf("warning: %s line %d is not valid UTF-8; skipped it", file, w.Line)
	case journal.WarnMissingField:
		return fmt.Sprintf("warning: %s line %d has no id or no op; skipped it", file, w.Line)
	case journal.WarnConflictMarker:
		return fmt.Sprintf("warning: %s line %d is a merge conflict marker; resolve the conflict (keep the lines of both sides)", file, w.Line)
	case journal.WarnNoTrailingNewline:
		return fmt.Sprintf("warning: %s line %d does not end with a line feed; it may still be being written", file, w.Line)
	default:
		return fmt.Sprintf("warning: %s line %d could not be read", file, w.Line)
	}
}

func msgMoreWarnings(n int) string {
	return fmt.Sprintf("warning: (%d more lines could not be read)", n)
}

// status

const statusLabelWidth = 20

func msgStatusLine(label string, value string) string {
	return padRight(label, statusLabelWidth) + value
}

// msgQuestionsValue is the count of open questions, and how many of them have an
// answer that nobody has confirmed by closing the question.
func msgQuestionsValue(open, awaiting int) string {
	if awaiting == 0 {
		return strconv.Itoa(open)
	}
	return fmt.Sprintf("%d  (%d awaiting confirmation)", open, awaiting)
}

// msgGlossaryValue is the count of entries, and of the words that more than one
// entry defines.
func msgGlossaryValue(entries, duplicateWords int) string {
	if duplicateWords == 0 {
		return strconv.Itoa(entries)
	}
	return fmt.Sprintf("%d  (%d with duplicate definitions)", entries, duplicateWords)
}

// list footers

func msgOpenFooter(open, done int, all bool) string {
	if all {
		return fmt.Sprintf("%d open, %d done", open, done)
	}
	return fmt.Sprintf("%d open (show done: --all)", open)
}

func msgMemoFooter(n int) string {
	if n == 1 {
		return "1 memo"
	}
	return fmt.Sprintf("%d memos", n)
}

func msgGlossaryFooter(n, duplicateWords int) string {
	line := fmt.Sprintf("%d terms", n)
	if n == 1 {
		line = "1 term"
	}
	if duplicateWords > 0 {
		line += fmt.Sprintf(" (%d with duplicate definitions)", duplicateWords)
	}
	return line
}

// msgQuestionState is the state of a question in a list: whether it has answers,
// and whether it is closed.
func msgQuestionState(answers int, done bool) string {
	count := fmt.Sprintf("%d answers", answers)
	if answers == 1 {
		count = "1 answer"
	}
	switch {
	case done && answers == 0:
		return "done without answers"
	case done:
		return count + ", done"
	case answers == 0:
		return "unanswered"
	default:
		return count + ", awaiting confirmation"
	}
}

// log

func msgLogFooter(shown, total int) string {
	switch {
	case shown < total:
		return fmt.Sprintf("%d of %d records (--limit 0 for all)", shown, total)
	case total == 1:
		return "1 record"
	default:
		return fmt.Sprintf("%d records", total)
	}
}

func msgBadLimit(value string) string {
	return fmt.Sprintf("Option --limit needs a whole number of 0 or more, not %q. Use 0 for all.", value)
}

func msgBadKind(value string) string {
	names := make([]string, len(kinds))
	for i, k := range kinds {
		names[i] = k.name
	}
	return fmt.Sprintf("Option --kind needs one of %s (or its letter), not %q.", strings.Join(names, ", "), value)
}

// show

const (
	msgShowAnswers = "Answers"
	msgShowEvents  = "Events"
)

func msgShowAnswerCount(n int) string { return fmt.Sprintf("%s (%d)", msgShowAnswers, n) }

func msgShowBy(author, when string) string { return fmt.Sprintf("by %s, %s", author, when) }

func msgShowToQuestion(id, text string) string {
	if text == "" {
		return "to question " + id
	}
	return "to question " + id + "  " + text
}

func msgShowWord(word string) string { return "Word: " + word }

func msgShowAnswerEvent(id string) string { return "answer " + id }

// version

func msgVersion(version string) string { return "mtqg " + version }

func msgFormatVersion(found, supported int, known bool) string {
	if !known {
		return "Repository format version: unknown (no .mtqg/ found)"
	}
	return fmt.Sprintf("Repository format version: %d (this mtqg supports up to %d)", found, supported)
}
