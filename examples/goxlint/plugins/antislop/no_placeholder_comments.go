package antislop

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/egoisutolabs/goxlint"
)

// placeholderPhrases mark code that is hypothetical, simplified or unfinished.
// Coding agents leave them behind when they stub out work they did not do.
var placeholderPhrases = []string{
	"in a real implementation",
	"in a real application",
	"in a real app",
	"in a real-world",
	"in a real world",
	"in a production environment",
	"in production, you",
	"in production you",
	"simplified implementation",
	"simplified version of",
	"placeholder implementation",
	"stub implementation",
	"this is a stub",
	"implement me",
	"todo: implement",
	"not yet implemented",
	"not implemented yet",
	"your code here",
	"your logic here",
	"... existing code",
	"existing code ...",
	"... rest of the",
}

// placeholderPanics are panic messages that stand in for missing code.
var placeholderPanics = []string{"not implemented", "unimplemented", "todo", "implement me"}

// NoPlaceholderComments rejects comments describing hypothetical or unfinished
// code (see placeholderPhrases) and panics that stand in for missing code,
// such as panic("not implemented").
var NoPlaceholderComments = &goxlint.Rule{
	Meta: goxlint.Meta{
		Name:        "no-placeholder-comments",
		Description: `Disallow placeholder comments such as "in a real implementation" and panic("not implemented").`,
		Category:    goxlint.Suspicious,
		Messages: map[string]string{
			"placeholderComment": "Comment says {{phrase}}, which describes code that is hypothetical or unfinished.",
			"placeholderPanic":   "panic({{message}}) stands in for missing code.",
		},
	},
	Create: func(ctx *goxlint.Context) goxlint.Visitor {
		return goxlint.Visit(
			goxlint.On(func(f *ast.File) {
				for _, cg := range f.Comments {
					for _, c := range cg.List {
						lower := strings.ToLower(commentText(c))
						for _, phrase := range placeholderPhrases {
							if strings.Contains(lower, phrase) {
								ctx.Report(goxlint.Diagnostic{
									Node:      c,
									MessageID: "placeholderComment",
									Data:      goxlint.Data{"phrase": strconv.Quote(phrase)},
									Help:      "Write the real implementation, or replace the comment with a tracked TODO that names the missing behavior.",
								})
								break
							}
						}
					}
				}
			}),
			goxlint.On(func(call *ast.CallExpr) {
				fn, ok := call.Fun.(*ast.Ident)
				if !ok || fn.Name != "panic" || len(call.Args) != 1 {
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
				msg = strings.ToLower(strings.TrimSpace(msg))
				for _, p := range placeholderPanics {
					if strings.HasPrefix(msg, p) {
						ctx.Report(goxlint.Diagnostic{
							Node:      call,
							MessageID: "placeholderPanic",
							Data:      goxlint.Data{"message": lit.Value},
							Help:      "Implement it, or return an error the caller can handle.",
						})
						return
					}
				}
			}),
		)
	},
}
