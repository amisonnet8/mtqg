package cli

import (
	"strings"

	"github.com/amisonnet8/mtqg/internal/journal"
)

// The grammar of the command line is data: the kinds, the commands and what
// each takes. Reading the arguments, the list of commands in help, and (later)
// the completion of the shell all come from these tables, so they cannot
// disagree.

// A kind of record, the letter it can be shortened to, and the type of the
// format that its records are written with. Questions and answers are the kind qa,
// bugs and replies the kind bug: the two are handled by the same code, which
// reads what it needs from this table.
type kindSpec struct {
	name  string
	short string
	typ   string
}

var kinds = []kindSpec{
	{"memo", "m", journal.TypeMemo},
	{"todo", "t", journal.TypeTodo},
	{"qa", "q", journal.TypeQA},
	{"bug", "b", journal.TypeBug},
	{"glossary", "g", journal.TypeGlossary},
	{"rule", "r", journal.TypeRule},
}

// hookEvents lists, for each agent mtqg has a hook adapter for (§11.3), the
// events it understands. "mtqg hook" and "mtqg init --agent" both read this
// table instead of having their own list.
var hookEvents = map[string][]string{
	"claude-code": {"session-start", "stop"},
}

// initAgents is the agents "mtqg init --agent" accepts, in a fixed order (for
// help and error messages).
var initAgents = []string{"claude-code"}

// What a command takes after its name.
type argMode int

const (
	// argsNone: nothing. Options may be anywhere.
	argsNone argMode = iota
	// argsID: one ID. Options may be anywhere.
	argsID
	// argsText: free text, joined with spaces. Options come before the text; from
	// its first word on, everything is text.
	argsText
	// argsIDText: an ID and then free text.
	argsIDText
	// argsAny: anything the command defines for itself. Options may be anywhere.
	argsAny
)

// idSet says which records the ID that a command takes is completed from.
type idSet int

const (
	// idsNone: the command takes no ID, or one that is not completed.
	idsNone idSet = iota
	// idsOpen: the open ones of the command's kind (done).
	idsOpen
	// idsDone: the done ones of the command's kind (reopen).
	idsDone
	// idsAll: every record in view (show, edit, delete).
	idsAll
	// idsParents: the questions or the bugs, open or done: the word that may be the
	// ID of the one to answer or reply to (add of qa and bug).
	idsParents
)

// command is one command of mtqg. A command with no run is known, so that it
// can be named and asked for, but not yet available.
type command struct {
	kind    string // the name of the kind, or "" for a command without one
	name    string // the verb, or the command word
	usage   string
	summary string
	args    argMode
	all     bool // accepts --all
	mark    bool // accepts --mark
	dryRun  bool // accepts -n and --dry-run

	// values are the options that take a value, as --limit 5 or --limit=5.
	values []string

	// minWords is how many words a command that takes a text must be given. A
	// command that takes text and no minimum opens the editor when it is given
	// none (memo add).
	minWords int

	// ids and choices are what the first word after the command is completed from
	// (candidates): the IDs of some records, or a fixed list of words.
	ids     idSet
	choices []string

	run func(c *ctx) int
}

// spec returns the kind that the command is for. A command without a kind gets
// the zero kindSpec.
func (cmd *command) spec() kindSpec {
	k, _ := findKind(cmd.kind)
	return k
}

// takesValue says whether the command has an option of this name that takes a
// value.
func (cmd *command) takesValue(name string) bool {
	for _, v := range cmd.values {
		if v == name {
			return true
		}
	}
	return false
}

// full names the command as one types it: "mtqg todo done".
func (cmd *command) full() string {
	if cmd.kind == "" {
		return "mtqg " + cmd.name
	}
	return "mtqg " + cmd.kind + " " + cmd.name
}

// commands lists every command of the first release, in the order help shows
// them. The ones that have no run function come in later steps.
//
// It is filled in by init: runHelp reads the table, so the table cannot be
// initialized as a variable that names runHelp.
var commands []*command

