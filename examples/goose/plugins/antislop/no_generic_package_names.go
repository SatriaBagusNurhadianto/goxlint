package antislop

import (
	"go/ast"
	"slices"
	"strings"

	"github.com/egoisutolabs/goose-lint/goose"
)

// NoGenericPackageNames rejects catch-all package names such as utils,
// helpers and common. They say nothing about what the package provides and
// attract unrelated code. The banned list is configurable with names.
var NoGenericPackageNames = &goose.Rule{
	Meta: goose.Meta{
		Name:        "no-generic-package-names",
		Description: "Disallow catch-all package names such as utils, helpers and common.",
		Category:    goose.Style,
		Messages: map[string]string{
			"generic": "Package name `{{name}}` says nothing about what the package provides.",
		},
	},
	Create: func(ctx *goose.Context) goose.Visitor {
		opts := struct {
			Names []string `json:"names"`
		}{Names: []string{"util", "utils", "helper", "helpers", "common", "misc", "shared", "lib", "base", "stuff"}}
		if !ctx.Options(&opts) {
			return nil
		}
		return goose.Visit(
			goose.On(func(f *ast.File) {
				name := strings.TrimSuffix(f.Name.Name, "_test")
				if !slices.Contains(opts.Names, name) {
					return
				}
				ctx.Report(goose.Diagnostic{
					Node:      f.Name,
					MessageID: "generic",
					Data:      goose.Data{"name": f.Name.Name},
					Help:      "Name the package after what it does (for example retry, slug or httpheader), and move unrelated helpers next to their callers.",
				})
			}),
		)
	},
}
