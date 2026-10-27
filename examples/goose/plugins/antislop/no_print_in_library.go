package antislop

import (
	"go/ast"
	"go/types"

	"github.com/egoisutolabs/goose-lint/goose"
)

// NoPrintInLibrary rejects fmt.Print, fmt.Printf, fmt.Println and the print
// and println builtins outside package main and tests. Library code that
// writes to stdout is usually leftover debugging, and callers cannot redirect
// or silence it.
var NoPrintInLibrary = &goose.Rule{
	Meta: goose.Meta{
		Name:        "no-print-in-library",
		Description: "Disallow printing to stdout from library code.",
		Category:    goose.Restriction,
		NeedsTypes:  true,
		Messages: map[string]string{
			"print": "`{{call}}` writes to stdout from library code.",
		},
	},
	Create: func(ctx *goose.Context) goose.Visitor {
		if isMainPackage(ctx) || ctx.IsTestFile() {
			return nil
		}
		return goose.Visit(
			goose.On(func(call *ast.CallExpr) {
				name := ""
				switch fun := ast.Unparen(call.Fun).(type) {
				case *ast.Ident:
					if _, builtin := ctx.ObjectOf(fun).(*types.Builtin); builtin && (fun.Name == "print" || fun.Name == "println") {
						name = fun.Name
					}
				case *ast.SelectorExpr:
					if ctx.IsPkgFunc(call, "fmt", "Print", "Printf", "Println") {
						name = "fmt." + fun.Sel.Name
					}
				}
				if name == "" {
					return
				}
				ctx.Report(goose.Diagnostic{
					Node:      call,
					MessageID: "print",
					Data:      goose.Data{"call": name},
					Help:      "Return the value, accept an io.Writer, or use a logger the caller configures.",
				})
			}),
		)
	},
}
