package main

import (
	"context"
	"os"
	"path/filepath"

	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/logging"

	"github.com/jessevdk/go-flags"
)

type UnsafeResetAllCmd struct {
	config.VegaHomeFlag
}

func (cmd *UnsafeResetAllCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	vegaPaths := paths.New(cmd.VegaHome)

	snapshotsPath := vegaPaths.StatePathFor(paths.SnapshotStateHome)
	if err := deleteAll(log, snapshotsPath); err != nil {
		log.Error("Unable to remove snapshot state", logging.Error(err))
	} else {
		log.Info("Removed snapshot state", logging.String("path", snapshotsPath))
	}

	checkpointsPath := vegaPaths.StatePathFor(paths.CheckpointStateHome)
	if err := deleteAll(log, checkpointsPath); err != nil {
		log.Error("Unable to remove checkpoint state", logging.Error(err))
	} else {
		log.Info("Removed checkpoint state", logging.String("path", checkpointsPath))
	}
	return nil
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
