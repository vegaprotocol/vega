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
	"errors"
	"fmt"
	"math"

	"code.vegaprotocol.io/vega/datanode/sqlstore"

	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"github.com/jessevdk/go-flags"
)

type InitCmd struct {
	config.VegaHomeFlag

	Force   bool `short:"f" long:"force" description:"Erase exiting vega configuration at the specified path"`
	Archive bool `short:"a" long:"archive" description:"Disable database retention policies. Keeps data and network history indefinitely"`
	Lite    bool `short:"l" long:"lite" description:"Set all database retention policies to one day only"`
}

var initCmd InitCmd

func (opts *InitCmd) Usage() string {
	return "<ChainID> [options]"
}

func (opts *InitCmd) Execute(args []string) error {
	logger := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer logger.AtExit()

	if len(args) != 1 {
		return errors.New("expected <chain ID>")
	}

	chainID := args[0]

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

	if opts.Archive && opts.Lite {
		return fmt.Errorf("specify either archive mode, lite mode - not both")
	}

	if opts.Archive {
		cfg.NetworkHistory.Store.HistoryRetentionBlockSpan = math.MaxInt64
		cfg.SQLStore.RetentionPeriod = sqlstore.RetentionPeriodArchive
	}

	if opts.Lite {
		cfg.SQLStore.RetentionPeriod = sqlstore.RetentionPeriodLite
	}

	cfg.ChainID = chainID

	if err := cfgLoader.Save(&cfg); err != nil {
		return fmt.Errorf("couldn't save configuration file: %w", err)
	}

	logger.Info("configuration generated successfully", logging.String("path", cfgLoader.ConfigFilePath()))

	return nil
}

func Init(ctx context.Context, parser *flags.Parser) error {
	initCmd = InitCmd{}

	short := "init <chain ID>"
	long := "Generate the minimal configuration required for a vega data-node to start. The Chain ID is required."

	_, err := parser.AddCommand("init", short, long, &initCmd)
	return err
}
