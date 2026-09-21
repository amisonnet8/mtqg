//go:build e2e

package e2e

// The examples in docs/reference/cli.md and cli_ja.md are run against mtqg, and what
// they show has to be what mtqg prints. Every example is a code block that starts
// with "$ ", and the line before it says where to run it:
//
//	<!-- mtqg:example repo=parser -->
//
// repo names a fixture (a directory of testdata/examples, or "none" for a git
// repository with no .mtqg/, or "empty" for one that has only had mtqg init). The
// other words are ids=any (the IDs that a command makes are not compared), path=
// (what to show for the directory of the repository), author=<kind>:<name>,
// binary (also run the built binary, and require the same output), setup="..." (a
// command to run first, without showing it, once for each) and skip="<why>".
//
// go test -tags e2e ./e2e -run '^TestDocExamples$' -update (make docs-examples)
// writes what mtqg prints into the documents. Nobody writes an example by hand.

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/amisonnet8/mtqg/internal/cli"
)

var update = flag.Bool("update", false, "write what mtqg prints into the examples of docs/reference/ (make docs-examples)")

// documents are the files of docs/reference/ that hold examples, English first.
var documents = []string{"cli.md", "cli_ja.md"}

// examplesNow is what "today" is in the examples: records of this day show as HH:MM.
var examplesNow = time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)

// repoName is the name of the directory of the repository, which context prints.
const repoName = "sample-parser"

// neutralFixtures hold no text that a person wrote, so that either language may use them.
var neutralFixtures = []string{"none", "empty", "refuse", "broken"}

var (
	markPattern = regexp.MustCompile(`^<!-- mtqg:example (.+) -->$`)
	// anyIDPattern finds what could be an ID (10 to 32 lowercase hex digits).
	anyIDPattern = regexp.MustCompile(`\b[0-9a-f]{10,32}\b`)
)

// example is a code block of a document that is run.
type example struct {
	line int // the line of the opening fence, from 1
	mark string
	opts exampleOptions
	cmds []exampleCommand
}

type exampleOptions struct {
	repo   string
	ids    bool // ids=any
	binary bool
	path   string
	author string // <kind>:<name>
	skip   string
	setup  []string
}

// exampleCommand is a "$ " line and the output the document shows after it, which is
// lines[outStart:outEnd] of the document (blank lines that end it are left out).
type exampleCommand struct {
	text     string
	outStart int
	outEnd   int
}

// document is a file that has been read and split into examples.
type document struct {
	name     string
	lines    []string
	examples []*example
	problems []string
}

func loadDocument(t *testing.T, name string) *document {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "docs", "reference", name))
	if err != nil {
		t.Fatal(err)
	}
	d := &document{name: name, lines: strings.Split(string(data), "\n")}
	d.parse()
	return d
}

func (d *document) problem(line int, format string, args ...any) {
	d.problems = append(d.problems, fmt.Sprintf("%s:%d: %s", d.name, line, fmt.Sprintf(format, args...)))
}

func (d *document) parse() {
	marked := map[int]bool{} // the marks that an example took
	for i := 0; i < len(d.lines); i++ {
		if !strings.HasPrefix(d.lines[i], "```") {
			continue
		}
		end := i + 1
		for end < len(d.lines) && !strings.HasPrefix(d.lines[end], "```") {
			end++
		}
		if end == len(d.lines) {
			d.problem(i+1, "a code block that is never closed")
			return
		}
		if end > i+1 && strings.HasPrefix(d.lines[i+1], "$ ") {
			d.example(i, end, marked)
		}
		i = end
	}
	for i, l := range d.lines {
		if strings.HasPrefix(l, "<!-- mtqg:example") && !marked[i] {
			d.problem(i+1, "an example mark that is not right before a code block that starts with $")
		}
	}
}

// example reads the block between the fences d.lines[open] and d.lines[end].
func (d *document) example(open, end int, marked map[int]bool) {
	ex := &example{line: open + 1}
	var m []string
	if open > 0 {
		m = markPattern.FindStringSubmatch(d.lines[open-1])
	}
	if m == nil {
		d.problem(ex.line, "an example (a code block that starts with $) without a line <!-- mtqg:example repo=... --> before it")
		return
	}
	marked[open-1] = true
	ex.mark = m[1]
	opts, err := parseOptions(ex.mark)
	if err != nil {
		d.problem(ex.line, "%v", err)
		return
	}
	ex.opts = opts

	for k := open + 1; k < end; k++ {
		if text, ok := strings.CutPrefix(d.lines[k], "$ "); ok {
			ex.cmds = append(ex.cmds, exampleCommand{text: text, outStart: k + 1})
		}
	}
	for i := range ex.cmds {
		stop := end
		if i+1 < len(ex.cmds) {
			stop = ex.cmds[i+1].outStart - 1
		}
		for stop > ex.cmds[i].outStart && strings.TrimSpace(d.lines[stop-1]) == "" {
			stop--
		}
		ex.cmds[i].outEnd = stop
	}
	if ex.opts.skip == "" {
		for _, c := range ex.cmds {
			if _, err := parseInvocation(c.text); err != nil {
				d.problem(ex.line, "%v", err)
			}
		}
	}
	d.examples = append(d.examples, ex)
}

