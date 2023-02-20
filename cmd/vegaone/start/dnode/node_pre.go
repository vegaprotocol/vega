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

package dnode

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"code.vegaprotocol.io/vega/libs/subscribers"

	"github.com/cenkalti/backoff"
	"google.golang.org/grpc"
	"gopkg.in/natefinch/lumberjack.v2"

	"code.vegaprotocol.io/vega/datanode/broker"
	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/datanode/networkhistory"
	"code.vegaprotocol.io/vega/datanode/networkhistory/snapshot"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/pprof"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	vegaprotoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"
)

func (l *DN) persistentPre([]string) (err error) {
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

	preLog := l.Log.Named("start.persistentPre")

	if conf.Pprof.Enabled {
		preLog.Info("vega is starting with pprof profile, this is not a recommended setting for production")
		l.pproffhandlr, err = pprof.New(l.Log, conf.Pprof)
		if err != nil {
			return
		}
		l.configWatcher.OnConfigUpdate(
			func(cfg config.Config) { l.pproffhandlr.ReloadConf(cfg.Pprof) },
		)
	}

	preLog.Info("Starting Vega Datanode",
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

	if l.conf.SQLStore.WipeOnStartup {
		if err = sqlstore.WipeDatabaseAndMigrateSchemaToLatestVersion(preLog, l.conf.SQLStore.ConnectionConfig, sqlstore.EmbedMigrations,
			bool(l.conf.SQLStore.VerboseMigration)); err != nil {
			return fmt.Errorf("failed to wiped database:%w", err)
		}
		preLog.Info("Wiped all existing data from the datanode")
	}

	initialisedFromNetworkHistory := false
	if l.conf.NetworkHistory.Enabled {
		preLog.Info("Initializing Network History")

		if l.conf.AutoInitialiseFromNetworkHistory {
			if err := networkhistory.KillAllConnectionsToDatabase(context.Background(), l.conf.SQLStore.ConnectionConfig); err != nil {
				return fmt.Errorf("failed to kill all connections to database: %w", err)
			}
		}

		err = l.initialiseNetworkHistory(preLog, l.conf.SQLStore.ConnectionConfig)
		if err != nil {
			return fmt.Errorf("failed to initialise network history:%w", err)
		}

		if l.conf.AutoInitialiseFromNetworkHistory {
			preLog.Info("Auto Initialising Datanode From Network History")
			apiPorts := []int{l.conf.API.Port}
			apiPorts = append(apiPorts, l.conf.NetworkHistory.Initialise.GrpcAPIPorts...)

			if err = networkhistory.InitialiseDatanodeFromNetworkHistory(l.ctx, l.conf.NetworkHistory.Initialise,
				preLog, l.conf.SQLStore.ConnectionConfig, l.networkHistoryService, apiPorts,
				bool(l.conf.SQLStore.VerboseMigration)); err != nil {
				return fmt.Errorf("failed to initialize datanode from network history: %w", err)
			}

			initialisedFromNetworkHistory = true
			preLog.Info("Initialized from network history")
		}
	}

	if !initialisedFromNetworkHistory {
		operation := func() (opErr error) {
			preLog.Info("Attempting to initialise database...")
			opErr = l.initialiseDatabase(preLog)
			if opErr != nil {
				preLog.Error("Failed to initialise database, retrying...", logging.Error(opErr))
			}
			preLog.Info("Database initialised")
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

	preLog.Info("Applying Data Retention Policies")

	err = sqlstore.ApplyDataRetentionPolicies(l.conf.SQLStore, preLog)
	if err != nil {
		return fmt.Errorf("failed to apply data retention policies:%w", err)
	}

	preLog.Info("Enabling SQL stores")

	l.transactionalConnectionSource, err = sqlstore.NewTransactionalConnectionSource(preLog, l.conf.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to create transactional connection source: %w", err)
	}

	logSqlstore := l.Log.Named("sqlstore")
	l.CreateAllStores(l.ctx, logSqlstore, l.transactionalConnectionSource, l.conf.CandlesV2.CandleStore)

	logService := l.Log.Named("service")
	logService.SetLevel(l.conf.Service.Level.Get())
	if err := l.SetupServices(l.ctx, logService, l.conf.CandlesV2); err != nil {
		return err
	}

	err = networkhistory.VerifyChainID(l.conf.ChainID, l.chainService)
	if err != nil {
		return fmt.Errorf("failed to verify chain id:%w", err)
	}

	l.SetupSQLSubscribers()

	return nil
}

func (l *DN) initialiseDatabase(preLog *logging.Logger) error {
	var err error
	conf := l.conf.SQLStore.ConnectionConfig
	conf.MaxConnPoolSize = 1
	pool, err := sqlstore.CreateConnectionPool(conf)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}
	defer pool.Close()

	hasVegaSchema, err := sqlstore.HasVegaSchema(l.ctx, pool)
	if err != nil {
		return fmt.Errorf("failed to check if database has schema: %w", err)
	}

	// If it's an empty database, recreate it with correct locale settings
	if !hasVegaSchema {
		err = sqlstore.RecreateVegaDatabase(l.ctx, preLog, l.conf.SQLStore.ConnectionConfig)
		if err != nil {
			return fmt.Errorf("failed to recreate vega schema: %w", err)
		}
	}

	err = sqlstore.MigrateToLatestSchema(preLog, l.conf.SQLStore)
	if err != nil {
		return fmt.Errorf("failed to migrate to latest schema:%w", err)
	}

	return nil
}

// we've already set everything up WRT arguments etc... just bootstrap the node.
func (l *DN) preRun([]string) (err error) {
	// ensure that context is cancelled if we return an error here
	defer func() {
		if err != nil {
			l.cancel()
		}
	}()

	preLog := l.Log.Named("start.preRun")
	brokerLog := l.Log.Named("broker")
	eventSourceLog := brokerLog.Named("eventsource")

	eventReceiverSender, err := broker.NewEventReceiverSender(l.conf.Broker, eventSourceLog, l.conf.ChainID)
	if err != nil {
		preLog.Error("unable to initialise event source", logging.Error(err))
		return err
	}

	var rawEventSource broker.RawEventReceiver = eventReceiverSender

	if l.conf.Broker.UseBufferedEventSource {
		bufferFilePath, err := l.vegaPaths.CreateStatePathFor(paths.DataNodeEventBufferHome)
		if err != nil {
			preLog.Error("failed to create path for buffered event source", logging.Error(err))
			return err
		}

		archiveFilesPath, err := l.vegaPaths.CreateStatePathFor(paths.DataNodeArchivedEventBufferHome)
		if err != nil {
			l.Log.Error("failed to create archive path for buffered event source", logging.Error(err))
			return err
		}

		rawEventSource, err = broker.NewBufferedEventSource(l.ctx, l.Log, l.conf.Broker.BufferedEventSourceConfig, eventReceiverSender,
			bufferFilePath, archiveFilesPath)
		if err != nil {
			preLog.Error("unable to initialise file buffered event source", logging.Error(err))
			return err
		}
	}

	var eventSource broker.EventReceiver
	eventSource = broker.NewDeserializer(rawEventSource)
	eventSource = broker.NewFanOutEventSource(eventSource, l.conf.SQLStore.FanOutBufferSize, 2)

	var onBlockCommittedHandler func(ctx context.Context, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool)
	var protocolUpgradeHandler broker.ProtocolUpgradeHandler

	if l.conf.NetworkHistory.Enabled {
		blockCommitHandler := networkhistory.NewBlockCommitHandler(l.Log, l.conf.NetworkHistory, l.snapshotService.SnapshotData,
			bool(l.conf.Broker.UseEventFile), l.conf.Broker.FileEventSourceConfig.TimeBetweenBlocks.Duration,
			5*time.Second, 6)
		onBlockCommittedHandler = blockCommitHandler.OnBlockCommitted
		protocolUpgradeHandler = networkhistory.NewProtocolUpgradeHandler(l.Log, l.protocolUpgradeService, eventReceiverSender,
			l.networkHistoryService.CreateAndPublishSegment)
	} else {
		onBlockCommittedHandler = func(ctx context.Context, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool) {}
		protocolUpgradeHandler = networkhistory.NewProtocolUpgradeHandler(l.Log, l.protocolUpgradeService, eventReceiverSender,
			func(ctx context.Context, chainID string, toHeight int64) error { return nil })
	}

	l.sqlBroker = broker.NewSQLStoreBroker(l.Log, l.conf.Broker, l.conf.ChainID, eventSource,
		l.transactionalConnectionSource,
		l.blockStore,
		onBlockCommittedHandler,
		protocolUpgradeHandler,
		l.GetSQLSubscribers(),
	)

	l.broker, err = broker.New(l.ctx, brokerLog, l.conf.Broker, l.conf.ChainID, eventSource)
	if err != nil {
		preLog.Error("unable to initialise broker", logging.Error(err))
		return err
	}

	// Event service as used by old and new world
	l.eventService = subscribers.NewService(preLog, l.broker, l.conf.Broker.EventBusClientBufferSize)

	nodeAddr := fmt.Sprintf("%v:%v", l.conf.API.CoreNodeIP, l.conf.API.CoreNodeGRPCPort)
	conn, err := grpc.Dial(nodeAddr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	l.vegaCoreServiceClient = vegaprotoapi.NewCoreServiceClient(conn)
	return nil
}

func (l *DN) initialiseNetworkHistory(preLog *logging.Logger, connConfig sqlstore.ConnectionConfig) error {
	// Want to pre-allocate some connections to ensure a connection is always available,
	// 3 is chosen to allow for the fact that pool size can temporarily drop below the min pool size.
	connConfig.MaxConnPoolSize = 3
	connConfig.MinConnPoolSize = 3

	networkHistoryPool, err := sqlstore.CreateConnectionPool(connConfig)
	if err != nil {
		return fmt.Errorf("failed to create network history connection pool: %w", err)
	}

	preNetworkHistoryLog := preLog.Named("networkHistory")
	networkHistoryLog := l.Log.Named("networkHistory")
	networkHistoryLog.SetLevel(l.conf.NetworkHistory.Level.Get())

	snapshotServiceLog := networkHistoryLog.Named("snapshot")
	networkHistoryServiceLog := networkHistoryLog.Named("service")

	l.snapshotService, err = snapshot.NewSnapshotService(snapshotServiceLog, l.conf.NetworkHistory.Snapshot,
		networkHistoryPool, l.vegaPaths.StatePathFor(paths.DataNodeNetworkHistorySnapshotCopyFrom),
		l.vegaPaths.StatePathFor(paths.DataNodeNetworkHistorySnapshotCopyTo), func(version int64) error {
			if err = sqlstore.MigrateToSchemaVersion(preNetworkHistoryLog, l.conf.SQLStore, version, sqlstore.EmbedMigrations); err != nil {
				return fmt.Errorf("failed to migrate to schema version %d: %w", version, err)
			}
			return nil
		})
	if err != nil {
		return fmt.Errorf("failed to create snapshot service:%w", err)
	}

	l.networkHistoryService, err = networkhistory.New(l.ctx, networkHistoryServiceLog, l.conf.NetworkHistory, l.vegaPaths.StatePathFor(paths.DataNodeNetworkHistoryHome),
		networkHistoryPool,
		l.conf.ChainID, l.snapshotService, l.conf.API.Port, l.vegaPaths.StatePathFor(paths.DataNodeNetworkHistorySnapshotCopyFrom),
		l.vegaPaths.StatePathFor(paths.DataNodeNetworkHistorySnapshotCopyTo), l.conf.MaxMemoryPercent)

	if err != nil {
		return fmt.Errorf("failed to create networkHistory service:%w", err)
	}

	return nil
}
