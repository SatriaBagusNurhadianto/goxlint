package maintainability

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/egoisutolabs/goxlint/goxlinttest"
)

func TestComplexityScore(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       int
	}{
		{"empty", "", 1},
		{"straight line", "x := 1; x++; return", 1},
		{"if else", "if x { f() } else { g() }", 2},
		{"else if", "if x { f() } else if y { g() } else { h() }", 3},
		{"loops", "for { break }; for range xs { f() }", 3},
		{"short circuit", "if a && b || c { f() }", 4},
		{"switch clauses", "switch x { case 1, 2: f(); case 3: g(); default: h() }", 3},
		{"type switch", "switch x.(type) { case int, string: f(); case bool: g(); default: h() }", 3},
		{"select", "select { case <-a: f(); case b <- 1: g(); default: h() }", 3},
		{"nested branches", "if a { for range xs { if b { return } } }", 4},
		{"closure independent", "f := func() { if a && b { return } }; if x { f() }", 2},
		{"bitwise operators", "x := a & b | c; _ = x", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), "p.go", "package p\nfunc f() {"+tc.body+"}", 0)
			if err != nil {
				t.Fatal(err)
			}
			fn, ok := file.Decls[0].(*ast.FuncDecl)
			if !ok {
				t.Fatal("missing function")
			}
			if score := complexity(fn.Body); score != tc.want {
				t.Fatalf("score = %d, want %d", score, tc.want)
			}
		})
	}
}

func TestCyclomaticComplexity(t *testing.T) {
	goxlinttest.Run(t, CyclomaticComplexity, goxlinttest.Cases{
		Valid: []goxlinttest.Case{
			{Code: "func f(x bool) { if x { return } }", Options: map[string]int{"max": 2}},
			{Code: "func f() {}", Options: map[string]int{"max": 1}},
			{Code: "func external()"},
		},
		Invalid: []goxlinttest.Case{
			{Code: "func f(x bool) {" + strings.Repeat("if x {};", 10) + "}", Errors: []goxlinttest.Error{{Message: "Function f has cyclomatic complexity 11 (maximum 10)."}}},
			{Code: "func f(x bool) { if x { return } }", Options: map[string]int{"max": 1}, Errors: []goxlinttest.Error{{Message: "Function f has cyclomatic complexity 2 (maximum 1).", Line: 1, Column: 6}}},
			{Code: "type T int\nfunc (t T) f() { if t > 0 { return } }", Options: map[string]int{"max": 1}, Errors: []goxlinttest.Error{{MessageID: "complex", Line: 2}}},
			{Code: "func outer() { _ = func(x bool) { if x { return } } }", Options: map[string]int{"max": 1}, Errors: []goxlinttest.Error{{Message: "Function <literal> has cyclomatic complexity 2 (maximum 1).", Line: 1}}},
			{Code: "func f() {}", Options: map[string]int{"max": 0}, Errors: []goxlinttest.Error{{Message: "invalid options: max must be at least 1"}}},
			{Code: "func f() {}", Options: map[string]int{"max": -1}, Errors: []goxlinttest.Error{{Message: "invalid options: max must be at least 1"}}},
			{Code: "func f() {}", Options: map[string]int{"typo": 10}, Errors: []goxlinttest.Error{{Message: "invalid options for test/rule: json: unknown field \"typo\""}}},
		},
	})
}
