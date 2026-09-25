package cli

import (
	"bytes"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// agentFileResult is what happened to one file that "mtqg init --agent" touches.
type agentFileResult struct {
	path   string
	status string // "created", "updated" or "unchanged"
}

const (
	claudeSettingsPath = ".claude/settings.json"
	claudeMCPPath      = ".mcp.json"
	claudeMemoryPath   = "CLAUDE.md"
)

// claudeMemoryLine is the one line "mtqg init --agent claude-code" adds to
// CLAUDE.md, if it is not there already. It is mtqg's own message (English,
// like everything else mtqg writes - cli-output.md), not something written on
// the user's behalf.
const claudeMemoryLine = "This repository records its development with mtqg: run `mtqg context` at the start of a session, and see `.mtqg/SCHEMA.md` for the data format."

// claudeHookCommand is the shell command "mtqg init --agent claude-code" puts
// into .claude/settings.json's hooks: nothing more than a call to mtqg (§11.3
// "安全性" - a hook that is committed runs on other people's machines the
// moment they clone).
func claudeHookCommand(event string) string {
	return "mtqg hook claude-code " + event
}

// wireClaudeCode sets up this repository's Claude Code configuration so that
// mtqg's hooks run at SessionStart and Stop, and the agent's author is
// MTQG_AUTHOR_* (§11.3 "mtqg init --agent <agent>"). dryRun writes nothing and
// only says what would change.
func wireClaudeCode(root string, dryRun bool) ([]agentFileResult, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("cli: %w", err)
	}
	defer func() { _ = r.Close() }()

	settings, err := mergeClaudeSettings(r, dryRun)
	if err != nil {
		return nil, err
	}
	mcpConfig, err := mergeClaudeMCP(r, dryRun)
	if err != nil {
		return nil, err
	}
	memory, err := appendClaudeMemory(r, dryRun)
	if err != nil {
		return nil, err
	}
	return []agentFileResult{settings, mcpConfig, memory}, nil
}

// mergeClaudeSettings adds mtqg's hooks and default MTQG_AUTHOR_* to
// .claude/settings.json, keeping everything else that is there. Running it
// again does not add a second copy of anything, and it never touches a value
// that is already set (an existing env value, another hook that happens to
// call the same command some other way is left alone; only a group whose
// command is exactly mtqg's own is recognized as "already there").
func mergeClaudeSettings(root *os.Root, dryRun bool) (agentFileResult, error) {
	path := claudeSettingsPath
	original, existed, err := readRootFile(root, path)
	if err != nil {
		return agentFileResult{}, err
	}

	settings := map[string]any{}
	if existed {
		if err := jsonv2.Unmarshal(original, &settings); err != nil {
			return agentFileResult{}, &failure{kindHookConfigInvalid, msgAgentSettingsInvalid(path, err)}
		}
	}

	ensureHookEvent(settings, "SessionStart", claudeHookCommand("session-start"))
	ensureHookEvent(settings, "Stop", claudeHookCommand("stop"))
	ensureEnvDefault(settings, "MTQG_AUTHOR_KIND", "ai")
	ensureEnvDefault(settings, "MTQG_AUTHOR_NAME", "claude-code")

	// Keys are written in a fixed (alphabetical) order so that the file is the
	// same however the map was walked: settings.json is committed, and its
	// diffs are read by people. The first run of "init --agent" on a
	// hand-written file therefore reorders its top-level keys once; a run after
	// that changes nothing and writes nothing (below).
	out, err := jsonv2.Marshal(settings, jsontext.WithIndent("  "), jsonv2.Deterministic(true))
	if err != nil {
		return agentFileResult{}, fmt.Errorf("cli: %w", err)
	}
	out = append(out, '\n')
	return writeRootFile(root, path, original, existed, out, dryRun)
}

// ensureHookEvent makes sure settings["hooks"][event] has one group whose
// command is cmd, adding it only if no group there already calls that command.
// No matcher is set, so the hook runs for every source ("startup", "resume",
// "clear" and "compact" alike): a session that begins any way, or is compacted,
// should see mtqg's context again.
func ensureHookEvent(settings map[string]any, event, cmd string) {
	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
		settings["hooks"] = hooks
	}
	groups, _ := hooks[event].([]any)
	for _, g := range groups {
		if group, ok := g.(map[string]any); ok && groupHasCommand(group, cmd) {
			return
		}
	}
	groups = append(groups, map[string]any{
		"hooks": []any{
			map[string]any{"type": "command", "command": cmd},
		},
	})
	hooks[event] = groups
}

func groupHasCommand(group map[string]any, cmd string) bool {
	items, _ := group["hooks"].([]any)
	for _, it := range items {
		if m, ok := it.(map[string]any); ok {
			if s, _ := m["command"].(string); s == cmd {
				return true
			}
		}
	}
	return false
}

