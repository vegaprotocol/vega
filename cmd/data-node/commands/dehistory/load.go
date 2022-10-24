package dehistory

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/dehistory/initialise"
	"code.vegaprotocol.io/vega/datanode/dehistory/store"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/datanode/dehistory"
	"code.vegaprotocol.io/vega/datanode/dehistory/snapshot"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"code.vegaprotocol.io/vega/datanode/config"
)

type loadCmd struct {
	config.VegaHomeFlag
	config.Config
}

func (cmd *loadCmd) Execute(_ []string) error {
	cfg := logging.NewDefaultConfig()
	cfg.Custom.Zap.Level = logging.InfoLevel
	cfg.Environment = "custom"
	log := logging.NewLoggerFromConfig(
		cfg,
	)
	defer log.AtExit()

	vegaPaths := paths.New(cmd.VegaHome)
	configFilePath, err := vegaPaths.CreateConfigPathFor(paths.DataNodeDefaultConfigFile)
	if err != nil {
		return fmt.Errorf("couldn't get path for %s: %w", paths.DataNodeDefaultConfigFile, err)
	}

	err = paths.ReadStructuredFile(configFilePath, &cmd.Config)
	if err != nil {
		return fmt.Errorf("failed to read configuration:%w", err)
	}

	err = verifyChainID(log, cmd.SQLStore.ConnectionConfig, cmd.ChainID)
	if err != nil {
		return fmt.Errorf("failed to verify chain id:%w", err)
	}

	if datanodeLive(cmd.Config) {
		return fmt.Errorf("datanode must be shutdown before data can be loaded")
	}

	snapshotsCopyFrom, snapshotsCopyTo := initialise.GetSnapshotPaths(bool(cmd.Config.SQLStore.UseEmbedded), cmd.Config.DeHistory.Snapshot, vegaPaths)

	snapshotService, err := snapshot.NewSnapshotService(log, cmd.Config.DeHistory.Snapshot, cmd.Config.SQLStore.ConnectionConfig, snapshotsCopyTo)
	if err != nil {
		return fmt.Errorf("failed to create snapshot service: %w", err)
	}

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	deHistoryStore, err := store.New(ctx, log, cmd.Config.ChainID, cmd.Config.DeHistory.Store, vegaPaths.StatePathFor(paths.DataNodeDeHistoryHome), false)
	if err != nil {
		return fmt.Errorf("failed to create decentralized history store:%w", err)
	}

	deHistoryService, err := dehistory.NewWithStore(ctx, log, cmd.Config.ChainID, cmd.Config.DeHistory,
		cmd.Config.SQLStore.ConnectionConfig, snapshotService, deHistoryStore, cmd.Config.API.Port, snapshotsCopyFrom, snapshotsCopyTo)
	if err != nil {
		return fmt.Errorf("failed new dehistory service:%w", err)
	}

	from, to, err := getSpanOfAllAvailableHistory(context.Background(), deHistoryService)
	if err != nil {
		return fmt.Errorf("failed to get span of all available history:%w", err)
	}

	datanodeFromHeight, datanodeToHeight, err := initialise.GetDatanodeBlockSpan(ctx, cmd.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to get datanode block span:%w", err)
	}

	if from == to || (from >= datanodeFromHeight && to <= datanodeToHeight) {
		fmt.Println("No history is available to load.  Data can be fetched using the fetch command")
		return nil
	}

	datanodeIsEmpty, err := initialise.DataNodeIsEmpty(ctx, cmd.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to check if datanode is empty:%w", err)
	}

	if datanodeIsEmpty {
		fmt.Printf("Datanode has no data, history from block height %d to %d is available to load\n",
			from, to)
	} else {
		fmt.Printf("History from block height %d to %d is available to load, current datanode block span is %d to %d\n",
			from, to, datanodeFromHeight, datanodeToHeight)
	}

	yes := flags.YesOrNo("Do you want to load this history?")

	if yes {
		fmt.Printf("Loading history from block %d to %d...\n", from, to)

		loadedFrom, loadedTo, err := deHistoryService.LoadAllAvailableHistoryIntoDatanode(context.Background())
		if err != nil {
			return fmt.Errorf("failed to load all available history:%w", err)
		}

		fmt.Printf("Loaded history from height %d to %d into the datanode\n", loadedFrom, loadedTo)
	}

	return nil
}

func getSpanOfAllAvailableHistory(ctx context.Context, dehistoryService *dehistory.Service) (from int64, to int64, err error) {
	contiguousHistory, err := dehistoryService.GetContiguousHistory(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get contiguous history data")
	}

	if len(contiguousHistory) == 0 {
		return 0, 0, nil
	}

	return contiguousHistory[0].HeightFrom, contiguousHistory[len(contiguousHistory)-1].HeightTo, nil
}
