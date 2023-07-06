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

package commands

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/blockexplorer/config"
	"code.vegaprotocol.io/vega/paths"
	"github.com/jessevdk/go-flags"
)

type InitCmd struct {
	config.VegaHomeFlag

	Force bool `description:"Erase exiting blockexplorer configuration at the specified path" long:"force" short:"f"`
}

func (opts *InitCmd) Execute(_ []string) error {
	paths := paths.New(opts.VegaHome)
	loader, err := config.NewLoader(paths)
	if err != nil {
		return fmt.Errorf("couldn't initialise configuration loader: %w", err)
	}

	configExists, err := loader.ConfigExists()
	if err != nil {
		return fmt.Errorf("couldn't verify configuration presence: %w", err)
	}

	if configExists && !opts.Force {
		return fmt.Errorf("configuration already exists at `%s` please remove it first or re-run using -f", loader.ConfigFilePath())
	}

	config := config.NewDefaultConfig()
	loader.Save(&config)
	fmt.Println("wrote config file: ", loader.ConfigFilePath())
	return nil
}

var initCmd InitCmd

func Init(ctx context.Context, parser *flags.Parser) error {
	initCmd = InitCmd{}

	short := "Create a default config file"
	long := "Generate the minimal configuration required for a block explorer to start"

	_, err := parser.AddCommand("init", short, long, &initCmd)
	return err
}
