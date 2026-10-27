# Example Goose linter

This example builds a Go linter from the Goose engine and two example plugins:

- `plugins/antislop`: opinionated rules for error handling, types, comments, and runtime behavior.
- `plugins/maintainability`: cyclomatic complexity and duplicate-function detection.

`main.go` registers the plugins with `cli.Main`. Both plugins use Goose's public
rule-authoring API; their namespaces remain `anti-slop` and `maintainability`.

From the repository root:

```sh
go run ./examples/goose rules
go run ./examples/goose ./...
```

To install the example as a `goose` command:

```sh
go install ./examples/goose
```

The repository's `.gooserc.json` configures self-linting with the example
anti-slop rules. Maintainability rules are opt-in:

```sh
go run ./examples/goose -W maintainability/cyclomatic-complexity \
  -W maintainability/no-duplicate-functions ./...
```

See [maintainability rules](maintainability.md) for options and behavior, and
[suppressions](../../docs/suppressions.md) for directive syntax.

To build your own linter, create a command that calls `cli.Main` with your
plugins. The example plugins are implementations to learn from or adapt;
the engine and CLI runner do not depend on them. The
[package-rule guide](../../docs/package-rules.md) covers cross-file analyzers.

The example stays in the repository's Go module, so normal checks include it:

```sh
go test ./...
go test ./examples/goose/...
```

The CLI integration tests live in `main_test.go`; each plugin's unit tests live
beside its implementation.
