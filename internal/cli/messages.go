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

func msgBadAt(value string) string {
	return fmt.Sprintf("Option --at needs <path> or <path>:<line> with a line of 1 or more, not %q.", value)
}

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

func msgMCPNoClientName() string {
	return "No author name: the connecting client gave no name in its initialize request."
}

// The text of a record.

func msgEmptyText() string { return "Aborting: the text is empty" }

func msgEditorFailed(err error) string { return fmt.Sprintf("The editor failed: %v", err) }

func msgBadEditorCommand(reason string) string { return fmt.Sprintf("Cannot read $EDITOR: %s", reason) }

func msgTextNotUTF8() string { return "The text is not valid UTF-8." }

func msgStdinFailed(err error) string { return fmt.Sprintf("Cannot read standard input: %v", err) }

// Finding a record.

func msgNotFound(prefix string) string { return fmt.Sprintf("No record matches %q", prefix) }

func msgIDTooShort(prefix string) string {
	return fmt.Sprintf("The ID %q is too short: give at least %d digits", prefix, model.MinIDDigits)
}

// threadWords are the words that a kind with replies is spoken of with: the
// question that is answered, the bug that is replied to.
type threadWords struct {
	parent  string // the record that is replied to
	replyTo string // what is done to it, after "the ID of the parent to"
	create  string // what one does to make a new one
}

func wordsOfThread(typ string) threadWords {
	if typ == journal.TypeBug {
		return threadWords{parent: "bug", replyTo: "reply to", create: "report a bug"}
	}
	return threadWords{parent: "question", replyTo: "answer", create: "ask a question"}
}

