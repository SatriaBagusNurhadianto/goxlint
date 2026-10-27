// Command goxlint demonstrates a linter built from the goxlint engine and example plugins.
package main

import (
	"github.com/egoisutolabs/goxlint/cli"
	"github.com/egoisutolabs/goxlint/examples/goxlint/plugins/antislop"
	"github.com/egoisutolabs/goxlint/examples/goxlint/plugins/maintainability"
)

func main() { cli.Main(antislop.Plugin, maintainability.Plugin) }