func init() {
	commands = []*command{
		{kind: "memo", name: "add", usage: "mtqg memo add <text>", summary: "Record a memo", args: argsText, run: runAddMemo},
		{kind: "memo", name: "list", usage: "mtqg memo list", summary: "List the memos", args: argsNone, run: runListMemos},
		{kind: "todo", name: "add", usage: "mtqg todo add <text>", summary: "Record something to do", args: argsText, run: runAddTodo},
		{kind: "todo", name: "list", usage: "mtqg todo list [--all]", summary: "List the todos that are open (--all: all)", args: argsNone, all: true, run: runListTodos},
		{kind: "todo", name: "done", usage: "mtqg todo done <id>", summary: "Mark a todo as done", args: argsID, ids: idsOpen, run: runDone},
		{kind: "todo", name: "reopen", usage: "mtqg todo reopen <id>", summary: "Mark a todo as open again", args: argsID, ids: idsDone, run: runReopen},

		{kind: "qa", name: "add", usage: "mtqg qa add <question> | mtqg qa add <question-id> <answer>", summary: "Ask a question, or answer one", args: argsText, ids: idsParents, run: runAddThread},
		{kind: "qa", name: "list", usage: "mtqg qa list [--all]", summary: "List the questions that are open (--all: all)", args: argsNone, all: true, run: runListThread},
		{kind: "qa", name: "done", usage: "mtqg qa done <question-id>", summary: "Close a question", args: argsID, ids: idsOpen, run: runDone},
		{kind: "qa", name: "reopen", usage: "mtqg qa reopen <question-id>", summary: "Open a question again", args: argsID, ids: idsDone, run: runReopen},
		{kind: "bug", name: "add", usage: "mtqg bug add <bug> | mtqg bug add <bug-id> <reply>", summary: "Report a bug, or reply to one", args: argsText, ids: idsParents, run: runAddThread},
		{kind: "bug", name: "list", usage: "mtqg bug list [--all]", summary: "List the bugs that are open (--all: all)", args: argsNone, all: true, run: runListThread},
		{kind: "bug", name: "done", usage: "mtqg bug done <bug-id>", summary: "Close a bug", args: argsID, ids: idsOpen, run: runDone},
		{kind: "bug", name: "reopen", usage: "mtqg bug reopen <bug-id>", summary: "Open a bug again", args: argsID, ids: idsDone, run: runReopen},
		{kind: "glossary", name: "add", usage: "mtqg glossary add <word> [<definition>]", summary: "Define a term", args: argsText, minWords: 1, run: runAddGlossary},
		{kind: "glossary", name: "list", usage: "mtqg glossary list", summary: "List the terms", args: argsNone, run: runListGlossary},
		{kind: "rule", name: "add", usage: "mtqg rule add <text>", summary: "Record a rule", args: argsText, run: runAddRule},
		{kind: "rule", name: "list", usage: "mtqg rule list", summary: "List the rules", args: argsNone, run: runListRules},

		{name: "init", usage: "mtqg init [--agent claude-code] [-n]", summary: "Create .mtqg/ in this repository, optionally wiring up an agent (-n: only report)", args: argsNone, dryRun: true, values: []string{"--agent"}, run: runInit},
		{name: "status", usage: "mtqg status", summary: "Show what is open and what is not committed", args: argsNone, run: runStatus},
		{name: "version", usage: "mtqg version", summary: "Show the version of mtqg and of the repository's format", args: argsNone, run: runVersion},
		{name: "help", usage: "mtqg help", summary: "List the commands", args: argsNone, run: runHelp},

		{name: "edit", usage: "mtqg edit <id> [<text>]", summary: "Replace the text of a record", args: argsIDText, ids: idsAll, run: runEdit},
		{name: "delete", usage: "mtqg delete <id>", summary: "Hide a record", args: argsID, ids: idsAll, run: runDelete},
		{name: "undo", usage: "mtqg undo", summary: "Remove the last line you wrote from this terminal", args: argsNone, run: runUndo},
		{name: "log", usage: "mtqg log [--limit N] [--kind K]", summary: "Show the newest records of all kinds", args: argsNone, values: []string{"--limit", "--kind"}, run: runLog},
		{name: "show", usage: "mtqg show <id>", summary: "Show a record in full, with its history", args: argsID, ids: idsAll, run: runShow},
		{name: "search", usage: "mtqg search <text>", summary: "Find the records whose text contains a text", args: argsText, minWords: 1, run: runSearch},
		{name: "review", usage: "mtqg review", summary: "Show concurrent changes, duplicate definitions and answers with no parent", args: argsNone, run: runReview},
		{name: "context", usage: "mtqg context [--max-tokens N]", summary: "Summarize the records for an AI agent", args: argsNone, values: []string{"--max-tokens"}, run: runContext},
		{name: "format", usage: "mtqg format [--mark] [file]", summary: "Show the event lines found in any text", args: argsAny, mark: true, run: runFormat},
		{name: "archive", usage: "mtqg archive <start>..<end> [-n]", summary: "Move the items of a date range out of view (-n: only report)", args: argsAny, dryRun: true, run: runArchive},

		{name: "hook", usage: "mtqg hook <agent> <event>", summary: "Run one hook event for an agent (called from the agent's own configuration)", args: argsAny, choices: initAgents, run: runHook},
		{name: "mcp", usage: "mtqg mcp", summary: "Run an MCP server on standard input and output, for an AI agent", args: argsNone, run: runMCP},
		{name: "completion", usage: "mtqg completion <shell>", summary: "Print the completion script of a shell (" + strings.Join(shells, ", ") + ")", args: argsAny, choices: shells, run: runCompletion},
		{name: "candidates", usage: "mtqg candidates [--word=<partial>] -- <word>...", summary: "List what can come next on a command line (the completion scripts call it)", args: argsAny, values: []string{"--word"}, run: runCandidates},
	}
}

