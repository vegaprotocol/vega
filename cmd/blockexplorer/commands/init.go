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
