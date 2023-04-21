package networkhistory

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"

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

	Force             bool   `short:"f" long:"force" description:"do not prompt for confirmation"`
	WipeExistingData  bool   `short:"w" long:"wipe-existing-data" description:"Erase all data from the node before loading from network history"`
	OptimiseForAppend string `short:"a" long:"optimise-for-append" required:"false" description:"if true the load will be optimised for appending new segments onto existing datanode data, this is the default if the node already contains data" choice:"default" choice:"true" choice:"false" default:"default"`
}

func (cmd *loadCmd) Execute(args []string) error {
	cfg := logging.NewDefaultConfig()
	cfg.Custom.Zap.Level = logging.WarnLevel
	cfg.Environment = "custom"
	log := logging.NewLoggerFromConfig(
		cfg,
	)
	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()
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

	if err := networkhistory.KillAllConnectionsToDatabase(ctx, cmd.SQLStore.ConnectionConfig); err != nil {
		return fmt.Errorf("failed to kill all connections to database: %w", err)
	}

	connPool, err := getCommandConnPool(cmd.Config.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to get command connection pool: %w", err)
	}
	defer connPool.Close()

	hasSchema, err := sqlstore.HasVegaSchema(ctx, connPool)
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

	networkHistoryService, err := createNetworkHistoryService(ctx, log, cmd.Config, connPool, vegaPaths)
	if err != nil {
		return fmt.Errorf("failed to created network history service: %w", err)
	}
	defer networkHistoryService.Stop()

	segments, err := networkHistoryService.ListAllHistorySegments()
	if err != nil {
		return fmt.Errorf("failed to list history segments: %w", err)
	}

	mostRecentContiguousHistory, err := segments.MostRecentContiguousHistory()
	if err != nil {
		fmt.Println("No history is available to load.  Data can be fetched using the fetch command")
		return nil //nolint:nilerr
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

	contiguousHistory, err := segments.ContiguousHistoryInRange(from, to)
	if err != nil {
		fmt.Printf("No contiguous history is available for block span %d to %d. From and To Heights must match "+
			"from and to heights of the segments in the contiguous history span.\nUse the show command with the '-s' "+
			"option to see all available segments\n", from, to)
		return nil //nolint:nilerr
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

	optimiseForAppend := false
	switch strings.ToLower(cmd.OptimiseForAppend) {
	case "true":
		optimiseForAppend = true
	case "false":
		optimiseForAppend = false
	default:
		optimiseForAppend = span.HasData
	}

	if optimiseForAppend {
		fmt.Println("Loading history, optimising for append")
	} else {
		fmt.Println("Loading history, optimising for bulk load")
	}

	loadLog := newLoadLog()
	defer loadLog.AtExit()
	loaded, err := networkHistoryService.LoadNetworkHistoryIntoDatanodeWithLog(ctx, loadLog, contiguousHistory,
		cmd.Config.SQLStore.ConnectionConfig, optimiseForAppend, bool(cmd.Config.SQLStore.VerboseMigration))
	if err != nil {
		return fmt.Errorf("failed to load all available history:%w", err)
	}

	fmt.Printf("Loaded history from height %d to %d into the datanode\n", loaded.LoadedFromHeight, loaded.LoadedToHeight)

	return nil
}

func createNetworkHistoryService(ctx context.Context, log *logging.Logger, vegaConfig config.Config, connPool *pgxpool.Pool, vegaPaths paths.Paths) (*networkhistory.Service, error) {
	snapshotService, err := snapshot.NewSnapshotService(log, vegaConfig.NetworkHistory.Snapshot, connPool,
		vegaPaths.StatePathFor(paths.DataNodeNetworkHistorySnapshotCopyTo), func(version int64) error {
			if err := sqlstore.MigrateUpToSchemaVersion(log, vegaConfig.SQLStore, version, sqlstore.EmbedMigrations); err != nil {
				return fmt.Errorf("failed to migrate to schema version %d: %w", version, err)
			}
			return nil
		},
		func(version int64) error {
			if err := sqlstore.MigrateDownToSchemaVersion(log, vegaConfig.SQLStore, version, sqlstore.EmbedMigrations); err != nil {
				return fmt.Errorf("failed to migrate down to schema version %d: %w", version, err)
			}
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot service: %w", err)
	}

	networkHistoryStore, err := store.New(ctx, log, vegaConfig.ChainID, vegaConfig.NetworkHistory.Store, vegaPaths.StatePathFor(paths.DataNodeNetworkHistoryHome),
		vegaConfig.MaxMemoryPercent)
	if err != nil {
		return nil, fmt.Errorf("failed to create network history store:%w", err)
	}

	networkHistoryService, err := networkhistory.NewWithStore(ctx, log, vegaConfig.ChainID, vegaConfig.NetworkHistory,
		connPool, snapshotService, networkHistoryStore, vegaConfig.API.Port,
		vegaPaths.StatePathFor(paths.DataNodeNetworkHistorySnapshotCopyTo))
	if err != nil {
		return nil, fmt.Errorf("failed new networkhistory service:%w", err)
	}
	return networkHistoryService, nil
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
