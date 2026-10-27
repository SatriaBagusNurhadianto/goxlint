# Suppressions

A suppression comment stops Goose from reporting a rule at one location. The
Goose engine processes suppressions. They work with all plugins.

## Directives

| Directive | Scope |
| --- | --- |
| `//goose:ignore <targets> [reason]` | The line of the comment and the next line. |
| `//goose:ignore-file <targets> [reason]` | The full file. |

```go
//goose:ignore acme/no-println The CLI output must go to stdout.
fmt.Println(report)

value := cache[key] //goose:ignore acme/no-map-lookup The key is always present.
```

```go
//goose:ignore-file acme/no-println This file is a command-line program.
package main
```

## Syntax

- The directive must be a `//` line comment. Goose does not read `/* */`
  comments as directives.
- Put one space after `//goose:ignore` or `//goose:ignore-file`.
- The first field after the directive is a list of targets. Separate the
  targets with commas. Do not put spaces in the list.
- The text after the list is the reason. The reason is optional, but always
  write it. A reason tells the next reader why the exception is necessary.

## Targets

| Target | Suppresses |
| --- | --- |
| `acme/no-println` | One rule. |
| `acme` | All rules of the plugin `acme`. |
| `*` | All rules. |

```go
//goose:ignore acme/no-println,acme/max-params Generated adapter.
```

## Scope of a line directive

`//goose:ignore` applies to two physical lines: the line of the comment, and
the line after it. A diagnostic agrees with the directive when its start
position is on one of these lines.

A blank line between the comment and the code stops the directive:

```go
//goose:ignore acme/no-println The reason.

fmt.Println(x) // Goose reports this line.
```

For a statement on more than one line, put the directive directly above the
line where the diagnostic starts.

## Unused suppressions

Goose reports a target that suppresses no diagnostic. The report has the rule
ID `goose/unused-suppression` and points to the target in the comment.

```go
//goose:ignore acme/no-foo,acme/no-bar Existing exception.
foo()
```

If only `acme/no-foo` reports a diagnostic on `foo()`, Goose reports
`acme/no-bar` as unused.

These targets are unused:

- A target for a rule that reported nothing on the lines.
- A target with a spelling error, or a target for a rule that does not exist.
- A target for a rule that is off for the file. Goose examines the rules that
  are enabled for each file, after overrides and command-line flags.
- A redundant target. When two directives or two targets agree with the same
  diagnostic, the first directive in the file gets the credit. In that
  directive, the first agreeing target gets the credit.

One target can suppress more than one diagnostic.

### Set the severity

The default severity is `warn`. Set it in `.gooserc.json`:

```json
{ "reportUnusedSuppressions": "error" }
```

Or set it for one run:

```sh
goose --report-unused-suppressions=error ./...
goose --report-unused-suppressions=off ./...
```

The accepted values are `off`, `warn`, and `error`, and the aliases `allow` and
`deny`. This setting is not a rule. Do not add `goose/unused-suppression` to
`rules`, and do not give it to `-D`.

The warning flags apply to these reports. `--deny-warnings` and
`--max-warnings` can make the run fail. `--quiet` hides the warnings, but Goose
still counts them.

### Properties of unused-suppression reports

- Goose adds the reports after it processes suppressions. Thus a suppression
  cannot hide an unused-suppression report.
- The reports do not have fixes. Remove or correct the target yourself.
- The reports appear in all output formats.

### When Goose does not examine suppressions

An absent diagnostic is not always evidence that a suppression is unused.
Thus Goose does not report unused suppressions for a file in these conditions:

- The package of the file did not load correctly. In typed mode, this also
  applies when the package imports a package that did not load correctly.
  Other packages are still examined.
- A rule failed with an internal error on the file.
- A rule needs type information, and the file has no type information.
- A package rule could not run on the package of the file.

## Suppressions and package rules

Goose processes suppressions after the file phase and the package phase. Thus
a suppression can hide a diagnostic from a package rule. Put the directive on
the line where the package rule reports.

## Recommendations

- Prefer a configuration override when a full directory needs an exception.
  Read [Configuration](configuration.md#overrides).
- Suppress one rule, not a plugin or `*`. A wide target can hide new problems.
- Set `reportUnusedSuppressions` to `error` in CI. Stale suppressions then
  cannot stay in the code.
