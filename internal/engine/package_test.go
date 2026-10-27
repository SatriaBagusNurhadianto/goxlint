package engine

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strings"
	"testing"

	"github.com/egoisutolabs/goose-lint/goose"
)

func packageFile(t *testing.T, name, src string) File {
	t.Helper()
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, name, src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return File{Fset: fs, AST: file, Filename: name, Source: []byte(src)}
}

func TestPackageAnalysisBeforeSuppressions(t *testing.T) {
	files := []File{
		packageFile(t, "p/b.go", "//goose:ignore-file t/package\npackage p\n"),
		packageFile(t, "p/a.go", "//goose:ignore-file t/package\npackage p\n"),
	}
	calls := 0
	rule := &goose.Rule{
		Create: func(ctx *goose.Context) goose.Visitor {
			return goose.Visit(goose.On(func(n *ast.File) { ctx.Report(goose.Diagnostic{Node: n.Name, Message: "file phase"}) }))
		},
		AnalyzePackage: func(ctx *goose.PackageContext) {
			calls++
			if ctx.Name != "p" || len(ctx.Files) != 2 || ctx.Files[0].Filename != "p/a.go" {
				t.Errorf("unexpected package: %+v", ctx)
			}
			for _, f := range ctx.Files {
				if f.Pkg != nil || f.TypesInfo != nil {
					t.Error("syntax-only analyzer received types")
				}
				f.Report(goose.Diagnostic{Node: f.File.Name, Message: "package phase"})
			}
		},
	}
	out := LintAll(files, func(string) []ConfiguredRule { return []ConfiguredRule{{ID: "t/package", Rule: rule, Severity: Warn}} }, LintOptions{ReportUnusedSuppressions: Warn})
	if calls != 1 || len(out[0]) != 0 || len(out[1]) != 0 {
		t.Fatalf("want both phases suppressed, got calls=%d, %+v", calls, out)
	}
}

func TestPackageAnalysisConfiguration(t *testing.T) {
	files := []File{
		packageFile(t, "p/a.go", "package p\n"),
		packageFile(t, "p/b.go", "package p\n"),
		packageFile(t, "p/c.go", "package p\n"),
	}
	rule := &goose.Rule{AnalyzePackage: func(pkg *goose.PackageContext) {
		if len(pkg.Files) != 3 || pkg.Files[2].Enabled {
			t.Error("disabled evidence file missing or enabled")
		}
		for _, f := range pkg.Files {
			opts := struct {
				Message string `json:"message"`
			}{}
			if f.Options(&opts) {
				f.Report(goose.Diagnostic{Node: f.File.Name, Message: opts.Message + "!"})
			}
		}
	}}
	out := LintAll(files, func(name string) []ConfiguredRule {
		if name == "p/c.go" {
			return nil
		}
		severity, options := Warn, json.RawMessage(`{"message":"first"}`)
		if name == "p/b.go" {
			severity, options = Error, json.RawMessage(`{"message":"second"}`)
		}
		return []ConfiguredRule{{ID: "t/package", Rule: rule, Severity: severity, Options: options}}
	}, LintOptions{})
	if len(out[0]) != 1 || out[0][0].Message != "first!" || out[0][0].Severity != Warn || out[0][0].Filename != "p/a.go" {
		t.Fatalf("first configuration: %+v", out)
	}
	if len(out[1]) != 1 || out[1][0].Message != "second!" || out[1][0].Severity != Error || out[1][0].Filename != "p/b.go" {
		t.Fatalf("second configuration: %+v", out)
	}
	if len(out[2]) != 0 {
		t.Fatalf("disabled file reported diagnostics: %+v", out[2])
	}
}

func TestPackageBoundaries(t *testing.T) {
	files := []File{
		packageFile(t, "one/a.go", "package p\n"),
		packageFile(t, "one/b.go", "package p\n"),
		packageFile(t, "two/a.go", "package p\n"),
		packageFile(t, "one/a_test.go", "package p_test\n"),
	}
	rule := &goose.Rule{AnalyzePackage: func(pkg *goose.PackageContext) {
		for _, f := range pkg.Files {
			var names []string
			for _, other := range pkg.Files {
				names = append(names, other.Filename)
			}
			f.Report(goose.Diagnostic{Node: f.File.Name, Message: strings.Join(names, ",")})
		}
	}}
	out := LintAll(files, func(string) []ConfiguredRule { return []ConfiguredRule{{ID: "t/package", Rule: rule, Severity: Warn}} }, LintOptions{})
	for i, want := range []string{"one/a.go,one/b.go", "one/a.go,one/b.go", "two/a.go", "one/a_test.go"} {
		if len(out[i]) != 1 || out[i][0].Message != want {
			t.Fatalf("file %d: want %s, got %+v", i, want, out[i])
		}
	}
}