// findKind returns the kind that a word names, spelled out or by its letter.
func findKind(word string) (kindSpec, bool) {
	for _, k := range kinds {
		if word == k.name || word == k.short {
			return k, true
		}
	}
	return kindSpec{}, false
}

func verbsOf(kind string) []string {
	var verbs []string
	for _, cmd := range commands {
		if cmd.kind == kind {
			verbs = append(verbs, cmd.name)
		}
	}
	return verbs
}

// commandsOf returns the commands of a kind, in the order help shows them, for
// "mtqg <kind> -h".
func commandsOf(kind string) []*command {
	var cmds []*command
	for _, cmd := range commands {
		if cmd.kind == kind {
			cmds = append(cmds, cmd)
		}
	}
	return cmds
}

// invocation is a command line, read.
type invocation struct {
	cmd *command

	// kindHelp is the name of a kind, when help was asked for with a kind but no
	// verb yet ("mtqg todo -h"): cmd is nil, and printKindHelp is what runs.
	kindHelp string

	dir     string // -C
	all     bool
	fullID  bool
	noColor bool
	json    bool
	mark    bool
	dryRun  bool
	help    bool

	// values holds the options that take a value, by their names (--limit).
	values map[string]string

	// words are what follows the command: the text of a record, or the ID.
	words []string
}

// usageError is a mistake in the command line.
type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

