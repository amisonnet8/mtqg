package cli

import (
	"strings"
)

// The grammar of the command line is data: the kinds, the commands and what
// each takes. Reading the arguments, the list of commands in help, and (later)
// the completion of the shell all come from these tables, so they cannot
// disagree.

// A kind of record and the letter it can be shortened to.
type kindSpec struct {
	name  string
	short string
}

var kinds = []kindSpec{
	{"memo", "m"},
	{"todo", "t"},
	{"qa", "q"},
	{"glossary", "g"},
}

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

// command is one command of mtqg. A command with no run is known, so that it
// can be named and asked for, but not yet available.
type command struct {
	kind    string // the name of the kind, or "" for a command without one
	name    string // the verb, or the command word
	usage   string
	summary string
	args    argMode
	all     bool // accepts --all

	// values are the options that take a value, as --limit 5 or --limit=5.
	values []string

	// minWords is how many words a command that takes a text must be given. A
	// command that takes text and no minimum opens the editor when it is given
	// none (memo add).
	minWords int

	run func(c *ctx) int
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
		{kind: "todo", name: "done", usage: "mtqg todo done <id>", summary: "Mark a todo as done", args: argsID, run: runDone},
		{kind: "todo", name: "reopen", usage: "mtqg todo reopen <id>", summary: "Mark a todo as open again", args: argsID, run: runReopen},

		{kind: "qa", name: "add", usage: "mtqg qa add <question> | mtqg qa add <question-id> <answer>", summary: "Ask a question, or answer one", args: argsText, run: runAddQA},
		{kind: "qa", name: "list", usage: "mtqg qa list [--all]", summary: "List the questions that are open (--all: all)", args: argsNone, all: true, run: runListQA},
		{kind: "qa", name: "done", usage: "mtqg qa done <question-id>", summary: "Close a question", args: argsID, run: runDone},
		{kind: "qa", name: "reopen", usage: "mtqg qa reopen <question-id>", summary: "Open a question again", args: argsID, run: runReopen},
		{kind: "glossary", name: "add", usage: "mtqg glossary add <word> <definition>", summary: "Define a term", args: argsText, minWords: 2, run: runAddGlossary},
		{kind: "glossary", name: "list", usage: "mtqg glossary list", summary: "List the terms", args: argsNone, run: runListGlossary},

		{name: "init", usage: "mtqg init", summary: "Create .mtqg/ in this repository", args: argsNone, run: runInit},
		{name: "status", usage: "mtqg status", summary: "Show what is open and what is not committed", args: argsNone, run: runStatus},
		{name: "version", usage: "mtqg version", summary: "Show the version of mtqg and of the repository's format", args: argsNone, run: runVersion},
		{name: "help", usage: "mtqg help", summary: "List the commands", args: argsNone, run: runHelp},

		{name: "edit", usage: "mtqg edit <id> <text>", summary: "Replace the text of a record", args: argsIDText},
		{name: "delete", usage: "mtqg delete <id>", summary: "Hide a record", args: argsID},
		{name: "undo", usage: "mtqg undo", summary: "Remove the last line written from this terminal", args: argsNone},
		{name: "log", usage: "mtqg log [--limit N] [--kind K]", summary: "Show the newest records of all kinds", args: argsNone, values: []string{"--limit", "--kind"}, run: runLog},
		{name: "show", usage: "mtqg show <id>", summary: "Show a record in full, with its history", args: argsID, run: runShow},
		{name: "search", usage: "mtqg search <text>", summary: "Search the text of the records", args: argsText},
		{name: "review", usage: "mtqg review", summary: "Show concurrent changes and duplicate definitions", args: argsNone},
		{name: "context", usage: "mtqg context [--max-tokens N]", summary: "Summarize the records for an AI agent", args: argsAny},
		{name: "format", usage: "mtqg format [file]", summary: "Show the event lines found in any text", args: argsAny},
		{name: "archive", usage: "mtqg archive <start>..<end> [-n]", summary: "Move finished items of a date range out of view", args: argsAny},
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

// invocation is a command line, read.
type invocation struct {
	cmd *command

	dir     string // -C
	all     bool
	fullID  bool
	noColor bool
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
// Options are -C <path>, --all, --full-id, --no-color and -h or --help. Before
// the command they may stand anywhere. After it, a command that takes a text
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
	if !inv.help {
		if err := checkArity(inv.cmd, inv.words); err != nil {
			return nil, err
		}
	}
	return inv, nil
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
