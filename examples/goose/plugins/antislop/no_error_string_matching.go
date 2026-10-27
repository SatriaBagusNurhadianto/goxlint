package antislop

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/egoisutolabs/goose-lint/goose"
)

// SAFETY: the universe type error is always an interface.
var errorInterface = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

// NoErrorStringMatching rejects branching on the text of err.Error(): passing
// it to strings.Contains and friends, comparing it with == or !=, or switching
// on it.
//
// Test files are skipped unless checkTests is set, because asserting on an
// error's message is a legitimate thing for a test to do.
var NoErrorStringMatching = &goose.Rule{
	Meta: goose.Meta{
		Name:        "no-error-string-matching",
		Description: "Disallow matching on err.Error() text; use errors.Is or errors.As.",
		Category:    goose.Suspicious,
		NeedsTypes:  true,
		Messages: map[string]string{
			"errorText": "Matching on the text of `{{expr}}` breaks when the message changes or gains context.",
		},
	},
	Create: func(ctx *goose.Context) goose.Visitor {
		opts := struct {
			CheckTests bool `json:"checkTests"`
		}{}
		if !ctx.Options(&opts) {
			return nil
		}
		if ctx.IsTestFile() && !opts.CheckTests {
			return nil
		}

		report := func(e ast.Expr) {
			ctx.Report(goose.Diagnostic{
				Node:      e,
				MessageID: "errorText",
				Data:      goose.Data{"expr": ctx.Snippet(e)},
				Help:      "Compare with errors.Is against a sentinel error, or extract a typed error with errors.As.",
			})
		}
		isErrorText := func(e ast.Expr) bool {
			call, ok := ast.Unparen(e).(*ast.CallExpr)
			if !ok || len(call.Args) != 0 {
				return false
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Error" {
				return false
			}
			t := ctx.TypeOf(sel.X)
			return t != nil && types.Implements(t, errorInterface)
		}

		return goose.Visit(
			goose.On(func(call *ast.CallExpr) {
				if !ctx.IsPkgFunc(call, "strings", "Contains", "HasPrefix", "HasSuffix", "EqualFold", "Index", "Compare") {
					return
				}
				for _, arg := range call.Args {
					if isErrorText(arg) {
						report(arg)
					}
				}
			}),
			goose.On(func(bin *ast.BinaryExpr) {
				if bin.Op != token.EQL && bin.Op != token.NEQ {
					return
				}
				for _, side := range []ast.Expr{bin.X, bin.Y} {
					if isErrorText(side) {
						report(side)
					}
				}
			}),
			goose.On(func(sw *ast.SwitchStmt) {
				if sw.Tag != nil && isErrorText(sw.Tag) {
					report(sw.Tag)
				}
			}),
		)
	},
}
