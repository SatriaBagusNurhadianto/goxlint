package antislop

import (
	"testing"

	"github.com/egoisutolabs/goose-lint/goosetest"
)

func TestNoAnyReturns(t *testing.T) {
	anyReturn := goosetest.Error{MessageID: "anyReturn"}

	goosetest.Run(t, NoAnyReturns, goosetest.Cases{
		Valid: []goosetest.Case{
			{Name: "concrete", Code: `func f() (int, error) { return 0, nil }`},
			{Name: "generic", Code: `func first[T any](xs []T) T { return xs[0] }`},
			{Name: "driver Valuer", Code: `
type ID int

func (i ID) Value() (any, error) { return int64(i), nil }`},
			{Name: "callback argument", Code: `
func apply(fn func() any) {}

func g() { apply(func() any { return 1 }) }`},
		},
		Invalid: []goosetest.Case{
			{Name: "any", Code: `func Load() any { return nil }`, Errors: []goosetest.Error{{
				Message: "`Load` returns `any`, so every caller has to type-assert the result.",
			}}},
			{Name: "with error", Code: `func Load() (interface{}, error) { return nil, nil }`, Errors: []goosetest.Error{anyReturn}},
			{Name: "named results", Code: `func Load() (v any, err error) { return }`, Errors: []goosetest.Error{anyReturn}},
			{Name: "method not ignored", Code: `
type C struct{}

func (C) Get(key string) any { return nil }`, Errors: []goosetest.Error{anyReturn}},
		},
	})
}

func TestNoAnyMapValues(t *testing.T) {
	anyMap := goosetest.Error{MessageID: "anyMap"}

	goosetest.Run(t, NoAnyMapValues, goosetest.Cases{
		Valid: []goosetest.Case{
			{Name: "concrete values", Code: `var m map[string]int`},
			{Name: "any keys", Code: `var m map[any]bool`},
			{Name: "generic", Code: `func keys[V any](m map[string]V) {}`},
		},
		Invalid: []goosetest.Case{
			{Name: "var", Code: `var m map[string]any`, Errors: []goosetest.Error{{
				Message: "`map[string]any` discards the value types; every read needs a type assertion.",
			}}},
			{Name: "literal", Code: `var m = map[string]interface{}{"a": 1}`, Errors: []goosetest.Error{anyMap}},
			{Name: "param", Code: `func f(meta map[string]any) {}`, Errors: []goosetest.Error{anyMap}},
			{Name: "nested", Code: `var m map[string]map[string]any`, Errors: []goosetest.Error{anyMap}},
		},
	})
}

func TestNoReflectDynamicAccess(t *testing.T) {
	dynamic := goosetest.Error{MessageID: "dynamic"}

	goosetest.Run(t, NoReflectDynamicAccess, goosetest.Cases{
		Valid: []goosetest.Case{
			{Name: "kind check", Code: `
import "reflect"

func f(v any) bool { return reflect.ValueOf(v).Kind() == reflect.Struct }`},
			{Name: "unrelated Call", Code: `
type R struct{}

func (R) Call() {}

func f(r R) { r.Call() }`},
		},
		Invalid: []goosetest.Case{
			{Name: "FieldByName", Code: `
import "reflect"

func f(v any) reflect.Value { return reflect.ValueOf(v).FieldByName("Name") }`, Errors: []goosetest.Error{{
				Message: "`reflect.Value.FieldByName` resolves members at runtime, where the compiler cannot check them.",
			}}},
			{Name: "MethodByName Call", Code: `
import "reflect"

func f(v any) { reflect.ValueOf(v).MethodByName("Run").Call(nil) }`, Errors: []goosetest.Error{dynamic, dynamic}},
			{Name: "Type FieldByName", Code: `
import "reflect"

func f(v any) bool {
	_, ok := reflect.TypeOf(v).FieldByName("ID")
	return ok
}`, Errors: []goosetest.Error{{Message: "`reflect.Type.FieldByName` resolves members at runtime, where the compiler cannot check them."}}},
		},
	})
}

func TestNoGenericPackageNames(t *testing.T) {
	goosetest.Run(t, NoGenericPackageNames, goosetest.Cases{
		Valid: []goosetest.Case{
			{Name: "specific", Code: "package retry"},
			{Name: "custom list", Code: "package utils", Options: map[string]any{"names": []string{"misc"}}},
		},
		Invalid: []goosetest.Case{
			{Name: "utils", Code: "package utils", Errors: []goosetest.Error{{
				Message: "Package name `utils` says nothing about what the package provides.",
				Line:    1, Column: 9,
			}}},
			{Name: "external test package", Code: "package helpers_test", Errors: []goosetest.Error{{MessageID: "generic"}}},
			{Name: "invalid options", Code: "package utils", Options: map[string]any{"names": 3}, Errors: []goosetest.Error{{
				Message: "invalid options for test/rule: json: cannot unmarshal number into Go struct field .names of type []string",
			}}},
			{Name: "unknown option", Code: "package utils", Options: map[string]any{"nmaes": []string{}}, Errors: []goosetest.Error{{
				Message: `invalid options for test/rule: json: unknown field "nmaes"`,
			}}},
		},
	})
}
