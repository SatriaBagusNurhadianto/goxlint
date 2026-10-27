package antislop

import (
	"go/ast"
	"slices"

	"github.com/egoisutolabs/goxlint"
)

var reflectDynamicMethods = []string{"FieldByName", "FieldByNameFunc", "MethodByName", "Call", "CallSlice"}

// NoReflectDynamicAccess rejects looking up fields and methods by name and
// calling functions through reflect: reflect.Value's FieldByName,
// FieldByNameFunc, MethodByName, Call and CallSlice, and reflect.Type's
// FieldByName, FieldByNameFunc and MethodByName. The compiler cannot check the
// names or the argument types, so mistakes become runtime panics.
var NoReflectDynamicAccess = &goxlint.Rule{
	Meta: goxlint.Meta{
		Name:        "no-reflect-dynamic-access",
		Description: "Disallow reflect lookups by name and reflect calls.",
		Category:    goxlint.Restriction,
		NeedsTypes:  true,
		Messages: map[string]string{
			"dynamic": "`reflect.{{type}}.{{method}}` resolves members at runtime, where the compiler cannot check them.",
		},
	},
	Create: func(ctx *goxlint.Context) goxlint.Visitor {
		return goxlint.Visit(
			goxlint.On(func(call *ast.CallExpr) {
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || !slices.Contains(reflectDynamicMethods, sel.Sel.Name) {
					return
				}
				recv := ctx.TypeOf(sel.X)
				typeName := ""
				switch {
				case isNamed(recv, "reflect", "Value"):
					typeName = "Value"
				case isNamed(recv, "reflect", "Type"):
					typeName = "Type"
				default:
					return
				}
				ctx.Report(goxlint.Diagnostic{
					Node:      sel.Sel,
					MessageID: "dynamic",
					Data:      goxlint.Data{"type": typeName, "method": sel.Sel.Name},
					Help:      "Use typed field access and direct calls, or an interface the value implements.",
				})
			}),
		)
	},
}
