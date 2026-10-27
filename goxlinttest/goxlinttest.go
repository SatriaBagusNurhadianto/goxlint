// Package goxlinttest is the RuleTester for goxlint rules.
//
//	func TestNoFoo(t *testing.T) {
//		goxlinttest.Run(t, NoFoo, goxlinttest.Cases{
//			Valid: []goxlinttest.Case{
//				{Code: `var bar = 1`},
//			},
//			Invalid: []goxlinttest.Case{
//				{
//					Code:   `var foo = 1`,
//					Errors: []goxlinttest.Error{{MessageID: "foo", Line: 1}},
//					Output: `var bar = 1`,
//				},
//			},
//		})
//	}
//
// Code without a package clause is wrapped in "package p". Line numbers in
// Errors are relative to Code as written. Every case is type-checked, so code
// must compile, apart from unused variables and imports.
package goxlinttest

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"sync"
	"testing"

	"github.com/egoisutolabs/goxlint"
	"github.com/egoisutolabs/goxlint/internal/engine"
)

// Cases are the inputs for one rule.
type Cases struct {
	Valid   []Case
	Invalid []Case
}

// Case is one test input.
type Case struct {
	Name string
	Code string
	// Filename defaults to "p.go". Use a _test.go name to test test-only behavior.
	Filename string
	// Options are encoded to JSON and passed to the rule like config options.
	Options any
	// Errors are the expected diagnostics, in position order. Invalid cases only.
	Errors []Error
	// Output is the expected code after applying every fix. Leave empty when
	// the rule reports no fix.
	Output string
}

// Error is an expected diagnostic. Zero fields are not checked.
type Error struct {
	MessageID string
	Message   string
	Line      int
	Column    int
}

const ruleID = "test/rule"

// Run tests rule against cases.
func Run(t *testing.T, rule *goxlint.Rule, cases Cases) {
	t.Helper()
	for i, c := range cases.Valid {
		t.Run(caseName("valid", i, c), func(t *testing.T) {
			t.Helper()
			res := lint(t, rule, c)
			for _, d := range res.diags {
				t.Errorf("unexpected diagnostic at %d:%d: %s", res.line(d), d.Start.Column, d.Message)
			}
		})
	}
	for i, c := range cases.Invalid {
		t.Run(caseName("invalid", i, c), func(t *testing.T) {
			t.Helper()
			if len(c.Errors) == 0 {
				t.Fatal("invalid case must list Errors")
			}
			res := lint(t, rule, c)
			checkErrors(t, rule, res, c.Errors)
			checkOutput(t, res, c)
		})
	}
}

func caseName(kind string, i int, c Case) string {
	if c.Name != "" {
		return kind + "/" + c.Name
	}
	return fmt.Sprintf("%s/%d", kind, i)
}

type result struct {
	diags  []engine.Diagnostic
	src    []byte
	prefix string
}

// line converts a diagnostic line back to a line in Case.Code.
func (r result) line(d engine.Diagnostic) int {
	return d.Start.Line - strings.Count(r.prefix, "\n")
}

var (
	mu   sync.Mutex
	fset = token.NewFileSet()
	imp  types.Importer
)

