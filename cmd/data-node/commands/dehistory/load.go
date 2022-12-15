package dehistory

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/sqlstore"

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

	Force            bool `short:"f" long:"force" description:"do not prompt for confirmation"`
	WipeExistingData bool `short:"w" long:"wipe-existing-data" description:"Erase all data from the node before loading from dehistory"`
}

func (cmd *loadCmd) Execute(_ []string) error {
	cfg := logging.NewDefaultConfig()
	cfg.Custom.Zap.Level = logging.WarnLevel
	cfg.Environment = "custom"
	log := logging.NewLoggerFromConfig(
		cfg,
	)
	defer log.AtExit()

	vegaPaths := paths.New(cmd.VegaHome)
	fixConfig(&cmd.Config, vegaPaths)

	// Wiping data from dehistory before loading then loading the data should never happen in any circumstance
	cmd.Config.DeHistory.WipeOnStartup = false

	hasSchema, err := sqlstore.HasVegaSchema(context.Background(), cmd.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to check for existing schema:%w", err)
	}

	if hasSchema {
		err = verifyChainID(log, cmd.SQLStore.ConnectionConfig, cmd.ChainID)
		if err != nil {
			if !errors.Is(err, dehistory.ErrChainNotFound) {
				return fmt.Errorf("failed to verify chain id:%w", err)
			}
		}
	}

	if datanodeLive(cmd.Config) {
		return fmt.Errorf("datanode must be shutdown before data can be loaded")
	}

	if hasSchema && cmd.WipeExistingData {
		sqlstore.WipeDatabaseAndMigrateSchemaToVersion(log, cmd.Config.SQLStore.ConnectionConfig, 0, sqlstore.EmbedMigrations)
	}

	snapshotService, err := snapshot.NewSnapshotService(log, cmd.Config.DeHistory.Snapshot, cmd.Config.SQLStore.ConnectionConfig,
		vegaPaths.StatePathFor(paths.DataNodeDeHistorySnapshotCopyFrom),
		vegaPaths.StatePathFor(paths.DataNodeDeHistorySnapshotCopyTo), func(version int64) error {
			if err = sqlstore.MigrateToSchemaVersion(log, cmd.Config.SQLStore, version, sqlstore.EmbedMigrations); err != nil {
				return fmt.Errorf("failed to migrate to schema version %d: %w", version, err)
			}
			return nil
		})
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
		cmd.Config.SQLStore.ConnectionConfig, snapshotService, deHistoryStore, cmd.Config.API.Port,
		vegaPaths.StatePathFor(paths.DataNodeDeHistorySnapshotCopyFrom),
		vegaPaths.StatePathFor(paths.DataNodeDeHistorySnapshotCopyTo))
	if err != nil {
		return fmt.Errorf("failed new dehistory service:%w", err)
	}

	from, to, err := getSpanOfAllAvailableHistory(deHistoryService)
	if err != nil {
		return fmt.Errorf("failed to get span of all available history:%w", err)
	}

	span, err := sqlstore.GetDatanodeBlockSpan(ctx, cmd.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to get datanode block span:%w", err)
	}

	if from == to || (from >= span.FromHeight && to <= span.ToHeight) {
		fmt.Println("No history is available to load.  Data can be fetched using the fetch command")
		return nil
	}

	fmt.Printf("Decentralized history from block height %d to %d is available to load, current datanode block span is %d to %d\n",
		from, to, span.FromHeight, span.ToHeight)

	yes := true
	if !cmd.Force {
		yes = flags.YesOrNo("Do you want to load this history?")
	}

	if yes {
		fmt.Printf("Loading history from block %d to %d...\n", from, to)

		loaded, err := deHistoryService.LoadAllAvailableHistoryIntoDatanode(context.Background())
		if err != nil {
			return fmt.Errorf("failed to load all available history:%w", err)
		}

		fmt.Printf("Loaded history from height %d to %d into the datanode\n", loaded.LoadedFromHeight, loaded.LoadedToHeight)
	}

	return nil
}

func getSpanOfAllAvailableHistory(dehistoryService *dehistory.Service) (from int64, to int64, err error) {
	contiguousHistory, err := dehistoryService.GetContiguousHistory()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get contiguous history data")
	}

	if len(contiguousHistory) == 0 {
		return 0, 0, nil
	}

	return contiguousHistory[0].HeightFrom, contiguousHistory[len(contiguousHistory)-1].HeightTo, nil
}
