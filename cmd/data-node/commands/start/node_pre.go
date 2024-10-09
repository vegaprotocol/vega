// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package start

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"code.vegaprotocol.io/vega/datanode/broker"
	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/datanode/networkhistory"
	"code.vegaprotocol.io/vega/datanode/networkhistory/ipfs"
	"code.vegaprotocol.io/vega/datanode/networkhistory/snapshot"
	"code.vegaprotocol.io/vega/datanode/networkhistory/store"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/fs"
	"code.vegaprotocol.io/vega/libs/pprof"
	"code.vegaprotocol.io/vega/libs/subscribers"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	vegaprotoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"

	"github.com/cenkalti/backoff"
	"google.golang.org/grpc"
	"gopkg.in/natefinch/lumberjack.v2"
)

func (l *NodeCommand) persistentPre([]string) (err error) {
	// ensure we cancel the context on error
	defer func() {
		if err != nil {
			l.cancel()
		}
	}()

	conf := l.configWatcher.Get()

	// reload logger with the setup from configuration
	l.Log = logging.NewLoggerFromConfig(conf.Logging).Named(l.Log.GetName())

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
		if ResetDatabaseAndNetworkHistory(l.ctx, l.Log, l.vegaPaths, l.conf.SQLStore.ConnectionConfig); err != nil {
			return fmt.Errorf("failed to reset database and network history: %w", err)
		}
	} else if !l.conf.SQLStore.WipeOnStartup && l.conf.NetworkHistory.Enabled {
		ipfsDir := filepath.Join(l.vegaPaths.StatePathFor(paths.DataNodeNetworkHistoryHome), "store", "ipfs")
		ipfsExists, err := fs.PathExists(ipfsDir)
		if err != nil {
			return fmt.Errorf("failed to check if ipfs store is already initialized")
		}

		// We do not care for migration when the ipfs store does not exist on the local file system
		if ipfsExists {
			preLog.Info("Migrating the IPFS storage to the latest version")
			if err := ipfs.MigrateIpfsStorageVersion(preLog, ipfsDir); err != nil {
				return fmt.Errorf("failed to migrate the ipfs version")
			}
			preLog.Info("Migrating the IPFS storage finished")
		} else {
			preLog.Info("IPFS store not initialized. Migration not needed")
		}
	}

	initialisedFromNetworkHistory := false
	if l.conf.NetworkHistory.Enabled {
		preLog.Info("Initializing Network History")

		if l.conf.AutoInitialiseFromNetworkHistory {
			if err := networkhistory.KillAllConnectionsToDatabase(l.ctx, l.conf.SQLStore.ConnectionConfig); err != nil {
				return fmt.Errorf("failed to kill all connections to database: %w", err)
			}
		}

		err = l.initialiseNetworkHistory(preLog, l.conf.SQLStore.ConnectionConfig)
		if err != nil {
			l.Log.Error("Failed to initialise network history", logging.Error(err))
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

	// check that the schema version matches the latest migration, because if it doesn't queries might fail if rows/tables
	// it expects to exist don't
	if err := sqlstore.CheckSchemaVersionsSynced(l.Log, conf.SQLStore.ConnectionConfig, sqlstore.EmbedMigrations); err != nil {
		return err
	}

	preLog.Info("Enabling SQL stores")

	l.transactionalConnectionSource, err = sqlstore.NewTransactionalConnectionSource(l.ctx, preLog, l.conf.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to create transactional connection source: %w", err)
	}

	logSqlstore := l.Log.Named("sqlstore")
	l.CreateAllStores(l.ctx, logSqlstore, l.transactionalConnectionSource, l.conf.CandlesV2.CandleStore)

	logService := l.Log.Named("service")
	logService.SetLevel(l.conf.Service.Level.Get())
	if err := l.SetupServices(l.ctx, logService, l.conf.Service, l.conf.CandlesV2); err != nil {
		return err
	}

	err = networkhistory.VerifyChainID(l.conf.ChainID, l.chainService)
	if err != nil {
		return fmt.Errorf("failed to verify chain id:%w", err)
	}

	l.SetupSQLSubscribers()

	return nil
}

func (l *NodeCommand) initialiseDatabase(preLog *logging.Logger) error {
	var err error
	conf := l.conf.SQLStore.ConnectionConfig
	conf.MaxConnPoolSize = 1
	pool, err := sqlstore.CreateConnectionPool(l.ctx, conf)
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
func (l *NodeCommand) preRun([]string) (err error) {
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

func (l *NodeCommand) initialiseNetworkHistory(preLog *logging.Logger, connConfig sqlstore.ConnectionConfig) error {
	// Want to pre-allocate some connections to ensure a connection is always available,
	// 3 is chosen to allow for the fact that pool size can temporarily drop below the min pool size.
	connConfig.MaxConnPoolSize = 3
	connConfig.MinConnPoolSize = 3

	networkHistoryPool, err := sqlstore.CreateConnectionPool(l.ctx, connConfig)
	if err != nil {
		return fmt.Errorf("failed to create network history connection pool: %w", err)
	}

	preNetworkHistoryLog := preLog.Named("networkHistory")
	networkHistoryLog := l.Log.Named("networkHistory")
	networkHistoryLog.SetLevel(l.conf.NetworkHistory.Level.Get())

	snapshotServiceLog := networkHistoryLog.Named("snapshot")
	networkHistoryServiceLog := networkHistoryLog.Named("service")
	home := l.vegaPaths.StatePathFor(paths.DataNodeNetworkHistoryHome)

	networkHistoryStore, err := store.New(l.ctx, networkHistoryServiceLog, l.conf.ChainID, l.conf.NetworkHistory.Store, home,
		l.conf.MaxMemoryPercent)
	if err != nil {
		return fmt.Errorf("failed to create network history store: %w", err)
	}

	l.snapshotService, err = snapshot.NewSnapshotService(snapshotServiceLog, l.conf.NetworkHistory.Snapshot,
		networkHistoryPool, networkHistoryStore,
		l.vegaPaths.StatePathFor(paths.DataNodeNetworkHistorySnapshotCopyTo), func(version int64) error {
			if err = sqlstore.MigrateUpToSchemaVersion(preNetworkHistoryLog, l.conf.SQLStore, version, sqlstore.EmbedMigrations); err != nil {
				return fmt.Errorf("failed to migrate up to schema version %d: %w", version, err)
			}
			return nil
		},
		func(version int64) error {
			if err = sqlstore.MigrateDownToSchemaVersion(preNetworkHistoryLog, l.conf.SQLStore, version, sqlstore.EmbedMigrations); err != nil {
				return fmt.Errorf("failed to migrate down to schema version %d: %w", version, err)
			}
			return nil
		})
	if err != nil {
		return fmt.Errorf("failed to create snapshot service:%w", err)
	}

	l.networkHistoryService, err = networkhistory.New(l.ctx, networkHistoryServiceLog, l.conf.ChainID, l.conf.NetworkHistory,
		networkHistoryPool,
		l.snapshotService,
		networkHistoryStore,
		l.conf.API.Port,
		l.vegaPaths.StatePathFor(paths.DataNodeNetworkHistorySnapshotCopyTo))
	if err != nil {
		return fmt.Errorf("failed to create networkHistory service:%w", err)
	}

	return nil
}

func ResetDatabaseAndNetworkHistory(ctx context.Context, log *logging.Logger, vegaPaths paths.Paths, connConfig sqlstore.ConnectionConfig) error {
	err := os.RemoveAll(vegaPaths.StatePathFor(paths.DataNodeNetworkHistoryHome))
	if err != nil {
		return fmt.Errorf("failed to remove network history dir: %w", err)
	}

	log.Info("Wiped all network history")

	if err := sqlstore.RecreateVegaDatabase(ctx, log, connConfig); err != nil {
		return fmt.Errorf("failed to wipe database:%w", err)
	}
	log.Info("Wiped all existing data from the database")
	return nil
}
