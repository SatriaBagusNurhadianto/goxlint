package antislop

import (
	"go/ast"
	"go/token"
	"reflect"
	"strings"
	"unicode"

	"github.com/egoisutolabs/goxlint"
)

// NoRestatingComments rejects short comments inside function bodies whose
// every meaningful word already appears in the statement below them:
//
//	// Create a new client
//	client := NewClient()
//
// It only considers single-line // comments on their own line directly above
// a statement, with at most maxRestatingWords words. Comments starting with a
// marker such as "TODO:" or "NOTE:", and comments quoting code, are skipped.
var NoRestatingComments = &goxlint.Rule{
	Meta: goxlint.Meta{
		Name:        "no-restating-comments",
		Description: "Disallow comments that restate the statement below them.",
		Category:    goxlint.Style,
		Messages: map[string]string{
			"restates": "Comment {{comment}} restates the code below it.",
		},
	},
	Create: func(ctx *goxlint.Context) goxlint.Visitor {
		return goxlint.Visit(
			goxlint.On(func(block *ast.BlockStmt) {
				prevEnd := block.Lbrace
				for _, stmt := range block.List {
					checkRestating(ctx, stmt, prevEnd)
					prevEnd = stmt.End()
				}
			}),
			goxlint.On(func(cc *ast.CaseClause) {
				prevEnd := cc.Colon
				for _, stmt := range cc.Body {
					checkRestating(ctx, stmt, prevEnd)
					prevEnd = stmt.End()
				}
			}),
		)
	},
}

const maxRestatingWords = 8

func checkRestating(ctx *goxlint.Context, stmt ast.Stmt, after token.Pos) {
	cg := ctx.CommentsBefore(stmt.Pos())
	if cg == nil || cg.Pos() < after || len(cg.List) != 1 || !strings.HasPrefix(cg.List[0].Text, "//") || !ctx.OwnLine(cg.Pos()) {
		return
	}
	text := commentText(cg.List[0])
	if hasMarker(text) || looksLikeCode(text) {
		return
	}
	words := commentWords(text)
	if len(words) < 2 || len(words) > maxRestatingWords {
		return
	}
	covered := statementWords(stmt)
	content := 0
	for _, w := range words {
		if restatingStopwords[w] {
			continue
		}
		content++
		if !covered[w] {
			return
		}
	}
	if content < 2 {
		return
	}
	ctx.Report(goxlint.Diagnostic{
		Node:      cg,
		MessageID: "restates",
		Data:      goxlint.Data{"comment": "`// " + text + "`"},
		Help:      "Explain why the code does this, or delete the comment.",
	})
}

// looksLikeCode reports whether text quotes code, such as "x[:0:0]" or
// "unsafe.Pointer -> *value". Such comments annotate rather than narrate.
func looksLikeCode(text string) bool {
	text = strings.TrimRight(text, ".?!")
	return strings.ContainsAny(text, "()[]{}*=<>+\"`'.")
}

// hasMarker reports whether text starts with an uppercase marker such as "TODO:".
func hasMarker(text string) bool {
	word, _, found := strings.Cut(text, ":")
	if !found || word == "" || strings.ContainsAny(word, " \t") {
		return false
	}
	return strings.ToUpper(word) == word
}

var restatingStopwords = stemmedSet(
	"a", "an", "the", "to", "of", "for", "and", "or", "is", "are", "it", "its", "this", "that",
	"these", "those", "we", "our", "us", "in", "on", "at", "with", "from", "into", "by", "as",
	"be", "then", "now", "here", "up", "all", "each", "every", "given", "some", "variable", "value",
)

