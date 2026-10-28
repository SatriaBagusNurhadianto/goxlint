package cli

import (
	"bytes"
	"fmt"
	"go/ast"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoisutolabs/goxlint"
)

func TestRelatedSourceSnapshots(t *testing.T) {
	for _, typed := range []bool{false, true} {
		for _, stdin := range []bool{false, true} {
			for _, fix := range []bool{false, true} {
				t.Run(fmt.Sprintf("typed=%t/stdin=%t/fix=%t", typed, stdin, fix), func(t *testing.T) {
					t.Chdir(t.TempDir())
					writeTestFile(t, "go.mod", "module test.local/related\n\ngo 1.26.0\n")
					writeTestFile(t, ".goxlintrc.json", `{"rules":{"test/related":"warn"},"overrides":[{"files":["a.go"],"rules":{"test/related":"off"}}]}`)
					writeTestFile(t, "a.go", "package p\nvar original = 1\n")
					source := "package p\n//line virtual.go:900\nvar bad = 1\n"
					disk := source
					if stdin {
						disk = "package p\nvar disk = 0\n"
					}
					writeTestFile(t, "b.go", disk)
					rule := &goxlint.Rule{Meta: goxlint.Meta{Name: "related", NeedsTypes: typed, Fixable: goxlint.FixSafe}, AnalyzePackage: func(pkg *goxlint.PackageContext) {
						if len(pkg.Files) != 2 {
							t.Errorf("missing package files: %d", len(pkg.Files))
							return
						}
						first, last := pkg.Files[0], pkg.Files[1]
						if first.Enabled || filepath.Base(last.Filename) != "b.go" {
							t.Error("wrong contexts")
						}
						reference := first.File.Decls[0].(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Names[0]
						primary := last.File.Decls[0].(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Names[0]
						d := goxlint.Diagnostic{Node: primary, Message: "finding", Related: []goxlint.Related{{Node: reference, Message: "original"}, {Node: primary, Message: "current source"}}}
						if primary.Name == "bad" {
							d.Fix = &goxlint.Fix{Kind: goxlint.FixSafe, Edits: []goxlint.Edit{goxlint.ReplaceNode(primary, "good")}}
						}
						last.Report(d)
					}}
					var args []string
					if stdin {
						args = append(args, "--stdin-filename=b.go")
					} else {
						args = append(args, ".")
					}
					if fix {
						args = append([]string{"--fix"}, args...)
					}
					var stdout, stderr bytes.Buffer
					code := run(args, strings.NewReader(source), &stdout, &stderr, goxlint.Plugin{Name: "test", Rules: []*goxlint.Rule{rule}})
					output := stdout.String()
					if stdin && fix {
						output = stderr.String()
					} else if !fix && stderr.Len() != 0 {
						t.Fatalf("stderr: %s", &stderr)
					}
					wantName := "bad"
					if fix {
						wantName = "good"
					}
					if code != 0 {
						t.Fatalf("exit %d: %s; %s", code, &stdout, &stderr)
					}
					for _, want := range []string{"[b.go:3:5]", "[a.go:2:5]", "var original = 1", "var " + wantName + " = 1", "related: original", "related: current source", "Found 1 warning and 0 errors"} {
						if !strings.Contains(output, want) {
							t.Errorf("missing %q: %s", want, output)
						}
					}
					if strings.Contains(output, "var disk") || strings.Count(output, "test(related)") != 1 {
						t.Fatalf("stale snapshot or duplicate diagnostics: %s", output)
					}
					if stdin && fix && stdout.String() != strings.ReplaceAll(source, "bad", "good") {
						t.Fatalf("wrong fixed buffer: %s", &stdout)
					}
				})
			}
		}
	}
}
