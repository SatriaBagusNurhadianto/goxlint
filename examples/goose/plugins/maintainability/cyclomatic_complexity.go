package maintainability

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/egoisutolabs/goose-lint/goose"
)

// CyclomaticComplexity limits independent paths through each function or closure.
// The score is 1 plus each if, for, range, non-default case or communication
// clause, and short-circuit boolean operator. Nested closures are scored separately.
var CyclomaticComplexity = &goose.Rule{
	Meta: goose.Meta{
		Name: "cyclomatic-complexity", Category: goose.Pedantic,
		Description: "Limit cyclomatic complexity per function or closure.",
		Messages: map[string]string{
			"complex": "Function {{name}} has cyclomatic complexity {{score}} (maximum {{max}}).",
		},
	},
	Create: func(ctx *goose.Context) goose.Visitor {
		opts := struct {
			Max int `json:"max"`
		}{Max: 10}
		if !ctx.Options(&opts) {
			return nil
		}
		if opts.Max < 1 {
			ctx.Report(goose.Diagnostic{Pos: ctx.File.Package, Message: "invalid options: max must be at least 1"})
			return nil
		}
		check := func(name string, node ast.Node, body *ast.BlockStmt) {
			if body == nil {
				return
			}
			score := complexity(body)
			if score > opts.Max {
				ctx.Report(goose.Diagnostic{
					Node: node, MessageID: "complex",
					Data: goose.Data{"name": name, "score": fmt.Sprint(score), "max": fmt.Sprint(opts.Max)},
					Help: "Simplify branching or extract a cohesive operation into a separate function.",
				})
			}
		}
		return goose.Visit(
			goose.On(func(fn *ast.FuncDecl) { check(fn.Name.Name, fn.Name, fn.Body) }),
			goose.On(func(fn *ast.FuncLit) { check("<literal>", fn.Type, fn.Body) }),
		)
	},
}

func complexity(body *ast.BlockStmt) int {
	score := 1
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
			score++
		case *ast.CaseClause:
			if n.List != nil {
				score++
			}
		case *ast.CommClause:
			if n.Comm != nil {
				score++
			}
		case *ast.BinaryExpr:
			if n.Op == token.LAND || n.Op == token.LOR {
				score++
			}
		}
		return true
	})
	return score
}