// statementSynonyms maps a word found in code to comment words that restate it.
var statementSynonyms = map[string][]string{
	"return":    {"return", "returning"},
	"if":        {"if", "check", "checking", "whether", "when", "verify", "ensure", "handle"},
	"loop":      {"loop", "iterate", "iterating", "for", "over", "through", "go"},
	"++":        {"increment", "increase", "add", "one", "bump"},
	"--":        {"decrement", "decrease", "subtract", "one"},
	"=":         {"set", "assign", "store", "save", "update"},
	":=":        {"set", "assign", "store", "save", "create", "initialize", "init", "declare", "define", "get", "make", "new"},
	"defer":     {"defer", "deferred"},
	"go":        {"start", "launch", "run", "spawn", "goroutine", "background"},
	"new":       {"new", "create", "make", "initialize", "construct", "instance"},
	"make":      {"make", "create", "new", "initialize", "allocate"},
	"err":       {"err", "error"},
	"nil":       {"nil", "null"},
	"!=":        {"not"},
	"==":        {"equal", "equals", "match", "matches"},
	"<":         {"less", "than", "smaller"},
	">":         {"greater", "than", "larger", "more"},
	"len":       {"length", "size", "count", "number"},
	"append":    {"append", "add", "push"},
	"print":     {"print", "log", "output", "write"},
	"println":   {"print", "log", "output", "write"},
	"printf":    {"print", "log", "output", "write", "format"},
	"ok":        {"ok", "exist", "exists", "found", "present"},
	"call":      {"call", "invoke", "run", "execute"},
	"close":     {"close"},
	"unlock":    {"unlock", "release"},
	"lock":      {"lock", "acquire"},
	"sprintf":   {"format", "build"},
	"fmt":       {"format"},
	"strconv":   {"convert"},
	"itoa":      {"convert", "string"},
	"atoi":      {"convert", "parse", "integer", "int"},
	"break":     {"break", "stop", "exit"},
	"continue":  {"continue", "skip"},
	"switch":    {"switch", "check", "handle"},
	"delete":    {"delete", "remove"},
	"wait":      {"wait"},
	"done":      {"done", "finish", "finished"},
	"wg":        {"wait", "group", "waitgroup"},
	"ctx":       {"context"},
	"req":       {"request"},
	"resp":      {"response"},
	"res":       {"result", "response"},
	"cfg":       {"config", "configuration"},
	"conf":      {"config", "configuration"},
	"db":        {"database"},
	"msg":       {"message"},
	"idx":       {"index"},
	"i":         {"index"},
	"n":         {"number", "count"},
	"buf":       {"buffer"},
	"str":       {"string"},
	"num":       {"number"},
	"cnt":       {"count"},
	"tmp":       {"temporary"},
	"val":       {"value"},
	"args":      {"argument", "arguments"},
	"params":    {"parameter", "parameters"},
	"init":      {"initialize"},
	"marshal":   {"serialize", "encode", "marshal"},
	"unmarshal": {"deserialize", "decode", "parse", "unmarshal"},
}

// statementWords returns the stemmed words that restate stmt: identifier
// parts, operators and keywords, expanded through statementSynonyms. For
// compound statements only the header counts.
func statementWords(stmt ast.Stmt) map[string]bool {
	words := map[string]bool{}
	add := func(w string) {
		w = strings.ToLower(w)
		words[stem(w)] = true
		for _, s := range statementSynonyms[w] {
			words[stem(s)] = true
		}
	}
	var parts []ast.Node
	switch s := stmt.(type) {
	case *ast.IfStmt:
		add("if")
		parts = []ast.Node{s.Init, s.Cond}
	case *ast.ForStmt:
		add("loop")
		parts = []ast.Node{s.Init, s.Cond, s.Post}
	case *ast.RangeStmt:
		add("loop")
		parts = []ast.Node{s.Key, s.Value, s.X}
	case *ast.SwitchStmt:
		add("switch")
		parts = []ast.Node{s.Init, s.Tag}
	case *ast.TypeSwitchStmt:
		add("switch")
		parts = []ast.Node{s.Init, s.Assign}
	default:
		parts = []ast.Node{stmt}
	}
	for _, p := range parts {
		if p == nil || reflect.ValueOf(p).IsNil() { // optional fields hold typed nils
			continue
		}
		ast.Inspect(p, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.Ident:
				for _, w := range splitIdent(n.Name) {
					add(w)
				}
			case *ast.ReturnStmt:
				add("return")
			case *ast.DeferStmt:
				add("defer")
			case *ast.GoStmt:
				add("go")
			case *ast.CallExpr:
				add("call")
			case *ast.IncDecStmt:
				add(n.Tok.String())
			case *ast.AssignStmt:
				add(n.Tok.String())
			case *ast.BinaryExpr:
				add(n.Op.String())
			case *ast.BranchStmt:
				add(n.Tok.String())
			case *ast.FuncLit:
				return false
			}
			return true
		})
	}
	return words
}

// commentWords splits text into lowercase stemmed words.
func commentWords(text string) []string {
	var out []string
	for _, f := range strings.FieldsFunc(text, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		out = append(out, stem(strings.ToLower(f)))
	}
	return out
}

// splitIdent splits camelCase, PascalCase and snake_case identifiers:
// "NewHTTPClient" -> ["New", "HTTP", "Client"].
func splitIdent(name string) []string {
	var parts []string
	for _, seg := range strings.Split(name, "_") {
		runes := []rune(seg)
		start := 0
		for i := 1; i < len(runes); i++ {
			lowerToUpper := unicode.IsLower(runes[i-1]) && unicode.IsUpper(runes[i])
			acronymEnd := unicode.IsUpper(runes[i-1]) && unicode.IsUpper(runes[i]) && i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if lowerToUpper || acronymEnd {
				parts = append(parts, string(runes[start:i]))
				start = i
			}
		}
		if start < len(runes) {
			parts = append(parts, string(runes[start:]))
		}
	}
	return parts
}

// stem drops a plural or third-person "s" so "items" matches "item".
func stem(w string) string {
	if len(w) > 3 && strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss") {
		return w[:len(w)-1]
	}
	return w
}

func stemmedSet(words ...string) map[string]bool {
	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[stem(w)] = true
	}
	return m
}
