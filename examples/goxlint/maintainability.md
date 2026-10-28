# Maintainability rules

The example `maintainability` plugin provides two checks independently of
`anti-slop`. Both belong to the `pedantic` category and are opt-in with goxlint's
default configuration. `goxlint rules` lists their analysis scope and type needs.

Enable them in `.goxlintrc.json`:

```json
{
  "rules": {
    "maintainability/cyclomatic-complexity": ["warn", { "max": 10 }],
    "maintainability/no-duplicate-functions": ["warn", {
      "minStatements": 5,
      "checkTests": false
    }]
  }
}
```

Or use their defaults for one run:

```sh
goxlint -W maintainability/cyclomatic-complexity -W maintainability/no-duplicate-functions ./...
```

`-D` makes findings errors; `-A` disables a rule. File overrides and the usual
warning controls apply. Neither rule offers automatic fixes, even with
`--fix-dangerously`.

## Cyclomatic complexity

`maintainability/cyclomatic-complexity` analyzes functions, methods, and function
literals without requiring type information. The score starts at **1** and adds
one for each:

- `if` (including each `else if`), `for`, and `range`;
- non-default `case` clause in a value or type switch;
- non-default communication clause in a `select`;
- `&&` and `||` operator.

A case with several alternatives counts once. `else`, `default`, and a plain
`switch` or `select` add nothing on their own. Nested closures are scored
separately and do not inflate the enclosing function's score.

```go
func accept(active, ready bool) bool {
    if active && ready { // 1 for if, 1 for &&, plus baseline 1 = 3
        return true
    }
    return false
}
```

The rule reports only when the score **exceeds** `max` (default **10**, minimum
**1**). Functions without bodies are skipped. Test functions are included;
use a file override to exclude them if desired.

## Duplicate functions

`maintainability/no-duplicate-functions` uses the package-analysis API and type
information to compare named functions and methods across files in the same
package. It also detects duplicates within a single file.

```go
// a.go
func existing(x int) int {
    next := x + 1
    return next
}

// b.go
func copied(value int) int {
    result := value + 1
    return result
}
```

With `minStatements: 2`, goxlint reports `copied`, pointing to `existing` in
`a.go`. The diagnostic includes the original function name as a related
location. Code frames, JSON, SARIF, and the go/analysis adapter preserve this
location. The short reference in the message also works in line-based output.

The comparison ignores comments, whitespace, optional trailing
punctuation, and names of local bindings. Type information distinguishes
shadowed variables and preserves the order and identity of parameter uses.
Import aliases and recursive self references are normalized too.

The complete function structure is compared, including its signature. Changes
to operators, literals, external calls, globals, field names, or types keep
functions distinct. This is conservative structural matching, not a proof that
two functions have equivalent behavior. It does not find arbitrary equivalent
algorithms, fragments of functions, or independently compare function literals.
Literal spellings and signature grouping can also keep otherwise equivalent
functions distinct.

`minStatements` defaults to **5** and must be at least **1**. It counts statement
nodes, excluding blocks, empty statements, case/communication labels, and the
contents of nested closures. Empty functions and declarations without bodies
are skipped. `_test.go` files are excluded as both evidence and report targets
unless `checkTests` is enabled for them.

A duplicate group uses the first function in filename/source order as its
reference. If the rule is disabled for a matching non-test file, that file can
still supply the reference implementation; reports appear only in enabled
files, using each file's severity and minimum size. This ordering is stable,
but it does not claim to know which function was written first.

The typed loader respects build tags, excludes generated and ignored files
from analysis, and separates directories and external test packages. It may
load the containing package when given an explicit `.go` file. Only loaded
files can provide comparison evidence. Package checks are skipped for a
package that failed to load or imports one that did; other packages are
still checked.

Both rules work with [suppression directives](../../docs/suppressions.md). Suppression
usage is evaluated after file and package analysis, so a directive that hides a
duplicate finding is counted as used.

For custom binaries, register the plugin alongside any others:

```go
func main() {
    cli.Main(antislop.Plugin, maintainability.Plugin)
}
```

Import `cli` from `github.com/egoisutolabs/goxlint/cli`. The example plugins
are in `github.com/egoisutolabs/goxlint/examples/goxlint/plugins/antislop` and
`github.com/egoisutolabs/goxlint/examples/goxlint/plugins/maintainability`.
See the [example setup](README.md) for build and run commands.
