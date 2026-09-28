package cli

import (
	"context"
	"errors"
	"slices"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

// The tools mtqg mcp exposes (docs/reference/cli.md "MCP server"). Each one
// that changes something matches one CLI command and returns the same JSON
// object that command's --json would (json.go); context and search are
// shaped for size instead. What is not exposed - delete, undo, archive,
// review, format - stays a command typed by a person.

// Input types. Their json struct tags give the field names of a tool call's
// arguments; their jsonschema tags become the description a client shows.

type textInput struct {
	Text string `json:"text" jsonschema:"the text of the record"`
}

type idInput struct {
	ID string `json:"id" jsonschema:"the record's ID (a prefix of at least 4 hex digits is enough)"`
}

type idTextInput struct {
	ID   string `json:"id" jsonschema:"the record's ID (a prefix of at least 4 hex digits is enough)"`
	Text string `json:"text" jsonschema:"the new text"`
}

type glossaryInput struct {
	Word       string `json:"word" jsonschema:"the term being defined"`
	Definition string `json:"definition" jsonschema:"the definition"`
}

type contextInput struct {
	MaxTokens int `json:"max_tokens,omitempty" jsonschema:"the approximate token budget (default 2000)"`
}

type searchInput struct {
	Query string `json:"query" jsonschema:"the text to search for"`
	Limit int    `json:"limit,omitempty" jsonschema:"the maximum number of records to return, newest first (default 20)"`
}

// addTools registers every tool on server. env and dir (what -C gave mtqg
// mcp, "" to look from the working directory) are fixed for the life of the
// server; the repository and the author are worked out fresh on every call
// (mcp.go's mcpCtx).
func addTools(server *mcp.Server, env Env, dir string) {
	ctxFor := func(req *mcp.CallToolRequest, kind, name string) (*ctx, *mcp.CallToolResult) {
		return mcpCtx(env, dir, mcpClientInfo(req), kind, name)
	}

	add := func(name, kind, cmdName, description string, create func(text string) (journal.Event, error)) {
		mcp.AddTool(server, &mcp.Tool{Name: name, Description: description}, func(_ context.Context, req *mcp.CallToolRequest, in textInput) (*mcp.CallToolResult, any, error) {
			c, errResult := ctxFor(req, kind, cmdName)
			if errResult != nil {
				return errResult, nil, nil
			}
			return mcpAdd(c, create, in.Text)
		})
	}
	add("memo_add", "memo", "add", "Record a memo: an observation or something worth remembering. Not for a defect (something that should have worked but did not) — use bug_report for that, even if already fixed.", model.MemoCreate)
	add("rule_add", "rule", "add", "Record a rule: something that, once read, can be followed as it stands.", model.RuleCreate)
	add("todo_add", "todo", "add", "Record a todo.", model.TodoCreate)
	add("qa_ask", "qa", "add", "Ask a question that will wait for an answer.", func(text string) (journal.Event, error) {
		return model.ParentCreate(journal.TypeQA, text)
	})
	add("bug_report", "bug", "add", "Report a bug: something that should have worked but did not, including one already fixed (fixing it is not a reason to skip recording it, or to record it as a memo instead).", func(text string) (journal.Event, error) {
		return model.ParentCreate(journal.TypeBug, text)
	})

	changeStatus := func(name, kind, cmdName, description, typ, status string) {
		mcp.AddTool(server, &mcp.Tool{Name: name, Description: description}, func(_ context.Context, req *mcp.CallToolRequest, in idInput) (*mcp.CallToolResult, any, error) {
			c, errResult := ctxFor(req, kind, cmdName)
			if errResult != nil {
				return errResult, nil, nil
			}
			return mcpChangeStatus(c, typ, in.ID, status)
		})
	}
	changeStatus("todo_done", "todo", "done", "Mark a todo done.", journal.TypeTodo, journal.StatusDone)
	changeStatus("todo_reopen", "todo", "reopen", "Open a todo again.", journal.TypeTodo, journal.StatusOpen)
	changeStatus("qa_done", "qa", "done", "Close a question.", journal.TypeQA, journal.StatusDone)
	changeStatus("qa_reopen", "qa", "reopen", "Open a question again.", journal.TypeQA, journal.StatusOpen)
	changeStatus("bug_done", "bug", "done", "Close a bug.", journal.TypeBug, journal.StatusDone)
	changeStatus("bug_reopen", "bug", "reopen", "Open a bug again.", journal.TypeBug, journal.StatusOpen)

	mcp.AddTool(server, &mcp.Tool{Name: "qa_answer", Description: "Answer an existing question."}, func(_ context.Context, req *mcp.CallToolRequest, in idTextInput) (*mcp.CallToolResult, any, error) {
		c, errResult := ctxFor(req, "qa", "add")
		if errResult != nil {
			return errResult, nil, nil
		}
		return mcpReplyAdd(c, journal.TypeQA, in.ID, in.Text)
	})
	mcp.AddTool(server, &mcp.Tool{Name: "bug_reply", Description: "Reply to an existing bug."}, func(_ context.Context, req *mcp.CallToolRequest, in idTextInput) (*mcp.CallToolResult, any, error) {
		c, errResult := ctxFor(req, "bug", "add")
		if errResult != nil {
			return errResult, nil, nil
		}
		return mcpReplyAdd(c, journal.TypeBug, in.ID, in.Text)
	})

	mcp.AddTool(server, &mcp.Tool{Name: "glossary_define", Description: "Define a term."}, func(_ context.Context, req *mcp.CallToolRequest, in glossaryInput) (*mcp.CallToolResult, any, error) {
		c, errResult := ctxFor(req, "glossary", "add")
		if errResult != nil {
			return errResult, nil, nil
		}
		return mcpGlossaryDefine(c, in.Word, in.Definition)
	})

	mcp.AddTool(server, &mcp.Tool{Name: "edit", Description: "Replace the text of a record with new text given directly (never opens an editor)."}, func(_ context.Context, req *mcp.CallToolRequest, in idTextInput) (*mcp.CallToolResult, any, error) {
		c, errResult := ctxFor(req, "", "edit")
		if errResult != nil {
			return errResult, nil, nil
		}
		return mcpEdit(c, in.ID, in.Text)
	})

	mcp.AddTool(server, &mcp.Tool{Name: "context", Description: "Summarize the records, the same as mtqg context."}, func(_ context.Context, req *mcp.CallToolRequest, in contextInput) (*mcp.CallToolResult, any, error) {
		c, errResult := ctxFor(req, "", "context")
		if errResult != nil {
			return errResult, nil, nil
		}
		return mcpContext(c, in.MaxTokens)
	})

	mcp.AddTool(server, &mcp.Tool{Name: "show", Description: "Show one record in full, with its history."}, func(_ context.Context, req *mcp.CallToolRequest, in idInput) (*mcp.CallToolResult, any, error) {
		c, errResult := ctxFor(req, "", "show")
		if errResult != nil {
			return errResult, nil, nil
		}
		return mcpShow(c, in.ID)
	})

	mcp.AddTool(server, &mcp.Tool{Name: "search", Description: "Find the records whose text contains a text, newest first."}, func(_ context.Context, req *mcp.CallToolRequest, in searchInput) (*mcp.CallToolResult, any, error) {
		c, errResult := ctxFor(req, "", "search")
		if errResult != nil {
			return errResult, nil, nil
		}
		return mcpSearch(c, in.Query, in.Limit)
	})
}

// mcpAdd is the MCP equivalent of ctx.add: it writes a new record from text
// given directly (never $EDITOR, never standard input), and returns the
// result as a tool result instead of printing it.
func mcpAdd(c *ctx, create func(text string) (journal.Event, error), rawText string) (*mcp.CallToolResult, any, error) {
	text, err := cleanText(rawText)
	if err != nil {
		return mcpError(c, err)
	}
	ev, err := create(text)
	if err != nil {
		return mcpError(c, err)
	}
	w, _, _, err := c.writerAs()
	if err != nil {
		return mcpError(c, err)
	}
	written, err := w.Append(ev)
	if err != nil {
		return mcpError(c, err)
	}
	rec := model.Build([]journal.Event{written}).Record(written.ID)
	return mcpResult(jsonRecordResult{Command: c.inv.cmd.label(), Record: recordJSON(rec)})
}

// mcpReplyAdd is the MCP equivalent of runAddThread's reply path: an answer to
// a question, or a reply to a bug, found by ID rather than guessed from the
// shape of the first word (docs/reference/cli.md "MCP server").
func mcpReplyAdd(c *ctx, typ, id, rawText string) (*mcp.CallToolResult, any, error) {
	j, _, _, err := c.writerAs()
	if err != nil {
		return mcpError(c, err)
	}
	state, err := c.load(j)
	if err != nil {
		return mcpError(c, err)
	}
	parent, err := state.ResolveKind(id, model.ParentKind(typ))
	if err != nil {
		return mcpError(c, err)
	}
	text, err := cleanText(rawText)
	if err != nil {
		return mcpError(c, err)
	}
	ev, err := model.ReplyCreate(parent, text)
	if err != nil {
		return mcpError(c, err)
	}
	written, err := j.Append(ev)
	if err != nil {
		return mcpError(c, err)
	}
	rec := model.Build([]journal.Event{written}).Record(written.ID)
	return mcpResult(jsonRecordResult{Command: c.inv.cmd.label(), Record: recordJSON(rec)})
}

// mcpChangeStatus is the MCP equivalent of ctx.changeStatus.
func mcpChangeStatus(c *ctx, typ, id, status string) (*mcp.CallToolResult, any, error) {
	j, err := c.reader()
	if err != nil {
		return mcpError(c, err)
	}
	state, err := c.load(j)
	if err != nil {
		return mcpError(c, err)
	}
	rec, err := state.ResolveKind(id, model.ParentKind(typ))
	if err != nil {
		return mcpError(c, err)
	}
	ev, err := model.SetStatus(rec, status)
	if errors.Is(err, model.ErrNoChange) {
		return mcpResult(jsonChangeResult{Command: c.inv.cmd.label(), Record: recordJSON(rec), Changed: false})
	}
	if err != nil {
		return mcpError(c, err)
	}
	w, _, _, err := c.writerAs()
	if err != nil {
		return mcpError(c, err)
	}
	if _, err := w.Append(ev); err != nil {
		return mcpError(c, err)
	}
	current := rec
	if result, err := w.Read(); err == nil {
		if r := model.Build(result.Events).Record(rec.ID); r != nil {
			current = r
		}
	}
	return mcpResult(jsonChangeResult{Command: c.inv.cmd.label(), Record: recordJSON(current), Changed: true})
}

// mcpGlossaryDefine is the MCP equivalent of runAddGlossary.
func mcpGlossaryDefine(c *ctx, word, rawText string) (*mcp.CallToolResult, any, error) {
	text, err := cleanText(rawText)
	if err != nil {
		return mcpError(c, err)
	}
	ev, err := model.GlossaryCreate(word, text)
	if err != nil {
		return mcpError(c, err)
	}
	w, _, _, err := c.writerAs()
	if err != nil {
		return mcpError(c, err)
	}
	written, err := w.Append(ev)
	if err != nil {
		return mcpError(c, err)
	}
	rec := model.Build([]journal.Event{written}).Record(written.ID)
	return mcpResult(jsonRecordResult{Command: c.inv.cmd.label(), Record: recordJSON(rec)})
}

// mcpEdit is the MCP equivalent of runEdit: the new text is always given
// directly, so $EDITOR never opens.
func mcpEdit(c *ctx, id, rawText string) (*mcp.CallToolResult, any, error) {
	j, _, _, err := c.writerAs()
	if err != nil {
		return mcpError(c, err)
	}
	state, err := c.load(j)
	if err != nil {
		return mcpError(c, err)
	}
	rec, err := state.Resolve(id)
	if err != nil {
		return mcpError(c, err)
	}
	text, err := cleanText(rawText)
	if err != nil {
		return mcpError(c, err)
	}
	ev, err := model.EditText(rec, text)
	if errors.Is(err, model.ErrNoChange) {
		return mcpResult(jsonChangeResult{Command: c.inv.cmd.label(), Record: recordJSON(rec), Changed: false})
	}
	if err != nil {
		return mcpError(c, err)
	}
	written, err := j.Append(ev)
	if err != nil {
		return mcpError(c, err)
	}
	current := model.Build(append(slices.Clone(rec.Events), written)).Record(rec.ID)
	return mcpResult(jsonChangeResult{Command: c.inv.cmd.label(), Record: recordJSON(current), Changed: true})
}

// mcpContext is the MCP equivalent of runContext: the same text mtqg context
// prints, as the tool's text content (not JSON: context is one of the two
// exceptions, docs/reference/cli.md "MCP server").
func mcpContext(c *ctx, maxTokens int) (*mcp.CallToolResult, any, error) {
	budget := defaultContextTokens
	if maxTokens > 0 {
		budget = maxTokens
	}
	text, _, _, _, err := c.contextDocument(budget)
	if err != nil {
		return mcpError(c, err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, nil, nil
}

// mcpShow is the MCP equivalent of runShow's --json path.
func mcpShow(c *ctx, id string) (*mcp.CallToolResult, any, error) {
	j, err := c.reader()
	if err != nil {
		return mcpError(c, err)
	}
	state, err := c.load(j)
	if err != nil {
		return mcpError(c, err)
	}
	rec, err := state.Resolve(id)
	if err != nil {
		return mcpError(c, err)
	}
	history := state.History(rec)
	events := make([]journal.Event, len(history))
	for i, e := range history {
		events[i] = e.Event
	}
	record := recordJSON(rec)
	if rec.CanHaveReplies() {
		record = threadJSON(state, rec)
	}
	return mcpResult(jsonShow{Command: c.inv.cmd.label(), Record: record, Events: events})
}

// searchSummaryLimit is how many characters of a match's text the search tool
// includes: context.go's contextText makes the same cut, but the search tool
// also reports whether it truncated (jsonSearchSummaryRecord.Truncated).
const searchSummaryLimit = 100

// defaultSearchLimit is how many matches the search tool returns when limit is
// not given.
const defaultSearchLimit = 20

// mcpSearch is the MCP equivalent of runSearch, shaped for an agent's context
// instead of full --json: fewer records (limit), each cut to
// searchSummaryLimit characters.
func mcpSearch(c *ctx, rawQuery string, limit int) (*mcp.CallToolResult, any, error) {
	query, err := cleanText(rawQuery)
	if err != nil {
		return mcpError(c, err)
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	j, err := c.reader()
	if err != nil {
		return mcpError(c, err)
	}
	state, err := c.load(j)
	if err != nil {
		return mcpError(c, err)
	}
	found := state.Search(query)
	slices.Reverse(found) // newest first
	count := len(found)
	if len(found) > limit {
		found = found[:limit]
	}
	records := make([]jsonSearchSummaryRecord, len(found))
	for i, r := range found {
		text, truncated := searchSummaryText(r.Text)
		records[i] = jsonSearchSummaryRecord{
			ID: r.ID, Kind: showKind(r), Word: r.Word, Status: r.Status,
			Text: text, Truncated: truncated, Created: jsonTime(r.Created),
		}
	}
	return mcpResult(jsonSearchSummary{Command: c.inv.cmd.label(), Query: query, Records: records, Count: count, Shown: len(records)})
}

// searchSummaryText is oneLine(s), cut to searchSummaryLimit characters if it
// is longer, and whether it was cut.
func searchSummaryText(s string) (text string, truncated bool) {
	line := oneLine(s)
	if utf8.RuneCountInString(line) <= searchSummaryLimit {
		return line, false
	}
	return string([]rune(line)[:searchSummaryLimit]), true
}
