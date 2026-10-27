package antislop

import (
	"go/ast"
	"go/types"
	"slices"

	"github.com/egoisutolabs/goose-lint/goose"
)

// NoAnyParameters rejects parameters typed as any or interface{}.
//
// Allowed:
//   - type parameters constrained by any (func f[T any](v T))
//   - variadic ...any, the fmt-style convention (disable with allowVariadic: false)
//   - function literals passed directly as arguments, whose signature the callee dictates
//   - methods named in ignoreMethods, whose signature an interface dictates (default: Scan)
//   - boundary parameters, whose every use narrows the value with a type switch
//     or type assertion, forwards it to another any parameter such as
//     json.Marshal's, or returns it as an any result; at least one use must
//     narrow or forward (disable with allowNarrowing: false)
//
// Test files are skipped unless checkTests is set.
var NoAnyParameters = &goose.Rule{
	Meta: goose.Meta{
		Name:        "no-any-parameters",
		Description: "Disallow parameters typed as any or interface{} unless the function narrows them.",
		Category:    goose.Restriction,
		NeedsTypes:  true,
		Messages: map[string]string{
			"anyParameter": "Parameter `{{name}}` is typed as `{{type}}`, which accepts anything and proves nothing.",
		},
	},
	Create: func(ctx *goose.Context) goose.Visitor {
		opts := struct {
			AllowVariadic  bool     `json:"allowVariadic"`
			AllowNarrowing bool     `json:"allowNarrowing"`
			IgnoreMethods  []string `json:"ignoreMethods"`
			CheckTests     bool     `json:"checkTests"`
		}{AllowVariadic: true, AllowNarrowing: true, IgnoreMethods: []string{"Scan"}}
		if !ctx.Options(&opts) {
			return nil
		}
		if ctx.IsTestFile() && !opts.CheckTests {
			return nil
		}

		check := func(ft *ast.FuncType, body *ast.BlockStmt) {
			for _, field := range ft.Params.List {
				typ := field.Type
				if ell, ok := typ.(*ast.Ellipsis); ok {
					if opts.AllowVariadic {
						continue
					}
					typ = ell.Elt
				}
				if !isEmptyInterface(ctx.TypeOf(typ)) {
					continue
				}
				names := field.Names
				if len(names) == 0 {
					names = []*ast.Ident{{Name: "_"}}
				}
				for _, name := range names {
					if opts.AllowNarrowing && body != nil && name.Name != "_" && onlyNarrowed(ctx, ft, body, ctx.ObjectOf(name)) {
						continue
					}
					ctx.Report(goose.Diagnostic{
						Node:      field.Type,
						MessageID: "anyParameter",
						Data:      goose.Data{"name": name.Name, "type": ctx.Snippet(field.Type)},
						Help:      "Accept a concrete type, a small interface naming the behavior you need, or a type parameter. At a decoding boundary, narrow it immediately with a type switch.",
					})
				}
			}
		}

		return goose.Visit(
			goose.On(func(fd *ast.FuncDecl) {
				if fd.Recv != nil && slices.Contains(opts.IgnoreMethods, fd.Name.Name) {
					return
				}
				check(fd.Type, fd.Body)
			}),
			goose.On(func(fl *ast.FuncLit) {
				if _, isArg := ctx.Parent().(*ast.CallExpr); isArg {
					return
				}
				check(fl.Type, fl.Body)
			}),
		)
	},
}

// onlyNarrowed reports whether every use of obj in body is the operand of a
// type switch or type assertion, an argument to a parameter that is itself
// any, or a returned any result, and at least one use is one of the first two.
func onlyNarrowed(ctx *goose.Context, ft *ast.FuncType, body *ast.BlockStmt, obj types.Object) bool {
	if obj == nil {
		return false
	}
	allowed := map[*ast.Ident]bool{}
	// Returning the value unchanged as an any result is allowed, but does not
	// by itself make the parameter a boundary.
	passThrough := map[*ast.Ident]bool{}
	var results []types.Type
	if ft.Results != nil {
		for _, field := range ft.Results.List {
			for range max(len(field.Names), 1) {
				results = append(results, ctx.TypeOf(field.Type))
			}
		}
	}
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncLit:
			return false // its returns belong to a different signature
		case *ast.ReturnStmt:
			if len(n.Results) != len(results) {
				return true
			}
			for i, r := range n.Results {
				if id, ok := ast.Unparen(r).(*ast.Ident); ok && isEmptyInterface(results[i]) {
					passThrough[id] = true
				}
			}
		}
		return true
	})
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.TypeAssertExpr: // includes the v.(type) of a type switch
			if id, ok := ast.Unparen(n.X).(*ast.Ident); ok {
				allowed[id] = true
			}
		case *ast.CallExpr:
			sig, ok := ctx.TypeOf(n.Fun).(*types.Signature)
			if !ok {
				return true
			}
			for i, arg := range n.Args {
				if id, ok := ast.Unparen(arg).(*ast.Ident); ok && isEmptyInterface(paramType(sig, i, n.Ellipsis.IsValid())) {
					allowed[id] = true
				}
			}
		}
		return true
	})
	strong := 0
	narrowed := true
	ast.Inspect(body, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && ctx.ObjectOf(id) == obj {
			switch {
			case allowed[id]:
				strong++
			case !passThrough[id]:
				narrowed = false
			}
		}
		return narrowed
	})
	return strong > 0 && narrowed
}

// paramType returns the type a call's i-th argument is passed as, accounting
// for variadic parameters, or nil.
func paramType(sig *types.Signature, i int, spread bool) types.Type {
	params := sig.Params()
	if sig.Variadic() && i >= params.Len()-1 {
		last := params.At(params.Len() - 1).Type()
		if spread {
			return last
		}
		if s, ok := last.(*types.Slice); ok {
			return s.Elem()
		}
		return nil
	}
	if i < params.Len() {
		return params.At(i).Type()
	}
	return nil
}
