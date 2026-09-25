package cli

import (
	"slices"
	"strings"

	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

// candidateWidth is how many columns of a record the description of a candidate
// has: it is for the menu of a shell.
const candidateWidth = 60

// shortIDDigits is how many digits of an ID are offered when they are enough.
const shortIDDigits = 10

// candidate is one thing that can come next on a command line.
type candidate struct {
	value       string
	description string
}

// globalOptions are the options that every command takes (and that stand before
// the command too), spelled as they are completed. Whether parseArgs takes each
// of them is tested (TestCandidateOptionsAreTheOnesParseArgsTakes).
var globalOptions = []string{"-C", "--full-id", "--no-color", "--json", "-h", "--help"}

// sameOption maps an option to the one that means the same, so that one is not
// offered when the other is on the line.
var sameOption = map[string]string{"--dry-run": "-n", "--help": "-h"}

// runCandidates lists what can come next on a command line. The words after -- are
// the line so far, without mtqg, and --word is the one being typed. It is called by
// the scripts of the shells at every TAB, so it never complains: whatever the
// words are and whatever the repository holds, it exits 0 and says nothing on
// standard error.
func runCandidates(c *ctx) int {
	partial := c.inv.values["--word"]
	at := walk(c.inv.words, c.inv.dir)
	list := c.candidatesAt(at, partial)

	if c.inv.json {
		out := jsonCandidates{Command: c.inv.cmd.label(), Candidates: make([]jsonCandidate, len(list)), Count: len(list)}
		for i, cand := range list {
			out.Candidates[i] = jsonCandidate{Value: cand.value, Description: cand.description}
		}
		return c.emit(out)
	}
	for _, cand := range list {
		if cand.description == "" {
			c.println(cand.value)
		} else {
			c.println(cand.value + "\t" + cand.description)
		}
	}
	return exitOK
}

// position is what a walk over the words of a line finds out about where the next
// word stands.
type position struct {
	dir      string // where -C says to look for .mtqg/, or "" for where mtqg is
	kind     *kindSpec
	cmd      *command // nil until the command is known
	unknown  bool     // a kind, verb or command that mtqg does not have
	seen     map[string]bool
	pending  string // an option that waits for its value
	words    int    // the words after the command that are not options
	optsDone bool   // no option can come any more: after -- or the first word of a text
}

// walk reads the words the way parseArgs does, but forgives everything: a line
// that is being typed is wrong more often than not. It shares the tables of
// args.go with parseArgs and does not have a grammar of its own.
func walk(words []string, dir string) position {
	p := position{dir: dir, seen: map[string]bool{}}
	for _, w := range words {
		if p.pending != "" {
			// The value of an option: whatever it looks like.
			if p.pending == "-C" {
				p.dir = w
			}
			p.pending = ""
			continue
		}
		if w == "--" {
			if p.cmd != nil {
				p.optsDone = true
			}
			continue
		}
		if !p.optsDone && len(w) >= 2 && w[0] == '-' {
			p.option(w)
			continue
		}
		p.word(w)
	}
	return p
}

// option reads a word that is an option.
func (p *position) option(w string) {
	name, _, hasValue := strings.Cut(w, "=")
	switch {
	case p.cmd != nil && p.cmd.takesValue(name):
		p.seen[name] = true
		p.pending = name
		if hasValue {
			p.pending = ""
		}
	case w == "-C":
		p.seen["-C"] = true
		p.pending = "-C"
	case strings.HasPrefix(w, "-C"):
		p.seen["-C"] = true
		p.dir = w[2:]
	default:
		if same, ok := sameOption[name]; ok {
			name = same
		}
		p.seen[name] = true
	}
}

// word reads a word that is not an option: the kind and the verb, the command, or
// what the command takes.
func (p *position) word(w string) {
	switch {
	case p.cmd != nil:
		p.words++
		if p.cmd.args == argsText || p.cmd.args == argsIDText {
			p.optsDone = true
		}
	case p.kind == nil:
		if k, ok := findKind(w); ok {
			p.kind = &k
			return
		}
		p.cmd = lookup("", w)
		p.unknown = p.cmd == nil
	default:
		p.cmd = lookup(p.kind.name, w)
		p.unknown = p.cmd == nil
	}
}

// candidatesAt lists what can be typed at a position, of those that start with
// what has been typed of the word.
func (c *ctx) candidatesAt(p position, partial string) []candidate {
	var list []candidate
	switch {
	case p.unknown, p.pending == "-C":
		// Nothing to offer, or a path, which the shell knows about.
	case p.pending == "--kind":
		for _, k := range kinds {
			list = append(list, candidate{value: k.name})
		}
	case p.pending == "--agent":
		for _, a := range initAgents {
			list = append(list, candidate{value: a})
		}
	case p.pending != "":
		// The value of another option: a number.
	case strings.HasPrefix(partial, "-") && !p.optsDone:
		list = optionsAt(p)
	case p.cmd == nil && p.kind == nil:
		list = firstWords()
	case p.cmd == nil:
		for _, cmd := range commands {
			if cmd.kind == p.kind.name && cmd.run != nil {
				list = append(list, candidate{value: cmd.name, description: cmd.summary})
			}
		}
	case p.words == 0:
		list = c.firstArgument(p, partial)
	}
	return withPrefix(list, partial)
}

// withPrefix keeps the candidates that start with what has been typed.
func withPrefix(list []candidate, partial string) []candidate {
	return slices.DeleteFunc(list, func(cand candidate) bool { return !strings.HasPrefix(cand.value, partial) })
}

// firstWords are the kinds and the commands that have no kind. A kind is not
// offered as the letter it can be shortened to.
func firstWords() []candidate {
	var list []candidate
	for _, k := range kinds {
		list = append(list, candidate{value: k.name})
	}
	for _, cmd := range commands {
		if cmd.kind == "" && cmd.run != nil {
			list = append(list, candidate{value: cmd.name, description: cmd.summary})
		}
	}
	return list
}

// optionsAt are the options that can stand at a position and are not on the line
// yet: those of the command, and the ones that every command has.
func optionsAt(p position) []candidate {
	var names []string
	if p.cmd != nil {
		if p.cmd.all {
			names = append(names, "--all")
		}
		if p.cmd.mark {
			names = append(names, "--mark")
		}
		if p.cmd.dryRun {
			names = append(names, "-n", "--dry-run")
		}
		names = append(names, p.cmd.values...)
	}
	names = append(names, globalOptions...)

	var list []candidate
	for _, name := range names {
		key := name
		if same, ok := sameOption[name]; ok {
			key = same
		}
		if !p.seen[key] {
			list = append(list, candidate{value: name})
		}
	}
	return list
}

// firstArgument is what the first word after the command can be: one of a few
// words, or the ID of a record.
func (c *ctx) firstArgument(p position, partial string) []candidate {
	if p.cmd.choices != nil {
		list := make([]candidate, len(p.cmd.choices))
		for i, w := range p.cmd.choices {
			list[i] = candidate{value: w}
		}
		return list
	}
	if p.cmd.ids == idsNone {
		return nil
	}
	state := c.quietState(p.dir)
	if state == nil {
		return nil
	}
	return idCandidates(state, p.cmd, partial)
}

// quietState reads the journal for a completion. Anything that goes wrong (no
// .mtqg/, a format from the future) is no records, and what was skipped is not
// said: a warning at every TAB would break the line being typed.
func (c *ctx) quietState(dir string) *model.State {
	if dir == "" {
		wd, err := c.env.Getwd()
		if err != nil {
			return nil
		}
		dir = wd
	}
	j, err := journal.Open(dir, journal.Options{})
	if err != nil {
		return nil
	}
	result, err := j.Read()
	if err != nil {
		return nil
	}
	return model.Build(result.Events)
}

// idCandidates are the records that a command takes the ID of, newest first, each
// with its ID and its text. The ID is its first 10 digits, which is what every list
// shows; the full ID when more has been typed than that, or when the 10 digits fit
// another record too, so that what is put in names one record.
func idCandidates(state *model.State, cmd *command, partial string) []candidate {
	all := state.All()
	digits := map[string]int{}
	for _, r := range all {
		digits[prefix(r.ID, shortIDDigits)]++
	}

	typ := cmd.spec().typ
	var list []candidate
	for _, r := range slices.Backward(all) {
		if !takesID(cmd.ids, typ, r) || !strings.HasPrefix(r.ID, partial) {
			continue
		}
		value := prefix(r.ID, shortIDDigits)
		if len(partial) > shortIDDigits || digits[value] > 1 {
			value = r.ID
		}
		list = append(list, candidate{value: value, description: describe(r)})
	}
	return list
}

// takesID says whether a record is one that a command takes the ID of.
func takesID(set idSet, typ string, r *model.Record) bool {
	if set == idsAll {
		return true
	}
	if r.Type != typ || r.Kind() != model.ParentKind(typ) {
		return false
	}
	switch set {
	case idsOpen:
		return r.Status == journal.StatusOpen
	case idsDone:
		return r.Status == journal.StatusDone
	default: // idsParents: the questions or the bugs, in either state
		return true
	}
}

func prefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// describe is what a record is called in the menu of a shell: its text on one line,
// and for an entry of the glossary the word before it.
func describe(r *model.Record) string {
	text := r.Text
	if r.Word != "" {
		text = r.Word + ": " + text
	}
	return truncate(oneLine(text), candidateWidth)
}
