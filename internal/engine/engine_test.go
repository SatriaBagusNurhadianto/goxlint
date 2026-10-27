package engine

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/egoisutolabs/goose-lint/goose"
)

// identRule reports every identifier named "bad" and offers a fix renaming it.
var identRule = &goose.Rule{
	Meta: goose.Meta{Name: "ident", Fixable: goose.FixSafe, Messages: map[string]string{"bad": "bad ident"}},
	Create: func(ctx *goose.Context) goose.Visitor {
		return goose.Visit(goose.On(func(id *ast.Ident) {
			if id.Name == "bad" {
				ctx.Report(goose.Diagnostic{Node: id, MessageID: "bad", Fix: &goose.Fix{
					Kind: goose.FixSafe, Edits: []goose.Edit{goose.ReplaceNode(id, "good")},
				}})
			}
		}))
	},
}

// stmtRule counts statements through an interface-typed handler.
var stmtRule = &goose.Rule{
	Meta: goose.Meta{Name: "stmt", Messages: map[string]string{"stmt": "stmt"}},
	Create: func(ctx *goose.Context) goose.Visitor {
		return goose.Visit(goose.On(func(s ast.Stmt) {
			if _, ok := s.(*ast.ReturnStmt); ok {
				ctx.Report(goose.Diagnostic{Node: s, MessageID: "stmt"})
			}
		}))
	},
}

var panicRule = &goose.Rule{
	Meta: goose.Meta{Name: "panics"},
	Create: func(ctx *goose.Context) goose.Visitor {
		return goose.Visit(goose.On(func(*ast.FuncDecl) { panic("boom") }))
	},
}

func lintSource(t *testing.T, src string, rules ...*goose.Rule) ([]Diagnostic, []byte) {
	t.Helper()
	fset := token.NewFileSet()
	af, err := parser.ParseFile(fset, "x.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var crs []ConfiguredRule
	for _, r := range rules {
		crs = append(crs, ConfiguredRule{ID: "t/" + r.Meta.Name, Plugin: "t", Rule: r, Severity: Error})
	}
	return LintFile(File{Fset: fset, AST: af, Filename: "x.go", Source: []byte(src)}, crs, LintOptions{}), []byte(src)
}

func TestSuppression(t *testing.T) {
	src := `package p

var bad = 1 //goose:ignore t/ident trailing

//goose:ignore t reason
var _ = bad

var _ = bad
`
	diags, _ := lintSource(t, src, identRule)
	if len(diags) != 1 || diags[0].Start.Line != 8 {
		t.Fatalf("want one diagnostic on line 8, got %+v", diags)
	}

	diags, _ = lintSource(t, "//goose:ignore-file *\npackage p\n\nvar bad = 1\n", identRule)
	if len(diags) != 0 {
		t.Fatalf("file-wide ignore: got %+v", diags)
	}
}

func TestApplyFixes(t *testing.T) {
	diags, src := lintSource(t, "package p\n\nvar bad = 1\nvar _ = bad\n", identRule)
	out, applied := ApplyFixes(src, diags, goose.FixSafe)
	if len(applied) != 2 || string(out) != "package p\n\nvar good = 1\nvar _ = good\n" {
		t.Fatalf("got %q (%d applied)", out, len(applied))
	}
	if _, applied := ApplyFixes(src, diags, goose.FixNone); len(applied) != 0 {
		t.Fatal("FixNone applied fixes")
	}

	overlapping := []Diagnostic{
		{Fix: &Fix{Kind: goose.FixSafe, Edits: []Edit{{Start: 0, End: 4, NewText: "AAAA"}}}},
		{Fix: &Fix{Kind: goose.FixSafe, Edits: []Edit{{Start: 2, End: 6, NewText: "BBBB"}}}},
	}
	out, applied = ApplyFixes([]byte("0123456789"), overlapping, goose.FixSafe)
	if string(out) != "AAAA456789" || len(applied) != 1 {
		t.Fatalf("overlap: got %q (%d applied)", out, len(applied))
	}
}

func TestInterfaceHandlersAndPanics(t *testing.T) {
	diags, _ := lintSource(t, "package p\n\nfunc f() int { return 1 }\nfunc g() {}\n", stmtRule, panicRule, identRule)
	var stmts, internal int
	for _, d := range diags {
		switch {
		case d.RuleID == "t/stmt":
			stmts++
		case d.RuleID == "t/panics" && strings.Contains(d.Message, "boom"):
			internal++
		}
	}
	if stmts != 1 || internal != 1 {
		t.Fatalf("want 1 stmt and 1 internal error, got %+v", diags)
	}
}

func TestInvalidFixPositions(t *testing.T) {
	for _, kind := range []string{"missing", "before file", "after file", "reversed"} {
		t.Run(kind, func(t *testing.T) {
			rule := &goose.Rule{
				Meta: goose.Meta{Name: "invalid-fix", Fixable: goose.FixSafe},
				Create: func(ctx *goose.Context) goose.Visitor {
					return goose.Visit(goose.On(func(id *ast.Ident) {
						if id.Name != "bad" {
							return
						}
						tf := ctx.Fset.File(id.Pos())
						edit := goose.ReplaceNode(id, "good")
						switch kind {
						case "missing":
							edit.Pos, edit.End = token.NoPos, token.NoPos
						case "before file":
							edit.Pos = token.Pos(tf.Base() - 1)
						case "after file":
							edit.End = token.Pos(tf.Base() + tf.Size() + 1)
						case "reversed":
							edit.Pos, edit.End = edit.End, edit.Pos
						}
						ctx.Report(goose.Diagnostic{Node: id, Message: "bad fix", Fix: &goose.Fix{
							Kind: goose.FixSafe, Edits: []goose.Edit{edit},
						}})
					}))
				},
			}
			diags, src := lintSource(t, "package p\nvar bad = 1\n", rule, identRule)
			if len(diags) != 2 || !strings.Contains(diags[0].Message, "invalid fix edit") || diags[0].Fix != nil {
				t.Fatalf("want internal error without a fix and a healthy rule diagnostic, got %+v", diags)
			}
			out, applied := ApplyFixes(src, diags, goose.FixSafe)
			if string(out) != "package p\nvar good = 1\n" || len(applied) != 1 {
				t.Fatalf("want only healthy rule's fix applied, got %q, %+v", out, applied)
			}
		})
	}
}

func TestLineDirectivesKeepPhysicalPositions(t *testing.T) {
	diags, _ := lintSource(t, "package p\n\n//line template.tmpl:100\nvar bad = 1\n", identRule)
	if len(diags) != 1 {
		t.Fatalf("want 1 diagnostic, got %+v", diags)
	}
	if d := diags[0]; d.Start.Filename != "x.go" || d.Start.Line != 4 || d.Start.Column != 5 || d.End.Column != 8 {
		t.Fatalf("want x.go:4:5-8, got %s:%d:%d-%d", d.Start.Filename, d.Start.Line, d.Start.Column, d.End.Column)
	}
}
