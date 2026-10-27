# Command line

Your linter gets its command line from `cli.Main`. This document uses the name
`goose` for the linter command. Use the name of your command.

## Commands

| Command | Action |
| --- | --- |
| `goose [flags] [patterns]` | Lint the packages or files that the patterns select. |
| `goose rules` | Show a table of all registered rules. |
| `goose init` | Write `.gooserc.json` in the current directory. |
| `goose -h` | Show the usage text. |

### Lint

```sh
goose ./...
goose ./internal/... ./cmd/server
goose main.go handler.go
```

If you do not give a pattern, Goose uses `./...`. You can put flags before or
after the patterns.

| Pattern | Selects |
| --- | --- |
| `./...` | All packages in the current directory and below it. |
| `dir/...` | All packages in `dir` and below it. |
| `dir` | The package in `dir` only. |
| `file.go` | One file. In typed mode, Goose loads the package that contains the file. |

### rules

`goose rules` shows one line for each rule:

| Column | Contents |
| --- | --- |
| `RULE` | The rule ID. |
| `CATEGORY` | The category of the rule. |
| `SCOPE` | `file`, `package`, or `file+package`. |
| `FIX` | The most dangerous fix level of the rule. Empty when the rule has no fixes. |
| `TYPES` | `yes` when the rule needs type information. |
| `DESCRIPTION` | The description from the rule metadata. |

### init

`goose init` writes `.gooserc.json` in the current directory. The file enables
the `correctness` category with `warn`, and sets each registered rule to
`error`. If the file exists, the command does not change it and exits with `2`.

## Flags

### Configuration

| Flag | Value | Action |
| --- | --- | --- |
| `-c` | path | Use this configuration file. If you do not use `-c`, Goose finds the nearest `.gooserc.json`. |
| `-D` | rule or category | Set the severity to `error`. You can use the flag more than one time. |
| `-W` | rule or category | Set the severity to `warn`. You can use the flag more than one time. |
| `-A` | rule or category | Set the severity to `off`. You can use the flag more than one time. |
| `--report-unused-suppressions` | `off`, `warn`, `error` | Set the severity of unused-suppression reports. |
| `--tags` | list | Use these build tags in typed mode, for example `--tags integration,e2e`. |

The `-D`, `-W`, and `-A` flags apply in the order that you give them. They
change the configuration before Goose resolves it. A later flag for the same
rule or category replaces an earlier flag. A rule setting is more specific than
a category setting. Thus `-A style -D acme/no-foo` enables `acme/no-foo` also
when it is a style rule.

These flags do not change the options of a rule. File overrides in the
configuration still apply after the flags.

### Fixes

| Flag | Applies |
| --- | --- |
| `--fix` | Safe fixes. |
| `--fix-suggestions` | Safe fixes and suggestion fixes. |
| `--fix-dangerously` | All fixes, also dangerous fixes. |

For more information, read [Fixes](fixes.md).

### Output and exit status

| Flag | Value | Action |
| --- | --- | --- |
| `--format` | `default`, `unix`, `json`, `github` | Select the output format. The default is `default`. |
| `--quiet` | | Do not show warnings. Goose still counts them. |
| `--deny-warnings` | | Exit with `1` when there are warnings. |
| `--max-warnings` | number | Exit with `1` when the number of warnings is more than this number. |

## Output formats

### default

The `default` format shows a code frame for each diagnostic. Then it shows a
summary:

```text
  ⚠ acme(no-foo): Do not use `foo`.
   ╭─[api/client.go:12:6]
11 │
12 │ func foo() {}
   ·      ───
13 │
   ╰────
  help: Rename the function.

Found 1 warning and 0 errors.
Finished in 85ms on 12 files with 4 rules using 8 threads (syntax mode).
```

An error has the `×` symbol. A warning has the `⚠` symbol. Goose uses colors
only when the output is a terminal and the `NO_COLOR` environment variable is
not set.

### unix

One line for each diagnostic. Many editors can read this format.

```text
api/client.go:12:6: Do not use `foo`. [Warning/acme/no-foo]
```

### json

One JSON array. Use this format for scripts and coding agents.

```json
[
  {
    "rule": "acme/no-foo",
    "severity": "warning",
    "file": "api/client.go",
    "line": 12,
    "column": 6,
    "endLine": 12,
    "endColumn": 9,
    "message": "Do not use `foo`.",
    "help": "Rename the function.",
    "fix": "safe"
  }
]
```

The `severity` field is `warning` or `error`. The `help` and `fix` fields are
not present when they are empty. The `fix` field gives the fix level.

### github

GitHub Actions workflow commands. GitHub shows each diagnostic as an annotation
on the pull request.

```text
::warning file=api/client.go,line=12,endLine=12,col=6,endColumn=9,title=acme/no-foo::Do not use `foo`.
```

All formats show paths relative to the current directory when possible. Line
and column numbers start at 1. Columns count bytes.

## Exit codes

| Code | Condition |
| --- | --- |
| `0` | No errors. The warning limits are not exceeded. |
| `1` | One or more errors, or a warning limit is exceeded. |
| `2` | A usage error, a configuration error, or a load error. |

A load error occurs when a file cannot be read or parsed, or when a package
has type errors. Goose shows each load error on stderr with the prefix
`goose: load:`. Goose still lints the packages that loaded correctly, and it
shows their diagnostics. For more information, read
[How Goose works](how-goose-works.md#load-errors).

## Use in CI

This GitHub Actions step shows diagnostics as annotations. The step fails when
there are errors.

```yaml
- name: Lint
  run: go run ./tools/lint --format github ./...
```

To make warnings fail the step, add `--deny-warnings`.
