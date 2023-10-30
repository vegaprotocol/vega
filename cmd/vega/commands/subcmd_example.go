// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

//go:build nobuild
// +build nobuild

package commands

import (
	"context"

	"code.vegaprotocol.io/vega/core/config"
	"github.com/jessevdk/go-flags"
)

// ExampleCmd describes a command (in this case `vega example`)
// It holds global variables that its sub-commands will use and the
// sub-commands itself.
type ExampleCmd struct {
	// Global variables
	config.VegaHomeFlag

	// Subcommands.
	Foo exampleFoo `command:"foo"`
}

var exampleCmd ExampleCmd

// Example is the registration function, the name of this function should
// follow the command name.
// This function is invoked from `Register` in main.go
func Example(ctx context.Context, parser *flags.Parser) error {
	// here we initialize the global exampleCmd with needed default values.
	exampleCmd = ExampleCmd{}
	_, err := parser.AddCommand("example", "short desc", "long desc", &exampleCmd)
	return err
}

// exampleFoo is an `example` sub-command.
type exampleFoo struct{}

func (opts *exampleFoo) Execute(args []string) error {
	return nil
}
