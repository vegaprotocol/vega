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
	"errors"
	"fmt"
	"math"
	"time"

	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/datanode/config/encoding"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"github.com/jessevdk/go-flags"
)

type InitCmd struct {
	config.VegaHomeFlag

	Force            bool   `description:"Erase exiting vega configuration at the specified path" long:"force"     short:"f"`
	RetentionProfile string `choice:"archive"                                                     choice:"minimal" choice:"conservative" default:"archive" description:"Set which mode to initialise the data node with, will affect retention policies" long:"retention-profile" short:"r"`
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

	if opts.RetentionProfile == "archive" {
		cfg.NetworkHistory.Store.HistoryRetentionBlockSpan = math.MaxInt64
		cfg.SQLStore.RetentionPeriod = sqlstore.RetentionPeriodArchive
		cfg.NetworkHistory.Initialise.TimeOut = encoding.Duration{Duration: 96 * time.Hour}
		cfg.NetworkHistory.Initialise.MinimumBlockCount = -1
	}

	if opts.RetentionProfile == "minimal" {
		cfg.SQLStore.RetentionPeriod = sqlstore.RetentionPeriodLite
		cfg.NetworkHistory.Initialise.TimeOut = encoding.Duration{Duration: 1 * time.Minute}
		cfg.NetworkHistory.Initialise.MinimumBlockCount = 1
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