// ensureEnvDefault sets settings["env"][key] to value, unless it is already
// set to something (whatever that is, it is left alone).
func ensureEnvDefault(settings map[string]any, key, value string) {
	env, _ := settings["env"].(map[string]any)
	if env == nil {
		env = map[string]any{}
		settings["env"] = env
	}
	if _, ok := env[key]; !ok {
		env[key] = value
	}
}

// mergeClaudeMCP adds an mcpServers.mtqg entry that runs "mtqg mcp" to
// .mcp.json, keeping everything else that is there: an existing mcpServers.mtqg
// entry is left exactly as it is, whatever it contains (docs/reference/cli.md
// "MCP server").
func mergeClaudeMCP(root *os.Root, dryRun bool) (agentFileResult, error) {
	path := claudeMCPPath
	original, existed, err := readRootFile(root, path)
	if err != nil {
		return agentFileResult{}, err
	}

	config := map[string]any{}
	if existed {
		if err := jsonv2.Unmarshal(original, &config); err != nil {
			return agentFileResult{}, &failure{kindHookConfigInvalid, msgAgentSettingsInvalid(path, err)}
		}
	}

	ensureMCPServer(config, "mtqg", map[string]any{
		"type":    "stdio",
		"command": "mtqg",
		"args":    []any{"mcp"},
	})

	out, err := jsonv2.Marshal(config, jsontext.WithIndent("  "), jsonv2.Deterministic(true))
	if err != nil {
		return agentFileResult{}, fmt.Errorf("cli: %w", err)
	}
	out = append(out, '\n')
	return writeRootFile(root, path, original, existed, out, dryRun)
}

// ensureMCPServer makes sure config["mcpServers"][name] exists, adding it as
// entry only if nothing is there yet under that name (the same "leave what is
// already there alone" rule as ensureEnvDefault).
func ensureMCPServer(config map[string]any, name string, entry map[string]any) {
	servers, _ := config["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
		config["mcpServers"] = servers
	}
	if _, ok := servers[name]; !ok {
		servers[name] = entry
	}
}

// appendClaudeMemory adds claudeMemoryLine to CLAUDE.md, unless it is already
// there.
func appendClaudeMemory(root *os.Root, dryRun bool) (agentFileResult, error) {
	path := claudeMemoryPath
	original, existed, err := readRootFile(root, path)
	if err != nil {
		return agentFileResult{}, err
	}
	if existed && bytes.Contains(original, []byte(claudeMemoryLine)) {
		return agentFileResult{path: path, status: "unchanged"}, nil
	}

	out := append([]byte{}, original...)
	if len(out) > 0 && out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	if len(out) > 0 {
		out = append(out, '\n')
	}
	out = append(out, []byte(claudeMemoryLine+"\n")...)
	return writeRootFile(root, path, original, existed, out, dryRun)
}

// readRootFile reads a file relative to root. A missing file is not an error:
// existed says so, and data is nil.
func readRootFile(root *os.Root, path string) (data []byte, existed bool, err error) {
	data, err = root.ReadFile(path)
	if err == nil {
		return data, true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	return nil, false, fmt.Errorf("cli: %w", err)
}

// writeRootFile decides what happened to a file (created, updated or
// unchanged, by comparing out to what was there) and, unless dryRun or nothing
// changed, writes it: to a temporary file next to it, replaced into place, the
// same shape of write journal.Journal uses for its own files.
func writeRootFile(root *os.Root, path string, original []byte, existed bool, out []byte, dryRun bool) (agentFileResult, error) {
	status := "updated"
	if !existed {
		status = "created"
	} else if bytes.Equal(original, out) {
		return agentFileResult{path: path, status: "unchanged"}, nil
	}
	if dryRun {
		return agentFileResult{path: path, status: status}, nil
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := root.MkdirAll(dir, 0o750); err != nil {
			return agentFileResult{}, fmt.Errorf("cli: %w", err)
		}
	}
	if err := writeFileAtomic(root, path, out); err != nil {
		return agentFileResult{}, err
	}
	return agentFileResult{path: path, status: status}, nil
}

// writeFileAtomic writes data to a fresh file next to path and puts it in
// place of path, so that a reader (or a crash) never sees a half-written file.
func writeFileAtomic(root *os.Root, path string, data []byte) error {
	tmp := path + ".tmp"
	_ = root.Remove(tmp) // a leftover from a write that crashed earlier

	f, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("cli: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = root.Remove(tmp)
		return fmt.Errorf("cli: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = root.Remove(tmp)
		return fmt.Errorf("cli: %w", err)
	}
	if err := root.Rename(tmp, path); err != nil {
		_ = root.Remove(tmp)
		return fmt.Errorf("cli: %w", err)
	}
	return nil
}

func agentFilesJSON(files []agentFileResult) []jsonAgentFile {
	out := make([]jsonAgentFile, len(files))
	for i, f := range files {
		out[i] = jsonAgentFile{Path: f.path, Result: f.status}
	}
	return out
}
