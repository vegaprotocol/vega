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

package start

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/cenkalti/backoff"
	"google.golang.org/grpc"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"

	"code.vegaprotocol.io/vega/datanode/api"
	"code.vegaprotocol.io/vega/datanode/broker"
	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/datanode/dehistory"
	"code.vegaprotocol.io/vega/datanode/dehistory/snapshot"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/subscribers"
	"code.vegaprotocol.io/vega/libs/pprof"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	vegaprotoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"
)

func (l *NodeCommand) persistentPre([]string) (err error) {
	// this shouldn't happen...
	if l.cancel != nil {
		l.cancel()
	}
	// ensure we cancel the context on error
	defer func() {
		if err != nil {
			l.cancel()
		}
	}()
	l.ctx, l.cancel = context.WithCancel(context.Background())

	conf := l.configWatcher.Get()

	// reload logger with the setup from configuration
	l.Log = logging.NewLoggerFromConfig(conf.Logging)

	if conf.Pprof.Enabled {
		l.Log.Info("vega is starting with pprof profile, this is not a recommended setting for production")
		l.pproffhandlr, err = pprof.New(l.Log, conf.Pprof)
		if err != nil {
			return
		}
		l.configWatcher.OnConfigUpdate(
			func(cfg config.Config) { l.pproffhandlr.ReloadConf(cfg.Pprof) },
		)
	}

	l.Log.Info("Starting Vega Datanode",
		logging.String("version", l.Version),
		logging.String("version-hash", l.VersionHash))

	if l.conf.SQLStore.UseEmbedded {
		logDir := l.vegaPaths.StatePathFor(paths.DataNodeLogsHome)
		postgresLogger := &lumberjack.Logger{
			Filename: filepath.Join(logDir, "embedded-postgres.log"),
			MaxSize:  l.conf.SQLStore.LogRotationConfig.MaxSize,
			MaxAge:   l.conf.SQLStore.LogRotationConfig.MaxAge,
			Compress: true,
		}

		runtimeDir := l.vegaPaths.StatePathFor(paths.DataNodeEmbeddedPostgresRuntimeDir)
		l.embeddedPostgres, err = sqlstore.StartEmbeddedPostgres(l.Log, l.conf.SQLStore,
			runtimeDir, postgresLogger)

		if err != nil {
			return fmt.Errorf("failed to start embedded postgres: %w", err)
		}

		go func() {
			for range l.ctx.Done() {
				l.embeddedPostgres.Stop()
			}
		}()
	}

	if l.conf.DeHistory.Enabled {
		l.Log.Info("Initializing Decentralized History")
		err = l.initialiseDecentralizedHistory()
		if err != nil {
			return fmt.Errorf("failed to initialise decentralized history:%w", err)
		}
	}

	if l.conf.SQLStore.WipeOnStartup {
		if err = sqlstore.WipeDatabaseAndMigrateSchemaToLatestVersion(l.Log, l.conf.SQLStore.ConnectionConfig, sqlstore.EmbedMigrations); err != nil {
			return fmt.Errorf("failed to wiped database:%w", err)
		}
		l.Log.Info("Wiped all existing data from the datanode")
	}

	initialisedFromDeHistory := false
	if bool(l.conf.DeHistory.Enabled) && bool(l.conf.AutoInitialiseFromDeHistory) {
		l.Log.Info("Auto Initialising Datanode From Decentralized History")
		initialisedFromDeHistory = true
		apiPorts := []int{l.conf.API.Port}
		apiPorts = append(apiPorts, l.conf.DeHistory.Initialise.GrpcAPIPorts...)
		blockSpan, err := sqlstore.GetDatanodeBlockSpan(l.ctx, l.conf.SQLStore.ConnectionConfig)
		if err != nil {
			return fmt.Errorf("failed to get datanode block span:%w", err)
		}

		if blockSpan.HasData {
			l.Log.Infof("Datanode has data from block height %d to %d", blockSpan.FromHeight, blockSpan.ToHeight)
		} else {
			l.Log.Info("Datanode is empty")
		}

		if err = dehistory.DatanodeFromDeHistory(l.ctx, l.conf.DeHistory.Initialise,
			l.Log, l.deHistoryService, blockSpan, apiPorts); err != nil {
			if errors.Is(err, dehistory.ErrDeHistoryNotAvailable) {
				initialisedFromDeHistory = false
				l.Log.Info("Unable to initialize from decentralized history, no history is available")
			} else {
				return fmt.Errorf("failed to initialize datanode from decentralized history:%w", err)
			}
		}
	}

	if initialisedFromDeHistory {
		l.Log.Info("Initialized from decentralized history")
	} else {
		operation := func() (opErr error) {
			l.Log.Info("Attempting to initialise database...")
			opErr = l.initialiseDatabase()
			if opErr != nil {
				l.Log.Error("Failed to initialise database, retrying...", logging.Error(opErr))
			}
			l.Log.Info("Database initialised")
			return opErr
		}

		retryConfig := l.conf.SQLStore.ConnectionRetryConfig

		expBackoff := backoff.NewExponentialBackOff()
		expBackoff.InitialInterval = retryConfig.InitialInterval
		expBackoff.MaxInterval = retryConfig.MaxInterval
		expBackoff.MaxElapsedTime = retryConfig.MaxElapsedTime

		err = backoff.Retry(operation, backoff.WithMaxRetries(expBackoff, retryConfig.MaxRetries))
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
	}

	l.Log.Info("Applying Data Retention Policies")

	err = sqlstore.ApplyDataRetentionPolicies(l.conf.SQLStore)
	if err != nil {
		return fmt.Errorf("failed to apply data retention policies:%w", err)
	}

	l.Log.Info("Enabling SQL stores")

	transactionalConnectionSource, err := sqlstore.NewTransactionalConnectionSource(l.Log, l.conf.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection source:%w", err)
	}

	l.transactionalConnectionSource = transactionalConnectionSource

	l.CreateAllStores(l.ctx, l.Log, transactionalConnectionSource, l.conf.CandlesV2.CandleStore)

	log := l.Log.Named("service")
	log.SetLevel(l.conf.Service.Level.Get())
	if err := l.SetupServices(l.ctx, log, l.conf.CandlesV2); err != nil {
		return err
	}

	err = dehistory.VerifyChainID(l.conf.ChainID, l.chainService)
	if err != nil {
		return fmt.Errorf("failed to verify chain id:%w", err)
	}

	l.SetupSQLSubscribers(l.ctx, l.Log)

	return nil
}

