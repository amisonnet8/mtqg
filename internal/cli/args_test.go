package cli

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		cmd   string // command.full()
		words []string
		check func(*testing.T, *invocation)
	}{
		{name: "kind spelled out", args: []string{"todo", "add", "Skip", "block", "comments"}, cmd: "mtqg todo add", words: []string{"Skip", "block", "comments"}},
		{name: "kind by its letter", args: []string{"t", "add", "x"}, cmd: "mtqg todo add", words: []string{"x"}},
		{name: "memo by letter", args: []string{"m", "add", "x"}, cmd: "mtqg memo add", words: []string{"x"}},
		{name: "qa by letter", args: []string{"q", "list"}, cmd: "mtqg qa list"},
		{name: "glossary by letter", args: []string{"g", "add", "token", "a unit"}, cmd: "mtqg glossary add", words: []string{"token", "a unit"}},
		{name: "a command without a kind", args: []string{"status"}, cmd: "mtqg status"},
		{
			name: "from the first word of the text on, everything is text",
			args: []string{"t", "add", "fix", "the", "-x", "flag", "--full-id"},
			cmd:  "mtqg todo add", words: []string{"fix", "the", "-x", "flag", "--full-id"},
			check: func(t *testing.T, inv *invocation) {
				if inv.fullID {
					t.Error("--full-id after the text is text, not an option")
				}
			},
		},
		{
			name: "options before the text are options",
			args: []string{"t", "add", "--full-id", "text"},
			cmd:  "mtqg todo add", words: []string{"text"},
			check: func(t *testing.T, inv *invocation) {
				if !inv.fullID {
					t.Error("--full-id before the text should be an option")
				}
			},
		},
		{name: "-- ends the options", args: []string{"t", "add", "--", "-1", "is", "not", "allowed"}, cmd: "mtqg todo add", words: []string{"-1", "is", "not", "allowed"}},
		{name: "a single - is a word", args: []string{"m", "add", "-"}, cmd: "mtqg memo add", words: []string{"-"}},
		{
			name: "-C before the kind and after the verb",
			args: []string{"-C", "../other", "t", "list"}, cmd: "mtqg todo list",
			check: func(t *testing.T, inv *invocation) {
				if inv.dir != "../other" {
					t.Errorf("dir = %q", inv.dir)
				}
			},
		},
		{
			name: "-C after the command",
			args: []string{"t", "list", "-C", "dir"}, cmd: "mtqg todo list",
			check: func(t *testing.T, inv *invocation) {
				if inv.dir != "dir" {
					t.Errorf("dir = %q", inv.dir)
				}
			},
		},
		{
			name: "-C with the path joined to it",
			args: []string{"-Cdir", "status"}, cmd: "mtqg status",
			check: func(t *testing.T, inv *invocation) {
				if inv.dir != "dir" {
					t.Errorf("dir = %q", inv.dir)
				}
			},
		},
		{
			name: "--all between the kind and the verb",
			args: []string{"t", "--all", "list"}, cmd: "mtqg todo list",
			check: func(t *testing.T, inv *invocation) {
				if !inv.all {
					t.Error("--all should be set")
				}
			},
		},
		{name: "an option after the ID", args: []string{"t", "done", "6cad", "--full-id"}, cmd: "mtqg todo done", words: []string{"6cad"}, check: func(t *testing.T, inv *invocation) {
			if !inv.fullID {
				t.Error("--full-id should be set")
			}
		}},
		{name: "--no-color before the command", args: []string{"--no-color", "status"}, cmd: "mtqg status", check: func(t *testing.T, inv *invocation) {
			if !inv.noColor {
				t.Error("--no-color should be set")
			}
		}},
		{
			name: "an option that takes a value, as two words",
			args: []string{"log", "--limit", "5", "--kind", "q"}, cmd: "mtqg log",
			check: func(t *testing.T, inv *invocation) {
				if inv.values["--limit"] != "5" || inv.values["--kind"] != "q" {
					t.Errorf("values = %v", inv.values)
				}
			},
		},
		{
			name: "an option that takes a value, joined with =",
			args: []string{"log", "--limit=0", "--kind=glossary", "--full-id"}, cmd: "mtqg log",
			check: func(t *testing.T, inv *invocation) {
				if inv.values["--limit"] != "0" || inv.values["--kind"] != "glossary" || !inv.fullID {
					t.Errorf("values = %v, fullID %v", inv.values, inv.fullID)
				}
			},
		},
		{
			name: "the value is the next word, whatever it looks like",
			args: []string{"log", "--kind", "-1"}, cmd: "mtqg log",
			check: func(t *testing.T, inv *invocation) {
				if inv.values["--kind"] != "-1" {
					t.Errorf("values = %v", inv.values)
				}
			},
		},
		{
			name: "--at after the text is text, like any other option",
			args: []string{"m", "add", "fix", "--at", "x"}, cmd: "mtqg memo add",
			words: []string{"fix", "--at", "x"},
			check: func(t *testing.T, inv *invocation) {
				if _, ok := inv.values["--at"]; ok {
					t.Error("--at after the text should be text, not an option")
				}
			},
		},
		{
			name: "--at before the text is an option, and comes before the id of a reply",
			args: []string{"qa", "add", "--at", "f", "1012", "ans"}, cmd: "mtqg qa add",
			words: []string{"1012", "ans"},
			check: func(t *testing.T, inv *invocation) {
				if inv.values["--at"] != "f" {
					t.Errorf("values = %v", inv.values)
				}
			},
		},
		{name: "a glossary entry with a word and a definition", args: []string{"g", "add", "token", "a", "unit"}, cmd: "mtqg glossary add", words: []string{"token", "a", "unit"}},
		{name: "help asked for by -h", args: []string{"-h"}, cmd: "mtqg help", check: func(t *testing.T, inv *invocation) {
			if !inv.help {
				t.Error("help should be set")
			}
		}},
		{name: "help asked for on a command, without its arguments", args: []string{"t", "done", "--help"}, cmd: "mtqg todo done", check: func(t *testing.T, inv *invocation) {
			if !inv.help {
				t.Error("help should be set")
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv, err := parseArgs(tt.args)
			if err != nil {
				t.Fatalf("parseArgs(%q): %v", tt.args, err)
			}
			if got := inv.cmd.full(); got != tt.cmd {
				t.Errorf("command = %q, want %q", got, tt.cmd)
			}
			if len(inv.words)+len(tt.words) > 0 && !reflect.DeepEqual(inv.words, tt.words) {
				t.Errorf("words = %q, want %q", inv.words, tt.words)
			}
			if tt.check != nil {
				tt.check(t, inv)
			}
		})
	}
}

