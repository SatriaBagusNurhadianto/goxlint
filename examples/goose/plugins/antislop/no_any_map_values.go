package antislop

import (
	"go/ast"

	"github.com/egoisutolabs/goose-lint/goose"
)

// NoAnyMapValues rejects map types whose values are any or interface{}, such
// as map[string]any. They replace a type the compiler can check with string
// keys and runtime assertions. At a genuine JSON or template boundary, use
// //goose:ignore with the reason.
var NoAnyMapValues = &goose.Rule{
	Meta: goose.Meta{
		Name:        "no-any-map-values",
		Description: "Disallow map types with any or interface{} values.",
		Category:    goose.Restriction,
		NeedsTypes:  true,
		Messages: map[string]string{
			"anyMap": "`{{type}}` discards the value types; every read needs a type assertion.",
		},
	},
	Create: func(ctx *goose.Context) goose.Visitor {
		return goose.Visit(
			goose.On(func(m *ast.MapType) {
				if !isEmptyInterface(ctx.TypeOf(m.Value)) {
					return
				}
				ctx.Report(goose.Diagnostic{
					Node:      m,
					MessageID: "anyMap",
					Data:      goose.Data{"type": ctx.Snippet(m)},
					Help:      "Use a struct with named fields, or a map with a concrete value type.",
				})
			}),
		)
	},
}
