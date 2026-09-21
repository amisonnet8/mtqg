package cli

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"time"

	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

// The shape of --json is here and nowhere else. It is a promise to programs
// (docs/reference/cli.md "JSON output"): one object with "command" first, full
// IDs, UTC times, text as it was written, and no field taken away once it is
// there. A command turns what it found into these types and gives them to emit.

// jsonOptions spells the escaping out, as the journal does: only what JSON
// requires is escaped, so a text with < or & or U+2028 reads as it was written.
var jsonOptions = jsonv2.JoinOptions(
	jsontext.EscapeForHTML(false),
	jsontext.EscapeForJS(false),
)

// emit writes one object to standard output, indented with two spaces.
func (c *ctx) emit(v any) int {
	b, err := jsonv2.Marshal(v, jsonOptions, jsontext.WithIndent("  "))
	if err != nil {
		// Only text that is not valid UTF-8 cannot be written, and the journal does
		// not hold any.
		return c.fail(err)
	}
	_, _ = c.env.Stdout.Write(append(b, '\n'))
	return exitOK
}

// emitLine writes one object to standard error on one line: an error or a
// warning.
func (c *ctx) emitLine(v any) {
	b, err := jsonv2.Marshal(v, jsonOptions)
	if err != nil {
		return
	}
	_, _ = c.env.Stderr.Write(append(b, '\n'))
}

// A record, as programs see it.
type jsonRecord struct {
	ID      string       `json:"id"`
	Kind    string       `json:"kind"`
	Word    string       `json:"word,omitempty"`
	Text    string       `json:"text,omitempty"`
	Re      string       `json:"re,omitempty"`
	Status  string       `json:"status,omitempty"`
	Author  jsonAuthor   `json:"author"`
	Created string       `json:"created,omitempty"`
	Updated string       `json:"updated,omitempty"`
	Replies []jsonRecord `json:"replies,omitzero"`
}

type jsonAuthor struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// jsonTime is a time in UTC as the journal writes it, or "" if it is not known.
func jsonTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// recordJSON is a record as it is, without replies.
func recordJSON(r *model.Record) jsonRecord {
	return jsonRecord{
		ID:      r.ID,
		Kind:    showKind(r),
		Word:    r.Word,
		Text:    r.Text,
		Re:      r.Re,
		Status:  r.Status,
		Author:  jsonAuthor{Kind: r.Author.Kind, Name: r.Author.Name},
		Created: jsonTime(r.Created),
		Updated: jsonTime(r.Updated),
	}
}

// recordsJSON converts a list. The result is never nil, so that an empty list is
// [] and not left out.
func recordsJSON(records []*model.Record) []jsonRecord {
	out := make([]jsonRecord, len(records))
	for i, r := range records {
		out[i] = recordJSON(r)
	}
	return out
}

// threadJSON is a question or a bug with all of its answers or replies, oldest
// first. The list is there, and empty, when there are none.
func threadJSON(state *model.State, r *model.Record) jsonRecord {
	out := recordJSON(r)
	out.Replies = recordsJSON(state.Replies(r.ID))
	return out
}

// label is how a command is named in the output: the kind and the verb, or the
// command word.
func (cmd *command) label() string {
	if cmd.kind == "" {
		return cmd.name
	}
	return cmd.kind + " " + cmd.name
}

// What the commands print. Each type is the object of one command or of a few
// with the same shape; the fields are in the order they are written.

type jsonRecordResult struct {
	Command string     `json:"command"`
	Record  jsonRecord `json:"record"`
}

type jsonChangeResult struct {
	Command string     `json:"command"`
	Record  jsonRecord `json:"record"`
	Changed bool       `json:"changed"`
}

type jsonMemoList struct {
	Command string       `json:"command"`
	Records []jsonRecord `json:"records"`
	Count   int          `json:"count"`
}

type jsonTodoList struct {
	Command string       `json:"command"`
	Records []jsonRecord `json:"records"`
	Open    int          `json:"open"`
	Done    int          `json:"done"`
}

type jsonGlossaryList struct {
	Command        string       `json:"command"`
	Records        []jsonRecord `json:"records"`
	Entries        int          `json:"entries"`
	DuplicateWords int          `json:"duplicate_words"`
}

