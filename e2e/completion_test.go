//go:build e2e

package e2e

// The completion scripts are run by the real shells, with the real mtqg on the PATH:
// a line is typed as far as the cursor, and what the shell would offer is read back.
// Each shell is a different program with different rules about words (an empty one may
// be dropped, `=` may split one), so this is what shows that the scripts pass the line
// to `mtqg candidates` intact. What is offered is decided by mtqg and tested in
// internal/cli; here it is only checked that all the shells offer the same.
//
// A shell that is not installed is skipped. On the machine of a developer that is
// bash, zsh, fish and PowerShell (the devcontainer has all four); on a runner of CI it
// is what the image has.

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// completionRepo is a repository with two todos, one of them done, and a memo.
type completionRepo struct {
	*repo
	open, done string // the short IDs of the two todos
	memo       string
}

func newCompletionRepo(t *testing.T) *completionRepo {
	t.Helper()
	r := newRepo(t)
	r.mtqg("init")
	c := &completionRepo{repo: r}
	c.open = trimmed(r.mtqg("t", "add", "Skip block comments"))
	c.done = trimmed(r.mtqg("t", "add", "Skip line comments"))
	r.mtqg("t", "done", c.done)
	c.memo = trimmed(r.mtqg("m", "add", "Policy: use English for all error messages"))
	return c
}

// completionCase is a line as typed, with the cursor at its end, and the values that
// the shells offer for it.
type completionCase struct {
	line string
	want []string
}

func (c *completionRepo) cases() []completionCase {
	return []completionCase{
		{"mtqg t", []string{"todo"}},
		{"mtqg t re", []string{"reopen"}},
		{"mtqg todo ", []string{"add", "list", "done", "reopen"}},
		{"mtqg t done ", []string{c.open}},
		{"mtqg t reopen ", []string{c.done}},
		{"mtqg show ", []string{c.open, c.done, c.memo}},
		{"mtqg show " + c.memo[:6], []string{c.memo}},
		{"mtqg log --kind ", []string{"memo", "todo", "qa", "bug", "glossary"}},
		{"mtqg log --l", []string{"--limit"}},
		{"mtqg log --limit 5 --k", []string{"--kind"}},
		{"mtqg completion ", []string{"bash", "zsh", "fish", "powershell"}},
		{"mtqg t add x", nil},
		{"mtqg t add fix the -", nil},
	}
}

// shellEnv is the environment of a shell that runs mtqg from the PATH.
func (c *completionRepo) shellEnv() []string {
	path := filepath.Dir(binary) + string(os.PathListSeparator) + os.Getenv("PATH")
	return append(append([]string{}, c.env...),
		"PATH="+path, "TERM=dumb", "POWERSHELL_UPDATECHECK=Off", "POWERSHELL_TELEMETRY_OPTOUT=1", "DOTNET_CLI_TELEMETRY_OPTOUT=1")
}

// runShell runs a program with a script on its standard input, in the repository, and
// returns what it printed.
func (c *completionRepo) runShell(t *testing.T, script string, program string, args ...string) string {
	t.Helper()
	cmd := exec.Command(program, args...)
	cmd.Dir = c.dir
	cmd.Env = c.shellEnv()
	cmd.Stdin = strings.NewReader(script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s: %v\n%s\n%s", program, err, stdout.String(), stderr.String())
	}
	// What a shell says on standard error is not a candidate.
	if stderr.Len() > 0 {
		t.Logf("%s said on standard error:\n%s", program, stderr.String())
	}
	return stdout.String()
}

// shell finds a shell, or skips the test. A shell that is there but does not see the
// built mtqg is a failure, not a skip: the test would have run and shown nothing.
func (c *completionRepo) shell(t *testing.T, name string, probe string) string {
	t.Helper()
	path, err := exec.LookPath(name)
	if err == nil && name == "bash" {
		path = gitBash(path)
	}
	if err != nil {
		t.Skipf("%s is not installed", name)
	}
	if got := strings.TrimSpace(c.runShell(t, probe, path, probeArgs(name)...)); !strings.HasPrefix(got, "mtqg ") {
		t.Fatalf("%s does not run the mtqg that was built: %q", name, got)
	}
	return path
}

