package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mcpHarness runs mtqg mcp's tools in memory, against a real repository (the
// same harness the rest of internal/cli's tests use), through a real MCP
// client session (mcp.NewInMemoryTransports): what a real client sends and
// receives, without a subprocess.
type mcpHarness struct {
	t  *testing.T
	h  *harness
	cs *mcp.ClientSession
}

// newMCPTest makes a repository with mtqg init already run, and connects one
// client named clientName to a server whose tools see that repository (dir
// "": every call looks for .mtqg/ from the working directory, the harness's
// root).
func newMCPTest(t *testing.T, clientName string) *mcpHarness {
	t.Helper()
	h := initialized(t)
	return newMCPTestIn(t, h, "", clientName)
}

func newMCPTestIn(t *testing.T, h *harness, dir, clientName string) *mcpHarness {
	t.Helper()
	env := h.env
	env.Getenv = func(k string) string { return h.vars[k] }
	env.Getwd = func() (string, error) { return h.root, nil }
	env.Now = func() time.Time { return h.now }
	if env.Location == nil {
		env.Location = time.UTC
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "mtqg", Version: "test"}, nil)
	addTools(server, env, dir)

	client := mcp.NewClient(&mcp.Implementation{Name: clientName, Version: "1.0"}, nil)
	ct, st := mcp.NewInMemoryTransports()
	ctx := context.Background()
	go func() { _, _ = server.Connect(ctx, st, nil) }()
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return &mcpHarness{t: t, h: h, cs: cs}
}

func (m *mcpHarness) call(name string, args map[string]any) *mcp.CallToolResult {
	m.t.Helper()
	res, err := m.cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		m.t.Fatal(err)
	}
	return res
}

// text is the tool result's one TextContent, whether it is a success (JSON,
// or context's plain text) or an error result.
func (m *mcpHarness) text(res *mcp.CallToolResult) string {
	m.t.Helper()
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			return tc.Text
		}
	}
	m.t.Fatal("no text content in result")
	return ""
}

func (m *mcpHarness) decode(res *mcp.CallToolResult, v any) {
	m.t.Helper()
	if err := json.Unmarshal([]byte(m.text(res)), v); err != nil {
		m.t.Fatalf("decode %q: %v", m.text(res), err)
	}
}

func TestMCPToolsWriteRecords(t *testing.T) {
	cases := []struct {
		tool string
		args map[string]any
		kind string
	}{
		{"memo_add", map[string]any{"text": "a memo"}, "memo"},
		{"rule_add", map[string]any{"text": "a rule"}, "rule"},
		{"todo_add", map[string]any{"text": "a todo"}, "todo"},
		{"qa_ask", map[string]any{"text": "a question?"}, "question"},
		{"bug_report", map[string]any{"text": "a bug"}, "bug"},
	}
	for _, c := range cases {
		t.Run(c.tool, func(t *testing.T) {
			m := newMCPTest(t, "test-agent")
			res := m.call(c.tool, c.args)
			if res.IsError {
				t.Fatalf("isError: %s", m.text(res))
			}
			var out jsonRecordResult
			m.decode(res, &out)
			if out.Record.Kind != c.kind {
				t.Errorf("kind = %q, want %q", out.Record.Kind, c.kind)
			}
			if out.Record.Author.Kind != "ai" || out.Record.Author.Name != "test-agent" {
				t.Errorf("author = %+v, want ai/test-agent", out.Record.Author)
			}
			if !strings.Contains(m.h.readJournal(), out.Record.ID) {
				t.Error("the record was not written to the journal")
			}
		})
	}
}

