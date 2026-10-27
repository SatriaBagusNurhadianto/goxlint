package antislop

import (
	"go/ast"
	"slices"

	"github.com/egoisutolabs/goose-lint/goose"
)

// NoAnyReturns rejects functions whose results are typed as any or
// interface{}, which forces every caller to type-assert. Methods named in
// ignoreMethods are exempt because an interface dictates their signature
// (default: Value for database/sql/driver.Valuer, MarshalYAML for yaml.Marshaler).
var NoAnyReturns = &goose.Rule{
	Meta: goose.Meta{
		Name:        "no-any-returns",
		Description: "Disallow results typed as any or interface{}.",
		Category:    goose.Restriction,
		NeedsTypes:  true,
		Messages: map[string]string{
			"anyReturn": "`{{name}}` returns `{{type}}`, so every caller has to type-assert the result.",
		},
	},
	Create: func(ctx *goose.Context) goose.Visitor {
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
					ctx.Report(goose.Diagnostic{
						Node:      field.Type,
						MessageID: "anyReturn",
						Data:      goose.Data{"name": name, "type": ctx.Snippet(field.Type)},
						Help:      "Return a concrete type, a small interface, or make the function generic.",
					})
				}
			}
		}

		return goose.Visit(
			goose.On(func(fd *ast.FuncDecl) {
				if fd.Recv != nil && slices.Contains(opts.IgnoreMethods, fd.Name.Name) {
					return
				}
				check(fd.Name.Name, fd.Type)
			}),
			goose.On(func(fl *ast.FuncLit) {
				if _, isArg := ctx.Parent().(*ast.CallExpr); isArg {
					return
				}
				check("function literal", fl.Type)
			}),
		)
	},
}
