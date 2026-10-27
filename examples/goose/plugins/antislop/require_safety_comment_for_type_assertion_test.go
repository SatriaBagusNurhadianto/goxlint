package antislop

import (
	"testing"

	"github.com/egoisutolabs/goose-lint/goosetest"
)

func TestRequireSafetyCommentForTypeAssertion(t *testing.T) {
	missing := goosetest.Error{MessageID: "missingComment"}

	goosetest.Run(t, RequireSafetyCommentForTypeAssertion, goosetest.Cases{
		Valid: []goosetest.Case{
			{Name: "checked assignment", Code: `
func f(v any) {
	s, ok := v.(string)
	_, _ = s, ok
}`},
			{Name: "checked var", Code: `
func f(v any) {
	var s, ok = v.(string)
	_, _ = s, ok
}`},
			{Name: "checked if init", Code: `
func f(v any) {
	if s, ok := v.(string); ok {
		_ = s
	}
}`},
			{Name: "type switch", Code: `
func f(v any) {
	switch v.(type) {
	}
}`},
			{Name: "comment above", Code: `
func f(v any) string {
	// SAFETY: callers only pass strings; see parseArgs.
	return v.(string)
}`},
			{Name: "trailing comment", Code: `
func f(v any) string {
	return v.(string) // SAFETY: callers only pass strings.
}`},
			{Name: "comment above multi-line statement", Code: `
func g(a, b string) string { return a + b }

func f(v any) string {
	// SAFETY: callers only pass strings.
	return g(
		"x",
		v.(string),
	)
}`},
			{Name: "test file", Filename: "x_test.go", Code: `
func f(v any) string { return v.(string) }`},
			{Name: "custom marker", Options: map[string]any{"markers": []string{"INVARIANT"}}, Code: `
func f(v any) string {
	// INVARIANT: v is always a string here.
	return v.(string)
}`},
		},
		Invalid: []goosetest.Case{
			{Name: "unchecked", Code: `
func f(v any) string {
	return v.(string)
}`, Errors: []goosetest.Error{{
				Message: "Unchecked type assertion `v.(string)` panics if the value is not `string`.",
				Line:    3, Column: 9,
			}}},
			{Name: "empty justification", Code: `
func f(v any) string {
	// SAFETY:
	return v.(string)
}`, Errors: []goosetest.Error{missing}},
			{Name: "unrelated comment", Code: `
func f(v any) string {
	// convert to string
	return v.(string)
}`, Errors: []goosetest.Error{missing}},
			{Name: "detached by blank line", Code: `
func f(v any) string {
	// SAFETY: callers only pass strings.

	return v.(string)
}`, Errors: []goosetest.Error{missing}},
			{Name: "comment on function does not cover body", Code: `
// SAFETY: callers only pass strings.
func f(v any) string {
	return v.(string)
}`, Errors: []goosetest.Error{missing}},
			{Name: "wrong marker", Options: map[string]any{"markers": []string{"INVARIANT"}}, Code: `
func f(v any) string {
	// SAFETY: callers only pass strings.
	return v.(string)
}`, Errors: []goosetest.Error{missing}},
			{Name: "test file with checkTests", Filename: "x_test.go", Options: map[string]any{"checkTests": true},
				Code: `func f(v any) string { return v.(string) }`, Errors: []goosetest.Error{missing}},
			{Name: "single-value assignment", Code: `
func f(v any) {
	s := v.(string)
	_ = s
}`, Errors: []goosetest.Error{missing}},
		},
	})
}
