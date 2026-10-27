// Package maintainability provides configurable complexity and duplication checks.
package maintainability

import "github.com/egoisutolabs/goose-lint/goose"

// Plugin registers the maintainability rules. They are pedantic checks, so they
// are opt-in unless a repository enables the pedantic category.
var Plugin = goose.Plugin{
	Name:  "maintainability",
	Rules: []*goose.Rule{CyclomaticComplexity, NoDuplicateFunctions},
}
