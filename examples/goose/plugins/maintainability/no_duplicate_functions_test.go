package maintainability

import (
	"encoding/json"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"slices"
	"strings"
	"testing"

	"github.com/egoisutolabs/goose-lint/internal/engine"
)

func lintDuplicates(t *testing.T, sources map[string]string, configure func(string) []engine.ConfiguredRule) []engine.Diagnostic {
	t.Helper()
	var names []string
	for name := range sources {
		names = append(names, name)
	}
	slices.Sort(names)
	fs := token.NewFileSet()
	var files []engine.File
	var syntax []*ast.File
	for _, name := range names {
		file, err := parser.ParseFile(fs, name, sources[name], parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		syntax = append(syntax, file)
		files = append(files, engine.File{Fset: fs, AST: file, Filename: name, Source: []byte(sources[name])})
	}
	info := &types.Info{
		Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{}, Implicits: map[ast.Node]types.Object{},
	}
	conf := types.Config{Importer: importer.ForCompiler(fs, "source", nil)}
	pkg, err := conf.Check("example.com/p", fs, syntax, info)
	if err != nil {
		t.Fatal(err)
	}
	for i := range files {
		files[i].Pkg, files[i].Info = pkg, info
	}
	if configure == nil {
		configure = func(string) []engine.ConfiguredRule { return duplicateRuleConfig(`{"minStatements":1}`) }
	}
	var out []engine.Diagnostic
	for _, diags := range engine.LintAll(files, configure, engine.LintOptions{ReportUnusedSuppressions: engine.Warn}) {
		out = append(out, diags...)
	}
	return out
}

func duplicateRuleConfig(options string) []engine.ConfiguredRule {
	return []engine.ConfiguredRule{{ID: "maintainability/no-duplicate-functions", Plugin: "maintainability", Rule: NoDuplicateFunctions, Severity: engine.Warn, Options: json.RawMessage(options)}}
}

func TestDuplicateFunctionNormalization(t *testing.T) {
	for _, tc := range []struct {
		name, code string
		duplicates int
	}{
		{"renamed locals", "func first(x int) int { y := x+1; return y }; func second(a int) int { b := a+1; return b }", 1},
		{"format and comments", "func first(x int) int { /* why */ return x+1 }\nfunc second(y int) int {\n// a reason\nreturn y + 1\n}", 1},
		{"multiline trailing comma", "var call func(int,int) int; func a(x int) int { return call(x,1) }; func b(y int) int { return call(\ny,\n1,\n) }", 1},
		{"three copies", "func a(x int) int { return x+1 }; func b(y int) int { return y+1 }; func c(z int) int { return z+1 }", 2},
		{"named results", "func a(x int) (out int) { out=x+1; return }; func b(y int) (result int) { result=y+1; return }", 1},
		{"methods", "type T struct{ N int }; func (t T) First() int { return t.N+1 }; func (r T) Second() int { return r.N+1 }", 1},
		{"generics", "func a[T ~int](x T) T { return x+1 }; func b[U ~int](y U) U { return y+1 }", 1},
		{"nested closures", "func a(x int) func(int) int { return func(y int) int { return x+y } }; func b(n int) func(int) int { return func(m int) int { return n+m } }", 1},
		{"recursion", "func a(x int) int { if x==0 { return 1 }; return x*a(x-1) }; func b(y int) int { if y==0 { return 1 }; return y*b(y-1) }", 1},
		{"labels", "func a(x int) int {again: x--; if x>0 {goto again}; return x}; func b(y int) int {retry: y--; if y>0 {goto retry}; return y}", 1},
		{"shadowed locals", "func a(x int) int { y:=x; if x>0 { y:=2; _=y }; return y }; func b(n int) int { m:=n; if n>0 { m:=2; _=m }; return m }", 1},
		{"different operators", "func a(x int) int { return x+1 }; func b(y int) int { return y-1 }", 0},
		{"different literals", "func a(x int) int { return x+1 }; func b(y int) int { return y+2 }", 0},
		{"different types", "func a(x int) int { return x+1 }; func b(y int64) int64 { return y+1 }", 0},
		{"different globals", "var left,right int; func a(x int) int { return x+left }; func b(y int) int { return y+right }", 0},
		{"different calls", "var first,second func(int) int; func a(x int) int { return first(x) }; func b(y int) int { return second(y) }", 0},
		{"parameter order", "func a(x,y int) int { return x-y }; func b(n,m int) int { return m-n }", 0},
		{"different fields", "type T struct{ A,B int }; func a(t T) int { return t.A }; func b(r T) int { return r.B }", 0},
		{"different field keys", "type T struct{ A,B int }; func a(x int) T { return T{A:x} }; func b(y int) T { return T{B:y} }", 0},
		{"different receivers", "type T int; type U int; func (t T) A() int { return int(t) }; func (u U) B() int { return int(u) }", 0},
		{"different shadow binding", "func a(x int) int { y:=x; if x>0 { x:=2; _=y; _=x }; return y }; func b(n int) int { m:=n; if n>0 { n:=2; _=n; _=n }; return m }", 0},
		{"empty and bodyless", "func a() {}; func b() {}; func external()", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diags := lintDuplicates(t, map[string]string{"p.go": "package p\n" + tc.code}, nil)
			if len(diags) != tc.duplicates {
				t.Fatalf("want %d duplicates, got %+v", tc.duplicates, diags)
			}
			for _, d := range diags {
				if d.RuleID != "maintainability/no-duplicate-functions" || !strings.Contains(d.Message, "duplicates") || d.Fix != nil || !strings.Contains(d.Message, "p.go:") {
					t.Errorf("unexpected duplicate diagnostic: %+v", d)
				}
			}
		})
	}
}

