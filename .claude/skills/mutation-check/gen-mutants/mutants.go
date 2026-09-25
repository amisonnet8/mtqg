package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
)

// A Mutant is one syntactic change to try: replacing the bytes at [Start,End)
// of the original source with Replacement. Applying it and nothing else must
// still be valid Go (mutants_test.go checks this on a sample).
type Mutant struct {
	ID          string // "<line>:<col>:<operator>", unique within one file
	Operator    string // the mutation family, e.g. "ROR", "bool-literal"
	Description string // e.g. "== -> !="
	Start, End  int    // byte offsets into the original source
	Replacement string
}

// The operator tables. Each one is a purely syntactic substitution: given the
// token found, what to replace it with. Kept small and conventional (the
// reduced "relational operator replacement" set, not every possible pairing)
// so a survivor is easy to read as one sentence, the same way a hand-written
// mutate.sh entry is (SKILL.md).
var (
	rorTable = map[token.Token]token.Token{
		token.EQL: token.NEQ,
		token.NEQ: token.EQL,
		token.LSS: token.LEQ,
		token.LEQ: token.LSS,
		token.GTR: token.GEQ,
		token.GEQ: token.GTR,
	}
	logicalTable = map[token.Token]token.Token{
		token.LAND: token.LOR,
		token.LOR:  token.LAND,
	}
	arithTable = map[token.Token]token.Token{
		token.ADD: token.SUB,
		token.SUB: token.ADD,
		token.MUL: token.QUO,
		token.QUO: token.MUL,
	}
	incDecTable = map[token.Token]token.Token{
		token.INC: token.DEC,
		token.DEC: token.INC,
	}
	assignTable = map[token.Token]token.Token{
		token.ADD_ASSIGN: token.SUB_ASSIGN,
		token.SUB_ASSIGN: token.ADD_ASSIGN,
	}
)

// find walks file (parsed from src with fset) and returns every mutation this
// catalog knows how to make. Only mutations that need no type information are
// attempted: an operator swap, a boolean literal flip, removing a leading "!",
// or shifting an integer literal by one. Anything that needs the words of the
// specification to describe (see testing.md's mutation lists) is still a job
// for a person and mutate.sh; this tool only covers what syntax alone can
// tell it.
func find(fset *token.FileSet, file *ast.File, src []byte) []Mutant {
	var out []Mutant

	// A struct field or method could be literally named "true"/"false" (its
	// own identifier namespace, not the predeclared one): both the field's
	// declaration (ast.Field.Names) and a selector's use of it (x.Sel) read
	// the same as the boolean literal but are not it. Collect those idents
	// first so the main pass can skip them.
	skipSel := map[*ast.Ident]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.SelectorExpr:
			skipSel[node.Sel] = true
		case *ast.Field:
			for _, name := range node.Names {
				skipSel[name] = true
			}
		}
		return true
	})

	add := func(pos token.Pos, operator, description string, oldLen int, replacement string) {
		p := fset.Position(pos)
		start := p.Offset
		out = append(out, Mutant{
			ID:          fmt.Sprintf("%d:%d:%s", p.Line, p.Column, operator),
			Operator:    operator,
			Description: description,
			Start:       start,
			End:         start + oldLen,
			Replacement: replacement,
		})
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.BinaryExpr:
			if to, ok := rorTable[node.Op]; ok {
				add(node.OpPos, "ROR", node.Op.String()+" -> "+to.String(), len(node.Op.String()), to.String())
			} else if to, ok := logicalTable[node.Op]; ok {
				add(node.OpPos, "logical", node.Op.String()+" -> "+to.String(), len(node.Op.String()), to.String())
			} else if to, ok := arithTable[node.Op]; ok {
				add(node.OpPos, "arithmetic", node.Op.String()+" -> "+to.String(), len(node.Op.String()), to.String())
			}
		case *ast.IncDecStmt:
			if to, ok := incDecTable[node.Tok]; ok {
				add(node.TokPos, "inc-dec", node.Tok.String()+" -> "+to.String(), len(node.Tok.String()), to.String())
			}
		case *ast.AssignStmt:
			if to, ok := assignTable[node.Tok]; ok {
				add(node.TokPos, "compound-assign", node.Tok.String()+" -> "+to.String(), len(node.Tok.String()), to.String())
			}
		case *ast.UnaryExpr:
			if node.Op == token.NOT {
				add(node.OpPos, "negation", "remove the leading !", len("!"), "")
			}
		case *ast.Ident:
			if skipSel[node] {
				break
			}
			switch node.Name {
			case "true":
				add(node.Pos(), "bool-literal", "true -> false", len("true"), "false")
			case "false":
				add(node.Pos(), "bool-literal", "false -> true", len("false"), "true")
			}
		case *ast.BasicLit:
			if node.Kind == token.INT {
				if v, err := strconv.ParseInt(node.Value, 0, 64); err == nil {
					for _, delta := range [2]int64{1, -1} {
						nv := v + delta
						add(node.Pos(), "boundary", fmt.Sprintf("%s -> %d", node.Value, nv), len(node.Value), strconv.FormatInt(nv, 10))
					}
				}
			}
		}
		return true
	})
	return out
}

// apply returns src with one mutation spliced in.
func apply(src []byte, m Mutant) []byte {
	out := make([]byte, 0, len(src)-(m.End-m.Start)+len(m.Replacement))
	out = append(out, src[:m.Start]...)
	out = append(out, m.Replacement...)
	out = append(out, src[m.End:]...)
	return out
}