type jsonLog struct {
	Command string       `json:"command"`
	Records []jsonRecord `json:"records"`
	Shown   int          `json:"shown"`
	Total   int          `json:"total"`
}

// jsonShow gives the events as they are in the journal: the format of
// docs/reference/schema.md.
type jsonShow struct {
	Command string          `json:"command"`
	Record  jsonRecord      `json:"record"`
	Events  []journal.Event `json:"events"`
}

type jsonStatus struct {
	Command                       string `json:"command"`
	OpenTodos                     int    `json:"open_todos"`
	OpenQuestions                 int    `json:"open_questions"`
	QuestionsAwaitingConfirmation int    `json:"questions_awaiting_confirmation"`
	OpenBugs                      int    `json:"open_bugs"`
	BugsAwaitingConfirmation      int    `json:"bugs_awaiting_confirmation"`
	GlossaryEntries               int    `json:"glossary_entries"`
	DuplicateWords                int    `json:"duplicate_words"`
	UncommittedRecords            *int   `json:"uncommitted_records"`
}

type jsonInit struct {
	Command string `json:"command"`
	Root    string `json:"root"`
}

type jsonVersion struct {
	Command string            `json:"command"`
	Mtqg    string            `json:"mtqg"`
	Format  jsonFormatVersion `json:"format"`
}

type jsonFormatVersion struct {
	Repository *int `json:"repository"`
	Supported  int  `json:"supported"`
}

type jsonHelp struct {
	Command  string            `json:"command"`
	Kinds    []jsonKind        `json:"kinds"`
	Commands []jsonCommandInfo `json:"commands"`
}

type jsonKind struct {
	Name  string `json:"name"`
	Short string `json:"short"`
}

type jsonCommandInfo struct {
	Command   string `json:"command"`
	Usage     string `json:"usage"`
	Summary   string `json:"summary"`
	Available bool   `json:"available"`
}

// What context prints with --json: the same content as the text, after the same
// cuts, in sections that each say how many there were.

type jsonContext struct {
	Command         string              `json:"command"`
	Repository      string              `json:"repository"`
	Branch          string              `json:"branch,omitempty"`
	Attention       []jsonAttention     `json:"attention"`
	OpenTodos       jsonSection         `json:"open_todos"`
	OpenQuestions   jsonThreadSection   `json:"open_questions"`
	OpenBugs        jsonThreadSection   `json:"open_bugs"`
	Recent          jsonSection         `json:"recent"`
	Glossary        jsonGlossarySection `json:"glossary"`
	Truncated       bool                `json:"truncated"`
	MaxTokens       *int                `json:"max_tokens"`
	EstimatedTokens int                 `json:"estimated_tokens"`
}

type jsonAttention struct {
	Kind  string `json:"kind"`
	Word  string `json:"word,omitempty"`
	Count int    `json:"count,omitzero"`
}

type jsonSection struct {
	Total   int          `json:"total"`
	Records []jsonRecord `json:"records"`
}

type jsonThreadSection struct {
	Total   int                `json:"total"`
	Records []jsonThreadRecord `json:"records"`
}

// jsonThreadRecord is a question or a bug, with how many answers or replies it
// has and the latest of them (if it was not left out).
type jsonThreadRecord struct {
	jsonRecord
	ReplyCount  int         `json:"reply_count"`
	LatestReply *jsonRecord `json:"latest_reply,omitempty"`
}

type jsonGlossarySection struct {
	Total   int                `json:"total"`
	Records []jsonGlossaryItem `json:"records"`
}

// jsonGlossaryItem is an entry of the glossary, or only its word when the
// definitions were left out (no id, no text).
type jsonGlossaryItem struct {
	ID          string      `json:"id,omitempty"`
	Kind        string      `json:"kind,omitempty"`
	Word        string      `json:"word"`
	Text        string      `json:"text,omitempty"`
	Author      *jsonAuthor `json:"author,omitempty"`
	Created     string      `json:"created,omitempty"`
	Updated     string      `json:"updated,omitempty"`
	Definitions int         `json:"definitions"`
}