func TestDuplicatesAcrossFilesAndImportAliases(t *testing.T) {
	diags := lintDuplicates(t, map[string]string{
		"a.go": "package p\nimport s \"strings\"\nfunc first(x string) string { return s.TrimSpace(x) }",
		"b.go": "package p\nimport text \"strings\"\nfunc second(y string) string { return text.TrimSpace(y) }",
	}, nil)
	if len(diags) != 1 || diags[0].Filename != "b.go" || !strings.Contains(diags[0].Message, "first at a.go:3:6") {
		t.Fatalf("missing cross-file reference: %+v", diags)
	}
}

func TestDuplicateOptionsAndDisabledEvidence(t *testing.T) {
	sources := map[string]string{
		"a.go": "package p\nfunc first(x int) int { y:=x+1; return y }",
		"b.go": "package p\nfunc second(y int) int { z:=y+1; return z }",
	}
	for _, tc := range []struct {
		name, options, disabled string
		want                    int
	}{
		{"default minimum", "", "", 0},
		{"at minimum", `{"minStatements":2}`, "", 1},
		{"below minimum", `{"minStatements":3}`, "", 0},
		{"disabled reference first", `{"minStatements":1}`, "a.go", 1},
		{"disabled reference last", `{"minStatements":1}`, "b.go", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diags := lintDuplicates(t, sources, func(name string) []engine.ConfiguredRule {
				if name == tc.disabled {
					return nil
				}
				return duplicateRuleConfig(tc.options)
			})
			if len(diags) != tc.want {
				t.Fatalf("want %d, got %+v", tc.want, diags)
			}
			if tc.disabled != "" && len(diags) == 1 && diags[0].Filename == tc.disabled {
				t.Fatalf("reported disabled file: %+v", diags)
			}
		})
	}
	for _, options := range []string{`{"minStatements":0}`, `{"minStatements":-1}`, `{"typo":5}`} {
		diags := lintDuplicates(t, map[string]string{"p.go": "package p\nfunc f() {}"}, func(string) []engine.ConfiguredRule { return duplicateRuleConfig(options) })
		if len(diags) != 1 || !strings.Contains(diags[0].Message, "invalid options") {
			t.Fatalf("invalid options accepted: %+v", diags)
		}
	}
}

func TestDuplicateSuppressionAndOverrides(t *testing.T) {
	sources := map[string]string{
		"a.go": "package p\nfunc first(x int) int { return x+1 }",
		"b.go": "package p\n//goose:ignore maintainability/no-duplicate-functions shared implementation required\nfunc second(y int) int { return y+1 }",
	}
	if diags := lintDuplicates(t, sources, nil); len(diags) != 0 {
		t.Fatalf("duplicate suppression was not used: %+v", diags)
	}
	sources["b.go"] = "package p\nfunc second(y int) int { return y+1 }"
	sources["c.go"] = "package p\nfunc third(z int) int { return z+1 }"
	diags := lintDuplicates(t, sources, func(name string) []engine.ConfiguredRule {
		if name == "b.go" {
			return duplicateRuleConfig(`{"minStatements":2}`)
		}
		return duplicateRuleConfig(`{"minStatements":1}`)
	})
	if len(diags) != 1 || diags[0].Filename != "c.go" {
		t.Fatalf("per-file options ignored: %+v", diags)
	}
}

func TestDuplicateTestFilePolicy(t *testing.T) {
	sources := map[string]string{
		"p.go":      "package p\nfunc original(x int) int { return x+1 }",
		"p_test.go": "package p\nfunc testCopy(y int) int { return y+1 }",
	}
	if diags := lintDuplicates(t, sources, nil); len(diags) != 0 {
		t.Fatalf("test files should be excluded by default: %+v", diags)
	}
	diags := lintDuplicates(t, sources, func(string) []engine.ConfiguredRule {
		return duplicateRuleConfig(`{"minStatements":1,"checkTests":true}`)
	})
	if len(diags) != 1 || diags[0].Filename != "p_test.go" {
		t.Fatalf("checkTests did not include the test helper: %+v", diags)
	}
}

func TestDuplicateDefaultMinimum(t *testing.T) {
	for _, count := range []int{4, 5} {
		source := "package p\nfunc first(x int) int {" + strings.Repeat("x++;", count-1) + "return x}\nfunc second(y int) int {" + strings.Repeat("y++;", count-1) + "return y}"
		diags := lintDuplicates(t, map[string]string{"p.go": source}, func(string) []engine.ConfiguredRule { return duplicateRuleConfig("") })
		if len(diags) != count-4 {
			t.Fatalf("default minimum boundary for %d statements: %+v", count, diags)
		}
	}
}
