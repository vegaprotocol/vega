package networkhistory

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	vgterm "code.vegaprotocol.io/vega/libs/term"

	"go.uber.org/zap"

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

	Force                       bool   `short:"f" long:"force" description:"do not prompt for confirmation"`
	WipeExistingData            bool   `short:"w" long:"wipe-existing-data" description:"Erase all data from the node before loading from network history"`
	WithIndexesAndOrderTriggers string `short:"i" long:"with-indexes-and-order-triggers" required:"false" description:"if true the load will not drop indexes and order triggers when loading data, this is usually the best option when appending new segments onto existing datanode data" choice:"default" choice:"true" choice:"false" default:"default"`
}

func (cmd *loadCmd) Execute(args []string) error {
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

	if !cmd.Force && vgterm.HasTTY() {
		if !flags.YesOrNo("Running this command will kill all existing database connections, do you want to continue?") {
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

	// Wiping data from network history before loading then trying to load the data should never happen in any circumstance
	cmd.Config.NetworkHistory.WipeOnStartup = false

	hasSchema, err := sqlstore.HasVegaSchema(context.Background(), connPool)
	if err != nil {
		return fmt.Errorf("failed to check for existing schema:%w", err)
	}

	if hasSchema {
		err = verifyChainID(cmd.SQLStore.ConnectionConfig, cmd.ChainID)
		if err != nil {
			if !errors.Is(err, networkhistory.ErrChainNotFound) {
				return fmt.Errorf("failed to verify chain id:%w", err)
			}
		}
	}

	if hasSchema && cmd.WipeExistingData {
		err := sqlstore.WipeDatabaseAndMigrateSchemaToVersion(log, cmd.Config.SQLStore.ConnectionConfig, 0,
			sqlstore.EmbedMigrations, bool(cmd.Config.SQLStore.VerboseMigration))
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

	networkHistoryStore, err := store.New(ctx, log, cmd.Config.ChainID, cmd.Config.NetworkHistory.Store, vegaPaths.StatePathFor(paths.DataNodeNetworkHistoryHome),
		false, cmd.Config.MaxMemoryPercent)
	if err != nil {
		return fmt.Errorf("failed to create network history store:%w", err)
	}
	defer networkHistoryStore.Stop()

	networkHistoryService, err := networkhistory.NewWithStore(ctx, log, cmd.Config.ChainID, cmd.Config.NetworkHistory,
		connPool, snapshotService, networkHistoryStore, cmd.Config.API.Port,
		vegaPaths.StatePathFor(paths.DataNodeNetworkHistorySnapshotCopyFrom),
		vegaPaths.StatePathFor(paths.DataNodeNetworkHistorySnapshotCopyTo))
	if err != nil {
		return fmt.Errorf("failed new networkhistory service:%w", err)
	}

	segments, err := networkHistoryService.ListAllHistorySegments()
	mostRecentContiguousHistory := networkhistory.GetMostRecentContiguousHistory(segments)

	if mostRecentContiguousHistory == nil {
		fmt.Println("No history is available to load.  Data can be fetched using the fetch command")
		return nil
	}

	from := mostRecentContiguousHistory.HeightFrom
	if len(args) >= 1 {
		from, err = strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse from height: %w", err)
		}
	}

	to := mostRecentContiguousHistory.HeightTo
	if len(args) == 2 {
		to, err = strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse to height: %w", err)
		}
	}

	contiguousHistory := networkhistory.GetContiguousHistoryForSpan(networkhistory.GetContiguousHistories(segments), from, to)
	if contiguousHistory == nil {
		fmt.Printf("No contiguous history is available for block span %d to %d. From and To Heights must match "+
			"from and to heights of the segments in the contiguous history span.\nUse the show command with the '-s' "+
			"option to see all available segments\n", from, to)
		return nil
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
			" \"--wipe-existing-data\" flag to wipe existing data from the datanode before loading.\n\n", from, to, span.FromHeight, span.ToHeight)

		return nil
	}

	if from < span.FromHeight {
		fmt.Printf("Available Network History data spans height %d to %d. However as the datanode already contains"+
			" data from height %d to %d only the history from height %d to %d will be loaded.  To load all the available history data"+
			" run the load command with the \"wipe-existing-data\" flag which will empty the data node before restoring it from the history data\n\n",
			from, to, span.FromHeight, span.ToHeight, span.ToHeight+1, to)

		if !cmd.Force && vgterm.HasTTY() {
			if !flags.YesOrNo(fmt.Sprintf("Do you wish to continue and load all history from height %d to %d ?", span.ToHeight+1, to)) {
				return nil
			}
		}
	} else {
		fmt.Printf("Network history from block height %d to %d is available to load, current datanode block span is %d to %d\n\n",
			from, to, span.FromHeight, span.ToHeight)

		if !cmd.Force && vgterm.HasTTY() {
			if !flags.YesOrNo("Do you want to load this history?") {
				return nil
			}
		}
	}

	withIndexesAndOrderTriggers := false
	switch cmd.WithIndexesAndOrderTriggers {
	case "true":
		withIndexesAndOrderTriggers = true
	case "default":
		withIndexesAndOrderTriggers = span.HasData
	}

	if withIndexesAndOrderTriggers {
		fmt.Println("Loading history with indexes and order triggers...")
	} else {
		fmt.Println("Loading history...")
	}

	loadLog := newLoadLog()
	defer loadLog.AtExit()
	loaded, err := networkHistoryService.LoadNetworkHistoryIntoDatanodeWithLog(context.Background(), loadLog, *contiguousHistory,
		cmd.Config.SQLStore.ConnectionConfig, withIndexesAndOrderTriggers, bool(cmd.Config.SQLStore.VerboseMigration))
	if err != nil {
		return fmt.Errorf("failed to load all available history:%w", err)
	}

	fmt.Printf("Loaded history from height %d to %d into the datanode\n", loaded.LoadedFromHeight, loaded.LoadedToHeight)

	return nil
}

type loadLog struct {
	log *logging.Logger
}

func newLoadLog() *loadLog {
	cfg := logging.NewDefaultConfig()
	cfg.Custom.Zap.Level = logging.InfoLevel
	cfg.Environment = "custom"

	return &loadLog{
		log: logging.NewLoggerFromConfig(cfg),
	}
}

func (l *loadLog) AtExit() {
	l.log.AtExit()
}

func (l *loadLog) Infof(s string, args ...interface{}) {
	currentTime := time.Now()
	argsWithTime := []any{currentTime.Format("2006-01-02 15:04:05")}
	argsWithTime = append(argsWithTime, args...)
	fmt.Printf("%s "+s+"\n", argsWithTime...)
}

func (l *loadLog) Info(msg string, fields ...zap.Field) {
	l.log.Info(msg, fields...)
}