func parseOptions(mark string) (exampleOptions, error) {
	words, err := splitWords(mark)
	if err != nil {
		return exampleOptions{}, err
	}
	var o exampleOptions
	for _, w := range words {
		key, val, hasValue := strings.Cut(w, "=")
		switch {
		case key == "repo" && hasValue:
			o.repo = val
		case key == "ids" && val == "any":
			o.ids = true
		case key == "binary" && !hasValue:
			o.binary = true
		case key == "path" && hasValue:
			o.path = val
		case key == "author" && strings.Contains(val, ":"):
			o.author = val
		case key == "skip" && val != "":
			o.skip = val
		case key == "setup" && val != "":
			o.setup = append(o.setup, val)
		default:
			return exampleOptions{}, fmt.Errorf("cannot read %q in the mark of the example", w)
		}
	}
	if o.skip == "" && o.repo == "" {
		return exampleOptions{}, errors.New("an example needs repo=... (or skip=\"why\")")
	}
	if o.binary && (o.ids || o.path != "" || o.author != "") {
		return exampleOptions{}, errors.New("binary needs an output that is the same each time: no ids=any, path= or author=")
	}
	return o, nil
}

// splitWords splits a line at spaces, the way a shell does for the simple cases:
// 'text' and "text" are one word, or a part of one, and nothing is an escape.
func splitWords(s string) ([]string, error) {
	var words []string
	var cur strings.Builder
	inWord := false
	var quote rune
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote, inWord = r, true
		case r == ' ' || r == '\t':
			if inWord {
				words = append(words, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("a quote is not closed in %q", s)
	}
	if inWord {
		words = append(words, cur.String())
	}
	return words, nil
}

// invocation is a command of an example: mtqg with arguments, which may be given the
// output of a git command as its input, and may have its output thrown away.
type invocation struct {
	git           []string // git and its arguments, whose output is the input of mtqg
	args          []string // the arguments of mtqg
	discardStdout bool
}

func parseInvocation(text string) (invocation, error) {
	words, err := splitWords(text)
	if err != nil {
		return invocation{}, err
	}
	var inv invocation
	if n := len(words); n > 0 && words[n-1] == ">/dev/null" {
		inv.discardStdout = true
		words = words[:n-1]
	}
	if i := slices.Index(words, "|"); i >= 0 {
		inv.git, words = words[:i], words[i+1:]
		if len(inv.git) == 0 || inv.git[0] != "git" {
			return invocation{}, fmt.Errorf("only git may come before | in %q", text)
		}
	}
	if len(words) == 0 || words[0] != "mtqg" {
		return invocation{}, fmt.Errorf("a command of an example starts with mtqg: %q", text)
	}
	inv.args = words[1:]
	return inv, nil
}

// TestDocExamplesAreMarkedAndMatch checks the documents without running anything: that no
// example is left without a mark, and that the two languages have the same examples.
func TestDocExamplesAreMarkedAndMatch(t *testing.T) {
	var docs []*document
	for _, name := range documents {
		d := loadDocument(t, name)
		for _, p := range d.problems {
			t.Error(p)
		}
		docs = append(docs, d)
	}

	for _, d := range docs {
		ja := strings.HasSuffix(d.name, "_ja.md")
		for _, ex := range d.examples {
			if ex.opts.repo == "" || slices.Contains(neutralFixtures, ex.opts.repo) {
				continue
			}
			if strings.HasSuffix(ex.opts.repo, "_ja") != ja {
				t.Errorf("%s:%d: the fixture %q is of the other language (the English document has English records, the Japanese one Japanese)", d.name, ex.line, ex.opts.repo)
			}
		}
	}

	en, ja := docs[0], docs[1]
	if len(en.examples) != len(ja.examples) {
		t.Fatalf("%s has %d examples and %s has %d: a change to one has to be made to the other", en.name, len(en.examples), ja.name, len(ja.examples))
	}
	for i := range en.examples {
		a, b := strings.ReplaceAll(en.examples[i].mark, "_ja", ""), strings.ReplaceAll(ja.examples[i].mark, "_ja", "")
		if a != b {
			t.Errorf("example %d differs in its mark: %s:%d has %q, %s:%d has %q", i+1, en.name, en.examples[i].line, a, ja.name, ja.examples[i].line, b)
		}
		if len(en.examples[i].cmds) != len(ja.examples[i].cmds) {
			t.Errorf("example %d has %d commands in %s (line %d) and %d in %s (line %d)", i+1, len(en.examples[i].cmds), en.name, en.examples[i].line, len(ja.examples[i].cmds), ja.name, ja.examples[i].line)
		}
	}
}

// TestDocExamples runs every example and compares what it prints with the document.
func TestDocExamples(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	// mtqg prints the path that git and the file system give, with links and short
	// names resolved (/private/var on macOS, long names on Windows), so the directory
	// the examples run in is resolved too. Otherwise path= would not find it.
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r := &exampleRunner{root: root, templates: map[string]string{}}
	for _, name := range documents {
		d := loadDocument(t, name)
		if len(d.problems) > 0 {
			t.Fatalf("%s has %d problems; TestDocExamplesAreMarkedAndMatch lists them", name, len(d.problems))
		}
		var edits []edit
		for _, ex := range d.examples {
			t.Run(fmt.Sprintf("%s:%d", name, ex.line), func(t *testing.T) {
				if ex.opts.skip != "" {
					t.Skipf("skipped: %s", ex.opts.skip)
				}
				got := r.outputs(t, ex, false)
				if ex.opts.binary {
					built := r.outputs(t, ex, true)
					for i := range got {
						if built[i] != got[i] {
							t.Errorf("`$ %s`: the built binary prints something else than Run does:\n--- Run\n%s\n--- binary\n%s", ex.cmds[i].text, got[i], built[i])
						}
					}
				}
				for i, c := range ex.cmds {
					shown := strings.Join(d.lines[c.outStart:c.outEnd], "\n")
					if same(shown, got[i], ex.opts.ids) {
						continue
					}
					edits = append(edits, edit{c.outStart, c.outEnd, lines(got[i])})
					if !*update {
						t.Errorf("`$ %s` prints something else than the document shows (make docs-examples writes what it prints into the document):\n--- document\n%s\n--- mtqg\n%s", c.text, shown, got[i])
					}
				}
			})
		}
		if *update && len(edits) > 0 {
			d.apply(t, edits)
		}
	}
}

// same says whether an example shows what mtqg printed. With ids, IDs are not compared.
func same(shown, got string, ids bool) bool {
	if ids {
		shown, got = anyIDPattern.ReplaceAllString(shown, "<id>"), anyIDPattern.ReplaceAllString(got, "<id>")
	}
	return shown == got
}

func lines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// edit replaces lines[start:end] of a document.
type edit struct {
	start, end int
	with       []string
}

// apply writes the edits into the document. The rest of the file is not touched.
func (d *document) apply(t *testing.T, edits []edit) {
	t.Helper()
	sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	out := slices.Clone(d.lines)
	for _, e := range edits {
		out = slices.Concat(out[:e.start], e.with, out[e.end:])
	}
	path := filepath.Join("..", "docs", "reference", d.name)
	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d examples into %s", len(edits), path)
}

// exampleRunner runs examples in copies of repositories built from fixtures.
type exampleRunner struct {
	root      string
	templates map[string]string // the name of a fixture -> the repository built from it
	copies    int
}

// outputs runs the commands of an example, in a fresh copy of its repository, and
// returns what each printed (standard output, then standard error), with the path of the
// repository and the like changed as the mark says. With useBinary, the built mtqg runs.
func (r *exampleRunner) outputs(t *testing.T, ex *example, useBinary bool) []string {
	t.Helper()
	dir := r.copyOf(t, ex.opts.repo)
	vars := map[string]string{}
	if kind, name, ok := strings.Cut(ex.opts.author, ":"); ok {
		vars["MTQG_AUTHOR_KIND"], vars["MTQG_AUTHOR_NAME"] = kind, name
	}
	for _, text := range ex.opts.setup {
		if _, code := run(t, dir, vars, text, useBinary); code != 0 {
			t.Fatalf("the setup %q failed with exit code %d", text, code)
		}
	}
	var got []string
	for _, c := range ex.cmds {
		out, _ := run(t, dir, vars, c.text, useBinary)
		if ex.opts.path != "" {
			out = strings.ReplaceAll(out, dir, ex.opts.path)
		}
		got = append(got, strings.TrimRight(out, "\n"))
	}
	return got
}

// run runs one command of an example in dir.
func run(t *testing.T, dir string, vars map[string]string, text string, useBinary bool) (string, int) {
	t.Helper()
	inv, err := parseInvocation(text)
	if err != nil {
		t.Fatal(err)
	}
	stdin := ""
	if inv.git != nil {
		stdin = gitIn(t, dir, inv.git[1:]...)
	}
	var stdout, stderr string
	var code int
	if useBinary {
		stdout, stderr, code = runBinary(t, dir, vars, stdin, inv.args...)
	} else {
		stdout, stderr, code = runInProcess(dir, vars, stdin, inv.args...)
	}
	if inv.discardStdout {
		stdout = ""
	}
	return stdout + stderr, code
}

// runInProcess calls Run with the clock, the time zone and the terminal fixed, and
// with only the environment variables that vars holds.
func runInProcess(dir string, vars map[string]string, stdin string, args ...string) (stdout, stderr string, code int) {
	var out, errOut bytes.Buffer
	env := cli.Env{
		Stdin:    strings.NewReader(stdin),
		Stdout:   &out,
		Stderr:   &errOut,
		Getenv:   func(k string) string { return vars[k] },
		Getwd:    func() (string, error) { return dir, nil },
		Now:      func() time.Time { return examplesNow },
		Location: time.UTC,
		ReadFile: func(name string) ([]byte, error) {
			if !filepath.IsAbs(name) {
				name = filepath.Join(dir, name)
			}
			return os.ReadFile(name)
		},
		RunEditor: func([]string) error { return errors.New("the examples do not run an editor") },
	}
	code = cli.Run(env, args)
	return out.String(), errOut.String(), code
}

// runBinary runs the built mtqg. It has the real clock, so only an example that does
// not show what "today" is may use it.
func runBinary(t *testing.T, dir string, vars map[string]string, stdin string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	cmd.Env = append(cleanEnv(), "TZ=UTC")
	for k, v := range vars {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	cmd.Stdin = strings.NewReader(stdin)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("cannot run mtqg: %v", err)
		}
		code = exit.ExitCode()
	}
	return out.String(), errOut.String(), code
}

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return string(out)
}

