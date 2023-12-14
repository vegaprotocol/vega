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

	"code.vegaprotocol.io/vega/core/checkpoint"
	"code.vegaprotocol.io/vega/core/config"
	snapshotdbs "code.vegaprotocol.io/vega/core/snapshot/databases"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"github.com/jessevdk/go-flags"
)

type UnsafeResetAllCmd struct {
	config.VegaHomeFlag
}

//nolint:unparam
func (cmd *UnsafeResetAllCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	vegaPaths := paths.New(cmd.VegaHome)

	if err := snapshotdbs.RemoveAll(vegaPaths); err != nil {
		log.Error("Could not remove local snapshots databases", logging.Error(err))
		log.Error("Skipping removal of the local snapshots")
	} else {
		log.Info("Local snapshots have been removed")
	}

	if err := checkpoint.RemoveAll(vegaPaths); err != nil {
		log.Error("Could not remove local checkpoints", logging.Error(err))
		log.Error("Skipping removal of the local checkpoints")
	} else {
		log.Info("Local checkpoints have been removed")
	}

	return nil
}

var unsafeResetCmd UnsafeResetAllCmd

func UnsafeResetAll(_ context.Context, parser *flags.Parser) error {
	unsafeResetCmd = UnsafeResetAllCmd{}

	_, err := parser.AddCommand("unsafe_reset_all", "(unsafe) Remove all application state", "(unsafe) Remove all vega application state (checkpoints and snapshots)", &unsafeResetCmd)
	return err
}
