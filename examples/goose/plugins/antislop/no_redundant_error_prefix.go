package antislop

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/egoisutolabs/goose-lint/goose"
)

// redundantPrefixes restate that something failed, which every error already
// says. Wrapped errors then read "failed to load: failed to open: failed to read".
var redundantPrefixes = []string{"failed to ", "unable to ", "could not ", "couldn't ", "error: ", "error "}

// NoRedundantErrorPrefix rejects error messages created with errors.New or
// fmt.Errorf that start with a phrase such as "failed to".
var NoRedundantErrorPrefix = &goose.Rule{
	Meta: goose.Meta{
		Name:        "no-redundant-error-prefix",
		Description: `Disallow error messages starting with "failed to", "unable to" and similar.`,
		Category:    goose.Style,
		NeedsTypes:  true,
		Fixable:     goose.FixSuggestion,
		Messages: map[string]string{
			"redundantPrefix": "Error message starts with {{prefix}}; every error already says something failed.",
		},
	},
	Create: func(ctx *goose.Context) goose.Visitor {
		return goose.Visit(
			goose.On(func(call *ast.CallExpr) {
				if len(call.Args) == 0 || !(ctx.IsPkgFunc(call, "errors", "New") || ctx.IsPkgFunc(call, "fmt", "Errorf")) {
					return
				}
				lit, ok := call.Args[0].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return
				}
				msg, err := strconv.Unquote(lit.Value)
				if err != nil {
					return
				}
				for _, prefix := range redundantPrefixes {
					if len(msg) <= len(prefix) || !strings.EqualFold(msg[:len(prefix)], prefix) {
						continue
					}
					d := goose.Diagnostic{
						Node:      lit,
						MessageID: "redundantPrefix",
						Data:      goose.Data{"prefix": strconv.Quote(strings.TrimSpace(prefix))},
						Help:      `Name the operation instead, for example "open config: %w".`,
					}
					// The prefix is plain ASCII, so it appears verbatim after the opening quote.
					if strings.EqualFold(lit.Value[1:1+len(prefix)], prefix) {
						d.Fix = &goose.Fix{
							Kind:    goose.FixSuggestion,
							Message: "Remove " + strconv.Quote(strings.TrimSpace(prefix)),
							Edits:   []goose.Edit{goose.ReplaceNode(lit, lit.Value[:1]+lit.Value[1+len(prefix):])},
						}
					}
					ctx.Report(d)
					return
				}
			}),
		)
	},
}
