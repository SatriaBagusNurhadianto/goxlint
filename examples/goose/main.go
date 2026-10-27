// Command goose demonstrates a linter built from the Goose engine and example plugins.
package main

import (
	"github.com/egoisutolabs/goose-lint/cli"
	"github.com/egoisutolabs/goose-lint/examples/goose/plugins/antislop"
	"github.com/egoisutolabs/goose-lint/examples/goose/plugins/maintainability"
)

func main() { cli.Main(antislop.Plugin, maintainability.Plugin) }
