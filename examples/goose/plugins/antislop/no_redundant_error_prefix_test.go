package antislop

import (
	"testing"

	"github.com/egoisutolabs/goose-lint/goosetest"
)

func TestNoRedundantErrorPrefix(t *testing.T) {
	goosetest.Run(t, NoRedundantErrorPrefix, goosetest.Cases{
		Valid: []goosetest.Case{
			{Name: "operation", Code: `
import "fmt"

func f(err error) error { return fmt.Errorf("open config: %w", err) }`},
			{Name: "other function", Code: `
import "fmt"

var s = fmt.Sprintf("failed to %s", "x")`},
			{Name: "prefix only", Code: `
import "errors"

var err = errors.New("failed to ")`},
			{Name: "non-literal", Code: `
import "errors"

const msg = "failed to open"

var err = errors.New(msg)`},
		},
		Invalid: []goosetest.Case{
			{
				Name: "fmt.Errorf",
				Code: `
import "fmt"

func f(err error) error { return fmt.Errorf("failed to open config: %w", err) }`,
				Errors: []goosetest.Error{{Message: `Error message starts with "failed to"; every error already says something failed.`, Line: 4}},
				Output: `
import "fmt"

func f(err error) error { return fmt.Errorf("open config: %w", err) }`,
			},
			{
				Name: "errors.New capitalized",
				Code: `
import "errors"

var ErrMissing = errors.New("Unable to find user")`,
				Errors: []goosetest.Error{{MessageID: "redundantPrefix"}},
				Output: `
import "errors"

var ErrMissing = errors.New("find user")`,
			},
			{
				Name:   "raw string",
				Code:   "import \"errors\"\n\nvar err = errors.New(`could not parse`)",
				Errors: []goosetest.Error{{MessageID: "redundantPrefix"}},
				Output: "import \"errors\"\n\nvar err = errors.New(`parse`)",
			},
		},
	})
}
