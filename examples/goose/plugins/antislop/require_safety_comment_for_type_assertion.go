package antislop

import (
	"go/ast"
	"strings"

	"github.com/egoisutolabs/goose-lint/goose"
)

// RequireSafetyCommentForTypeAssertion requires a justification comment before
// every unchecked type assertion x.(T), which panics when x is not a T. The
// two-value form v, ok := x.(T) and type switches need no comment.
//
// The comment must be on the line above the statement or the assertion, or at
// the end of the assertion's line, and read "<MARKER>: <reason>". Markers are
// configurable and default to SAFETY.
//
// Test files are skipped unless checkTests is set: a failed assertion there
// panics the test, which reports the failure anyway.
var RequireSafetyCommentForTypeAssertion = &goose.Rule{
	Meta: goose.Meta{
		Name:        "require-safety-comment-for-type-assertion",
		Description: "Require a SAFETY comment justifying each unchecked type assertion.",
		Category:    goose.Restriction,
		Messages: map[string]string{
			"missingComment": "Unchecked type assertion `{{expr}}` panics if the value is not `{{type}}`.",
		},
	},
	Create: func(ctx *goose.Context) goose.Visitor {
		opts := struct {
			Markers    []string `json:"markers"`
			CheckTests bool     `json:"checkTests"`
		}{Markers: []string{"SAFETY"}}
		if !ctx.Options(&opts) || (ctx.IsTestFile() && !opts.CheckTests) {
			return nil
		}

		justified := func(cg *ast.CommentGroup) bool {
			if cg == nil {
				return false
			}
			for _, c := range cg.List {
				text := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(c.Text, "//"), "/*"), "*/"))
				for _, m := range opts.Markers {
					if reason, ok := strings.CutPrefix(text, m+":"); ok && strings.TrimSpace(reason) != "" {
						return true
					}
				}
			}
			return false
		}

		return goose.Visit(
			goose.On(func(ta *ast.TypeAssertExpr) {
				if ta.Type == nil || isChecked(ta, ctx.Parent()) {
					return
				}
				if justified(ctx.CommentsBefore(ta.Pos())) || justified(ctx.TrailingComment(ta.End())) {
					return
				}
				if stmt := enclosingStatement(ctx.Ancestors()); stmt != nil && justified(ctx.CommentsBefore(stmt.Pos())) {
					return
				}
				ctx.Report(goose.Diagnostic{
					Node:      ta,
					MessageID: "missingComment",
					Data:      goose.Data{"expr": ctx.Snippet(ta), "type": ctx.Snippet(ta.Type)},
					Help:      "Use the checked form `v, ok := x.(T)`, or add `// " + opts.Markers[0] + ": <why this cannot fail>` above it.",
				})
			}),
		)
	},
}

// enclosingStatement returns the nearest statement, spec or declaration, or nil.
func enclosingStatement(ancestors []ast.Node) ast.Node {
	for _, a := range ancestors {
		switch a.(type) {
		case *ast.BlockStmt:
		case ast.Stmt, ast.Spec, ast.Decl:
			return a
		}
	}
	return nil
}

// isChecked reports whether ta is the single value in a two-value assignment.
func isChecked(ta *ast.TypeAssertExpr, parent ast.Node) bool {
	switch p := parent.(type) {
	case *ast.AssignStmt:
		return len(p.Lhs) == 2 && len(p.Rhs) == 1 && p.Rhs[0] == ta
	case *ast.ValueSpec:
		return len(p.Names) == 2 && len(p.Values) == 1 && p.Values[0] == ta
	}
	return false
}