func lint(t *testing.T, rule *goxlint.Rule, c Case) result {
	t.Helper()
	prefix := ""
	if !hasPackageClause(c.Code) {
		prefix = "package p\n\n"
	}
	src := []byte(prefix + c.Code)
	filename := c.Filename
	if filename == "" {
		filename = "p.go"
	}

	var opts json.RawMessage
	if c.Options != nil {
		b, err := json.Marshal(c.Options)
		if err != nil {
			t.Fatalf("encoding options: %v", err)
		}
		opts = b
	}

	mu.Lock()
	defer mu.Unlock()
	if imp == nil {
		imp = importer.ForCompiler(fset, "source", nil)
	}
	af, err := parser.ParseFile(fset, filename, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	info := &types.Info{
		Types:        map[ast.Expr]types.TypeAndValue{},
		Defs:         map[*ast.Ident]types.Object{},
		Uses:         map[*ast.Ident]types.Object{},
		Implicits:    map[ast.Node]types.Object{},
		Instances:    map[*ast.Ident]types.Instance{},
		Selections:   map[*ast.SelectorExpr]*types.Selection{},
		Scopes:       map[ast.Node]*types.Scope{},
		FileVersions: map[*ast.File]string{},
	}
	conf := types.Config{Importer: imp, Error: func(err error) {
		msg := err.Error()
		if strings.Contains(msg, "declared and not used") || strings.Contains(msg, "imported and not used") {
			return
		}
		t.Errorf("type error: %v", err)
	}}
	// Type errors were already reported through conf.Error.
	pkg, _ := conf.Check(af.Name.Name, fset, []*ast.File{af}, info)
	if t.Failed() {
		t.FailNow()
	}

	cr := engine.ConfiguredRule{ID: ruleID, Plugin: "test", Rule: rule, Severity: engine.Error, Options: opts}
	diags := engine.LintFile(engine.File{Fset: fset, AST: af, Filename: filename, Source: src, Pkg: pkg, Info: info}, []engine.ConfiguredRule{cr}, engine.LintOptions{})
	return result{diags: diags, src: src, prefix: prefix}
}

func hasPackageClause(code string) bool {
	for line := range strings.SplitSeq(code, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		return strings.HasPrefix(line, "package ")
	}
	return false
}

func checkErrors(t *testing.T, rule *goxlint.Rule, res result, want []Error) {
	t.Helper()
	for _, d := range res.diags {
		if strings.HasPrefix(d.Message, "internal error") {
			t.Fatalf("%s", d.Message)
		}
	}
	if len(res.diags) != len(want) {
		var got []string
		for _, d := range res.diags {
			got = append(got, fmt.Sprintf("  %d:%d %s", res.line(d), d.Start.Column, d.Message))
		}
		t.Fatalf("got %d diagnostics, want %d:\n%s", len(res.diags), len(want), strings.Join(got, "\n"))
	}
	for i, w := range want {
		d := res.diags[i]
		if w.MessageID != "" {
			tmpl, ok := rule.Meta.Messages[w.MessageID]
			if !ok {
				t.Errorf("error %d: rule has no message ID %q", i, w.MessageID)
			} else if !matchesTemplate(tmpl, d.Message) {
				t.Errorf("error %d: message %q does not come from message ID %q", i, d.Message, w.MessageID)
			}
		}
		if w.Message != "" && w.Message != d.Message {
			t.Errorf("error %d: message = %q, want %q", i, d.Message, w.Message)
		}
		if w.Line != 0 && w.Line != res.line(d) {
			t.Errorf("error %d: line = %d, want %d", i, res.line(d), w.Line)
		}
		if w.Column != 0 && w.Column != d.Start.Column {
			t.Errorf("error %d: column = %d, want %d", i, d.Start.Column, w.Column)
		}
	}
}

func checkOutput(t *testing.T, res result, c Case) {
	t.Helper()
	fixed, applied := engine.ApplyFixes(res.src, res.diags, goxlint.FixDangerous)
	if len(applied) == 0 {
		if c.Output != "" {
			t.Errorf("expected a fix producing:\n%s\nbut the rule reported no fix", c.Output)
		}
		return
	}
	got := strings.TrimPrefix(string(fixed), res.prefix)
	if c.Output == "" {
		t.Errorf("rule reported a fix; set Output to:\n%s", got)
		return
	}
	if got != c.Output {
		t.Errorf("fixed output mismatch\n--- got ---\n%s\n--- want ---\n%s", got, c.Output)
	}
}

// matchesTemplate reports whether msg could be produced from tmpl by
// substituting its {{placeholders}}.
func matchesTemplate(tmpl, msg string) bool {
	var parts []string
	for {
		i := strings.Index(tmpl, "{{")
		if i < 0 {
			parts = append(parts, tmpl)
			break
		}
		j := strings.Index(tmpl[i:], "}}")
		if j < 0 {
			parts = append(parts, tmpl)
			break
		}
		parts = append(parts, tmpl[:i])
		tmpl = tmpl[i+j+2:]
	}
	if !strings.HasPrefix(msg, parts[0]) {
		return false
	}
	msg = msg[len(parts[0]):]
	for _, p := range parts[1:] {
		k := strings.Index(msg, p)
		if p == parts[len(parts)-1] {
			return strings.HasSuffix(msg, p)
		}
		if k < 0 {
			return false
		}
		msg = msg[k+len(p):]
	}
	return msg == ""
}
