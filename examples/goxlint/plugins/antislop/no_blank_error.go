package antislop

import (
	"go/ast"
	"go/types"

	"github.com/egoisutolabs/goxlint"
)

// NoBlankError rejects discarding an error with the blank identifier
// (`_ = f()`, `v, _ := f()`) unless a comment on the line above or at the end
// of the line says why the failure cannot matter.
//
// Test files are skipped unless checkTests is set: tests routinely discard
// errors from setup calls whose failure the assertions would catch anyway.
var NoBlankError = &goxlint.Rule{
	Meta: goxlint.Meta{
		Name:        "no-blank-error",
		Description: "Disallow discarding errors with `_` without a comment explaining why.",
		Category:    goxlint.Suspicious,
		NeedsTypes:  true,
		Messages: map[string]string{
			"blankError": "Error from `{{expr}}` is discarded with `_`.",
		},
	},
	Create: func(ctx *goxlint.Context) goxlint.Visitor {
		opts := struct {
			CheckTests bool `json:"checkTests"`
		}{}
		if !ctx.Options(&opts) {
			return nil
		}
		if ctx.IsTestFile() && !opts.CheckTests {
			return nil
		}

		check := func(stmt ast.Node, lhs []ast.Expr, rhs []ast.Expr) {
			if hasComment(ctx, stmt) {
				return
			}
			report := func(value ast.Expr) {
				ctx.Report(goxlint.Diagnostic{
					Node:      value,
					MessageID: "blankError",
					Data:      goxlint.Data{"expr": ctx.Snippet(value)},
					Help:      "Handle or return the error, or add a comment explaining why the failure cannot matter.",
				})
			}
			if len(lhs) == len(rhs) {
				for i, l := range lhs {
					if isBlank(l) && isErrorType(ctx.TypeOf(rhs[i])) {
						report(rhs[i])
					}
				}
				return
			}
			if len(rhs) != 1 {
				return
			}
			tuple, ok := ctx.TypeOf(rhs[0]).(*types.Tuple)
			if !ok || tuple.Len() != len(lhs) {
				return
			}
			for i, l := range lhs {
				if isBlank(l) && isErrorType(tuple.At(i).Type()) {
					report(rhs[0])
					return
				}
			}
		}

		return goxlint.Visit(
			goxlint.On(func(as *ast.AssignStmt) { check(as, as.Lhs, as.Rhs) }),
			goxlint.On(func(vs *ast.ValueSpec) {
				lhs := make([]ast.Expr, len(vs.Names))
				for i, n := range vs.Names {
					lhs[i] = n
				}
				check(vs, lhs, vs.Values)
			}),
		)
	},
}

func isBlank(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "_"
}