func TestMCPReplyTools(t *testing.T) {
	t.Run("qa_answer answers a question found by ID", func(t *testing.T) {
		m := newMCPTest(t, "test-agent")
		var q jsonRecordResult
		m.decode(m.call("qa_ask", map[string]any{"text": "why?"}), &q)

		res := m.call("qa_answer", map[string]any{"id": q.Record.ID, "text": "because"})
		if res.IsError {
			t.Fatalf("isError: %s", m.text(res))
		}
		var a jsonRecordResult
		m.decode(res, &a)
		if a.Record.Kind != "answer" || a.Record.Re != q.Record.ID {
			t.Errorf("got %+v", a.Record)
		}
	})

	t.Run("bug_reply replies to a bug found by ID", func(t *testing.T) {
		m := newMCPTest(t, "test-agent")
		var b jsonRecordResult
		m.decode(m.call("bug_report", map[string]any{"text": "it crashes"}), &b)

		res := m.call("bug_reply", map[string]any{"id": b.Record.ID, "text": "fixed"})
		var r jsonRecordResult
		m.decode(res, &r)
		if r.Record.Kind != "reply" || r.Record.Re != b.Record.ID {
			t.Errorf("got %+v", r.Record)
		}
	})

	t.Run("qa_answer given a todo's ID is wrong_kind, not a new question", func(t *testing.T) {
		m := newMCPTest(t, "test-agent")
		var todo jsonRecordResult
		m.decode(m.call("todo_add", map[string]any{"text": "do the thing"}), &todo)

		res := m.call("qa_answer", map[string]any{"id": todo.Record.ID, "text": "an answer"})
		if !res.IsError {
			t.Fatalf("expected isError, got %s", m.text(res))
		}
		var body jsonErrorBody
		m.decode(res, &body)
		if body.Kind != kindWrongKind {
			t.Errorf("error kind = %q, want %q", body.Kind, kindWrongKind)
		}
		if strings.Contains(m.h.readJournal(), "an answer") {
			t.Error("a reply should not have been written for the wrong kind")
		}
	})
}

func TestMCPChangeStatus(t *testing.T) {
	t.Run("todo_done marks a todo done, todo_reopen opens it again", func(t *testing.T) {
		m := newMCPTest(t, "test-agent")
		var todo jsonRecordResult
		m.decode(m.call("todo_add", map[string]any{"text": "do the thing"}), &todo)

		var done jsonChangeResult
		m.decode(m.call("todo_done", map[string]any{"id": todo.Record.ID}), &done)
		if !done.Changed || done.Record.Status != "done" {
			t.Errorf("got %+v", done)
		}

		var reopen jsonChangeResult
		m.decode(m.call("todo_reopen", map[string]any{"id": todo.Record.ID}), &reopen)
		if !reopen.Changed || reopen.Record.Status != "open" {
			t.Errorf("got %+v", reopen)
		}
	})

	t.Run("marking an already-done todo done writes nothing and says changed:false", func(t *testing.T) {
		m := newMCPTest(t, "test-agent")
		var todo jsonRecordResult
		m.decode(m.call("todo_add", map[string]any{"text": "do the thing"}), &todo)
		m.call("todo_done", map[string]any{"id": todo.Record.ID})
		before := m.h.readJournal()

		var again jsonChangeResult
		m.decode(m.call("todo_done", map[string]any{"id": todo.Record.ID}), &again)
		if again.Changed {
			t.Error("changed should be false for a todo that is done already")
		}
		if m.h.readJournal() != before {
			t.Error("the journal should not have changed")
		}
	})

	t.Run("qa_done and bug_done close a question and a bug", func(t *testing.T) {
		m := newMCPTest(t, "test-agent")
		var q jsonRecordResult
		m.decode(m.call("qa_ask", map[string]any{"text": "why?"}), &q)
		var qDone jsonChangeResult
		m.decode(m.call("qa_done", map[string]any{"id": q.Record.ID}), &qDone)
		if !qDone.Changed || qDone.Record.Status != "done" {
			t.Errorf("qa_done: got %+v", qDone)
		}

		var b jsonRecordResult
		m.decode(m.call("bug_report", map[string]any{"text": "it crashes"}), &b)
		var bDone jsonChangeResult
		m.decode(m.call("bug_done", map[string]any{"id": b.Record.ID}), &bDone)
		if !bDone.Changed || bDone.Record.Status != "done" {
			t.Errorf("bug_done: got %+v", bDone)
		}
	})
}

func TestMCPGlossaryDefine(t *testing.T) {
	m := newMCPTest(t, "test-agent")
	res := m.call("glossary_define", map[string]any{"word": "token", "definition": "the smallest unit"})
	if res.IsError {
		t.Fatalf("isError: %s", m.text(res))
	}
	var out jsonRecordResult
	m.decode(res, &out)
	if out.Record.Kind != "glossary" || out.Record.Word != "token" || out.Record.Text != "the smallest unit" {
		t.Errorf("got %+v", out.Record)
	}
}

