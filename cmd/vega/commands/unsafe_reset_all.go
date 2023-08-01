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
