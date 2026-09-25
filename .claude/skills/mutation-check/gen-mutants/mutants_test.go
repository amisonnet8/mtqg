package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

const sample = `package sample

func f(a, b int, ok bool) int {
	if a == b && ok {
		a++
	} else if a != b || !ok {
		a += 1
	}
	if a < 4 {
		return 0
	}
	if ok == true {
		return -1
	}
	return a + b
}

type T struct{ true bool }

func g(t T) bool {
	return t.true
}
`

func parseSample(t *testing.T, src string) (*token.FileSet, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "sample.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return fset, f
}

// Every mutant this catalog proposes must, applied alone, still be valid Go:
// a mutation that does not compile is exit code 3 in mutate.sh's vocabulary,
// which says nothing about the tests, so the catalog should not manufacture
// them where the AST already knows the syntax is fine.
func TestFindProducesParseableMutants(t *testing.T) {
	fset, f := parseSample(t, sample)
	src := []byte(sample)
	mutants := find(fset, f, src)
	if len(mutants) == 0 {
		t.Fatal("expected some mutants in the sample")
	}
	for _, m := range mutants {
		mutated := apply(src, m)
		if _, err := parser.ParseFile(token.NewFileSet(), "sample.go", mutated, 0); err != nil {
			t.Errorf("mutant %s (%s) produced invalid Go: %v\n%s", m.ID, m.Description, err, mutated)
		}
	}
}

// Applying a mutation and then applying its own reversal must reproduce the
// original bytes exactly: the tool restores the real file the same way.
func TestApplyRoundTrips(t *testing.T) {
	fset, f := parseSample(t, sample)
	src := []byte(sample)
	for _, m := range find(fset, f, src) {
		mutated := apply(src, m)
		reverted := apply(mutated, Mutant{
			Start:       m.Start,
			End:         m.Start + len(m.Replacement),
			Replacement: string(src[m.Start:m.End]),
		})
		if string(reverted) != string(src) {
			t.Errorf("mutant %s did not restore cleanly", m.ID)
		}
	}
}

// A struct field literally named "true" is its own identifier, not the
// boolean literal, even though it reads the same. Only the real literal
// (ok == true) should be found.
func TestFindSkipsSelectorFieldNamedLikeABoolLiteral(t *testing.T) {
	fset, f := parseSample(t, sample)
	src := []byte(sample)
	var boolMutants []Mutant
	for _, m := range find(fset, f, src) {
		if m.Operator == "bool-literal" {
			boolMutants = append(boolMutants, m)
		}
	}
	if len(boolMutants) != 1 {
		t.Fatalf("got %d bool-literal mutants, want 1 (t.true must not count): %+v", len(boolMutants), boolMutants)
	}
	if got := string(src[boolMutants[0].Start:boolMutants[0].End]); got != "true" {
		t.Errorf("the one bool-literal mutant points at %q, want the literal in `ok == true`", got)
	}
}

// Sanity check that each operator family this catalog claims to cover is
// actually found somewhere in the sample, so the table above cannot silently
// stop matching without a test noticing.
func TestFindCoversEveryOperatorFamily(t *testing.T) {
	fset, f := parseSample(t, sample)
	seen := map[string]bool{}
	for _, m := range find(fset, f, []byte(sample)) {
		seen[m.Operator] = true
	}
	for _, want := range []string{"ROR", "logical", "arithmetic", "inc-dec", "compound-assign", "negation", "bool-literal", "boundary"} {
		if !seen[want] {
			t.Errorf("no %q mutant found in the sample", want)
		}
	}
}

func TestApply(t *testing.T) {
	src := []byte("abcXYZdef")
	m := Mutant{Start: 3, End: 6, Replacement: "123"}
	got := string(apply(src, m))
	if want := "abc123def"; got != want {
		t.Errorf("apply() = %q, want %q", got, want)
	}
}