// Help on a kind with no verb yet is its own case: cmd is nil (there is no one
// command to name), so it does not fit the table above, which reads cmd.full().
func TestParseArgsHelpOnAKind(t *testing.T) {
	tests := []struct {
		name string
		args []string
		kind string
	}{
		{"the kind spelled out, -h", []string{"todo", "-h"}, "todo"},
		{"the kind's letter, --help", []string{"t", "--help"}, "todo"},
		{"qa", []string{"qa", "-h"}, "qa"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv, err := parseArgs(tt.args)
			if err != nil {
				t.Fatalf("parseArgs(%q): %v", tt.args, err)
			}
			if !inv.help || inv.kindHelp != tt.kind || inv.cmd != nil {
				t.Errorf("help=%v kindHelp=%q cmd=%v, want help=true kindHelp=%q cmd=nil", inv.help, inv.kindHelp, inv.cmd, tt.kind)
			}
		})
	}
}

func TestParseArgsMistakes(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string // a part of the message
	}{
		{"no command", nil, "No command given"},
		{"unknown command", []string{"frobnicate"}, `Unknown command "frobnicate"`},
		{"a kind without a verb", []string{"t"}, "needs a verb"},
		{"an unknown verb", []string{"t", "frobnicate"}, `Unknown verb "frobnicate"`},
		{"a verb of another kind", []string{"m", "done", "x"}, `Unknown verb "done" for `},
		{"too many words for a list", []string{"t", "list", "extra"}, "Too many arguments"},
		{"no ID", []string{"t", "done"}, "Missing argument"},
		{"two IDs", []string{"t", "done", "a", "b"}, "Too many arguments"},
		{"an unknown option before the command", []string{"--bogus", "status"}, "Unknown option --bogus"},
		{"a text that starts with an option", []string{"t", "add", "-x", "flag"}, "Unknown option -x"},
		{"-C without a path", []string{"status", "-C"}, "needs a value"},
		{"--limit with no value", []string{"log", "--limit"}, "Option --limit needs a value"},
		{"--limit on a command that has none", []string{"t", "list", "--limit", "5"}, "Unknown option --limit"},
		{"--limit before the command", []string{"--limit", "5", "log"}, "Unknown option --limit"},
		{"a glossary entry with no word", []string{"g", "add"}, "Missing argument"},
		{"words for log", []string{"log", "extra"}, "Too many arguments"},
		{"--all where it means nothing", []string{"t", "add", "--all", "x"}, "Unknown option --all"},
		{"--at with no value", []string{"m", "add", "--at"}, "Option --at needs a value"},
		{"--at on a command that has none", []string{"t", "list", "--at", "x"}, "Unknown option --at"},
		{"--before with no value", []string{"log", "--before"}, "Option --before needs a value"},
		{"--before on a command that has none", []string{"m", "list", "--before", "x"}, "Unknown option --before"},
		{"--events on a command that has none", []string{"t", "list", "--events"}, "Unknown option --events"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseArgs(tt.args)
			var usage *usageError
			if !errors.As(err, &usage) {
				t.Fatalf("err = %v, want a usage error", err)
			}
			if !strings.Contains(usage.msg, tt.want) {
				t.Errorf("message %q does not contain %q", usage.msg, tt.want)
			}
		})
	}
}

// The tables must agree with each other: every kind has the verbs help shows,
// every command names itself in its usage, and no two commands are the same.
func TestGrammarTablesAreConsistent(t *testing.T) {
	seen := map[string]bool{}
	for _, cmd := range commands {
		if cmd.usage == "" || cmd.summary == "" {
			t.Errorf("%s has no usage or summary", cmd.full())
		}
		if !strings.Contains(cmd.usage, cmd.full()) {
			t.Errorf("the usage %q does not name %q", cmd.usage, cmd.full())
		}
		if seen[cmd.full()] {
			t.Errorf("%s is listed twice", cmd.full())
		}
		seen[cmd.full()] = true
		if cmd.kind != "" {
			if _, ok := findKind(cmd.kind); !ok {
				t.Errorf("%s names a kind that does not exist", cmd.full())
			}
		}
	}
	for _, k := range kinds {
		if len(verbsOf(k.name)) == 0 {
			t.Errorf("kind %s has no verbs", k.name)
		}
	}
}
