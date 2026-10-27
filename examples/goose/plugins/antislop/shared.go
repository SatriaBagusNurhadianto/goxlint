package antislop

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/egoisutolabs/goose-lint/goose"
)

// isErrorType reports whether t is an interface or concrete type implementing error.
func isErrorType(t types.Type) bool {
	if t == nil {
		return false
	}
	if b, ok := t.(*types.Basic); ok && b.Kind() == types.UntypedNil {
		return false
	}
	return types.Implements(t, errorInterface)
}

// isNamed reports whether t is the named type pkgPath.name, or a pointer to it.
func isNamed(t types.Type, pkgPath, name string) bool {
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	n, ok := types.Unalias(t).(*types.Named)
	return ok && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == pkgPath && n.Obj().Name() == name
}

// isMainPackage reports whether the file belongs to package main.
func isMainPackage(ctx *goose.Context) bool { return ctx.File.Name.Name == "main" }

// enclosingFuncDecl returns the top-level function containing the current node, or nil.
func enclosingFuncDecl(ctx *goose.Context) *ast.FuncDecl {
	for _, a := range ctx.Ancestors() {
		if fd, ok := a.(*ast.FuncDecl); ok {
			return fd
		}
	}
	return nil
}

// enclosingSignature returns the signature of the nearest function or function
// literal containing the current node, or nil.
func enclosingSignature(ctx *goose.Context) *types.Signature {
	for _, a := range ctx.Ancestors() {
		switch fn := a.(type) {
		case *ast.FuncLit:
			sig, _ := ctx.TypeOf(fn).(*types.Signature)
			return sig
		case *ast.FuncDecl:
			if obj := ctx.ObjectOf(fn.Name); obj != nil {
				sig, _ := obj.Type().(*types.Signature)
				return sig
			}
			return nil
		}
	}
	return nil
}

// errNotNil returns the error variable x when cond is `x != nil`, or nil.
func errNotNil(ctx *goose.Context, cond ast.Expr) *ast.Ident {
	bin, ok := ast.Unparen(cond).(*ast.BinaryExpr)
	if !ok || bin.Op != token.NEQ {
		return nil
	}
	x, y := ast.Unparen(bin.X), ast.Unparen(bin.Y)
	if isNilIdent(x) {
		x, y = y, x
	}
	id, ok := x.(*ast.Ident)
	if !ok || !isNilIdent(y) || !isErrorType(ctx.TypeOf(id)) {
		return nil
	}
	return id
}

func isNilIdent(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "nil"
}

// references reports whether node mentions obj.
func references(ctx *goose.Context, node ast.Node, obj types.Object) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && ctx.ObjectOf(id) == obj {
			found = true
		}
		return !found
	})
	return found
}

// hasComment reports whether a comment sits on the line above node or at the
// end of its last line.
func hasComment(ctx *goose.Context, node ast.Node) bool {
	return ctx.CommentsBefore(node.Pos()) != nil || ctx.TrailingComment(node.End()) != nil
}

// commentText strips comment markers and surrounding space.
func commentText(c *ast.Comment) string {
	text := strings.TrimPrefix(c.Text, "//")
	text = strings.TrimPrefix(text, "/*")
	text = strings.TrimSuffix(text, "*/")
	return strings.TrimSpace(text)
}
