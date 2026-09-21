package cli

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

// defaultContextTokens is how much context is meant to be, when it is not told:
// enough to carry what is open and what is agreed, and little enough to read at
// the start of every session.
const defaultContextTokens = 2000

// contextTextLimit is how many characters a text of a record is cut to. One line
// of a record must not take the budget of a section.
const contextTextLimit = 100

// maxContextWords is how many words with conflicting definitions Attention names.
const maxContextWords = 5

// runContext prints what an agent reads when a session starts. The model layer
// says what there is and what is cut first; this measures the words and cuts until
// they are within the budget.
func runContext(c *ctx) int {
	budget := defaultContextTokens
	if v, ok := c.inv.values["--max-tokens"]; ok {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return c.usageFailure(msgBadMaxTokens(v))
		}
		budget = n
	}

	j, err := c.reader()
	if err != nil {
		return c.fail(err)
	}
	state, err := c.load(j)
	if err != nil {
		return c.fail(err)
	}

	// What git says: how many records are not committed, and the branch. If git
	// cannot be run, neither is known, and neither is shown.
	root := j.Location().Root
	uncommitted, branch := 0, ""
	events, err := j.UncommittedEvents()
	switch {
	case err == nil:
		uncommitted = model.UncommittedRecords(events)
		if b, err := journal.GitBranch(root); err == nil {
			branch = b
		}
	case errors.Is(err, journal.ErrGitUnavailable):
		c.warning(warningReport{kind: kindGitUnavailable, message: "warning: " + msgGitUnavailable(errors.Unwrap(err))})
	default:
		return c.fail(err)
	}

	d := state.Context(uncommitted)
	repository := filepath.Base(root)
	view := contextView{c: c, repository: repository, branch: branch, now: c.env.Now()}
	cuts := fitToBudget(d.Steps(), budget, func(n int) int {
		return estimateTokens(strings.Join(view.lines(d.Reduced(n)), "\n") + "\n")
	})
	reduced := d.Reduced(cuts)
	lines := view.lines(reduced)
	text := strings.Join(lines, "\n") + "\n"

	if c.inv.json {
		return c.emit(view.json(c.inv.cmd.label(), reduced, cuts > 0, budget, estimateTokens(text)))
	}
	_, _ = c.env.Stdout.Write([]byte(text))
	return exitOK
}

// estimateTokens says roughly how many tokens a text is, from the number of
// characters: 4 characters of ASCII count as 1 token, and every other character
// counts as 1. It is an estimate, as no tokenizer is among the things mtqg may
// depend on, and it is not the number of tokens of any model.
func estimateTokens(s string) int {
	ascii, other := 0, 0
	for _, r := range s {
		if r < utf8.RuneSelf {
			ascii++
		} else {
			other++
		}
	}
	return (ascii+3)/4 + other
}

// fitToBudget returns how many cuts to make to get within the budget: none if
// the text fits already or there is no limit (0), the fewest that do, or all of
// them if even that is too long. cost is the size after a number of cuts. Cuts
// take away, but a note that says what was left out can make a cut cost a little
// more than the one before it, so the number found is one that fits, and not
// always the fewest.
func fitToBudget(steps, budget int, cost func(cuts int) int) int {
	if budget <= 0 || cost(0) <= budget {
		return 0
	}
	if cost(steps) > budget {
		return steps
	}
	lo, hi := 0, steps // cost(lo) is over the budget, cost(hi) is not
	for hi-lo > 1 {
		mid := (lo + hi) / 2
		if cost(mid) <= budget {
			hi = mid
		} else {
			lo = mid
		}
	}
	return hi
}

// contextView puts the data of context into words.
type contextView struct {
	c          *ctx
	repository string
	branch     string
	now        time.Time
}

// text is the first line of a record, made safe to show and cut to the limit.
func contextText(s string) string {
	line := oneLine(s)
	if utf8.RuneCountInString(line) <= contextTextLimit {
		return line
	}
	return string([]rune(line)[:contextTextLimit-3]) + "..."
}

func (v contextView) id(full string) string { return shortID(full, v.c.inv.fullID) }

func (v contextView) when(t time.Time) string { return formatTime(t, v.now, v.c.env.Location) }

