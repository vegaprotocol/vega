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

package networkhistory

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/datanode/networkhistory"
	vgterm "code.vegaprotocol.io/vega/libs/term"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
)

type rollbackCmd struct {
	config.VegaHomeFlag
	config.Config

	Force bool `description:"do not prompt for confirmation" long:"force" short:"f"`
}

func (cmd *rollbackCmd) Execute(args []string) error {
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()
	cfg := logging.NewDefaultConfig()
	cfg.Custom.Zap.Level = logging.WarnLevel
	cfg.Environment = "custom"
	log := logging.NewLoggerFromConfig(
		cfg,
	)
	defer log.AtExit()

	if len(args) != 1 {
		return errors.New("expected <rollback-to-height>")
	}

	rollbackToHeight, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse rollback to height: %w", err)
	}

	vegaPaths := paths.New(cmd.VegaHome)
	err = fixConfig(&cmd.Config, vegaPaths)
	if err != nil {
		return fmt.Errorf("failed to fix config:%w", err)
	}

	if datanodeLive(cmd.Config) {
		return fmt.Errorf("datanode must be shutdown before it can be rolled back")
	}

	if !cmd.Force && vgterm.HasTTY() {
		if !flags.YesOrNo("Running this command will kill all existing database connections, do you want to continue?") {
			return nil
		}
	}

	if err := networkhistory.KillAllConnectionsToDatabase(ctx, cmd.SQLStore.ConnectionConfig); err != nil {
		return fmt.Errorf("failed to kill all connections to database: %w", err)
	}

	connPool, err := getCommandConnPool(ctx, cmd.Config.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to get command connection pool: %w", err)
	}
	defer connPool.Close()

	networkHistoryService, err := createNetworkHistoryService(ctx, log, cmd.Config, connPool, vegaPaths)
	if err != nil {
		return fmt.Errorf("failed to created network history service: %w", err)
	}
	defer networkHistoryService.Stop()

	loadLog := newLoadLog()
	defer loadLog.AtExit()
	err = networkHistoryService.RollbackToHeight(ctx, loadLog, rollbackToHeight)
	if err != nil {
		return fmt.Errorf("failed to rollback datanode: %w", err)
	}

	fmt.Printf("Rolled back datanode to height %d\n", rollbackToHeight)

	return nil
}
