package engine

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/egoisutolabs/goxlint"
)

func parseRelatedFiles(t *testing.T, sources map[string]string) []File {
	t.Helper()
	fset := token.NewFileSet()
	var files []File
	for _, name := range slices.Sorted(maps.Keys(sources)) {
		astFile, err := parser.ParseFile(fset, name, sources[name], parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, File{Fset: fset, AST: astFile, Filename: name, Source: []byte(sources[name])})
	}
	return files
}

func TestRelatedLoadedPackageFiles(t *testing.T) {
	files := parseRelatedFiles(t, map[string]string{
		"a.go": "package p\n//line virtual.go:900:20\nvar original = 1\n",
		"b.go": "package p\nvar copied = 1\n",
	})
	node := files[0].AST.Decls[0].(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Names[0]
	tf := files[0].Fset.File(node.Pos())
	report := func(ctx *goxlint.Context) {
		ctx.Report(goxlint.Diagnostic{Node: ctx.File.Name, Message: "primary", Related: []goxlint.Related{
			{Node: node, Pos: token.NoPos, End: token.Pos(1 << 30), Message: "original"},
			{Pos: tf.Pos(tf.Size()), Message: "end of file"},
			{Node: ctx.File.Name, Message: "same file"},
		}})
	}
	rule := &goxlint.Rule{
		Create: func(ctx *goxlint.Context) goxlint.Visitor { report(ctx); return nil },
		AnalyzePackage: func(pkg *goxlint.PackageContext) {
			if pkg.Files[0].Enabled {
				t.Error("reference file should be disabled")
			}
			report(pkg.Files[1])
		},
	}
	out := LintAll(files, func(name string) []ConfiguredRule {
		if name == "a.go" {
			return nil
		}
		return []ConfiguredRule{{ID: "t/r", Rule: rule, Severity: Warn}}
	}, LintOptions{})
	if len(out[0]) != 0 || len(out[1]) != 2 {
		t.Fatalf("wrong diagnostics: %+v", out)
	}
	for _, d := range out[1] {
		if d.internal || d.Filename != "b.go" || len(d.Related) != 3 {
			t.Fatalf("invalid related diagnostic: %+v", d)
		}
		r := d.Related[0]
		if r.Filename != "a.go" || r.Start.Filename != "a.go" || r.End.Filename != "a.go" || r.Start.Line != 3 || r.Start.Column != 5 || r.End.Column != 13 || r.Message != "original" {
			t.Fatalf("wrong physical related location: %+v", r)
		}
		if d.Related[1].Start.Offset != tf.Size() || d.Related[1].End != d.Related[1].Start || d.Related[2].Filename != "b.go" {
			t.Fatalf("point or same-file range lost: %+v", d.Related)
		}
	}
}

func TestInvalidRelatedLocations(t *testing.T) {
	files := parseRelatedFiles(t, map[string]string{
		"a.go":             "package p\nvar reference = 1\n",
		"b.go":             "//goxlint:ignore-file t/unused\npackage p\n",
		"external_test.go": "package p_test\n",
		"other/c.go":       "package p\n",
	})
	fset := files[0].Fset
	stale, err := parser.ParseFile(fset, "a.go", "package p\nvar old = 1\n", 0)
	if err != nil {
		t.Fatal(err)
	}
	excluded, err := parser.ParseFile(fset, "excluded.go", "package p\n", 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		related goxlint.Related
	}{
		{"missing", goxlint.Related{}},
		{"negative start", goxlint.Related{Pos: -1}},
		{"negative end", goxlint.Related{Pos: files[0].AST.Name.Pos(), End: -1}},
		{"reversed", goxlint.Related{Pos: files[0].AST.Name.End(), End: files[0].AST.Name.Pos()}},
		{"cross-file end", goxlint.Related{Pos: files[0].AST.Name.Pos(), End: files[1].AST.Name.End()}},
		{"outside file set", goxlint.Related{Pos: 1 << 30}},
		{"excluded file", goxlint.Related{Node: excluded.Name}},
		{"stale parse with same filename", goxlint.Related{Node: stale.Name}},
		{"external test package", goxlint.Related{Node: files[2].AST.Name}},
		{"other directory", goxlint.Related{Node: files[3].AST.Name}},
	} {
		for _, packagePhase := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/package=%t", tc.name, packagePhase), func(t *testing.T) {
				report := func(ctx *goxlint.Context) {
					ctx.Report(goxlint.Diagnostic{
						Node: ctx.File.Name, Message: "bad related", Related: []goxlint.Related{{Node: ctx.File.Name}, tc.related},
						Fix: &goxlint.Fix{Kind: goxlint.FixSafe, Edits: []goxlint.Edit{goxlint.ReplaceNode(ctx.File.Name, "q")}},
					})
				}
				rule := &goxlint.Rule{Meta: goxlint.Meta{Fixable: goxlint.FixSafe}}
				if packagePhase {
					rule.AnalyzePackage = func(pkg *goxlint.PackageContext) { report(pkg.Files[1]) }
				} else {
					rule.Create = func(ctx *goxlint.Context) goxlint.Visitor { report(ctx); return nil }
				}
				healthy := &goxlint.Rule{Create: func(ctx *goxlint.Context) goxlint.Visitor {
					ctx.Report(goxlint.Diagnostic{Node: ctx.File.Name, Message: "healthy"})
					return nil
				}}
				out := LintAll(files, func(name string) []ConfiguredRule {
					if name != "b.go" {
						return nil
					}
					return []ConfiguredRule{{ID: "t/bad", Rule: rule, Severity: Warn}, {ID: "t/healthy", Rule: healthy, Severity: Warn}}
				}, LintOptions{ReportUnusedSuppressions: Warn})
				if len(out[1]) != 2 {
					t.Fatalf("bad report hid healthy rule or triggered audit: %+v", out)
				}
				bad := 0
				for _, d := range out[1] {
					if d.RuleID != "t/bad" {
						continue
					}
					bad++
					if !d.internal || d.Severity != Error || !strings.Contains(d.Message, "related positions") || d.Fix != nil || len(d.Related) != 0 {
						t.Fatalf("invalid report escaped validation: %+v", d)
					}
				}
				if bad != 1 {
					t.Fatalf("missing internal error: %+v", out)
				}
			})
		}
	}
}