func TestPackageFailures(t *testing.T) {
	for _, kind := range []string{"panic", "missing types", "incomplete load"} {
		t.Run(kind, func(t *testing.T) {
			files := []File{packageFile(t, "p/a.go", "package p\n//goose:ignore t/package\nvar x = 1\n"), packageFile(t, "p/b.go", "package p\n//goose:ignore t/package\nvar y = 1\n")}
			// One incomplete file makes the whole package group unreliable evidence.
			files[0].Incomplete = kind == "incomplete load"
			rule := &goose.Rule{Meta: goose.Meta{NeedsTypes: kind == "missing types"}, AnalyzePackage: func(pkg *goose.PackageContext) {
				if kind != "panic" {
					t.Error("analyzer ran despite incomplete evidence")
				}
				pkg.Files[0].Report(goose.Diagnostic{Node: pkg.Files[0].File.Name, Message: "partial report"})
				panic("boom")
			}}
			out := LintAll(files, func(string) []ConfiguredRule { return []ConfiguredRule{{ID: "t/package", Rule: rule, Severity: Warn}} }, LintOptions{ReportUnusedSuppressions: Warn})
			if kind == "panic" {
				if len(out[0]) != 1 || !out[0][0].internal || !strings.Contains(out[0][0].Message, "boom") {
					t.Fatalf("want one internal error, got %+v", out)
				}
			} else if len(out[0]) != 0 {
				t.Fatalf("unexpected diagnostics: %+v", out)
			}
			if len(out[1]) != 0 {
				t.Fatalf("audited incomplete package: %+v", out)
			}
		})
	}
}

func TestPackageFixRoutingAndOrder(t *testing.T) {
	files := []File{packageFile(t, "p/b.go", "package p\nvar b = 1\n"), packageFile(t, "p/a.go", "package p\nvar a = 1\n")}
	rule := &goose.Rule{Meta: goose.Meta{Fixable: goose.FixSafe}, AnalyzePackage: func(pkg *goose.PackageContext) {
		for _, f := range pkg.Files {
			f.Report(goose.Diagnostic{Node: f.File.Name, Message: "rename", Fix: &goose.Fix{Kind: goose.FixSafe, Edits: []goose.Edit{goose.ReplaceNode(f.File.Name, "renamed")}}})
		}
	}}
	out := LintAll(files, func(string) []ConfiguredRule { return []ConfiguredRule{{ID: "t/package", Rule: rule, Severity: Warn}} }, LintOptions{})
	for i, f := range files {
		fixed, applied := ApplyFixes(f.Source, out[i], goose.FixSafe)
		want := strings.Replace(string(f.Source), "package p", "package renamed", 1)
		if len(applied) != 1 || string(fixed) != want {
			t.Fatalf("file %s: %q, %+v", f.Filename, fixed, applied)
		}
	}
	if !slices.Equal([]string{out[0][0].Filename, out[1][0].Filename}, []string{"p/b.go", "p/a.go"}) {
		t.Fatal("results lost input order")
	}
}

func TestPackagePanicDoesNotStopOtherAnalyzers(t *testing.T) {
	f := packageFile(t, "p.go", "package p\n")
	broken := &goose.Rule{AnalyzePackage: func(*goose.PackageContext) { panic("boom") }}
	healthy := &goose.Rule{AnalyzePackage: func(pkg *goose.PackageContext) {
		pkg.Files[0].Report(goose.Diagnostic{Node: pkg.Files[0].File.Name, Message: "healthy"})
	}}
	out := LintFile(f, []ConfiguredRule{{ID: "t/a-broken", Rule: broken, Severity: Warn}, {ID: "t/b-healthy", Rule: healthy, Severity: Warn}}, LintOptions{})
	if len(out) != 2 || !out[0].internal || out[1].Message != "healthy" {
		t.Fatalf("one analyzer failure stopped another: %+v", out)
	}
}

func TestPackageForeignDiagnostic(t *testing.T) {
	fs := token.NewFileSet()
	var files []File
	for _, name := range []string{"a.go", "b.go"} {
		const src = "package p\n"
		file, err := parser.ParseFile(fs, name, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, File{Fset: fs, AST: file, Filename: name, Source: []byte(src)})
	}
	rule := &goose.Rule{AnalyzePackage: func(pkg *goose.PackageContext) {
		pkg.Files[0].Report(goose.Diagnostic{Node: pkg.Files[1].File.Name, Message: "wrong context"})
	}}
	out := LintAll(files, func(string) []ConfiguredRule { return []ConfiguredRule{{ID: "t/package", Rule: rule, Severity: Warn}} }, LintOptions{})
	if len(out[0]) != 1 || !out[0][0].internal || !strings.Contains(out[0][0].Message, "current file") || len(out[1]) != 0 {
		t.Fatalf("foreign node was not rejected: %+v", out)
	}
}
