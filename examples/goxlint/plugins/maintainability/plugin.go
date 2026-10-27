// Package maintainability provides configurable complexity and duplication checks.
package maintainability

import "github.com/egoisutolabs/goxlint"

// Plugin registers the maintainability rules. They are pedantic checks, so they
// are opt-in unless a repository enables the pedantic category.
var Plugin = goxlint.Plugin{
	Name:  "maintainability",
	Rules: []*goxlint.Rule{CyclomaticComplexity, NoDuplicateFunctions},
}
