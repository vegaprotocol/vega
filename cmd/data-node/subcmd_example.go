// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

//go:build nobuild
// +build nobuild

package main

import (
	"context"

	"code.vegaprotocol.io/data-node/datanode/config"
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
type exampleFoo struct {
}

func (opts *exampleFoo) Execute(args []string) error {
	return nil
}