func TestMCPEdit(t *testing.T) {
	t.Run("replaces the text, given directly, never $EDITOR", func(t *testing.T) {
		m := newMCPTest(t, "test-agent")
		var memo jsonRecordResult
		m.decode(m.call("memo_add", map[string]any{"text": "first"}), &memo)

		var edited jsonChangeResult
		m.decode(m.call("edit", map[string]any{"id": memo.Record.ID, "text": "second"}), &edited)
		if !edited.Changed || edited.Record.Text != "second" {
			t.Errorf("got %+v", edited)
		}
	})

	t.Run("the same text changes nothing", func(t *testing.T) {
		m := newMCPTest(t, "test-agent")
		var memo jsonRecordResult
		m.decode(m.call("memo_add", map[string]any{"text": "first"}), &memo)
		before := m.h.readJournal()

		var edited jsonChangeResult
		m.decode(m.call("edit", map[string]any{"id": memo.Record.ID, "text": "first"}), &edited)
		if edited.Changed {
			t.Error("changed should be false for the same text")
		}
		if m.h.readJournal() != before {
			t.Error("the journal should not have changed")
		}
	})
}

func TestMCPEmptyText(t *testing.T) {
	m := newMCPTest(t, "test-agent")
	res := m.call("memo_add", map[string]any{"text": "   "})
	if !res.IsError {
		t.Fatalf("expected isError for empty text, got %s", m.text(res))
	}
	var body jsonErrorBody
	m.decode(res, &body)
	if body.Kind != kindEmptyText {
		t.Errorf("error kind = %q, want %q", body.Kind, kindEmptyText)
	}
}

func TestMCPAmbiguousID(t *testing.T) {
	// Fixed IDs sharing a 4-digit prefix, written directly (setJournal), so the
	// collision does not depend on chance the way two random UUIDs would.
	ambigA := "aaaa" + strings.Repeat("1", 28)
	ambigB := "aaaa" + strings.Repeat("2", 28)
	m := newMCPTest(t, "test-agent")
	m.h.setJournal(
		record(ambigA, "memo", "one", "tester", "2026-09-17T00:00:00Z"),
		record(ambigB, "memo", "two", "tester", "2026-09-17T00:01:00Z"),
	)

	res := m.call("show", map[string]any{"id": "aaaa"})
	if !res.IsError {
		t.Fatalf("expected isError for an ambiguous ID, got %s", m.text(res))
	}
	var body jsonErrorBody
	m.decode(res, &body)
	if body.Kind != kindAmbiguous || len(body.Candidates) != 2 {
		t.Errorf("got %+v", body)
	}
}

func TestMCPNoClientName(t *testing.T) {
	m := newMCPTest(t, "")
	res := m.call("memo_add", map[string]any{"text": "hello"})
	if !res.IsError {
		t.Fatalf("expected isError when the client gives no name, got %s", m.text(res))
	}
	var body jsonErrorBody
	m.decode(res, &body)
	if body.Kind != kindNoAuthor {
		t.Errorf("error kind = %q, want %q", body.Kind, kindNoAuthor)
	}
	if strings.TrimSpace(m.h.readJournal()) != "" {
		t.Error("nothing should have been written")
	}
}

func TestMCPNoMtqg(t *testing.T) {
	h := newHarness(t) // no init
	m := newMCPTestIn(t, h, "", "test-agent")
	res := m.call("memo_add", map[string]any{"text": "hello"})
	if !res.IsError {
		t.Fatalf("expected isError with no .mtqg/, got %s", m.text(res))
	}
	var body jsonErrorBody
	m.decode(res, &body)
	if body.Kind != kindNotInitialized {
		t.Errorf("error kind = %q, want %q", body.Kind, kindNotInitialized)
	}
}

func TestMCPDirFixesRepository(t *testing.T) {
	// -C's equivalent: dir fixed at server construction acts on that
	// repository regardless of the working directory the harness reports.
	h := initialized(t)
	other := t.TempDir()
	m := newMCPTestIn(t, h, h.root, "test-agent")
	h.env.Getwd = func() (string, error) { return other, nil } // would fail if used
	res := m.call("memo_add", map[string]any{"text": "hello"})
	if res.IsError {
		t.Fatalf("isError: %s", m.text(res))
	}
}

func TestMCPContext(t *testing.T) {
	t.Run("the same text as mtqg context", func(t *testing.T) {
		m := newMCPTest(t, "test-agent")
		m.call("memo_add", map[string]any{"text": "hello"})

		res := m.call("context", map[string]any{})
		got := m.text(res)
		_, want, _ := m.h.run("context")
		if got != want {
			t.Fatalf("context tool differs from mtqg context:\ngot:  %q\nwant: %q", got, want)
		}
	})

	t.Run("max_tokens is honored", func(t *testing.T) {
		m := newMCPTest(t, "test-agent")
		for i := range 20 {
			m.call("memo_add", map[string]any{"text": strings.Repeat("x", 50) + " " + string(rune('a'+i))})
		}
		full := m.text(m.call("context", map[string]any{}))
		cut := m.text(m.call("context", map[string]any{"max_tokens": 50}))
		if len(cut) >= len(full) {
			t.Errorf("a smaller max_tokens should cut the text: full=%d cut=%d", len(full), len(cut))
		}
	})
}