func (l *NodeCommand) initialiseDatabase() error {
	var err error

	hasVegaSchema, err := sqlstore.HasVegaSchema(l.ctx, l.conf.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to check if database has schema: %w", err)
	}

	// If it's an empty database, recreate it with correct locale settings
	if !hasVegaSchema {
		err = sqlstore.RecreateVegaDatabase(l.ctx, l.Log, l.conf.SQLStore.ConnectionConfig)
		if err != nil {
			return fmt.Errorf("failed to recreate vega schema: %w", err)
		}
	}

	err = sqlstore.MigrateToLatestSchema(l.Log, l.conf.SQLStore)
	if err != nil {
		return fmt.Errorf("failed to migrate to latest schema:%w", err)
	}

	return nil
}

// we've already set everything up WRT arguments etc... just bootstrap the node.
func (l *NodeCommand) preRun([]string) (err error) {
	// ensure that context is cancelled if we return an error here
	defer func() {
		if err != nil {
			l.cancel()
		}
	}()

	eventReceiverSender, err := broker.NewEventReceiverSender(l.conf.Broker, l.Log, l.conf.ChainID)
	if err != nil {
		l.Log.Error("unable to initialise event source", logging.Error(err))
		return err
	}

	var eventSource broker.EventReceiver
	if l.conf.Broker.UseBufferedEventSource {
		bufferFilePath, err := l.vegaPaths.CreateStatePathFor(paths.DataNodeEventBufferHome)
		if err != nil {
			l.Log.Error("failed to create path for buffered event source", logging.Error(err))
			return err
		}
		eventSource, err = broker.NewBufferedEventSource(l.Log, l.conf.Broker.BufferedEventSourceConfig, eventReceiverSender, bufferFilePath)
		if err != nil {
			l.Log.Error("unable to initialise file buffered event source", logging.Error(err))
			return err
		}
	}

	eventSource = broker.NewFanOutEventSource(eventReceiverSender, l.conf.SQLStore.FanOutBufferSize, 2)

	var onBlockCommittedHandler func(ctx context.Context, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool)
	var protocolUpgradeHandler broker.ProtocolUpgradeHandler

	if l.conf.DeHistory.Enabled {
		blockCommitHandler := dehistory.NewBlockCommitHandler(l.Log, l.conf.DeHistory, l.snapshotService.SnapshotData,
			bool(l.conf.Broker.UseEventFile), l.conf.Broker.FileEventSourceConfig.TimeBetweenBlocks.Duration)
		onBlockCommittedHandler = blockCommitHandler.OnBlockCommitted
		protocolUpgradeHandler = dehistory.NewProtocolUpgradeHandler(l.Log, l.protocolUpgradeService, eventReceiverSender,
			l.deHistoryService.CreateAndPublishSegment)
	} else {
		onBlockCommittedHandler = func(ctx context.Context, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool) {}
		protocolUpgradeHandler = dehistory.NewProtocolUpgradeHandler(l.Log, l.protocolUpgradeService, eventReceiverSender,
			func(ctx context.Context, chainID string, toHeight int64) error { return nil })
	}

	l.sqlBroker = broker.NewSQLStoreBroker(l.Log, l.conf.Broker, l.conf.ChainID, eventSource,
		l.transactionalConnectionSource,
		l.blockStore,
		onBlockCommittedHandler,
		protocolUpgradeHandler,
		l.GetSQLSubscribers(),
	)

	l.broker, err = broker.New(l.ctx, l.Log, l.conf.Broker, l.conf.ChainID, eventSource)
	if err != nil {
		l.Log.Error("unable to initialise broker", logging.Error(err))
		return err
	}

	// Event service as used by old and new world
	l.eventService = subscribers.NewService(l.broker)

	nodeAddr := fmt.Sprintf("%v:%v", l.conf.API.CoreNodeIP, l.conf.API.CoreNodeGRPCPort)
	conn, err := grpc.Dial(nodeAddr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	coreClient := vegaprotoapi.NewCoreServiceClient(conn)
	l.vegaCoreServiceClient = api.NewVegaCoreServiceClient(coreClient, conn.GetState)
	return nil
}

func (l *NodeCommand) initialiseDecentralizedHistory() error {
	deHistoryLog := l.Log.Named("deHistory")
	deHistoryLog.SetLevel(l.conf.DeHistory.Level.Get())

	snapshotServiceLog := deHistoryLog.Named("snapshot")
	deHistoryServiceLog := deHistoryLog.Named("service")

	var err error
	l.snapshotService, err = snapshot.NewSnapshotService(snapshotServiceLog, l.conf.DeHistory.Snapshot,
		l.conf.SQLStore.ConnectionConfig, l.vegaPaths.StatePathFor(paths.DataNodeDeHistorySnapshotCopyFrom),
		l.vegaPaths.StatePathFor(paths.DataNodeDeHistorySnapshotCopyTo), func(version int64) error {
			if err = sqlstore.MigrateToSchemaVersion(deHistoryLog, l.conf.SQLStore, version, sqlstore.EmbedMigrations); err != nil {
				return fmt.Errorf("failed to migrate to schema version %d: %w", version, err)
			}
			return nil
		})
	if err != nil {
		return fmt.Errorf("failed to create snapshot service:%w", err)
	}

	l.deHistoryService, err = dehistory.New(l.ctx, deHistoryServiceLog, l.conf.DeHistory, l.vegaPaths.StatePathFor(paths.DataNodeDeHistoryHome),
		l.conf.SQLStore.ConnectionConfig, l.conf.ChainID, l.snapshotService, l.conf.API.Port, l.vegaPaths.StatePathFor(paths.DataNodeDeHistorySnapshotCopyFrom),
		l.vegaPaths.StatePathFor(paths.DataNodeDeHistorySnapshotCopyTo))

	if err != nil {
		return fmt.Errorf("failed to create deHistory service:%w", err)
	}

	return nil
}
