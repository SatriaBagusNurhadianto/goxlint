package antislop

import (
	"go/ast"
	"slices"

	"github.com/egoisutolabs/goxlint"
)

// NoAnyReturns rejects functions whose results are typed as any or
// interface{}, which forces every caller to type-assert. Methods named in
// ignoreMethods are exempt because an interface dictates their signature
// (default: Value for database/sql/driver.Valuer, MarshalYAML for yaml.Marshaler).
var NoAnyReturns = &goxlint.Rule{
	Meta: goxlint.Meta{
		Name:        "no-any-returns",
		Description: "Disallow results typed as any or interface{}.",
		Category:    goxlint.Restriction,
		NeedsTypes:  true,
		Messages: map[string]string{
			"anyReturn": "`{{name}}` returns `{{type}}`, so every caller has to type-assert the result.",
		},
	},
	Create: func(ctx *goxlint.Context) goxlint.Visitor {
		opts := struct {
			IgnoreMethods []string `json:"ignoreMethods"`
		}{IgnoreMethods: []string{"Value", "MarshalYAML"}}
		if !ctx.Options(&opts) {
			return nil
		}

		check := func(name string, ft *ast.FuncType) {
			if ft.Results == nil {
				return
			}
			for _, field := range ft.Results.List {
				if isEmptyInterface(ctx.TypeOf(field.Type)) {
					ctx.Report(goxlint.Diagnostic{
						Node:      field.Type,
						MessageID: "anyReturn",
						Data:      goxlint.Data{"name": name, "type": ctx.Snippet(field.Type)},
						Help:      "Return a concrete type, a small interface, or make the function generic.",
					})
				}
			}
		}

		return goxlint.Visit(
			goxlint.On(func(fd *ast.FuncDecl) {
				if fd.Recv != nil && slices.Contains(opts.IgnoreMethods, fd.Name.Name) {
					return
				}
				check(fd.Name.Name, fd.Type)
			}),
			goxlint.On(func(fl *ast.FuncLit) {
				if _, isArg := ctx.Parent().(*ast.CallExpr); isArg {
					return
				}
				check("function literal", fl.Type)
			}),
		)
	},
}
