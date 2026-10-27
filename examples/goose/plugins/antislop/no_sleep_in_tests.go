package antislop

import (
	"go/ast"

	"github.com/egoisutolabs/goose-lint/goose"
)

// NoSleepInTests rejects time.Sleep in _test.go files, where it is used to
// wait for something to happen and makes tests slow and flaky. Sleeps inside
// a testing/synctest bubble are allowed, because time there is virtual.
var NoSleepInTests = &goose.Rule{
	Meta: goose.Meta{
		Name:        "no-sleep-in-tests",
		Description: "Disallow time.Sleep in tests outside synctest bubbles.",
		Category:    goose.Suspicious,
		NeedsTypes:  true,
		Messages: map[string]string{
			"sleep": "`time.Sleep` in a test guesses how long something takes, which makes the test slow and flaky.",
		},
	},
	Create: func(ctx *goose.Context) goose.Visitor {
		if !ctx.IsTestFile() {
			return nil
		}
		return goose.Visit(
			goose.On(func(call *ast.CallExpr) {
				if !ctx.IsPkgFunc(call, "time", "Sleep") {
					return
				}
				for _, a := range ctx.Ancestors() {
					if outer, ok := a.(*ast.CallExpr); ok && ctx.IsPkgFunc(outer, "testing/synctest", "Test", "Run") {
						return
					}
				}
				ctx.Report(goose.Diagnostic{
					Node:      call,
					MessageID: "sleep",
					Help:      "Wait on a channel or sync.WaitGroup, poll the condition with a deadline, or run the test in a testing/synctest bubble.",
				})
			}),
		)
	},
}
