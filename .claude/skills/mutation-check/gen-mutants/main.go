// Command gen-mutants finds the mutations mutants.go knows how to make in one
// Go source file, and for each one: applies it, runs a test command, puts the
// file back, and says whether the tests noticed (killed) or not (SURVIVED).
// It is the automatic half of the mutation-check skill (SKILL.md): the
// mutations here come from syntax alone (an operator swap, a boolean literal
// flip, an off-by-one on an integer literal, ...), not from a person
// describing a promise of the specification. mutate.sh is still how those are
// tried.
//
// Build once, run many times (it is its own module, kept out of the
// repository it mutates):
//
//	go build -o /tmp/gen-mutants ./.claude/skills/mutation-check/gen-mutants
//	/tmp/gen-mutants -C /path/to/repo -file internal/cli/candidates.go -list
//	/tmp/gen-mutants -C /path/to/repo -file internal/cli/candidates.go -- \
//		go test -count=1 ./internal/cli/ -run TestCandidates
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"strings"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr *os.File) int {
	fs := flag.NewFlagSet("gen-mutants", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("C", ".", "directory to run in (the repository root); -file and the test command are relative to it")
	file := fs.String("file", "", "the Go source file to mutate")
	list := fs.Bool("list", false, "only list the mutations found; do not run any test command")
	noBaseline := fs.Bool("no-baseline", false, "skip checking that the test command passes before any mutation")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: gen-mutants -C <dir> -file <path.go> [-list] [-no-baseline] -- <test command>...")
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *file == "" {
		fs.Usage()
		return 2
	}
	testCmd := fs.Args()
	if !*list && len(testCmd) == 0 {
		fs.Usage()
		fmt.Fprintln(stderr, "error: a test command is required after --, unless -list is given")
		return 2
	}

	if err := os.Chdir(*dir); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}

	src, err := os.ReadFile(*file)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, *file, src, parser.ParseComments)
	if err != nil {
		fmt.Fprintf(stderr, "error: parsing %s: %v\n", *file, err)
		return 2
	}

	mutants := find(fset, astFile, src)
	if *list {
		for _, m := range mutants {
			fmt.Fprintf(stdout, "%s:%s  %s\n", *file, m.ID, m.Description)
		}
		fmt.Fprintf(stdout, "%d mutation(s) found\n", len(mutants))
		return 0
	}
	if len(mutants) == 0 {
		fmt.Fprintln(stdout, "no mutations found")
		return 0
	}

	if !*noBaseline {
		if out, err := runCmd(testCmd); err != nil {
			fmt.Fprintf(stderr, "error: the tests fail before any mutation, so nothing they say means anything:\n%s\n", out)
			return 2
		}
	}

	killed, survived, notCompiled := 0, 0, 0
	for _, m := range mutants {
		mutated := apply(src, m)
		if err := os.WriteFile(*file, mutated, 0o644); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 2
		}
		out, runErr := runCmd(testCmd)
		if err := os.WriteFile(*file, src, 0o644); err != nil {
			fmt.Fprintf(stderr, "error: %s was not put back: %v\n", *file, err)
			return 2
		}

		name := fmt.Sprintf("%s:%s (%s)", *file, m.ID, m.Description)
		switch {
		case runErr == nil:
			fmt.Fprintln(stdout, "SURVIVED:", name)
			survived++
		case strings.Contains(out, "build failed") || strings.Contains(out, "setup failed"):
			fmt.Fprintln(stdout, "not compiled:", name, "(says nothing about the tests)")
			notCompiled++
		default:
			fmt.Fprintln(stdout, "killed:  ", name)
			killed++
		}
	}
	fmt.Fprintf(stdout, "\n%d killed, %d SURVIVED, %d not compiled (%d total)\n", killed, survived, notCompiled, len(mutants))
	if survived > 0 {
		return 1
	}
	return 0
}

func runCmd(args []string) (string, error) {
	cmd := exec.Command(args[0], args[1:]...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}
