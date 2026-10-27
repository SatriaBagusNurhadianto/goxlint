package antislop

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/egoisutolabs/goxlint"
)

// NoPanicInLibrary rejects panic outside package main and tests, where it
// crashes a program that only wanted to call a function. Allowed:
//   - Must* and must* functions, whose contract is to panic
//   - init functions
//   - panic("unreachable...") and panic("impossible...") marking provably dead code
var NoPanicInLibrary = &goxlint.Rule{
	Meta: goxlint.Meta{
		Name:        "no-panic-in-library",
		Description: "Disallow panic in library code; return an error instead.",
		Category:    goxlint.Restriction,
		Messages: map[string]string{
			"panic": "`panic` in library code crashes the caller's program.",
		},
	},
	Create: func(ctx *goxlint.Context) goxlint.Visitor {
		if isMainPackage(ctx) || ctx.IsTestFile() {
			return nil
		}
		return goxlint.Visit(
			goxlint.On(func(call *ast.CallExpr) {
				fn, ok := call.Fun.(*ast.Ident)
				if !ok || fn.Name != "panic" || len(call.Args) != 1 || isUnreachableMessage(call.Args[0]) {
					return
				}
				if fd := enclosingFuncDecl(ctx); fd != nil {
					name := fd.Name.Name
					if (fd.Recv == nil && name == "init") || strings.HasPrefix(name, "Must") || strings.HasPrefix(name, "must") {
						return
					}
				}
				ctx.Report(goxlint.Diagnostic{
					Node:      call,
					MessageID: "panic",
					Help:      "Return an error, or provide a Must variant whose name says it panics.",
				})
			}),
		)
	},
}

func isUnreachableMessage(arg ast.Expr) bool {
	lit, ok := arg.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return false
	}
	msg, err := strconv.Unquote(lit.Value)
	if err != nil {
		return false
	}
	msg = strings.ToLower(msg)
	return strings.HasPrefix(msg, "unreachable") || strings.HasPrefix(msg, "impossible")
}
