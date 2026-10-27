package antislop

import (
	"testing"

	"github.com/egoisutolabs/goose-lint/goosetest"
)

func TestNoPlaceholderComments(t *testing.T) {
	comment := goosetest.Error{MessageID: "placeholderComment"}
	panicErr := goosetest.Error{MessageID: "placeholderPanic"}

	goosetest.Run(t, NoPlaceholderComments, goosetest.Cases{
		Valid: []goosetest.Case{
			{Name: "ordinary comment", Code: `
// Load reads the config file and applies defaults.
func Load() {}`},
			{Name: "tracked todo", Code: `
// TODO(#42): support YAML configs.
func Load() {}`},
			{Name: "design simplification", Code: `
// For simplicity, we treat all methods as exported.
func f() {}`},
			{Name: "editing existing code", Code: `
// Fix rewrites existing code to use the new API.
func Fix() {}`},
			{Name: "advice in docs", Code: `
// In tests you would typically use a fake clock.
func f() {}`},
			{Name: "unreachable panic", Code: `func f() { panic("unreachable") }`},
			{Name: "panic with error", Code: `
import "errors"

func f() { panic(errors.New("boom")) }`},
		},
		Invalid: []goosetest.Case{
			{Name: "real implementation", Code: `
func Charge() error {
	// In a real implementation, this would call the payment API.
	return nil
}`, Errors: []goosetest.Error{{
				Message: `Comment says "in a real implementation", which describes code that is hypothetical or unfinished.`,
				Line:    3,
			}}},
			{Name: "simplified version", Code: `
// This is a simplified version of the real retry logic.
func retry() {}`, Errors: []goosetest.Error{comment}},
			{Name: "production advice", Code: `
func f() {
	// In production you would use a proper secret store.
}`, Errors: []goosetest.Error{comment}},
			{Name: "block comment", Code: `
/* This is a stub until the API is ready. */
func f() {}`, Errors: []goosetest.Error{comment}},
			{Name: "existing code marker", Code: `
func f() {
	// ... existing code ...
}`, Errors: []goosetest.Error{comment}},
			{Name: "not implemented panic", Code: `func f() { panic("not implemented") }`, Errors: []goosetest.Error{{
				Message: `panic("not implemented") stands in for missing code.`,
			}}},
			{Name: "todo panic", Code: `func f() { panic("TODO: handle refunds") }`, Errors: []goosetest.Error{panicErr}},
		},
	})
}

func TestNoRestatingComments(t *testing.T) {
	restates := goosetest.Error{MessageID: "restates"}

	goosetest.Run(t, NoRestatingComments, goosetest.Cases{
		Valid: []goosetest.Case{
			{Name: "explains why", Code: `
func f(counter int) int {
	// Retries share the counter so the backoff grows across attempts.
	counter++
	return counter
}`},
			{Name: "adds information", Code: `
type Body struct{}

func (Body) Close() error { return nil }

func f(b Body) {
	// Close the body so the connection can be reused.
	defer b.Close()
}`},
			{Name: "marker", Code: `
func f(n int) int {
	// NOTE: return n
	return n
}`},
			{Name: "single word", Code: `
func f(n int) int {
	// fallthrough
	return n
}`},
			{Name: "doc comment", Code: `
// Return the result
func Return() {}`},
			{Name: "trailing comment", Code: `
func f(n int) int {
	return n // return n
}`},
			{Name: "comment group", Code: `
func f(n int) int {
	// Callers rely on this exact value.
	// Return n.
	return n
}`},
			{Name: "trailing comment on line above", Code: `
func f(n int) int {
	switch n {
	case 1: // return n
		return n
	}
	return 0
}`},
			{Name: "quotes code", Code: `
func f(xs []int) []int {
	// xs[:0:0]
	return xs[:0:0]
}`},
			{Name: "literal text is not code", Code: `
func check(s string) {}

func f() {
	// wrong header
	check("wrong header")
}`},
			{Name: "words only in if body", Code: `
func f(err error) {
	// cleanup err
	if err == nil {
		cleanup()
	}
}

func cleanup() {}`},
		},
		Invalid: []goosetest.Case{
			{Name: "increment", Code: `
func f(counter int) int {
	// increment counter
	counter++
	return counter
}`, Errors: []goosetest.Error{{Message: "Comment `// increment counter` restates the code below it.", Line: 3}}},
			{Name: "create client", Code: `
type Client struct{}

func NewClient() *Client { return &Client{} }

func f() *Client {
	// Create a new client
	client := NewClient()
	return client
}`, Errors: []goosetest.Error{restates}},
			{Name: "check error", Code: `
func f(err error) error {
	// Check if err is not nil
	if err != nil {
		return err
	}
	return nil
}`, Errors: []goosetest.Error{restates}},
			{Name: "check nil error", Code: `
func f(err error) bool {
	// Check if the error is nil
	if err == nil {
		return true
	}
	return false
}`, Errors: []goosetest.Error{restates}},
			{Name: "return result", Code: `
func f(result int) int {
	// Return the result.
	return result
}`, Errors: []goosetest.Error{restates}},
			{Name: "loop", Code: `
func f(items []string) {
	// Loop through the items
	for _, item := range items {
		_ = item
	}
}`, Errors: []goosetest.Error{restates}},
			{Name: "case clause", Code: `
func f(n int) int {
	switch n {
	case 1:
		// return the count
		return n
	}
	return 0
}`, Errors: []goosetest.Error{restates}},
		},
	})
}
