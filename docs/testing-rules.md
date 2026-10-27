# Testing rules

The `goxlinttest` package tests a file rule with code samples. It has the same
function as the ESLint and oxlint `RuleTester`.

## A test

```go
package rules

import (
	"testing"

	"github.com/egoisutolabs/goxlint/goxlinttest"
)

func TestMaxParams(t *testing.T) {
	goxlinttest.Run(t, MaxParams, goxlinttest.Cases{
		Valid: []goxlinttest.Case{
			{Code: "func f(a, b int) {}"},
			{Code: "func f(a, b, c int) {}", Options: map[string]int{"max": 3}},
		},
		Invalid: []goxlinttest.Case{
			{
				Code:   "func f(a, b, c, d, e int) {}",
				Errors: []goxlinttest.Error{{MessageID: "tooMany", Line: 1, Column: 6}},
			},
		},
	})
}
```

`goxlinttest.Run` makes one subtest for each case. The subtest names are
`valid/0`, `invalid/0`, and so on. Set `Name` to give a case a better name.

## Cases

| Field | Description |
| --- | --- |
| `Name` | Optional. The subtest name. |
| `Code` | The Go source of the case. |
| `Filename` | Optional. The default is `p.go`. Use a `_test.go` name to test the behavior of the rule on test files. |
| `Options` | Optional. goxlint encodes this value as JSON and gives it to the rule as its options. |
| `Errors` | Invalid cases only. The diagnostics that you expect, in the order of their positions. |
| `Output` | Invalid cases only. The code that you expect after goxlint applies all fixes. |

A valid case fails if the rule reports a diagnostic. An invalid case fails if
the number of diagnostics is not the number of `Errors`, or if a diagnostic
does not agree with its `Error`.

## Expected errors

Each `goxlinttest.Error` can examine these fields. goxlint does not examine a field
with the zero value.

| Field | Examines |
| --- | --- |
| `MessageID` | The message agrees with this message template. |
| `Message` | The message is exactly this text. |
| `Line` | The line of the start position, relative to `Code`. |
| `Column` | The column of the start position, in bytes, starting at 1. |

Use `MessageID` for most cases. Use `Message` to examine the values in the
placeholders.

## The code

These conditions apply to the `Code` field:

- If `Code` has no `package` clause, `goxlinttest` adds `package p`. Line numbers
  stay relative to `Code`.
- `goxlinttest` type-checks all cases. The code must compile. Unused variables and
  unused imports are permitted.
- The code can import packages from the standard library.
- Each case is one file.

`goxlinttest` always type-checks the code. The rule gets the type information
only when it has `NeedsTypes`. This is the same behavior as in the linter.

## Fixes

Set `Output` to test a fix. `goxlinttest` applies all fixes of all levels and
compares the result with `Output`. This example uses a rule that removes the
prefix `failed to` from error messages:

```go
{
	Code:   "func f() error { return errors.New(\"failed to open\") }",
	Errors: []goxlinttest.Error{{MessageID: "redundantPrefix"}},
	Output: "func f() error { return errors.New(\"open\") }",
},
```

- If the rule makes a fix and `Output` is empty, the test fails. The failure
  message shows the fixed code.
- If `Output` is set and the rule makes no fix, the test fails.
- `Output` does not contain the `package p` line that `goxlinttest` adds.

## Options

Test the default options and each option. Also test options that are not
correct:

```go
{
	Code:    "func f() {}",
	Options: map[string]any{"typo": 1},
	Errors:  []goxlinttest.Error{{Message: `invalid options for test/rule: json: unknown field "typo"`}},
},
```

In tests, the rule ID is always `test/rule`.

## Test files

Set `Filename` to a name that ends with `_test.go`:

```go
{Code: "func f(v any) {}", Filename: "p_test.go"},
```

## Package rules

`goxlinttest` tests file rules only. It does not run `AnalyzePackage`. Test a
package rule with a Go test that loads a small package and examines the
diagnostics. The example plugin `examples/goxlint/plugins/maintainability` has a
test of this type.

## Recommendations

- Add a valid case for each pattern that the rule must not report. False
  positives are the most frequent rule defect.
- Add an invalid case for each form of the problem.
- Examine `Line` and `Column` in one or more cases. They make sure that the
  report is on the correct node.
- Run the tests with `-race`.