// parseArgs reads a command line: options, then the command (a kind and a verb,
// or a command word), then its arguments.
//
// Options are -C <path>, --all, --mark, -n or --dry-run, --full-id, --no-color,
// --json and -h or --help. Before the command they may stand anywhere. After it, a command that takes a text
// (memo add, todo add) reads options only up to the first word of the text: from
// there on every word is text, even one that starts with -. "--" ends the
// options. Other commands read options anywhere. A single "-" is a word (it
// means standard input), not an option.
func parseArgs(args []string) (*invocation, error) {
	inv := &invocation{}
	i := 0
	usage := "" // the command's usage, once it is known, for complaints

	// consumeOption reads args[i] if it is an option, and says whether it was.
	consumeOption := func() (bool, error) {
		arg := args[i]
		if len(arg) < 2 || arg[0] != '-' || arg == "--" {
			return false, nil
		}
		name, value, hasValue := strings.Cut(arg, "=")
		switch {
		case inv.cmd != nil && inv.cmd.takesValue(name):
			// --limit 5 or --limit=5. The value is the next word, whatever it looks
			// like.
			if !hasValue {
				if i+1 >= len(args) {
					return true, &usageError{msgOptionNeedsValue(name)}
				}
				value = args[i+1]
				i++
			}
			if inv.values == nil {
				inv.values = map[string]string{}
			}
			inv.values[name] = value
		case strings.HasPrefix(arg, "-C"):
			if arg == "-C" {
				if i+1 >= len(args) {
					return true, &usageError{msgOptionNeedsValue("-C")}
				}
				inv.dir = args[i+1]
				i += 2
				return true, nil
			}
			inv.dir = arg[2:]
		case arg == "--all":
			inv.all = true
		case arg == "--full-id":
			inv.fullID = true
		case arg == "--no-color":
			inv.noColor = true
		case arg == "--json":
			inv.json = true
		case arg == "--mark":
			inv.mark = true
		case arg == "-n" || arg == "--dry-run":
			inv.dryRun = true
		case arg == "-h" || arg == "--help":
			inv.help = true
		default:
			return true, &usageError{msgUnknownOption(arg, usage)}
		}
		i++
		return true, nil
	}

	// The command.
	var kind kindSpec
	haveKind := false
	for inv.cmd == nil {
		if i >= len(args) {
			switch {
			case inv.help && haveKind:
				inv.kindHelp = kind.name
				return inv, nil
			case inv.help:
				inv.cmd = helpCommand()
				return inv, nil
			case haveKind:
				return nil, &usageError{msgMissingVerb(kind.name, verbsOf(kind.name))}
			default:
				return nil, &usageError{msgNoCommand()}
			}
		}
		if isOpt, err := consumeOption(); err != nil {
			return nil, err
		} else if isOpt {
			continue
		}
		word := args[i]
		i++
		switch {
		case !haveKind:
			if k, ok := findKind(word); ok {
				kind, haveKind = k, true
				continue
			}
			inv.cmd = lookup("", word)
			if inv.cmd == nil {
				return nil, &usageError{msgUnknownCommand(word)}
			}
		default:
			inv.cmd = lookup(kind.name, word)
			if inv.cmd == nil {
				return nil, &usageError{msgUnknownVerb(kind.name, word, verbsOf(kind.name))}
			}
		}
	}

	// What follows the command.
	usage = inv.cmd.usage
	optionsOnlyBeforeText := inv.cmd.args == argsText || inv.cmd.args == argsIDText
	inOptions := true
	for i < len(args) {
		arg := args[i]
		if inOptions && arg == "--" {
			inOptions = false
			i++
			continue
		}
		if inOptions {
			if isOpt, err := consumeOption(); err != nil {
				return nil, err
			} else if isOpt {
				continue
			}
		}
		inv.words = append(inv.words, arg)
		i++
		if optionsOnlyBeforeText {
			inOptions = false
		}
	}

	if inv.all && !inv.cmd.all && !inv.help {
		return nil, &usageError{msgUnknownOption("--all", inv.cmd.usage)}
	}
	if inv.mark && !inv.cmd.mark && !inv.help {
		return nil, &usageError{msgUnknownOption("--mark", inv.cmd.usage)}
	}
	if inv.dryRun && !inv.cmd.dryRun && !inv.help {
		return nil, &usageError{msgUnknownOption("--dry-run", inv.cmd.usage)}
	}
	if !inv.help {
		if err := checkArity(inv.cmd, inv.words); err != nil {
			return nil, err
		}
	}
	return inv, nil
}

// wantsJSON says whether --json is among the arguments, up to a "--". It is
// asked when the command line could not be read, so that the complaint about it
// is in the form that was asked for.
func wantsJSON(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "--json":
			return true
		case "--":
			return false
		}
	}
	return false
}

// checkArity says whether a command was given the right number of words.
func checkArity(cmd *command, words []string) error {
	switch cmd.args {
	case argsText:
		if len(words) < cmd.minWords {
			return &usageError{msgMissingArgument(cmd.usage)}
		}
	case argsNone:
		if len(words) > 0 {
			return &usageError{msgTooManyArguments(cmd.usage)}
		}
	case argsID:
		switch {
		case len(words) == 0:
			return &usageError{msgMissingArgument(cmd.usage)}
		case len(words) > 1:
			return &usageError{msgTooManyArguments(cmd.usage)}
		}
	case argsIDText:
		// The ID alone is enough: with no text, the editor opens.
		if len(words) == 0 {
			return &usageError{msgMissingArgument(cmd.usage)}
		}
	}
	return nil
}

// lookup finds a command by its kind and name.
func lookup(kind, name string) *command {
	for _, cmd := range commands {
		if cmd.kind == kind && cmd.name == name {
			return cmd
		}
	}
	return nil
}

func helpCommand() *command { return lookup("", "help") }
