//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mcpConnect starts the real binary as "mtqg mcp" in r's repository and
// connects a real MCP client to it as a subprocess (mcp.CommandTransport):
// what an agent actually does, not a simulation of the protocol.
func (r *repo) mcpConnect(t *testing.T, clientName string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: clientName, Version: "1.0"}, nil)
	cmd := exec.Command(binary, "mcp")
	cmd.Dir = r.dir
	cmd.Env = r.env
	transport := &mcp.CommandTransport{Command: cmd}
	ctx := context.Background()
	cs, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("mcp connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func mcpText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			return tc.Text
		}
	}
	t.Fatal("no text content in result")
	return ""
}

// TestMCPServer runs a real agent-shaped conversation against the real
// binary: connect, list the tools, write a memo, and read it back, checking
// that a record written through MCP is exactly what "mtqg show --json" sees
// (docs/design/07-integrations.md §11.1: every entry point shares one core).
func TestMCPServer(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")

	cs := r.mcpConnect(t, "e2e-agent")
	ctx := context.Background()

	tools, err := cs.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("tools/list: %v", err)
	}
	if len(tools.Tools) == 0 {
		t.Fatal("tools/list returned no tools")
	}
	var hasMemoAdd bool
	for _, tool := range tools.Tools {
		if tool.Name == "memo_add" {
			hasMemoAdd = true
		}
	}
	if !hasMemoAdd {
		t.Fatalf("memo_add not offered: %+v", tools.Tools)
	}

	addRes, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "memo_add", Arguments: map[string]any{"text": "written through MCP"}})
	if err != nil {
		t.Fatalf("memo_add: %v", err)
	}
	if addRes.IsError {
		t.Fatalf("memo_add isError: %s", mcpText(t, addRes))
	}
	var added struct {
		Record struct {
			ID     string `json:"id"`
			Author struct {
				Kind string `json:"kind"`
				Name string `json:"name"`
			} `json:"author"`
		} `json:"record"`
	}
	if err := json.Unmarshal([]byte(mcpText(t, addRes)), &added); err != nil {
		t.Fatalf("decode memo_add result: %v\n%s", err, mcpText(t, addRes))
	}
	if added.Record.Author.Kind != "ai" || added.Record.Author.Name != "e2e-agent" {
		t.Errorf("author = %+v, want ai/e2e-agent", added.Record.Author)
	}

	showRes, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "show", Arguments: map[string]any{"id": added.Record.ID}})
	if err != nil {
		t.Fatalf("show: %v", err)
	}

	cliShow := r.mtqg("show", added.Record.ID, "--json")
	if strings.TrimSpace(mcpText(t, showRes)) != strings.TrimSpace(cliShow) {
		t.Fatalf("mcp show and mtqg show --json disagree:\nmcp: %s\ncli: %s", mcpText(t, showRes), cliShow)
	}
}

// TestMCPServerInitAfterStart checks that mtqg mcp looks for .mtqg/ on every
// call, not once at startup: a repository created after the server starts is
// still found.
func TestMCPServerInitAfterStart(t *testing.T) {
	r := newRepo(t) // no mtqg init yet

	cs := r.mcpConnect(t, "e2e-agent")
	ctx := context.Background()

	before, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "memo_add", Arguments: map[string]any{"text": "too early"}})
	if err != nil {
		t.Fatalf("memo_add before init: %v", err)
	}
	if !before.IsError {
		t.Fatalf("memo_add should fail before mtqg init: %s", mcpText(t, before))
	}

	r.mtqg("init")

	after, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "memo_add", Arguments: map[string]any{"text": "now it works"}})
	if err != nil {
		t.Fatalf("memo_add after init: %v", err)
	}
	if after.IsError {
		t.Fatalf("memo_add should succeed after mtqg init: %s", mcpText(t, after))
	}
}

// TestMCPServerExitsOnEOF checks that the server ends cleanly (exit code 0)
// when standard input closes, with no protocol traffic at all.
func TestMCPServerExitsOnEOF(t *testing.T) {
	r := newRepo(t)
	r.mtqg("init")

	cmd := exec.Command(binary, "mcp")
	cmd.Dir = r.dir
	cmd.Env = r.env
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err := stdin.Close(); err != nil { // EOF, no bytes ever written
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("mtqg mcp did not exit cleanly on EOF: %v", err)
		}
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("mtqg mcp did not exit within 10s of standard input closing")
	}
}