func TestRelatedSuppressionsUsePrimaryOnly(t *testing.T) {
	for _, primarySuppressed := range []bool{false, true} {
		for _, referenceSuppressed := range []bool{false, true} {
			t.Run(fmt.Sprintf("primary=%t/reference=%t", primarySuppressed, referenceSuppressed), func(t *testing.T) {
				primary, reference := "package p\n", "package p\n"
				if primarySuppressed {
					primary = "//goxlint:ignore-file t/r\n" + primary
				}
				if referenceSuppressed {
					reference = "//goxlint:ignore-file t/r\n" + reference
				}
				files := parseRelatedFiles(t, map[string]string{"a.go": reference, "b.go": primary})
				rule := &goxlint.Rule{AnalyzePackage: func(pkg *goxlint.PackageContext) {
					pkg.Files[1].Report(goxlint.Diagnostic{Node: pkg.Files[1].File.Name, Message: "finding", Related: []goxlint.Related{{Node: pkg.Files[0].File.Name}}})
				}}
				out := LintAll(files, func(string) []ConfiguredRule { return []ConfiguredRule{{ID: "t/r", Rule: rule, Severity: Warn}} }, LintOptions{ReportUnusedSuppressions: Warn})
				wantPrimary, wantReference := 1, 0
				if primarySuppressed {
					wantPrimary = 0
				}
				if referenceSuppressed {
					wantReference = 1
				}
				if len(out[1]) != wantPrimary || len(out[0]) != wantReference {
					t.Fatalf("related location changed suppression matching: %+v", out)
				}
				if referenceSuppressed && out[0][0].RuleID != UnusedSuppressionRuleID {
					t.Fatalf("reference suppression received credit: %+v", out)
				}
			})
		}
	}
}

func TestRelatedTypedTestVariant(t *testing.T) {
	root := t.TempDir()
	for name, source := range map[string]string{
		"go.mod":    "module test.local/related\n\ngo 1.26.0\n",
		"a.go":      "package p\nvar shared = 1\n",
		"b_test.go": "package p\nvar _ = shared\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	files, errs := Load(LoadOptions{Dir: root, Patterns: []string{"."}, Typed: true})
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	rule := &goxlint.Rule{Meta: goxlint.Meta{NeedsTypes: true}, Create: func(ctx *goxlint.Context) goxlint.Visitor {
		if !ctx.IsTestFile() {
			return nil
		}
		return goxlint.Visit(goxlint.On(func(id *ast.Ident) {
			if id.Name == "shared" {
				ctx.Report(goxlint.Diagnostic{Node: id, Message: "use", Related: []goxlint.Related{{Pos: ctx.ObjectOf(id).Pos(), Message: "declaration"}}})
			}
		}))
	}}
	count := 0
	for _, ds := range LintAll(files, func(string) []ConfiguredRule { return []ConfiguredRule{{ID: "t/r", Rule: rule, Severity: Warn}} }, LintOptions{}) {
		for _, d := range ds {
			count++
			if d.internal || len(d.Related) != 1 || filepath.Base(d.Related[0].Filename) != "a.go" || d.Related[0].Start.Line != 2 {
				t.Fatalf("test-variant declaration not resolved: %+v", d)
			}
		}
	}
	if count != 1 {
		t.Fatalf("got %d findings", count)
	}
}
