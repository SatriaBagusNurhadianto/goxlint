package antislop

import (
	"testing"

	"github.com/egoisutolabs/goose-lint/goosetest"
)

func TestNoSwallowedError(t *testing.T) {
	swallowed := goosetest.Error{MessageID: "swallowed"}

	goosetest.Run(t, NoSwallowedError, goosetest.Cases{
		Valid: []goosetest.Case{
			{Name: "returned", Code: `
func f(err error) (int, error) {
	if err != nil {
		return 0, err
	}
	return 1, nil
}`},
			{Name: "wrapped", Code: `
import "fmt"

func f(err error) error {
	if err != nil {
		return fmt.Errorf("load: %w", err)
	}
	return nil
}`},
			{Name: "justified", Code: `
func f(err error) error {
	if err != nil {
		// A missing cache entry is not a failure; fall back to the source.
		return nil
	}
	return nil
}`},
			{Name: "no error result", Code: `
func f(err error) bool {
	if err != nil {
		return false
	}
	return true
}`},
			{Name: "uses err in branch", Code: `
import "errors"

var errSkip = errors.New("skip")

func f(err error) error {
	if err != nil {
		if errors.Is(err, errSkip) {
			return nil
		}
		return err
	}
	return nil
}`},
		},
		Invalid: []goosetest.Case{
			{Name: "nil error", Code: `
func f(err error) error {
	if err != nil {
		return nil
	}
	return nil
}`, Errors: []goosetest.Error{{
				Message: "`err` is checked and then dropped: this branch returns a nil error.",
				Line:    4,
			}}},
			{Name: "multiple results", Code: `
import "strconv"

func parse(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, nil
	}
	return n, nil
}`, Errors: []goosetest.Error{swallowed}},
			{Name: "nil on left", Code: `
func f(err error) error {
	if nil != err {
		return nil
	}
	return nil
}`, Errors: []goosetest.Error{swallowed}},
			{Name: "closure", Code: `
var f = func(err error) error {
	if err != nil {
		return nil
	}
	return nil
}`, Errors: []goosetest.Error{swallowed}},
		},
	})
}

func TestNoLogAndReturnError(t *testing.T) {
	logged := goosetest.Error{MessageID: "logAndReturn"}

	goosetest.Run(t, NoLogAndReturnError, goosetest.Cases{
		Valid: []goosetest.Case{
			{Name: "return only", Code: `
import "fmt"

func f(err error) error {
	if err != nil {
		return fmt.Errorf("load: %w", err)
	}
	return nil
}`},
			{Name: "log only", Code: `
import "log"

func f(err error) {
	if err != nil {
		log.Printf("load: %v", err)
		return
	}
}`},
			{Name: "log unrelated", Code: `
import "log"

func f(err error) error {
	if err != nil {
		log.Print("retrying")
		return err
	}
	return nil
}`},
			{Name: "test report", Code: `
import "testing"

func check(t *testing.T, err error) error {
	if err != nil {
		t.Errorf("load: %v", err)
		return err
	}
	return nil
}`},
		},
		Invalid: []goosetest.Case{
			{Name: "log package", Code: `
import "log"

func f(err error) error {
	if err != nil {
		log.Printf("failed: %v", err)
		return err
	}
	return nil
}`, Errors: []goosetest.Error{{
				Message: "`err` is logged and then returned, so it will be reported twice.",
				Line:    6,
			}}},
			{Name: "slog and wrap", Code: `
import (
	"fmt"
	"log/slog"
)

func f(err error) error {
	if err != nil {
		slog.Error("load failed", "err", err)
		return fmt.Errorf("load: %w", err)
	}
	return nil
}`, Errors: []goosetest.Error{logged}},
			{Name: "logger method", Code: `
import "log/slog"

func f(logger *slog.Logger, err error) error {
	if err != nil {
		logger.Error("load failed", "err", err)
		return err
	}
	return nil
}`, Errors: []goosetest.Error{logged}},
		},
	})
}

func TestNoBlankError(t *testing.T) {
	blank := goosetest.Error{MessageID: "blankError"}

	goosetest.Run(t, NoBlankError, goosetest.Cases{
		Valid: []goosetest.Case{
			{Name: "handled", Code: `
import "strconv"

func f(s string) (int, error) {
	return strconv.Atoi(s)
}`},
			{Name: "comment above", Code: `
import "os"

func f() {
	// Best effort: the temp dir is cleaned up by the OS anyway.
	_ = os.Remove("tmp")
}`},
			{Name: "trailing comment", Code: `
import "os"

func f() {
	_ = os.Remove("tmp") // best effort, see cleanup()
}`},
			{Name: "test file", Filename: "x_test.go", Code: `
import "os"

func f() {
	_ = os.Remove("tmp")
}`},
			{Name: "non-error blank", Code: `
import "strconv"

func f(s string) error {
	_, err := strconv.Atoi(s)
	return err
}`},
		},
		Invalid: []goosetest.Case{
			{Name: "single", Code: `
import "os"

func f() {
	_ = os.Remove("tmp")
}`, Errors: []goosetest.Error{{
				Message: "Error from `os.Remove(\"tmp\")` is discarded with `_`.",
				Line:    5,
			}}},
			{Name: "tuple", Code: `
import "strconv"

func f(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}`, Errors: []goosetest.Error{blank}},
			{Name: "var", Code: `
import "os"

var _ = os.Remove("tmp")`, Errors: []goosetest.Error{blank}},
			{Name: "test file with checkTests", Filename: "x_test.go", Options: map[string]any{"checkTests": true}, Code: `
import "os"

func f() {
	_ = os.Remove("tmp")
}`, Errors: []goosetest.Error{blank}},
			{Name: "unrelated comment two lines up", Code: `
import "os"

func f() {
	// Clean up.

	_ = os.Remove("tmp")
}`, Errors: []goosetest.Error{blank}},
		},
	})
}
