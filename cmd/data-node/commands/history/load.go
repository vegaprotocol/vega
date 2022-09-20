package history

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/datanode/snapshot"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"code.vegaprotocol.io/vega/datanode/config"
)

type loadCmd struct {
	config.VegaHomeFlag
	config.Config
}

func (cmd *loadCmd) Execute(args []string) error {
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

	log.Info("Loading all available history...")

	connSource, err := sqlstore.NewTransactionalConnectionSource(log, cmd.Config.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}

	blockStore := sqlstore.NewBlocks(connSource)
	chainService := service.NewChain(sqlstore.NewChain(connSource), log)
	networkParamsService := service.NewNetworkParameter(sqlstore.NewNetworkParameters(connSource), log)

	snapshotService, err := snapshot.NewSnapshotService(log, cmd.Config.Snapshot, cmd.Config.Broker, blockStore,
		networkParamsService.GetByKey, chainService, cmd.Config.SQLStore.ConnectionConfig,
		vegaPaths.StatePathFor(paths.DataNodeSnapshotHome))
	if err != nil {
		return fmt.Errorf("failed to create snapshot service: %w", err)
	}

	loadedFrom, loadedTo, err := snapshotService.LoadAllAvailableHistory(context.Background())
	if err != nil {
		return fmt.Errorf("failed to load all available history:%w", err)
	}

	log.Infof("Loaded history from height %d to %d", loadedFrom, loadedTo)

	return nil
}