func (v contextView) lines(d *model.ContextData) []string {
	out := []string{msgContextTitle(v.repository, v.branch), ""}
	out = append(out, contextGuide...)

	section := func(lines ...string) {
		out = append(out, "")
		out = append(out, lines...)
	}

	if len(d.DuplicateWords) > 0 || d.Uncommitted > 0 {
		lines := []string{msgContextAttention}
		for i, word := range d.DuplicateWords {
			if i == maxContextWords {
				lines = append(lines, msgContextMoreWords(len(d.DuplicateWords)-maxContextWords))
				break
			}
			lines = append(lines, msgContextConflictingWord(oneLine(word)))
		}
		if d.Uncommitted > 0 {
			lines = append(lines, msgContextUncommitted(d.Uncommitted))
		}
		section(lines...)
	}

	if d.TodosTotal > 0 {
		lines := []string{msgContextHeading("Open todos", d.TodosTotal)}
		if left := d.TodosTotal - len(d.Todos); left > 0 {
			lines = append(lines, msgContextOlder(left, "todo list"))
		}
		for _, r := range d.Todos {
			lines = append(lines, msgContextItem(v.id(r.ID), contextText(r.Text), oneLine(r.Author.Name), v.when(r.Created)))
		}
		section(lines...)
	}

	if d.QuestionsTotal > 0 {
		section(v.threads("Open questions", journal.TypeQA, "qa list", d.QuestionsTotal, d.Questions, d.RepliesLeft)...)
	}
	if d.BugsTotal > 0 {
		section(v.threads("Open bugs", journal.TypeBug, "bug list", d.BugsTotal, d.Bugs, d.RepliesLeft)...)
	}

	if d.RecentTotal > 0 {
		lines := []string{msgContextRecent}
		lines = append(lines, v.recent(d.Recent)...)
		if left := d.RecentTotal - len(d.Recent); left > 0 {
			lines = append(lines, msgContextMoreRecent(left))
		}
		section(lines...)
	}

	if d.GlossaryTotal > 0 {
		lines := []string{msgContextHeading("Glossary", d.GlossaryTotal)}
		for _, it := range d.Glossary {
			if it.Entry == nil {
				lines = append(lines, msgContextWord(oneLine(it.Word), it.Definitions))
				continue
			}
			lines = append(lines, "- "+v.id(it.Entry.ID)+" "+oneLine(it.Word)+": "+contextText(it.Entry.Text))
		}
		if d.DefinitionsLeft {
			lines = append(lines, msgContextDefinitionsLeft)
		}
		section(lines...)
	}

	return append(out, "", "---", msgContextFooter)
}

// threads is a section of questions or of bugs: what is open, its state and the
// latest answer or reply under each.
func (v contextView) threads(name, typ, list string, total int, threads []model.Thread, repliesLeft bool) []string {
	lines := []string{msgContextHeading(name, total)}
	if left := total - len(threads); left > 0 {
		lines = append(lines, msgContextOlder(left, list))
	}
	anyReplies := false
	for _, t := range threads {
		p := t.Parent
		state := msgContextThreadState(typ, t.ReplyCount)
		lines = append(lines, msgContextThread(v.id(p.ID), contextText(p.Text), state, oneLine(p.Author.Name), v.when(p.Created)))
		if t.Latest != nil {
			lines = append(lines, msgContextReplyLine(contextText(t.Latest.Text), oneLine(t.Latest.Author.Name), oneLine(t.Latest.Author.Kind)))
		}
		anyReplies = anyReplies || t.ReplyCount > 0
	}
	if repliesLeft && anyReplies {
		lines = append(lines, msgContextRepliesLeft(typ))
	}
	return lines
}

// recent lays out the newest records in columns: the time, the author, the kind,
// the ID and the text. An answer or a reply ends with what it belongs to.
func (v contextView) recent(records []*model.Record) []string {
	rows := make([]tableRow, len(records))
	for i, r := range records {
		text := contextText(r.Text)
		switch {
		case r.IsReply():
			text += " (to " + v.id(r.Re) + ")"
		case r.Kind() == model.KindGlossary:
			text = oneLine(r.Word) + ": " + text
		}
		rows[i] = tableRow{cells: []string{v.when(r.Created), oneLine(r.Author.Name), showKind(r), v.id(r.ID), text}}
	}
	lines := formatTable(rows, 4, -1, 0, style{})
	for i := range lines {
		lines[i] = "- " + lines[i]
	}
	return lines
}

// json is the same content as lines, for --json: after the cuts that were made,
// with how many there were of each. budget is 0 for no limit, which is null.
func (v contextView) json(command string, d *model.ContextData, truncated bool, budget, estimated int) jsonContext {
	out := jsonContext{
		Command:         command,
		Repository:      v.repository,
		Branch:          v.branch,
		Attention:       []jsonAttention{},
		OpenTodos:       jsonSection{Total: d.TodosTotal, Records: recordsJSON(d.Todos)},
		OpenQuestions:   jsonThreadSection{Total: d.QuestionsTotal, Records: threadsJSON(d.Questions)},
		OpenBugs:        jsonThreadSection{Total: d.BugsTotal, Records: threadsJSON(d.Bugs)},
		Recent:          jsonSection{Total: d.RecentTotal, Records: recordsJSON(d.Recent)},
		Glossary:        jsonGlossarySection{Total: d.GlossaryTotal, Records: make([]jsonGlossaryItem, len(d.Glossary))},
		Truncated:       truncated,
		EstimatedTokens: estimated,
	}
	if budget > 0 {
		out.MaxTokens = &budget
	}
	for _, word := range d.DuplicateWords {
		out.Attention = append(out.Attention, jsonAttention{Kind: "duplicate_word", Word: word})
	}
	if d.Uncommitted > 0 {
		out.Attention = append(out.Attention, jsonAttention{Kind: "uncommitted", Count: d.Uncommitted})
	}
	for i, it := range d.Glossary {
		item := jsonGlossaryItem{Word: it.Word, Definitions: it.Definitions}
		if it.Entry != nil {
			r := recordJSON(it.Entry)
			item.ID, item.Kind, item.Text, item.Author, item.Created, item.Updated = r.ID, r.Kind, r.Text, &r.Author, r.Created, r.Updated
		}
		out.Glossary.Records[i] = item
	}
	return out
}

func threadsJSON(threads []model.Thread) []jsonThreadRecord {
	out := make([]jsonThreadRecord, len(threads))
	for i, t := range threads {
		out[i] = jsonThreadRecord{jsonRecord: recordJSON(t.Parent), ReplyCount: t.ReplyCount}
		if t.Latest != nil {
			r := recordJSON(t.Latest)
			out[i].LatestReply = &r
		}
	}
	return out
}
