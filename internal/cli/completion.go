package cli

import (
	"embed"
	"slices"
)

// The scripts are fixed text. They do not list the commands: at every TAB they
// ask `mtqg candidates`, which reads the tables of args.go, so a script cannot
// disagree with the mtqg that is installed.
//
//go:embed completions
var completionScripts embed.FS

// shells are the shells that have a script, and the files that hold the scripts.
var shells = []string{"bash", "zsh", "fish", "powershell"}

var scriptFiles = map[string]string{
	"bash":       "completions/mtqg.bash",
	"zsh":        "completions/mtqg.zsh",
	"fish":       "completions/mtqg.fish",
	"powershell": "completions/mtqg.ps1",
}

// runCompletion prints the completion script of a shell.
func runCompletion(c *ctx) int {
	words := c.inv.words
	switch {
	case len(words) == 0:
		return c.usageFailure(msgNoShell(shells))
	case len(words) > 1:
		return c.usageFailure(msgTooManyArguments(c.inv.cmd.usage))
	case !slices.Contains(shells, words[0]):
		return c.usageFailure(msgUnknownShell(words[0], shells))
	}
	script, err := completionScripts.ReadFile(scriptFiles[words[0]])
	if err != nil {
		return c.fail(err) // the files are embedded, so this is a build without them
	}
	if c.inv.json {
		return c.emit(jsonCompletion{Command: c.inv.cmd.label(), Shell: words[0], Script: string(script)})
	}
	_, _ = c.env.Stdout.Write(script)
	return exitOK
}
