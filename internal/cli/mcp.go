package cli

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/amisonnet8/mtqg/internal/journal"
)

// mtqg mcp runs an MCP server on standard input and output (docs/reference/cli.md
// "MCP server", docs/design/07-integrations.md §11.2). It is an entry point next
// to the CLI, not a separate layer: what it needs (error shapes, record JSON, the
// text of context, opening the journal) already lives in this package
// (.claude/rules/directory-structure.md "段階4b").

// runMCP starts the server and blocks until standard input ends.
func runMCP(c *ctx) int {
	server := mcp.NewServer(&mcp.Implementation{Name: "mtqg", Version: buildVersion()}, nil)
	addTools(server, c.env, c.inv.dir)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		return c.fail(err)
	}
	return exitOK
}

// mcpClientInfo returns the name a connecting client gave in its initialize
// request, or "" if it gave none.
func mcpClientInfo(req *mcp.CallToolRequest) string {
	info := req.ClientInfo()
	if info == nil {
		return ""
	}
	return strings.TrimSpace(info.Name)
}

// mcpCtx builds a ctx for one tool call. dir is what -C gave mtqg mcp, or ""
// (every call looks for .mtqg/ again: no repository is fixed at startup).
// author always comes from the connecting client, never from MTQG_AUTHOR_* or
// git (directory-structure.md "対象リポジトリ・記録者"). cmd is a stand-in
// command used only for its kind and name (reportOf's verb, and the label
// non-MCP callers would use); it is never looked up in the commands table.
func mcpCtx(env Env, dir, clientName, kind, name string) (*ctx, *mcp.CallToolResult) {
	if clientName == "" {
		return nil, mcpErrorResult(errorReport{kind: kindNoAuthor, lines: []string{msgMCPNoClientName()}})
	}
	author := journal.Author{Kind: journal.AuthorAI, Name: clientName}
	return &ctx{
		env:      env,
		inv:      &invocation{dir: dir, cmd: &command{kind: kind, name: name}},
		authorAs: &author,
	}, nil
}

// mcpResult marshals v (one of the json* result types already used for --json)
// as the tool's successful result.
func mcpResult(v any) (*mcp.CallToolResult, any, error) {
	b, err := marshalJSON(v)
	if err != nil {
		//nolint:nilerr // a marshal failure is reported as a tool error (IsError), not as the
		// function's own error return: the SDK would otherwise treat it as a protocol-level
		// failure rather than a tool error (mcp.go, cli-output.md "mtqg mcp").
		return mcpErrorResult(errorReport{kind: kindUnknown, lines: []string{err.Error()}}), nil, nil
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil, nil
}

// mcpError turns an error from the core into a tool error result: a domain
// error becomes isError:true with the same shape --json writes to standard
// error (cli-output.md "mtqg mcp"), never a protocol-level error, so the
// server keeps running and the agent can see what went wrong.
func mcpError(c *ctx, err error) (*mcp.CallToolResult, any, error) {
	return mcpErrorResult(c.reportOf(err)), nil, nil
}

func mcpErrorResult(rep errorReport) *mcp.CallToolResult {
	b, err := marshalJSON(rep.json())
	if err != nil {
		return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: rep.kind + ": " + strings.Join(rep.lines, "; ")}}}
	}
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}
}
