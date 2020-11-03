// +build nobuild

package main

import (
	"context"

	"code.vegaprotocol.io/vega/config"
	"github.com/jessevdk/go-flags"
)

// ExampleCmd describes a command (in this case `vega example`)
// It holds global variables that its sub-commands will use and the
// sub-commands itself.
type ExampleCmd struct {
	// Global variables
	config.RootPathFlag

	// Subcommands.
	Foo exampleFoo `command:"foo"`
}

var exampleCmd ExampleCmd

// Example is the registration function, the name of this function should
// follow the command name.
// This function is invoked from `Register` in main.go
func Example(ctx context.Context, parser *flags.Parser) error {

	// here we initialize the global exampleCmd with needed default values.
	exampleCmd = ExampleCmd{
		RootPathFlag: config.NewRootPathFlag(),
	}
	_, err := parser.AddCommand("example", "short desc", "long desc", &exampleCmd)
	return err
}

// exampleFoo is an `example` sub-command.
type exampleFoo struct {
}

func (opts *exampleFoo) Execute(args []string) error {
	return nil
}
