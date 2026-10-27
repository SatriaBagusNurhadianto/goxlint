# goxlint

goxlint is an engine to build your own Go linter.

> **Status: v0.0.2, early release.** The API of the `goxlint`, `cli`, and
> `goxlinttest` packages can change in each release before v1.0.0. Read the
> release notes before you update.
>
> goxlint was named goose-lint in v0.0.1. Change imports from
> `github.com/egoisutolabs/goose-lint` to `github.com/egoisutolabs/goxlint`.

You write rules in Go. goxlint loads the code, runs your rules, and shows the
results. goxlint also gives configuration, automatic fixes, suppression comments,
and a test harness for rules. You do not write this infrastructure yourself.

The rule API is similar to the [oxlint](https://oxc.rs/docs/guide/usage/linter)
and ESLint rule API. A rule registers a function for each AST node type that it
examines. When the rule finds a problem, it calls `ctx.Report`.

```go
var NoFoo = &goxlint.Rule{
	Meta: goxlint.Meta{
		Name:     "no-foo",
		Category: goxlint.Suspicious,
		Messages: map[string]string{"foo": "Do not use `{{name}}`."},
	},
	Create: func(ctx *goxlint.Context) goxlint.Visitor {
		return goxlint.Visit(
			goxlint.On(func(id *ast.Ident) {
				if id.Name == "foo" {
					ctx.Report(goxlint.Diagnostic{Node: id, MessageID: "foo", Data: goxlint.Data{"name": id.Name}})
				}
			}),
		)
	},
}
```

## Features

| Feature          | Description                                                                                       |
| ---------------- | ------------------------------------------------------------------------------------------------- |
| Visitor rules    | A rule registers handlers for AST node types. All rules share one walk of each file.              |
| Package rules    | A rule can also examine all files of a package together.                                          |
| Type information | A rule can request `go/types` data. goxlint loads types only when a rule needs them.                |
| Configuration    | A `.goxlintrc.json` file sets severities, rule options, ignore patterns, and per-file overrides.    |
| Fixes            | A rule can attach a fix. Fixes have three levels: safe, suggestion, and dangerous.                |
| Suppressions     | `//goxlint:ignore` comments disable a rule for a line or a file. goxlint reports unused suppressions. |
| Output           | Code frames for people. Also `unix`, `json`, and `github` formats for tools and CI.               |
| Rule tests       | The `goxlinttest` package tests a rule with valid and invalid code samples.                         |

## Quick start

goxlint is a library. You make a small `main` package that registers your
plugins, and that program is your linter.

1. Add goxlint to your module:

    ```sh
    go get github.com/egoisutolabs/goxlint
    ```

2. Make a plugin and the command:

    ```go
    // tools/lint/main.go
    package main

    import (
    	"github.com/egoisutolabs/goxlint/cli"
    	"github.com/egoisutolabs/goxlint"
    )

    var Plugin = goxlint.Plugin{Name: "acme", Rules: []*goxlint.Rule{NoFoo}}

    func main() { cli.Main(Plugin) }
    ```

3. Run your linter:

    ```sh
    go run ./tools/lint ./...
    ```

For the full procedure, read [Getting started](docs/getting-started.md).

## Example linter

The [`examples/goxlint`](examples/goxlint/README.md) directory has a complete linter
that is built with goxlint. It has two example plugins. Use them as a reference
when you write your own rules.

```sh
go run ./examples/goxlint rules
go run ./examples/goxlint ./...
```

## Documentation

| Document                                   | Contents                                                           |
| ------------------------------------------ | ------------------------------------------------------------------ |
| [Getting started](docs/getting-started.md) | Make a linter, write a rule, and run it.                           |
| [Command line](docs/cli.md)                | Commands, flags, output formats, and exit codes.                   |
| [Configuration](docs/configuration.md)     | The `.goxlintrc.json` file.                                          |
| [Writing rules](docs/writing-rules.md)     | The rule API: metadata, visitors, the context, and reports.        |
| [Package rules](docs/package-rules.md)     | Rules that examine all files of a package.                         |
| [Fixes](docs/fixes.md)                     | Fix levels and how goxlint applies fixes.                            |
| [Suppressions](docs/suppressions.md)       | `//goxlint:ignore` comments and unused-suppression reports.          |
| [Testing rules](docs/testing-rules.md)     | The `goxlinttest` package.                                           |
| [How goxlint works](docs/how-goxlint-works.md) | Loading, the walk, the package phase, and the order of operations. |

## Requirements

- Go 1.26 or later.

## Development

```sh
go test -race ./...
go vet ./...
```

## License

goxlint is available under the [MIT License](LICENSE).
