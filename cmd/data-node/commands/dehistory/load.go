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

	if from < span.FromHeight && to <= span.ToHeight {
		fmt.Printf("Available Decentralized History data spans height %d to %d.  The from height is before the datanodes' block span, %d to %d."+
			" To load history from before the datanodes oldest block you must specify the "+
			" \"wipe-existing-data\" flag to wipe existing data from the datanode before loading.\n\n", from, to, span.FromHeight, span.ToHeight)

		return nil
	}

	if from < span.FromHeight {
		fmt.Printf("Available Decentralized History data spans height %d to %d. However as the datanode already contains"+
			" data from height %d to %d only the history from height %d to %d will be loaded.  To load all the available history data"+
			" run the load command with the \"wipe-existing-data\" flag which will empty the data node before restoring it from the history data\n\n",
			from, to, span.FromHeight, span.ToHeight, span.ToHeight+1, to)

		if !cmd.Force {
			if !flags.YesOrNo(fmt.Sprintf("Do you wish to continue and load all history from height %d to %d ?", span.ToHeight+1, to)) {
				return nil
			}
		}
	} else {
		fmt.Printf("Decentralized history from block height %d to %d is available to load, current datanode block span is %d to %d\n\n",
			from, to, span.FromHeight, span.ToHeight)

		if !cmd.Force {
			if !flags.YesOrNo("Do you want to load this history?") {
				return nil
			}
		}
	}

	fmt.Println("Loading history...")

	loaded, err := deHistoryService.LoadDeHistoryIntoDatanode(context.Background())
	if err != nil {
		return fmt.Errorf("failed to load all available history:%w", err)
	}

	fmt.Printf("Loaded history from height %d to %d into the datanode\n", loaded.LoadedFromHeight, loaded.LoadedToHeight)

	return nil
}

func getSpanOfAllAvailableHistory(dehistoryService *dehistory.Service) (from int64, to int64, err error) {
	contiguousHistory, err := dehistoryService.GetContiguousHistoryFromHighestHeight()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get contiguous history data")
	}

	if len(contiguousHistory) == 0 {
		return 0, 0, nil
	}

	return contiguousHistory[0].HeightFrom, contiguousHistory[len(contiguousHistory)-1].HeightTo, nil
}
