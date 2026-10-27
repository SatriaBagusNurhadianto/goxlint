package engine

import (
	"go/parser"
	"go/token"
	"slices"
	"strings"
	"testing"

	"github.com/egoisutolabs/goose-lint/goose"
)

func suppressionFile(t *testing.T, src string) File {
	t.Helper()
	fset := token.NewFileSet()
	af, err := parser.ParseFile(fset, "x.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return File{Fset: fset, AST: af, Filename: "x.go", Source: []byte(src)}
}

func TestUnusedSuppressions(t *testing.T) {
	for _, tc := range []struct {
		name, src string
		unused    []string
		other     int
	}{
		{name: "own line", src: "package p\nvar bad = 1 //goose:ignore t/ident reason\n"},
		{name: "next line", src: "package p\n//goose:ignore t/ident reason\nvar bad = 1\n"},
		{name: "plugin", src: "package p\n//goose:ignore t\nvar bad = 1\n"},
		{name: "wildcard", src: "package p\n//goose:ignore *\nvar bad = 1\n"},
		{name: "file", src: "//goose:ignore-file t/ident\npackage p\nvar bad = 1\nvar _ = bad\n"},
		{name: "no diagnostics", src: "package p\n//goose:ignore t/ident reason\nvar good = 1\n", unused: []string{"t/ident"}},
		{name: "unused file", src: "//goose:ignore-file t\npackage p\n", unused: []string{"t"}},
		{name: "unused wildcard", src: "//goose:ignore-file *\npackage p\n", unused: []string{"*"}},
		{name: "unknown rule", src: "package p\n//goose:ignore t/typo\nvar bad = 1\n", unused: []string{"t/typo"}, other: 1},
		{name: "unknown plugin", src: "package p\n//goose:ignore typo\nvar bad = 1\n", unused: []string{"typo"}, other: 1},
		{name: "partial list", src: "package p\n//goose:ignore t/ident,t/typo explanation\nvar bad = 1\n", unused: []string{"t/typo"}},
		{name: "file partial list", src: "//goose:ignore-file t/typo,t/ident\npackage p\nvar bad = 1\n", unused: []string{"t/typo"}},
		{name: "duplicate target", src: "package p\n//goose:ignore t/ident,t/ident\nvar bad = 1\n", unused: []string{"t/ident"}},
		{name: "duplicate directive", src: "//goose:ignore-file t\npackage p\n//goose:ignore t/ident\nvar bad = 1\n", unused: []string{"t/ident"}},
		{name: "redundant wildcard", src: "package p\n//goose:ignore t/ident,*\nvar bad = 1\n", unused: []string{"*"}},
		{name: "out of scope", src: "package p\n//goose:ignore t/ident\n\nvar bad = 1\n", unused: []string{"t/ident"}, other: 1},
		{name: "does not suppress audit", src: "//goose:ignore-file *\npackage p\n//goose:ignore t/typo\nvar bad = 1\n", unused: []string{"t/typo"}},
		{name: "cannot suppress itself", src: "//goose:ignore-file goose/unused-suppression\npackage p\n", unused: []string{"goose/unused-suppression"}},
		{name: "multiple findings one target", src: "package p\n//goose:ignore t/ident\nvar bad = bad\n"},
		{name: "extra whitespace and CRLF", src: "package p\r\n//goose:ignore   t/typo reason\r\n", unused: []string{"t/typo"}},
		{name: "not a directive", src: "package p\n// goose:ignore t/ident\n/* //goose:ignore t/ident */\nvar bad = 1\n", other: 1},
		{name: "line directives", src: "package p\n//goose:ignore t/ident\n//line other.go:100\nvar bad = 1\n", unused: []string{"t/ident"}, other: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := suppressionFile(t, tc.src)
			rules := []ConfiguredRule{{ID: "t/ident", Plugin: "t", Rule: identRule, Severity: Error}}
			diags := LintFile(f, rules, LintOptions{ReportUnusedSuppressions: Warn})
			var unused []string
			other := 0
			for _, d := range diags {
				if d.RuleID != UnusedSuppressionRuleID {
					other++
					continue
				}
				target := string(f.Source[d.Start.Offset:d.End.Offset])
				unused = append(unused, target)
				if d.Severity != Warn || !strings.Contains(d.Message, target) || d.Help == "" || d.Fix != nil {
					t.Errorf("unexpected audit diagnostic: %+v", d)
				}
				if d.Filename != "x.go" || d.Start.Line < 1 || d.Start.Column < 1 {
					t.Errorf("invalid audit location: %+v", d)
				}
			}
			if !slices.Equal(unused, tc.unused) || other != tc.other {
				t.Fatalf("want unused %v and %d other diagnostics; got unused %v and %d: %+v", tc.unused, tc.other, unused, other, diags)
			}
		})
	}
}

func TestUnusedSuppressionSeverity(t *testing.T) {
	f := suppressionFile(t, "//goose:ignore-file t/ident\npackage p\n")
	for _, severity := range []Severity{Off, Warn, Error} {
		diags := LintFile(f, nil, LintOptions{ReportUnusedSuppressions: severity})
		if severity == Off {
			if len(diags) != 0 {
				t.Fatalf("disabled audit returned diagnostics: %+v", diags)
			}
		} else if len(diags) != 1 || diags[0].Severity != severity {
			t.Fatalf("want one %s diagnostic, got %+v", severity, diags)
		}
	}
}

func TestUnusedSuppressionsSkipIncompleteRules(t *testing.T) {
	typed := *identRule
	typed.Meta.NeedsTypes = true
	createPanic := &goose.Rule{Meta: goose.Meta{Name: "create-panic"}, Create: func(*goose.Context) goose.Visitor { panic("boom") }}
	for _, rule := range []*goose.Rule{panicRule, createPanic, &typed} {
		t.Run(rule.Meta.Name, func(t *testing.T) {
			f := suppressionFile(t, "package p\n//goose:ignore t/ident\nfunc good() {}\n")
			diags := LintFile(f, []ConfiguredRule{{ID: "t/" + rule.Meta.Name, Rule: rule, Severity: Error}}, LintOptions{ReportUnusedSuppressions: Warn})
			for _, d := range diags {
				if d.RuleID == UnusedSuppressionRuleID {
					t.Fatalf("audited incomplete rule execution: %+v", diags)
				}
			}
		})
	}
}

func TestUnusedSuppressionCreateOnlyRule(t *testing.T) {
	f := suppressionFile(t, "//goose:ignore-file t/create\npackage p\n")
	rule := &goose.Rule{Create: func(ctx *goose.Context) goose.Visitor {
		ctx.Report(goose.Diagnostic{Node: ctx.File, Message: "file finding"})
		return nil
	}}
	diags := LintFile(f, []ConfiguredRule{{ID: "t/create", Rule: rule, Severity: Warn}}, LintOptions{ReportUnusedSuppressions: Warn})
	if len(diags) != 0 {
		t.Fatalf("create-time finding should use suppression: %+v", diags)
	}
}

func TestUnusedSuppressionsAcrossRules(t *testing.T) {
	f := suppressionFile(t, "package p\n//goose:ignore t/ident,t/stmt\nfunc bad() { return }\n")
	rules := []ConfiguredRule{
		{ID: "t/ident", Rule: identRule, Severity: Error},
		{ID: "t/stmt", Rule: stmtRule, Severity: Warn},
	}
	if diags := LintFile(f, rules, LintOptions{ReportUnusedSuppressions: Warn}); len(diags) != 0 {
		t.Fatalf("both targets should be used: %+v", diags)
	}
}
