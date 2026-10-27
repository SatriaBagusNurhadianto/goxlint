# Package rules

A file rule examines one file. A package rule examines all files of a package
together. Use a package rule when the problem is across files, for example a
function that is a copy of a function in a different file.

A rule can have `Create`, `AnalyzePackage`, or both. Package rules use the same
metadata, message templates, options, and diagnostics as file rules. Read
[Writing rules](writing-rules.md) first.

## A package rule

This rule reports a package that has too many files.

```go
var LargePackage = &goose.Rule{
	Meta: goose.Meta{
		Name:        "large-package",
		Description: "Report packages with too many files.",
		Category:    goose.Pedantic,
		Messages: map[string]string{
			"large": "Package `{{name}}` has {{count}} files. The maximum is {{max}}.",
		},
	},
	AnalyzePackage: func(pkg *goose.PackageContext) {
		for _, file := range pkg.Files {
			if !file.Enabled {
				continue
			}
			opts := struct {
				MaxFiles int `json:"maxFiles"`
			}{MaxFiles: 20}
			if !file.Options(&opts) {
				continue
			}
			if len(pkg.Files) > opts.MaxFiles {
				file.Report(goose.Diagnostic{
					Node:      file.File.Name,
					MessageID: "large",
					Data: goose.Data{
						"name":  pkg.Name,
						"count": strconv.Itoa(len(pkg.Files)),
						"max":   strconv.Itoa(opts.MaxFiles),
					},
					Help: "Move different responsibilities to different packages.",
				})
			}
		}
	},
}
```

Register a package rule in a plugin, the same as a file rule. The rule ID is
`<plugin>/<rule>`.

## The package context

`AnalyzePackage` gets a `*goose.PackageContext`:

| Field | Description |
| --- | --- |
| `Name` | The Go package name from the `package` clause. |
| `Files` | One `*goose.Context` for each file. The list is in file name order. |

Each file context has the same fields and methods as in a file rule. Read
[The context](writing-rules.md#the-context).

## Which files are in a package

Goose puts files in the same group when they are in the same directory and have
the same package name.

- An external test package (`p_test`) is a different group from package `p`.
- Internal test files (`package p` in a `_test.go` file) are in the same group
  as the production files.
- Goose does not include generated files or ignored files.
- In typed mode, Goose includes each file one time. It does not include the
  copies from the test variants of the package.

The group contains only the files that Goose loaded. In syntax mode, a file
pattern such as `goose a.go` can select only part of a package. If the rule
must have the complete package, set `Meta.NeedsTypes: true`. Goose then uses
typed mode. Typed mode loads full packages, and it obeys build tags and
`--tags`.

## Enabled files and options

Each file has its own configuration, because overrides can change the rule for
some files.

- `file.Enabled` is `false` when the rule is off for that file. The file is
  still in `Files`. You can use it as evidence, for example as the original of
  a copied function.
- Goose ignores reports from a file where `Enabled` is `false`.
- `file.Options` decodes the options for that file, with overrides. A disabled
  file has no options.
- A report uses the severity of the file that contains the report.

The engine does not divide a package into groups with the same options. The
rule must decide how the options of different files apply.

## Reports

Report through the context of the file that contains the node. The diagnostic
and its fix must be in that file. If you use a node from a different file,
Goose reports an internal error for the rule.

To refer to a different file, put its location in the message or in the help
text:

```go
file.Report(goose.Diagnostic{
	Node:      fn.Name,
	MessageID: "duplicate",
	Data:      goose.Data{"name": fn.Name.Name, "location": "a.go:12:6"},
})
```

Report before `AnalyzePackage` returns. Goose ignores reports after that.

## Type information

Goose gives type information only when the rule has `NeedsTypes`. Use the `Pkg`
and `TypesInfo` fields of each file together. Test files can come from a test
variant of the package. That variant has different type-checking data.

Goose does not run a package rule with `NeedsTypes` if one file of the package
has no type information.

## When Goose does not run a package rule

Goose does not run a package rule on a package in these conditions:

- The package did not load correctly.
- In typed mode, the package imports a package that did not load correctly.
- The rule needs type information, and one or more files have none.

Goose also does not report unused suppressions for that package. Other packages
are not affected.

## Rules for package rules

- Keep all state in the `AnalyzePackage` call. Goose can analyze different
  packages at the same time.
- Do not change syntax trees, source, type information, or metadata.
- `Parent` and `Ancestors` are empty in a package rule. Walk the AST yourself,
  for example with `ast.Inspect`.

## Panics

If `AnalyzePackage` causes a panic, Goose reports an internal error for the
rule. Goose discards the reports that the call made before the panic. It does
not report unused suppressions for that package. Other rules continue to run.
Goose keeps the file-rule diagnostics.

## Order of operations

1. Goose finds the rules for each file and runs the file rules.
2. Goose puts the files in package groups. For each group, Goose runs each
   enabled package rule one time, in rule ID order.
3. Goose adds the diagnostics of the two phases together.
4. Goose applies suppressions one time, and reports unused suppressions.
5. Goose sorts the diagnostics by position.

## Test a package rule

`goosetest` does not run package rules. Read
[Testing rules](testing-rules.md#package-rules).

## Example

The example plugin [`maintainability`](../examples/goose/maintainability.md)
has a package rule, `no-duplicate-functions`. It compares functions across the
files of a package with type information.