// copyOf makes a copy of the repository of a fixture, in a directory named repoName,
// and returns its path. Every example has its own, so that what one writes is not
// there for the next.
func (r *exampleRunner) copyOf(t *testing.T, fixture string) string {
	t.Helper()
	src := r.template(t, fixture)
	r.copies++
	dst := filepath.Join(r.root, fmt.Sprintf("run-%d", r.copies), repoName)
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), info.Mode().Perm()|0o700)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		// Git makes its objects read-only, and a copy that cannot be removed is a
		// problem on Windows.
		return os.WriteFile(filepath.Join(dst, rel), data, info.Mode().Perm()|0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

// template builds the repository of a fixture once. The files of the fixture, in the
// order of their names, are added to the journal, and each is committed, except one
// whose name says it is not. That makes the last commit, and what has not been
// committed yet, part of the fixture.
func (r *exampleRunner) template(t *testing.T, fixture string) string {
	t.Helper()
	if dir, ok := r.templates[fixture]; ok {
		return dir
	}
	dir := filepath.Join(r.root, "template-"+fixture, repoName)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "init", "-q", "-b", "main")
	gitIn(t, dir, "config", "user.name", "yamada")
	gitIn(t, dir, "config", "user.email", "yamada@example.com")
	if fixture != "none" {
		if _, code := run(t, dir, nil, "mtqg init", false); code != 0 {
			t.Fatalf("mtqg init failed in the fixture %s", fixture)
		}
	}
	if fixture != "none" && fixture != "empty" {
		files, err := filepath.Glob(filepath.Join("testdata", "examples", fixture, "*.jsonl"))
		if err != nil || len(files) == 0 {
			t.Fatalf("the fixture %q has no *.jsonl files in e2e/testdata/examples (%v)", fixture, err)
		}
		sort.Strings(files)
		journal := filepath.Join(dir, ".mtqg", "journal.jsonl")
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			appendFile(t, journal, data)
			if !strings.Contains(filepath.Base(f), "uncommitted") {
				gitIn(t, dir, "add", ".mtqg")
				gitIn(t, dir, "commit", "-q", "-m", filepath.Base(f))
			}
		}
	}
	r.templates[fixture] = dir
	return dir
}

func appendFile(t *testing.T, path string, data []byte) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}
