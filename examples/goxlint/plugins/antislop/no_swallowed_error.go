package antislop

import (
	"go/ast"
	"go/types"

	"github.com/egoisutolabs/goxlint"
)

// NoSwallowedError rejects `if err != nil { return ..., nil }` in functions
// that return an error, when the branch never uses err. The failure is checked
// and then silently converted into success. A comment inside the branch
// explaining why the failure is safe to ignore exempts it.
var NoSwallowedError = &goxlint.Rule{
	Meta: goxlint.Meta{
		Name:        "no-swallowed-error",
		Description: "Disallow returning a nil error from inside an `if err != nil` branch.",
		Category:    goxlint.Suspicious,
		NeedsTypes:  true,
		Messages: map[string]string{
			"swallowed": "`{{err}}` is checked and then dropped: this branch returns a nil error.",
		},
	},
	Create: func(ctx *goxlint.Context) goxlint.Visitor {
		return goxlint.Visit(
			goxlint.On(func(ifs *ast.IfStmt) {
				errID := errNotNil(ctx, ifs.Cond)
				if errID == nil {
					return
				}
				sig := enclosingSignature(ctx)
				if sig == nil || sig.Results().Len() == 0 || !isErrorType(sig.Results().At(sig.Results().Len()-1).Type()) {
					return
				}
				errObj := ctx.ObjectOf(errID)
				if references(ctx, ifs.Body, errObj) || bodyHasComment(ctx, ifs.Body) {
					return
				}
				for _, stmt := range ifs.Body.List {
					ret, ok := stmt.(*ast.ReturnStmt)
					if !ok || len(ret.Results) != sig.Results().Len() {
						continue
					}
					last := ret.Results[len(ret.Results)-1]
					if id, ok := ast.Unparen(last).(*ast.Ident); ok {
						if _, isNil := ctx.ObjectOf(id).(*types.Nil); isNil {
							ctx.Report(goxlint.Diagnostic{
								Node:      ret,
								MessageID: "swallowed",
								Data:      goxlint.Data{"err": errID.Name},
								Help:      "Return or wrap " + errID.Name + ", or add a comment in the branch explaining why the failure is safe to ignore.",
							})
						}
					}
				}
			}),
		)
	},
}

// bodyHasComment reports whether any comment lies inside block.
func bodyHasComment(ctx *goxlint.Context, block *ast.BlockStmt) bool {
	for _, cg := range ctx.File.Comments {
		if cg.Pos() > block.Lbrace && cg.End() < block.Rbrace {
			return true
		}
	}
	return false
}
