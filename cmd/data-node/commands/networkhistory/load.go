package networkhistory

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/networkhistory/store"
	"code.vegaprotocol.io/vega/datanode/sqlstore"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	"code.vegaprotocol.io/vega/datanode/networkhistory"
	"code.vegaprotocol.io/vega/datanode/networkhistory/snapshot"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"code.vegaprotocol.io/vega/datanode/config"
)

type loadCmd struct {
	config.VegaHomeFlag
	config.Config

	Force            bool `short:"f" long:"force" description:"do not prompt for confirmation"`
	WipeExistingData bool `short:"w" long:"wipe-existing-data" description:"Erase all data from the node before loading from networkhistory"`
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
	err := fixConfig(&cmd.Config, vegaPaths)
	if err != nil {
		return fmt.Errorf("failed to fix config:%w", err)
	}

	if datanodeLive(cmd.Config) {
		return fmt.Errorf("datanode must be shutdown before data can be loaded")
	}

	if !cmd.Force {
		if !flags.YesOrNo("Running this command will kill all existing database connections, do you with to continue?") {
			return nil
		}
	}

	if err := networkhistory.KillAllConnectionsToDatabase(context.Background(), cmd.SQLStore.ConnectionConfig); err != nil {
		return fmt.Errorf("failed to kill all connections to database: %w", err)
	}

	connPool, err := getCommandConnPool(cmd.Config.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to get command connection pool: %w", err)
	}
	defer connPool.Close()

	// Wiping data from networkhistory before loading then trying to load the data should never happen in any circumstance
	cmd.Config.NetworkHistory.WipeOnStartup = false

	hasSchema, err := sqlstore.HasVegaSchema(context.Background(), connPool)
	if err != nil {
		return fmt.Errorf("failed to check for existing schema:%w", err)
	}

	if hasSchema {
		err = verifyChainID(log, cmd.SQLStore.ConnectionConfig, cmd.ChainID)
		if err != nil {
			if !errors.Is(err, networkhistory.ErrChainNotFound) {
				return fmt.Errorf("failed to verify chain id:%w", err)
			}
		}
	}

	if hasSchema && cmd.WipeExistingData {
		err := sqlstore.WipeDatabaseAndMigrateSchemaToVersion(log, cmd.Config.SQLStore.ConnectionConfig, 0, sqlstore.EmbedMigrations)
		if err != nil {
			return fmt.Errorf("failed to wipe database and migrate schema to version: %w", err)
		}
	}

	snapshotService, err := snapshot.NewSnapshotService(log, cmd.Config.NetworkHistory.Snapshot, connPool,
		vegaPaths.StatePathFor(paths.DataNodeNetworkHistorySnapshotCopyFrom),
		vegaPaths.StatePathFor(paths.DataNodeNetworkHistorySnapshotCopyTo), func(version int64) error {
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

	networkHistoryStore, err := store.New(ctx, log, cmd.Config.ChainID, cmd.Config.NetworkHistory.Store, vegaPaths.StatePathFor(paths.DataNodeNetworkHistoryHome), false)
	if err != nil {
		return fmt.Errorf("failed to create network history store:%w", err)
	}

	networkHistoryService, err := networkhistory.NewWithStore(ctx, log, cmd.Config.ChainID, cmd.Config.NetworkHistory,
		connPool, snapshotService, networkHistoryStore, cmd.Config.API.Port,
		vegaPaths.StatePathFor(paths.DataNodeNetworkHistorySnapshotCopyFrom),
		vegaPaths.StatePathFor(paths.DataNodeNetworkHistorySnapshotCopyTo))
	if err != nil {
		return fmt.Errorf("failed new networkhistory service:%w", err)
	}

	from, to, err := getSpanOfAllAvailableHistory(networkHistoryService)
	if err != nil {
		return fmt.Errorf("failed to get span of all available history:%w", err)
	}

	span, err := sqlstore.GetDatanodeBlockSpan(ctx, connPool)
	if err != nil {
		return fmt.Errorf("failed to get datanode block span:%w", err)
	}

	if from == to || (from >= span.FromHeight && to <= span.ToHeight) {
		fmt.Println("No history is available to load.  Data can be fetched using the fetch command")
		return nil
	}

	if from < span.FromHeight && to <= span.ToHeight {
		fmt.Printf("Available Network History data spans height %d to %d.  The from height is before the datanodes' block span, %d to %d."+
			" To load history from before the datanodes oldest block you must specify the "+
			" \"wipe-existing-data\" flag to wipe existing data from the datanode before loading.\n\n", from, to, span.FromHeight, span.ToHeight)

		return nil
	}

	if from < span.FromHeight {
		fmt.Printf("Available Network History data spans height %d to %d. However as the datanode already contains"+
			" data from height %d to %d only the history from height %d to %d will be loaded.  To load all the available history data"+
			" run the load command with the \"wipe-existing-data\" flag which will empty the data node before restoring it from the history data\n\n",
			from, to, span.FromHeight, span.ToHeight, span.ToHeight+1, to)

		if !cmd.Force {
			if !flags.YesOrNo(fmt.Sprintf("Do you wish to continue and load all history from height %d to %d ?", span.ToHeight+1, to)) {
				return nil
			}
		}
	} else {
		fmt.Printf("Network history from block height %d to %d is available to load, current datanode block span is %d to %d\n\n",
			from, to, span.FromHeight, span.ToHeight)

		if !cmd.Force {
			if !flags.YesOrNo("Do you want to load this history?") {
				return nil
			}
		}
	}

	fmt.Println("Loading history...")

	loaded, err := networkHistoryService.LoadNetworkHistoryIntoDatanode(context.Background(), cmd.Config.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to load all available history:%w", err)
	}

	fmt.Printf("Loaded history from height %d to %d into the datanode\n", loaded.LoadedFromHeight, loaded.LoadedToHeight)

	return nil
}

func getSpanOfAllAvailableHistory(networkhistoryService *networkhistory.Service) (from int64, to int64, err error) {
	contiguousHistory, err := networkhistoryService.GetContiguousHistoryFromHighestHeight()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get contiguous history data")
	}

	if len(contiguousHistory) == 0 {
		return 0, 0, nil
	}

	return contiguousHistory[0].HeightFrom, contiguousHistory[len(contiguousHistory)-1].HeightTo, nil
}