// gitBash is the bash to use on Windows, where the one that the PATH finds first may be
// the launcher of the Windows Subsystem for Linux, which cannot run the mtqg.exe that
// was built. Git for Windows has a bash of its own. Elsewhere, and where the PATH has
// no launcher, it is the bash that was found.
func gitBash(found string) string {
	if runtime.GOOS != "windows" || !strings.Contains(strings.ToLower(found), `\system32\`) {
		return found
	}
	own := filepath.Join(os.Getenv("ProgramFiles"), "Git", "bin", "bash.exe")
	if _, err := os.Stat(own); err == nil {
		return own
	}
	return found
}

func probeArgs(name string) []string {
	switch name {
	case "pwsh":
		return []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-Command", "-"}
	case "fish":
		return []string{"--no-config"}
	case "zsh":
		return []string{"-f"}
	}
	return []string{"--norc", "--noprofile"}
}

// parseAnswers reads the output of a run of several cases: a line "### n" starts the
// answer of case n, and each line after it is a candidate (before a tab, if there is
// a description, or a colon for zsh's _describe).
func parseAnswers(out string, count int, cut string) [][]string {
	answers := make([][]string, count)
	current := -1
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if rest, ok := strings.CutPrefix(line, "### "); ok {
			_, _ = fmt.Sscanf(rest, "%d", &current)
			continue
		}
		if current < 0 || line == "" {
			continue
		}
		value, _, _ := strings.Cut(line, cut)
		answers[current] = append(answers[current], value)
	}
	return answers
}

func (c *completionRepo) check(t *testing.T, cases []completionCase, answers [][]string) {
	t.Helper()
	for i, tc := range cases {
		got, want := slices.Clone(answers[i]), slices.Clone(tc.want)
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("%q:\n got  %q\n want %q", tc.line, got, want)
		}
	}
}

// The single quotes of each shell. Nothing typed here has one, but a line that had
// would otherwise end the string.
func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func quoteFish(s string) string {
	return "'" + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), "'", `\'`) + "'"
}

func quotePowerShell(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func TestCompletionInBash(t *testing.T) {
	c := newCompletionRepo(t)
	bash := c.shell(t, "bash", "mtqg version")
	script := c.mtqg("completion", "bash")
	cases := c.cases()

	var b strings.Builder
	b.WriteString(script)
	for i, tc := range cases {
		words := strings.Split(tc.line, " ") // the last is empty when the line ends with a space
		quoted := make([]string, len(words))
		for j, w := range words {
			quoted[j] = quote(w)
		}
		fmt.Fprintf(&b, "echo '### %d'\nCOMP_WORDS=(%s)\nCOMP_CWORD=%d\n_mtqg\nprintf '%%s\\n' \"${COMPREPLY[@]}\"\n",
			i, strings.Join(quoted, " "), len(words)-1)
	}
	out := c.runShell(t, b.String(), bash, probeArgs("bash")...)
	c.check(t, cases, parseAnswers(out, len(cases), "\t"))
}

func TestCompletionInFish(t *testing.T) {
	c := newCompletionRepo(t)
	fish := c.shell(t, "fish", "mtqg version")
	cases := c.cases()

	var b strings.Builder
	b.WriteString("mtqg completion fish | source\n")
	for i, tc := range cases {
		fmt.Fprintf(&b, "echo '### %d'\ncomplete -C %s\n", i, quoteFish(tc.line))
	}
	out := c.runShell(t, b.String(), fish, probeArgs("fish")...)
	c.check(t, cases, parseAnswers(out, len(cases), "\t"))
}

func TestCompletionInPowerShell(t *testing.T) {
	c := newCompletionRepo(t)
	pwsh := c.shell(t, "pwsh", "mtqg version")
	cases := c.cases()

	var b strings.Builder
	b.WriteString(c.mtqg("completion", "powershell"))
	b.WriteString("\n")
	for i, tc := range cases {
		fmt.Fprintf(&b, "'### %d'\n$line = %s\n(TabExpansion2 $line $line.Length).CompletionMatches | ForEach-Object { $_.CompletionText }\n",
			i, quotePowerShell(tc.line))
	}
	out := c.runShell(t, b.String(), pwsh, probeArgs("pwsh")...)
	c.check(t, cases, parseAnswers(out, len(cases), "\t"))
}

// zsh needs its completion system to offer anything, which a script cannot start
// without a terminal. So the function of the script is called as the system would
// call it (with `words` and `CURRENT` set) and what it hands to _describe is read.
// That is all of what the script does: what it passes to mtqg, and how it passes on
// what mtqg says.
func TestCompletionInZsh(t *testing.T) {
	c := newCompletionRepo(t)
	zsh := c.shell(t, "zsh", "mtqg version")
	script := c.mtqg("completion", "zsh")
	cases := c.cases()

	syntax := exec.Command(zsh, "-n")
	syntax.Stdin = strings.NewReader(script)
	if out, err := syntax.CombinedOutput(); err != nil {
		t.Fatalf("zsh -n: %v\n%s", err, out)
	}

	var b strings.Builder
	b.WriteString("compdef() { :; }\n")
	b.WriteString("_describe() { shift 3; print -l -- \"${(P@)1}\"; }\n") // -t candidates 'mtqg' <array>
	b.WriteString("_files() { :; }\n")
	b.WriteString(script)
	for i, tc := range cases {
		words := strings.Split(tc.line, " ")
		quoted := make([]string, len(words))
		for j, w := range words {
			quoted[j] = quote(w)
		}
		fmt.Fprintf(&b, "print '### %d'\nwords=(%s)\nCURRENT=%d\n_mtqg\n", i, strings.Join(quoted, " "), len(words))
	}
	out := c.runShell(t, b.String(), zsh, probeArgs("zsh")...)
	c.check(t, cases, parseAnswers(out, len(cases), ":"))
}

// The real binary, with the real operating system, exits 0 and says nothing on standard
// error for a line in a place with no .mtqg/, or with no repository at all.
func TestCandidatesAreQuietFromTheBinary(t *testing.T) {
	c := newCompletionRepo(t)
	res := c.run(nil, "", "candidates", "--word=", "--", "t", "done")
	if res.code != 0 || res.stdout != c.open+"\tSkip block comments\n" || res.stderr != "" {
		t.Errorf("in the repository: %+v", res)
	}

	elsewhere := t.TempDir()
	res = c.runIn(elsewhere, nil, "", "candidates", "--word=", "--", "t", "done")
	if res.code != 0 || res.stdout != "" || res.stderr != "" {
		t.Errorf("outside a repository: %+v", res)
	}
	res = c.runIn(elsewhere, nil, "", "candidates", "--word=st", "--")
	if res.code != 0 || !strings.HasPrefix(res.stdout, "status\t") || res.stderr != "" {
		t.Errorf("a command needs no repository: %+v", res)
	}

	// -C among the words, as the shell passes it.
	res = c.runIn(elsewhere, nil, "", "candidates", "--word=", "--", "-C", c.dir, "t", "reopen")
	if res.code != 0 || res.stdout != c.done+"\tSkip line comments\n" || res.stderr != "" {
		t.Errorf("-C: %+v", res)
	}
}
