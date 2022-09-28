package history

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/datanode/snapshot"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
)

type showCmd struct {
	config.VegaHomeFlag
	config.Config
}

func (cmd *showCmd) Execute(args []string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	vegaPaths := paths.New(cmd.VegaHome)

	configFilePath, err := vegaPaths.CreateConfigPathFor(paths.DataNodeDefaultConfigFile)
	if err != nil {
		return fmt.Errorf("couldn't get path for %s: %w", paths.DataNodeDefaultConfigFile, err)
	}

	paths.ReadStructuredFile(configFilePath, &cmd.Config)

	snapshotsPath := vegaPaths.StatePathFor(paths.DataNodeSnapshotHome)

	connSource, err := sqlstore.NewTransactionalConnectionSource(log, cmd.Config.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}

	blockStore := sqlstore.NewBlocks(connSource)

	err = cmd.showHistory(blockStore, snapshotsPath)
	if err != nil {
		return fmt.Errorf("failed to show history: %w", err)
	}

	return nil
}

func (cmd *showCmd) showHistory(blockStore *sqlstore.Blocks, snapshotsPath string) error {
	oldestHistoryBlock, lastBlock, err := snapshot.GetOldestHistoryBlockAndLastBlock(context.Background(), cmd.Config.SQLStore.ConnectionConfig, blockStore)
	if err != nil {
		return fmt.Errorf("failed to get oldest history block and last block:%w", err)
	}

	chainID, currentStateSnapshot, contiguousHistory, err := snapshot.GetAllAvailableHistory(snapshotsPath, oldestHistoryBlock, lastBlock)
	if err != nil {
		if errors.Is(err, snapshot.ErrNoCurrentStateSnapshotFound) {
			fmt.Printf("No history found and the datanode is currently empty\n")
			return nil
		}

		return fmt.Errorf("failed to get all available history: %w", err)
	}

	toHeight, fromHeight := snapshot.GetToAndFromHeightFromHistory(currentStateSnapshot, contiguousHistory)

	fmt.Printf("History for chain id %s is available from height %d to %d\n", chainID, fromHeight, toHeight)

	if oldestHistoryBlock == nil {
		fmt.Printf("The datanode currently has no history\n")
	} else {
		fmt.Printf("The datanode currently spans height %d to %d\n", oldestHistoryBlock.Height, lastBlock.Height)
	}
	return nil
}
