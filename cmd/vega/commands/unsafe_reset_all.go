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
	"os"
	"path/filepath"

	"code.vegaprotocol.io/vega/core/config"
	metadatadb "code.vegaprotocol.io/vega/core/snapshot/databases/metadata"
	snapshotdb "code.vegaprotocol.io/vega/core/snapshot/databases/snapshot"
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

	clearSnapshotDatabases(vegaPaths, log)

	checkpointsPath := vegaPaths.StatePathFor(paths.CheckpointStateHome)
	if err := deleteAll(log, checkpointsPath); err != nil {
		log.Error("Unable to remove checkpoint state", logging.Error(err))
	} else {
		log.Info("Removed checkpoint state", logging.String("path", checkpointsPath))
	}
	return nil
}

func clearSnapshotDatabases(vegaPaths paths.Paths, log *logging.Logger) {
	snapshotDB, err := snapshotdb.NewLevelDBDatabase(vegaPaths)
	if err != nil {
		log.Error("Could not initialize the local snapshot database", logging.Error(err))
		log.Error("Skipping clear up of the local snapshot database")
	} else {
		defer func() {
			if err := snapshotDB.Close(); err != nil {
				log.Warn("Could not close the local snapshot database cleanly", logging.Error(err))
			}
		}()
		if err := snapshotDB.Clear(); err != nil {
			log.Error("Could not clear up the local snapshot database", logging.Error(err))
		} else {
			log.Info("Removed local snapshots")
		}
	}

	metadataDB, err := metadatadb.NewLevelDBDatabase(vegaPaths)
	if err != nil {
		log.Error("Could not initialize the local snapshot metadata database adapter", logging.Error(err))
		log.Error("Skipping clear up of the local snapshot metadata database")
		return
	} else {
		defer func() {
			if err := metadataDB.Close(); err != nil {
				log.Warn("Could not close the local snapshot metadata database cleanly", logging.Error(err))
			}
		}()
		if err := metadataDB.Clear(); err != nil {
			log.Error("Could not clear the local snapshot metadata database", logging.Error(err))
		} else {
			log.Info("Removed local snapshot metadata")
		}
	}
}

func deleteAll(log *logging.Logger, dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	files, err := d.Readdir(0)
	if err != nil {
		return err
	}

	for _, f := range files {
		filePath := filepath.Join(dir, f.Name())
		if err := os.RemoveAll(filePath); err != nil {
			return err
		}
		log.Info("Removed file", logging.String("path", filePath))
	}

	return nil
}

var unsafeResetCmd UnsafeResetAllCmd

func UnsafeResetAll(ctx context.Context, parser *flags.Parser) error {
	unsafeResetCmd = UnsafeResetAllCmd{}

	_, err := parser.AddCommand("unsafe_reset_all", "(unsafe) Remove all application state", "(unsafe) Remove all vega application state (checkpoints and snapshots)", &unsafeResetCmd)
	return err
}