func TestMCPShow(t *testing.T) {
	m := newMCPTest(t, "test-agent")
	var memo jsonRecordResult
	m.decode(m.call("memo_add", map[string]any{"text": "hello"}), &memo)

	res := m.call("show", map[string]any{"id": memo.Record.ID})
	if res.IsError {
		t.Fatalf("isError: %s", m.text(res))
	}
	var out jsonShow
	m.decode(res, &out)
	if out.Record.ID != memo.Record.ID || len(out.Events) != 1 {
		t.Errorf("got %+v", out)
	}
}

func TestMCPSearch(t *testing.T) {
	t.Run("finds matches, newest first, cut to 100 characters", func(t *testing.T) {
		m := newMCPTest(t, "test-agent")
		m.call("memo_add", map[string]any{"text": "apple pie"})
		long := strings.Repeat("banana ", 20) + "apple"
		m.call("memo_add", map[string]any{"text": long})

		res := m.call("search", map[string]any{"query": "apple"})
		var out jsonSearchSummary
		m.decode(res, &out)
		if out.Count != 2 || len(out.Records) != 2 {
			t.Fatalf("got %+v", out)
		}
		// Newest first: the long one was added second.
		if !out.Records[0].Truncated || len([]rune(out.Records[0].Text)) != searchSummaryLimit {
			t.Errorf("first record = %+v, want truncated to %d chars", out.Records[0], searchSummaryLimit)
		}
		if out.Records[1].Truncated {
			t.Errorf("second record should not be truncated: %+v", out.Records[1])
		}
	})

	t.Run("limit caps how many are returned, but not the count", func(t *testing.T) {
		m := newMCPTest(t, "test-agent")
		for range 5 {
			m.call("memo_add", map[string]any{"text": "apple"})
		}
		res := m.call("search", map[string]any{"query": "apple", "limit": 2})
		var out jsonSearchSummary
		m.decode(res, &out)
		if out.Count != 5 || out.Shown != 2 || len(out.Records) != 2 {
			t.Fatalf("got %+v", out)
		}
	})

	t.Run("default limit is defaultSearchLimit", func(t *testing.T) {
		m := newMCPTest(t, "test-agent")
		for range defaultSearchLimit + 3 {
			m.call("memo_add", map[string]any{"text": "apple"})
		}
		res := m.call("search", map[string]any{"query": "apple"})
		var out jsonSearchSummary
		m.decode(res, &out)
		if out.Shown != defaultSearchLimit {
			t.Fatalf("shown = %d, want %d", out.Shown, defaultSearchLimit)
		}
	})
}

// TestMCPToolsMatchCommands checks that every write tool has a matching
// command in the commands table (docs/reference/cli.md "MCP server"'s "Same
// as" column is not a lie), and that the commands mtqg mcp deliberately does
// not expose (delete, undo, archive, review, format) have no tool.
func TestMCPToolsMatchCommands(t *testing.T) {
	registered := map[string]struct{ kind, name string }{
		"memo_add":        {"memo", "add"},
		"rule_add":        {"rule", "add"},
		"todo_add":        {"todo", "add"},
		"todo_done":       {"todo", "done"},
		"todo_reopen":     {"todo", "reopen"},
		"qa_ask":          {"qa", "add"},
		"qa_answer":       {"qa", "add"},
		"qa_done":         {"qa", "done"},
		"qa_reopen":       {"qa", "reopen"},
		"bug_report":      {"bug", "add"},
		"bug_reply":       {"bug", "add"},
		"bug_done":        {"bug", "done"},
		"bug_reopen":      {"bug", "reopen"},
		"glossary_define": {"glossary", "add"},
		"edit":            {"", "edit"},
	}
	for tool, want := range registered {
		found := false
		for _, cmd := range commands {
			if cmd.kind == want.kind && cmd.name == want.name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("tool %q claims to be %q %q, which is not in the commands table", tool, want.kind, want.name)
		}
	}

	notExposed := []string{"delete", "undo", "archive", "review", "format"}
	m := newMCPTest(t, "test-agent")
	for _, name := range notExposed {
		res, err := m.cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: map[string]any{}})
		if err == nil && res != nil && !res.IsError {
			t.Errorf("%q should not be a tool mtqg mcp exposes", name)
		}
	}
}
