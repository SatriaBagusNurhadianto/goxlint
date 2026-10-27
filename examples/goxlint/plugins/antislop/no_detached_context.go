package antislop

import (
	"go/ast"

	"github.com/egoisutolabs/goxlint"
)

// NoDetachedContext rejects context.TODO() outside tests, and
// context.Background() inside a function that already has a context.Context
// parameter in scope. Both detach the work from the caller's cancellation and
// deadline. Replacing Background() with the parameter is offered as a
// suggestion fix.
var NoDetachedContext = &goxlint.Rule{
	Meta: goxlint.Meta{
		Name:        "no-detached-context",
		Description: "Disallow context.TODO() and context.Background() when a context is available.",
		Category:    goxlint.Suspicious,
		NeedsTypes:  true,
		Fixable:     goxlint.FixSuggestion,
		Messages: map[string]string{
			"todo":       "`context.TODO()` is a placeholder for a context that was never plumbed through.",
			"background": "`context.Background()` ignores `{{param}}`, losing its cancellation and deadline.",
		},
	},
	Create: func(ctx *goxlint.Context) goxlint.Visitor {
		// contextParam returns the nearest named context.Context parameter of
		// an enclosing function or closure.
		contextParam := func() string {
			for _, a := range ctx.Ancestors() {
				var ft *ast.FuncType
				switch fn := a.(type) {
				case *ast.FuncLit:
					ft = fn.Type
				case *ast.FuncDecl:
					ft = fn.Type
				default:
					continue
				}
				for _, field := range ft.Params.List {
					if !isNamed(ctx.TypeOf(field.Type), "context", "Context") {
						continue
					}
					for _, name := range field.Names {
						if name.Name != "_" {
							return name.Name
						}
					}
				}
			}
			return ""
		}

		return goxlint.Visit(
			goxlint.On(func(call *ast.CallExpr) {
				switch {
				case ctx.IsPkgFunc(call, "context", "TODO"):
					if ctx.IsTestFile() {
						return
					}
					ctx.Report(goxlint.Diagnostic{
						Node:      call,
						MessageID: "todo",
						Help:      "Accept a `ctx context.Context` parameter and pass it through.",
					})
				case ctx.IsPkgFunc(call, "context", "Background"):
					param := contextParam()
					if param == "" {
						return
					}
					ctx.Report(goxlint.Diagnostic{
						Node:      call,
						MessageID: "background",
						Data:      goxlint.Data{"param": param},
						Help:      "Pass " + param + " through, or derive from it with context.WithoutCancel if the work must outlive the caller.",
						Fix: &goxlint.Fix{
							Kind:    goxlint.FixSuggestion,
							Message: "Use " + param,
							Edits:   []goxlint.Edit{goxlint.ReplaceNode(call, param)},
						},
					})
				}
			}),
		)
	},
}
