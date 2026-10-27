package maintainability

import (
	"fmt"
	"go/ast"
	"go/scanner"
	"go/token"
	"go/types"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/egoisutolabs/goose-lint/goose"
)

// NoDuplicateFunctions finds structurally identical named functions and methods
// in the loaded package. It normalizes local bindings using type information,
// preserves external references, and offers no automatic rewrite.
var NoDuplicateFunctions = &goose.Rule{
	Meta: goose.Meta{
		Name: "no-duplicate-functions", Category: goose.Pedantic, NeedsTypes: true,
		Description: "Find duplicate function implementations within a package.",
		Messages: map[string]string{
			"duplicate": "Function {{name}} duplicates {{other}} at {{location}}.",
		},
	},
	AnalyzePackage: findDuplicateFunctions,
}

type functionCandidate struct {
	ctx                 *goose.Context
	fn                  *ast.FuncDecl
	statements, minimum int
}

func findDuplicateFunctions(pkg *goose.PackageContext) {
	groups := map[string][]functionCandidate{}
	var order []string
	for _, ctx := range pkg.Files {
		opts := struct {
			MinStatements int  `json:"minStatements"`
			CheckTests    bool `json:"checkTests"`
		}{MinStatements: 5}
		if ctx.Enabled {
			if !ctx.Options(&opts) {
				continue
			}
			if opts.MinStatements < 1 {
				ctx.Report(goose.Diagnostic{Pos: ctx.File.Package, Message: "invalid options: minStatements must be at least 1"})
				continue
			}
		}
		if ctx.IsTestFile() && !opts.CheckTests {
			continue
		}
		for _, decl := range ctx.File.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			count := statementCount(fn.Body)
			if count == 0 {
				continue
			}
			key := normalizedFunction(ctx, fn)
			if _, exists := groups[key]; !exists {
				order = append(order, key)
			}
			groups[key] = append(groups[key], functionCandidate{ctx, fn, count, opts.MinStatements})
		}
	}
	for _, key := range order {
		functions := groups[key]
		if len(functions) < 2 {
			continue
		}
		original := 0
		// A disabled file can still supply an implementation to reuse. Prefer
		// it as the reference so enabled files are checked regardless of order.
		for i, fn := range functions {
			if !fn.ctx.Enabled {
				original = i
				break
			}
		}
		first := functions[original]
		pos := first.ctx.Fset.PositionFor(first.fn.Name.Pos(), false)
		location := fmt.Sprintf("%s:%d:%d", filepath.Base(first.ctx.Filename), pos.Line, pos.Column)
		for i, fn := range functions {
			if i == original || !fn.ctx.Enabled || fn.statements < fn.minimum {
				continue
			}
			fn.ctx.Report(goose.Diagnostic{
				Node: fn.fn.Name, MessageID: "duplicate",
				Data: goose.Data{"name": fn.fn.Name.Name, "other": first.fn.Name.Name, "location": location},
				Help: "Review whether the existing function can be reused or a shared helper extracted. Structural similarity does not prove behavioral equivalence.",
			})
		}
	}
}

func statementCount(body *ast.BlockStmt) int {
	count := 0
	ast.Inspect(body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.BlockStmt, *ast.EmptyStmt, *ast.CaseClause, *ast.CommClause:
		case ast.Stmt:
			count++
		}
		return true
	})
	return count
}

// normalizedFunction retains the complete token structure, including signature,
// types, literals and operators. Only identifiers bound inside this function,
// its declaration name/self references, and import aliases are canonicalized.
// Object identity distinguishes shadowed locals and keeps parameter order intact.
func normalizedFunction(ctx *goose.Context, fn *ast.FuncDecl) string {
	bindings := map[types.Object]string{}
	self := ctx.TypesInfo.Defs[fn.Name]
	ast.Inspect(fn, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok || id.Name == "_" {
			return true
		}
		obj := ctx.TypesInfo.Defs[id]
		if obj != nil && obj != self && localBinding(obj) {
			bindings[obj] = "$local:" + strconv.Itoa(len(bindings))
		}
		return true
	})
	tf := ctx.Fset.File(fn.Pos())
	start, end := tf.Offset(fn.Pos()), tf.Offset(fn.End())
	names := map[int]string{}
	ast.Inspect(fn, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		name := id.Name
		obj := ctx.ObjectOf(id)
		switch {
		case id == fn.Name || (self != nil && obj == self):
			name = "$self"
		case bindings[obj] != "":
			name = bindings[obj]
		default:
			if imported, ok := obj.(*types.PkgName); ok {
				name = "$package:" + imported.Imported().Path()
			} else if obj != nil && obj.Pkg() != nil {
				name = "$object:" + obj.Pkg().Path() + ":" + obj.Name()
			}
		}
		names[tf.Offset(id.Pos())-start] = name
		return true
	})
	var scan scanner.Scanner
	scanFile := token.NewFileSet().AddFile(ctx.Filename, -1, end-start)
	scan.Init(scanFile, ctx.Source[start:end], nil, 0)
	type part struct {
		token token.Token
		text  string
	}
	var parts []part
	for {
		pos, tok, literal := scan.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.IDENT {
			literal = names[scanFile.Offset(pos)]
		}
		if tok == token.SEMICOLON {
			literal = ";"
		}
		parts = append(parts, part{tok, literal})
	}
	var normalized strings.Builder
	for i, p := range parts {
		next := token.EOF
		if i+1 < len(parts) {
			next = parts[i+1].token
		}
		// Go permits omitted semicolons before closing delimiters and trailing
		// commas in multiline lists. Neither changes the parsed structure.
		if p.token == token.SEMICOLON && (next == token.RBRACE || next == token.RPAREN || next == token.EOF) {
			continue
		}
		if p.token == token.COMMA && (next == token.RBRACE || next == token.RPAREN || next == token.RBRACK) {
			continue
		}
		normalized.WriteString(strconv.Itoa(int(p.token)))
		normalized.WriteByte(':')
		normalized.WriteString(strconv.Quote(p.text))
	}
	return normalized.String()
}

func localBinding(obj types.Object) bool {
	switch obj := obj.(type) {
	case *types.Var:
		return !obj.IsField()
	case *types.Const, *types.Label, *types.TypeName:
		return true
	}
	return false
}
