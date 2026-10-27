package antislop

import (
	"testing"

	"github.com/egoisutolabs/goxlint/goxlinttest"
)

func TestNoErrorStringMatching(t *testing.T) {
	errText := goxlinttest.Error{MessageID: "errorText"}

	goxlinttest.Run(t, NoErrorStringMatching, goxlinttest.Cases{
		Valid: []goxlinttest.Case{
			{Name: "errors.Is", Code: `
import (
	"errors"
	"io"
)

func f(err error) bool { return errors.Is(err, io.EOF) }`},
			{Name: "logging the text", Code: `
import "fmt"

func f(err error) { fmt.Println(err.Error()) }`},
			{Name: "Error on a non-error", Code: `
import "strings"

type report struct{}

func (report) Error(s string) string { return s }

func f(r report) bool { return strings.Contains(r.Error("x"), "x") }`},
			{Name: "test file", Filename: "x_test.go", Code: `
func f(err error) bool { return err.Error() == "EOF" }`},
			{Name: "non-error type with Error method", Code: `
type T struct{}

func (T) Error() int { return 0 }

func f(t T) bool { return t.Error() == 0 }`},
		},
		Invalid: []goxlinttest.Case{
			{Name: "strings.Contains", Code: `
import "strings"

func f(err error) bool { return strings.Contains(err.Error(), "not found") }`,
				Errors: []goxlinttest.Error{{Message: "Matching on the text of `err.Error()` breaks when the message changes or gains context.", Line: 4}}},
			{Name: "HasPrefix", Code: `
import "strings"

func f(err error) bool { return strings.HasPrefix(err.Error(), "timeout") }`, Errors: []goxlinttest.Error{errText}},
			{Name: "test file with checkTests", Filename: "x_test.go", Options: map[string]any{"checkTests": true},
				Code: `func f(err error) bool { return err.Error() == "EOF" }`, Errors: []goxlinttest.Error{errText}},
			{Name: "equality", Code: `func f(err error) bool { return err.Error() == "EOF" }`, Errors: []goxlinttest.Error{errText}},
			{Name: "inequality reversed", Code: `func f(err error) bool { return "EOF" != err.Error() }`, Errors: []goxlinttest.Error{errText}},
			{Name: "switch", Code: `
func f(err error) int {
	switch err.Error() {
	case "EOF":
		return 1
	}
	return 0
}`, Errors: []goxlinttest.Error{{MessageID: "errorText", Line: 3}}},
			{Name: "concrete error type", Code: `
import "os"

func f(err *os.PathError) bool { return err.Error() == "x" }`, Errors: []goxlinttest.Error{errText}},
		},
	})
}
