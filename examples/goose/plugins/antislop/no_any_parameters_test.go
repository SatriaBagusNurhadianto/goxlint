package antislop

import (
	"testing"

	"github.com/egoisutolabs/goose-lint/goosetest"
)

func TestNoAnyParameters(t *testing.T) {
	errAny := goosetest.Error{MessageID: "anyParameter"}

	goosetest.Run(t, NoAnyParameters, goosetest.Cases{
		Valid: []goosetest.Case{
			{Name: "concrete", Code: `func f(n int, s string) {}`},
			{Name: "type parameter", Code: `func f[T any](v T) {}`},
			{Name: "variadic", Code: `func logf(format string, args ...any) {}`},
			{Name: "small interface", Code: `
import "io"

func f(r io.Reader) {}`},
			{Name: "callback argument", Code: `
func walk(fn func(any)) {}

func g() { walk(func(v any) {}) }`},
			{Name: "ignored method", Code: `
type ID int

func (i *ID) Scan(src any) error { return nil }`},
			{Name: "type switch boundary", Code: `
func toFloat64(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	}
	return 0, false
}`},
			{Name: "recursive type switch", Code: `
func ordered(value any) any {
	switch v := value.(type) {
	case []any:
		for i := range v {
			v[i] = ordered(v[i])
		}
		return v
	}
	return nil
}`},
			{Name: "default branch returns value", Code: `
func normalizeConst(v any) any {
	switch x := v.(type) {
	case int:
		return float64(x)
	default:
		return v
	}
}`},
			{Name: "comma-ok assertion", Code: `
func name(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}`},
			{Name: "forwarded to json.Marshal", Code: `
import "encoding/json"

func CanonicalSHA(v any) ([]byte, error) { return json.Marshal(v) }`},
			{Name: "forwarded to variadic any", Code: `
import "fmt"

func describe(v any) string { return fmt.Sprint("value: ", v) }`},
			{Name: "test file", Filename: "x_test.go", Code: `func helper(v any) {}`},
			{Name: "shadowed any", Code: `
type any int

func f(v any) {}`},
		},
		Invalid: []goosetest.Case{
			{Name: "any", Code: `func f(v any) {}`, Errors: []goosetest.Error{{
				Message: "Parameter `v` is typed as `any`, which accepts anything and proves nothing.",
				Line:    1, Column: 10,
			}}},
			{Name: "empty interface", Code: `func f(v interface{}) {}`, Errors: []goosetest.Error{errAny}},
			{Name: "alias", Code: `
type Payload = any

func f(p Payload) {}`, Errors: []goosetest.Error{{MessageID: "anyParameter", Line: 4}}},
			{Name: "named empty interface", Code: `
type Payload interface{}

func f(p Payload) {}`, Errors: []goosetest.Error{errAny}},
			{Name: "grouped names", Code: `func f(a, b any) {}`, Errors: []goosetest.Error{errAny, errAny}},
			{Name: "assigned literal", Code: `var f = func(v any) {}`, Errors: []goosetest.Error{errAny}},
			{Name: "unused", Code: `func f(v any) int { return 1 }`, Errors: []goosetest.Error{errAny}},
			{Name: "stored after switch", Code: `
var cache = map[string]bool{}

func put(k string, v any) {
	switch v.(type) {
	case string:
	}
	store(v)
}

func store(x any) {}

var _ = cache`, Errors: []goosetest.Error{errAny}},
			{Name: "returned unnarrowed", Code: `func wrap(v any) any { return v }`, Errors: []goosetest.Error{errAny}},
			{Name: "returned from closure", Code: `
func f(v any) func() any {
	switch v.(type) {
	case int:
	}
	return func() any { return v }
}`, Errors: []goosetest.Error{errAny}},
			{Name: "narrowing disallowed", Options: map[string]any{"allowNarrowing": false}, Code: `
func name(v any) string {
	s, _ := v.(string)
	return s
}`, Errors: []goosetest.Error{errAny}},
			{Name: "test file with checkTests", Filename: "x_test.go", Options: map[string]any{"checkTests": true},
				Code: `func helper(v any) {}`, Errors: []goosetest.Error{errAny}},
			{
				Name:    "variadic disallowed",
				Code:    `func logf(args ...any) {}`,
				Options: map[string]any{"allowVariadic": false},
				Errors:  []goosetest.Error{errAny},
			},
			{
				Name:    "method not ignored",
				Code:    "type ID int\n\nfunc (i *ID) Scan(src any) error { return nil }",
				Options: map[string]any{"ignoreMethods": []string{}},
				Errors:  []goosetest.Error{errAny},
			},
		},
	})
}
