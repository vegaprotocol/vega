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

package main

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/datanode/config"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/shared/paths"

	"github.com/jessevdk/go-flags"
)

type InitCmd struct {
	config.VegaHomeFlag

	Force bool `short:"f" long:"force" description:"Erase exiting vega configuration at the specified path"`
}

var initCmd InitCmd

func (opts *InitCmd) Execute(_ []string) error {
	logger := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer logger.AtExit()

	vegaPaths := paths.New(opts.VegaHome)

	cfgLoader, err := config.InitialiseLoader(vegaPaths)
	if err != nil {
		return fmt.Errorf("couldn't initialise configuration loader: %w", err)
	}

	configExists, err := cfgLoader.ConfigExists()
	if err != nil {
		return fmt.Errorf("couldn't verify configuration presence: %w", err)
	}

	if configExists && !opts.Force {
		return fmt.Errorf("configuration already exists at `%s` please remove it first or re-run using -f", cfgLoader.ConfigFilePath())
	}

	if configExists && opts.Force {
		cfgLoader.Remove()
	}

	cfg := config.NewDefaultConfig()

	if err := cfgLoader.Save(&cfg); err != nil {
		return fmt.Errorf("couldn't save configuration file: %w", err)
	}

	logger.Info("configuration generated successfully", logging.String("path", cfgLoader.ConfigFilePath()))

	return nil
}

func Init(ctx context.Context, parser *flags.Parser) error {
	initCmd = InitCmd{}

	short := "Initializes a vega node"
	long := "Generate the minimal configuration required for a vega data-node to start"

	_, err := parser.AddCommand("init", short, long, &initCmd)
	return err
}
