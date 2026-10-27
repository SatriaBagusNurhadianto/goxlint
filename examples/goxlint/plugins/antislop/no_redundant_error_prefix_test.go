package antislop

import (
	"testing"

	"github.com/egoisutolabs/goxlint/goxlinttest"
)

func TestNoRedundantErrorPrefix(t *testing.T) {
	goxlinttest.Run(t, NoRedundantErrorPrefix, goxlinttest.Cases{
		Valid: []goxlinttest.Case{
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
		Invalid: []goxlinttest.Case{
			{
				Name: "fmt.Errorf",
				Code: `
import "fmt"

func f(err error) error { return fmt.Errorf("failed to open config: %w", err) }`,
				Errors: []goxlinttest.Error{{Message: `Error message starts with "failed to"; every error already says something failed.`, Line: 4}},
				Output: `
import "fmt"

func f(err error) error { return fmt.Errorf("open config: %w", err) }`,
			},
			{
				Name: "errors.New capitalized",
				Code: `
import "errors"

var ErrMissing = errors.New("Unable to find user")`,
				Errors: []goxlinttest.Error{{MessageID: "redundantPrefix"}},
				Output: `
import "errors"

var ErrMissing = errors.New("find user")`,
			},
			{
				Name:   "raw string",
				Code:   "import \"errors\"\n\nvar err = errors.New(`could not parse`)",
				Errors: []goxlinttest.Error{{MessageID: "redundantPrefix"}},
				Output: "import \"errors\"\n\nvar err = errors.New(`parse`)",
			},
		},
	})
}
