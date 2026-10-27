package antislop

import (
	"go/ast"
	"slices"

	"github.com/egoisutolabs/goxlint"
)

// loggingMethods are method names treated as logging calls on any receiver,
// covering log, log/slog, zap, zerolog-style and logrus loggers.
var loggingMethods = []string{
	"Print", "Printf", "Println",
	"Error", "Errorf", "Errorw", "Warn", "Warnf", "Warnw", "Warning", "Warningf",
	"Info", "Infof", "Infow", "Debug", "Debugf", "Debugw",
	"ErrorContext", "WarnContext", "InfoContext", "DebugContext", "Log", "LogAttrs",
}

// NoLogAndReturnError rejects logging an error and then returning it from the
// same `if err != nil` branch. The caller will log or report it again, so one
// failure produces several log lines. Handle the error (log and stop) or
// return it with context, not both.
var NoLogAndReturnError = &goxlint.Rule{
	Meta: goxlint.Meta{
		Name:        "no-log-and-return-error",
		Description: "Disallow logging an error and also returning it.",
		Category:    goxlint.Suspicious,
		NeedsTypes:  true,
		Messages: map[string]string{
			"logAndReturn": "`{{err}}` is logged and then returned, so it will be reported twice.",
		},
	},
	Create: func(ctx *goxlint.Context) goxlint.Visitor {
		isLogCall := func(call *ast.CallExpr) bool {
			if ctx.IsPkgFunc(call, "log", "Print", "Printf", "Println") ||
				ctx.IsPkgFunc(call, "log/slog", "Error", "Warn", "Info", "Debug", "ErrorContext", "WarnContext", "InfoContext", "DebugContext", "Log", "LogAttrs") {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !slices.Contains(loggingMethods, sel.Sel.Name) {
				return false
			}
			fn := ctx.CalleeFunc(call)
			if fn == nil {
				return false
			}
			recv := ctx.TypeOf(sel.X)
			// Test failures are reports, not logs.
			return recv != nil && !isNamed(recv, "testing", "T") && !isNamed(recv, "testing", "B") &&
				!isNamed(recv, "testing", "TB") && !isNamed(recv, "testing", "F") && !isErrorType(recv)
		}

		return goxlint.Visit(
			goxlint.On(func(ifs *ast.IfStmt) {
				errID := errNotNil(ctx, ifs.Cond)
				if errID == nil {
					return
				}
				errObj := ctx.ObjectOf(errID)
				var logCall *ast.CallExpr
				returned := false
				for _, stmt := range ifs.Body.List {
					switch s := stmt.(type) {
					case *ast.ExprStmt:
						if call, ok := s.X.(*ast.CallExpr); ok && logCall == nil && isLogCall(call) && references(ctx, call, errObj) {
							logCall = call
						}
					case *ast.ReturnStmt:
						if references(ctx, s, errObj) {
							returned = true
						}
					}
				}
				if logCall != nil && returned {
					ctx.Report(goxlint.Diagnostic{
						Node:      logCall,
						MessageID: "logAndReturn",
						Data:      goxlint.Data{"err": errID.Name},
						Help:      "Either handle the error here (log it and stop), or return it with context and let the caller decide.",
					})
				}
			}),
		)
	},
}