// msgNoParentToReplyTo is what `qa add` and `bug add` say when their first word
// looks like an ID and names no record: a mistyped ID must not turn into a new
// question or bug, and the way to write one that really starts with such a word is
// to quote it. kind is the name of the command's kind (qa, bug).
func msgNoParentToReplyTo(typ, kind, word string) []string {
	w := wordsOfThread(typ)
	return []string{
		fmt.Sprintf("No record matches %q. A first word of %d or more hex digits is read as the ID of the %s to %s.", word, model.MinIDDigits, w.parent, w.replyTo),
		fmt.Sprintf("To %s that starts with it, put the whole text in quotes: mtqg %s add \"%s ...\"", w.create, kind, word),
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
// another command is the right one, it is named: the kind whose commands are for
// this record, when it has the verb (a reply is not, because a reply has no state
// to change).
func msgWrongKind(e *model.WrongKindError, verb string) string {
	id := shortID(e.Record.ID, false)
	msg := fmt.Sprintf("%s is %s, not %s", id, article(e.Record.Kind()), article(e.Want))
	if verb == "done" || verb == "reopen" {
		for _, k := range kinds {
			if model.ParentKind(k.typ) == e.Record.Kind() && lookup(k.name, verb) != nil {
				return fmt.Sprintf("%s; use `mtqg %s %s %s`", msg, k.name, verb, id)
			}
		}
	}
	return msg
}

func msgNoState(e *model.NoStateError, verb string) string {
	return fmt.Sprintf("%s is %s; %s has no state to change", shortID(e.Record.ID, false), article(e.Record.Kind()), article(e.Record.Kind()))
}

func msgNoReplies(e *model.NoRepliesError) string {
	return fmt.Sprintf("%s is %s; only a question or a bug can be answered or replied to", shortID(e.Record.ID, false), article(e.Record.Kind()))
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

// edit and delete

func msgEdited(id, text string) string { return fmt.Sprintf("Edited: %s  %s", id, text) }

func msgUnchanged(id, text string) string { return fmt.Sprintf("Unchanged: %s  %s", id, text) }

func msgDeleted(id, text string) string { return fmt.Sprintf("Deleted: %s  %s", id, text) }

// msgAlsoHidden says how many answers or replies went with a question or a bug
// that was deleted, and whose they were.
func msgAlsoHidden(typ string, n int, authors []string) string {
	many, one := "answers are", "answer is"
	if typ == journal.TypeBug {
		many, one = "replies are", "reply is"
	}
	count := fmt.Sprintf("%d %s", n, many)
	if n == 1 {
		count = "1 " + one
	}
	return fmt.Sprintf("%s also hidden (%s)", count, strings.Join(authors, ", "))
}

// msgDeleteNote is said after every delete: nothing was removed.
func msgDeleteNote() string { return "The lines remain in the journal and in git history" }

// undo

// msgUndone says what was removed: the type of the record, what the line did, the
// text (when there is one) and the ID.
func msgUndone(typ, what, text, id string) string {
	if text == "" {
		return fmt.Sprintf("Undone: %s %s (%s)", typ, what, id)
	}
	return fmt.Sprintf("Undone: %s %s %q (%s)", typ, what, text, id)
}

func msgNothingToUndo(author string) string {
	return fmt.Sprintf("Nothing to undo: .mtqg/journal.jsonl has no line written by %s from this terminal", author)
}

// msgHasLaterEvents refuses to undo the creation of a record that other events
// are about, and says what to do instead.
func msgHasLaterEvents(rec *model.Record, others int) []string {
	count := fmt.Sprintf("%d other events", others)
	if others == 1 {
		count = "1 other event"
	}
	id := shortID(rec.ID, false)
	return []string{
		fmt.Sprintf("Cannot undo: %s %s has %s, and undoing its creation would leave them without a record", rec.Kind(), id, count),
		"To hide it instead: mtqg delete " + id,
	}
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

// msgConflictsValue is the count of records that have concurrent changes.
func msgConflictsValue(n int) string {
	return fmt.Sprintf("%d  (concurrent changes; see mtqg review)", n)
}

func msgStatusLine(label string, value string) string {
	return padRight(label, statusLabelWidth) + value
}

// msgOpenValue is the count of open questions (or bugs), and how many of them have
// an answer (or a reply) that nobody has confirmed by closing the question.
func msgOpenValue(open, awaiting int) string {
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

func msgRuleFooter(n int) string {
	if n == 1 {
		return "1 rule"
	}
	return fmt.Sprintf("%d rules", n)
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

// msgThreadState is the state of a question or a bug in a list: whether it has
// answers (or replies), and whether it is closed.
func msgThreadState(typ string, replies int, done bool) string {
	many, one, none := "answers", "answer", "unanswered"
	if typ == journal.TypeBug {
		many, one, none = "replies", "reply", "no replies"
	}
	count := fmt.Sprintf("%d %s", replies, many)
	if replies == 1 {
		count = "1 " + one
	}
	switch {
	case done && replies == 0:
		return "done without " + many
	case done:
		return count + ", done"
	case replies == 0:
		return none
	default:
		return count + ", awaiting confirmation"
	}
}

// log

// msgLogFooter is log's last line: the count, and, with --before, that it is a
// count of records before the given ID.
func msgLogFooter(shown, total int, before string) string {
	suffix := ""
	if before != "" {
		suffix = " before " + before
	}
	switch {
	case shown < total:
		return fmt.Sprintf("%d of %d records%s (--limit 0 for all)", shown, total, suffix)
	case total == 1:
		return fmt.Sprintf("1 record%s", suffix)
	default:
		return fmt.Sprintf("%d records%s", total, suffix)
	}
}

func msgBadLimit(value string) string {
	return fmt.Sprintf("Option --limit needs a whole number of 0 or more, not %q. Use 0 for all.", value)
}

// completion

func msgNoShell(shells []string) string {
	return "Missing shell: `mtqg completion` needs one of " + strings.Join(shells, ", ") + "."
}

func msgUnknownShell(word string, shells []string) string {
	return fmt.Sprintf("Unknown shell %q: `mtqg completion` needs one of %s.", word, strings.Join(shells, ", "))
}

func msgBadKind(value string) string {
	names := make([]string, len(kinds))
	for i, k := range kinds {
		names[i] = k.name
	}
	return fmt.Sprintf("Option --kind needs one of %s (or its letter), not %q.", strings.Join(names, ", "), value)
}

// hook, init --agent

func msgUnknownAgent(word string, agents []string) string {
	return fmt.Sprintf("Unknown agent %q: mtqg knows %s.", word, strings.Join(agents, ", "))
}

func msgUnknownHookEvent(agent, word string, events []string) string {
	return fmt.Sprintf("Unknown event %q for %s: mtqg knows %s.", word, agent, strings.Join(events, ", "))
}

func msgHookBadInput(err error) string {
	return fmt.Sprintf("could not read the hook's input: %v", err)
}

// msgHookStopReason is what Claude Code shows the agent when a Stop hook keeps
// it from ending its turn (§11.3). It names what to do and the command to read
// what is already recorded, in one or two lines.
func msgHookStopReason() string {
	return "Before ending: record with mtqg anything worth keeping from this turn " +
		"(decisions made, questions asked and answered, bugs found - even ones already " +
		"fixed, things left to do). Run `mtqg context` to see what is already recorded."
}

func msgWouldCreateMtqg(root string) string {
	return fmt.Sprintf("Created (dry run): .mtqg/ in %s", root)
}

func msgAgentExisting(path string) string {
	return fmt.Sprintf(".mtqg/ already exists: %s (left as it is)", path)
}

func msgAgentFileResult(status, path string, dryRun bool) string {
	label := map[string]string{"created": "Created", "updated": "Updated", "unchanged": "Unchanged"}[status]
	if dryRun && status != "unchanged" {
		label += " (dry run)"
	}
	return label + ": " + path
}

func msgAgentSettingsInvalid(path string, err error) string {
	return fmt.Sprintf("%s is not valid JSON, so mtqg will not change it: %v", path, err)
}

// review

func msgReviewNothing() string { return "Nothing to review" }

func msgReviewConcurrent(n int) string {
	return fmt.Sprintf("Concurrent changes (%d)", n)
}

func msgReviewDuplicates(n int) string {
	return fmt.Sprintf("Duplicate glossary definitions (%d)", n)
}

func msgReviewUnattached(n int) string {
	return fmt.Sprintf("Answers and replies with no parent (%d)", n)
}

// msgReviewRecord names a record that has concurrent changes: what it is, its
// ID and its text.
func msgReviewRecord(kind, id, text string) string {
	return fmt.Sprintf("  %s %s %q", kind, id, text)
}

// msgReviewRe says what the re of an answer or a reply names, which is not a
// question or a bug of its own kind.
func msgReviewRe(id, what string) string { return fmt.Sprintf("    re %s: %s", id, what) }

// search

// msgSearchFooter counts what a search found and says what it was for.
func msgSearchFooter(n int, query string) string {
	q := fmt.Sprintf("%q", query)
	switch n {
	case 0:
		return "No records contain " + q
	case 1:
		return "1 record contains " + q
	default:
		return fmt.Sprintf("%d records contain %s", n, q)
	}
}

// format

func msgCannotReadFile(name string, err error) string {
	return fmt.Sprintf("Cannot read %s: %v", name, err)
}

// show

const msgShowEvents = "Events"

// msgShowReplyCount heads the answers of a question or the replies of a bug.
func msgShowReplyCount(typ string, n int) string {
	heading := "Answers"
	if typ == journal.TypeBug {
		heading = "Replies"
	}
	return fmt.Sprintf("%s (%d)", heading, n)
}

func msgShowBy(author, when string) string { return fmt.Sprintf("by %s, %s", author, when) }

// msgShowToParent names the question or bug that an answer or a reply is for, and
// what it says.
func msgShowToParent(kind, id, text string) string {
	if text == "" {
		return "to " + kind + " " + id
	}
	return "to " + kind + " " + id + "  " + text
}

// msgShowToMissing is for an answer or a reply whose re names nothing in the
// journal: the question or bug it was for is not there (it may be in an archive).
func msgShowToMissing(kind, id string) string {
	return "to " + kind + " " + id + "  (no such record)"
}

// msgShowToOther is for a record whose re names a record that cannot be replied to
// as it is: another kind, or an answer. It is not a reply to it, so no parent is
// named. what and want are what the record is and what it would have to be, each
// with its article (a question, a bug).
func msgShowToOther(id, what, want string) string {
	return "to " + id + "  (" + what + ", not " + want + ")"
}

func msgShowWord(word string) string { return "Word: " + word }

// msgShowReplyEvent says which answer or reply an event of the history is: its
// kind (answer, reply) and its ID.
func msgShowReplyEvent(kind, id string) string { return kind + " " + id }

// version

func msgVersion(version string) string { return "mtqg " + version }

func msgFormatVersion(found, supported int, known bool) string {
	if !known {
		return "Repository format version: unknown (no .mtqg/ found)"
	}
	return fmt.Sprintf("Repository format version: %d (this mtqg supports up to %d)", found, supported)
}

// context

// contextGuide tells the agent what it is reading and what to do with it.
var contextGuide = []string{
	"This is the process record of this project. Read the following before you start working.",
	"- Follow the rules listed under Rules",
	"- Respect answered questions",
	"- Do not decide open questions on your own; confirm them",
	"- Use terms as defined in the glossary",
	"- Record questions, decisions, findings, bugs, and todos with mtqg as they come up",
}

func msgContextTitle(repository, branch string) string {
	if branch == "" {
		return "# mtqg context — " + repository
	}
	return fmt.Sprintf("# mtqg context — %s (%s)", repository, branch)
}

const (
	msgContextAttention = "## Attention"
	msgContextRecent    = "## Recent records (newest first)"
	msgContextFooter    = "Read full entries with mtqg show <id>."
)

func msgContextHeading(name string, total int) string { return fmt.Sprintf("## %s (%d)", name, total) }

func msgContextConflictingWord(word string) string {
	return fmt.Sprintf("- Glossary term %q has conflicting definitions (see mtqg glossary list)", word)
}

// msgContextConcurrent names a record that has concurrent changes.
func msgContextConcurrent(kind, id, text string) string {
	return fmt.Sprintf("- %s %s %q has concurrent changes (see mtqg review)", kind, id, text)
}

func msgContextMoreConcurrent(n int) string {
	return fmt.Sprintf("- (%d more records have concurrent changes; see mtqg review)", n)
}

func msgContextMoreWords(n int) string {
	return fmt.Sprintf("- (%d more words have conflicting definitions; see mtqg glossary list)", n)
}

func msgContextUncommitted(n int) string {
	if n == 1 {
		return "- 1 mtqg record is not committed"
	}
	return fmt.Sprintf("- %d mtqg records are not committed", n)
}

// msgContextItem is a todo: what it is, who wrote it and when.
func msgContextItem(id, text, author, when string) string {
	return fmt.Sprintf("- %s %s (%s, %s)", id, text, author, when)
}

// msgContextThread is a question or a bug, with its state.
func msgContextThread(id, text, state, author, when string) string {
	return fmt.Sprintf("- %s %s (%s, %s, %s)", id, text, state, author, when)
}

// msgContextReplyLine is the latest answer or reply, under its question or bug.
func msgContextReplyLine(text, author, kind string) string {
	return fmt.Sprintf("    └ %s (%s, %s)", text, author, kind)
}

// msgContextThreadState says whether a question or a bug has anything under it
// that nobody has confirmed. typ is the type of the record, as in msgThreadState.
func msgContextThreadState(typ string, replies int) string {
	switch {
	case replies > 0:
		return "awaiting confirmation"
	case typ == journal.TypeBug:
		return "no replies"
	default:
		return "unanswered"
	}
}

func msgContextOlder(n int, list string) string {
	return fmt.Sprintf("- (%d older; see mtqg %s)", n, list)
}

func msgContextMoreRecent(n int) string { return fmt.Sprintf("- (%d more; see mtqg log)", n) }

func msgContextRepliesLeft(typ string) string {
	if typ == journal.TypeBug {
		return "- (latest replies left out; see mtqg show <id>)"
	}
	return "- (latest answers left out; see mtqg show <id>)"
}

const msgContextDefinitionsLeft = "- (definitions left out; see mtqg glossary list)"

func msgContextWord(word string, definitions int) string {
	if definitions > 1 {
		return fmt.Sprintf("- %s (%d definitions)", word, definitions)
	}
	return "- " + word
}

func msgBadMaxTokens(value string) string {
	return fmt.Sprintf("Option --max-tokens needs a whole number of 0 or more, not %q. Use 0 for no limit.", value)
}

// archive

func msgArchiveRangeLine(rng string, dryRun bool) string {
	if dryRun {
		return "Range: " + rng + " (dry run)"
	}
	return "Range: " + rng
}

const msgArchivedNothing = "Archived: nothing"

// msgArchivedLine says what moved and where to. parts is what is counted, as "12
// bugs".
func msgArchivedLine(parts []string, file string) string {
	return "Archived: " + strings.Join(parts, ", ") + " -> " + file
}

func msgSkippedLine(parts []string) string { return "Skipped: " + strings.Join(parts, ", ") }

// msgCount says how many of a thing there are, with the singular for one.
func msgCount(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}
	return fmt.Sprintf("%d %s", n, plural)
}

func msgRangeNotARange(arg string) string {
	return fmt.Sprintf("%q is not a range. Write <start>..<end>, for example 2021-01-01..2024-09-18. %s", arg, hintHelp)
}

func msgRangeBadCharacter(arg string) string {
	return fmt.Sprintf("%q is not a range: only digits, - and . are allowed", arg)
}

func msgRangeBadSide(side string) string {
	return fmt.Sprintf("%q is not a date, a month or a year: give 8, 6 or 4 digits (2024-09-18, 2024-09, 2024)", side)
}

// msgRangeMixedUnits complains that one side of a range is a year, a month or a
// day and the other is not; leftUnit and rightUnit are those words.
func msgRangeMixedUnits(left, leftUnit, right, rightUnit string) string {
	return fmt.Sprintf("The two sides of the range are not the same kind: %q is a %s and %q is a %s", left, leftUnit, right, rightUnit)
}

func msgRangeNoSuchDate(date string) string { return fmt.Sprintf("%s is not a date", date) }

func msgRangeBackwards(start, end string) string {
	return fmt.Sprintf("The range starts after it ends: %s..%s", start, end)
}
