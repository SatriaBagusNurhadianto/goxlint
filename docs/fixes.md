# Fixes

A rule can attach a fix to a diagnostic. A fix is a set of text edits. The user
applies fixes with a flag.

## Fix levels

Each fix has a level. The level tells how safe the fix is.

| Level | Constant | Use | Flag |
| --- | --- | --- | --- |
| Safe | `goose.FixSafe` | The fix does not change the behavior of the program. | `--fix` |
| Suggestion | `goose.FixSuggestion` | The fix can change the behavior. A person must examine it. | `--fix-suggestions` |
| Dangerous | `goose.FixDangerous` | The fix can break the code. | `--fix-dangerously` |

Each flag also applies the levels above it in the table. For example,
`--fix-suggestions` applies safe fixes and suggestion fixes.

## Add a fix to a rule

1. Set `Meta.Fixable` to the most dangerous level that the rule makes:

   ```go
   Meta: goose.Meta{
   	Name:    "no-todo-context",
   	Fixable: goose.FixSuggestion,
   	// ...
   },
   ```

2. Attach a `goose.Fix` to the diagnostic:

   ```go
   ctx.Report(goose.Diagnostic{
   	Node:      call,
   	MessageID: "todo",
   	Fix: &goose.Fix{
   		Kind:    goose.FixSuggestion,
   		Message: "Use the ctx parameter.",
   		Edits:   []goose.Edit{goose.ReplaceNode(call, "ctx")},
   	},
   })
   ```

If the fix level is more dangerous than `Meta.Fixable`, Goose reports an
internal error for the rule.

## Edits

A `goose.Edit` replaces the text between `Pos` and `End` with `NewText`.

| Helper | Edit |
| --- | --- |
| `goose.ReplaceNode(node, text)` | Replaces the node with `text`. |
| `goose.InsertBefore(pos, text)` | Inserts `text` at `pos`. |
| `goose.Edit{Pos: p, End: e, NewText: ""}` | Removes the text between `p` and `e`. |

A fix can contain more than one edit. The edits of one fix must not overlap.
All edits must be in the current file.

Make sure that the fixed code compiles and has the `gofmt` format. Goose does
not run `gofmt` after it applies fixes.

## How Goose applies fixes

1. Goose lints all files.
2. Goose sorts the fixes of each file by their first edit.
3. Goose applies each fix that is permitted by the flag. If an edit of a fix
   overlaps an edit that Goose applied before, Goose does not apply that fix.
   Goose applies all edits of a fix, or no edits of it.
4. Goose writes the changed files.
5. Goose shows `Applied N fix(es).` on stderr.
6. Goose loads and lints the files again. It shows the diagnostics that remain.

Goose applies fixes one time in each run. A fix that Goose did not apply
because of an overlap can apply in the next run. Run the command again until
no fixes apply.

Goose does not apply fixes to files of a package that did not load correctly.
Read [How Goose works](how-goose-works.md#load-errors).

## Show fixes without applying them

The `json` output format has a `fix` field. The field gives the level of the
fix. Other formats do not show fixes.

## Test fixes

Set `Output` in an invalid test case. `goosetest` applies all fixes and
compares the result with `Output`. Read [Testing rules](testing-rules.md#fixes).
