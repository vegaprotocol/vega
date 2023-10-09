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

	"code.vegaprotocol.io/vega/cmd/data-node/commands/start"
	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"github.com/jessevdk/go-flags"
)

type UnsafeResetAllCmd struct {
	config.VegaHomeFlag
	*config.Config
}

//nolint:unparam
func (cmd *UnsafeResetAllCmd) Execute(_ []string) error {
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	vegaPaths := paths.New(cmd.VegaHome)

	cfgLoader, err := config.InitialiseLoader(vegaPaths)
	if err != nil {
		return fmt.Errorf("couldn't initialise configuration loader: %w", err)
	}

	cmd.Config, err = cfgLoader.Get()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	connConfig := cmd.Config.SQLStore.ConnectionConfig
	if start.ResetDatabaseAndNetworkHistory(ctx, log, vegaPaths, connConfig); err != nil {
		return fmt.Errorf("failed to reset database and network history: %w", err)
	}

	return nil
}

var unsafeResetCmd UnsafeResetAllCmd

func UnsafeResetAll(_ context.Context, parser *flags.Parser) error {
	unsafeResetCmd = UnsafeResetAllCmd{}

	_, err := parser.AddCommand("unsafe_reset_all", "(unsafe) Remove all application state", "(unsafe) Remove all datanode application state (wipes the database and removes all network history)", &unsafeResetCmd)
	return err
}
