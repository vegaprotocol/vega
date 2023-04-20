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

func (d *DN) persistentPre([]string) (err error) {
	// this shouldn't happen...
	if d.cancel != nil {
		d.cancel()
	}
	// ensure we cancel the context on error
	defer func() {
		if err != nil {
			d.cancel()
		}
	}()
	d.ctx, d.cancel = context.WithCancel(context.Background())

	conf := d.configWatcher.Get()

	preLog := d.Log.Named("start.persistentPre")

	if conf.Pprof.Enabled {
		preLog.Info("vega is starting with pprof profile, this is not a recommended setting for production")
		d.pproffhandlr, err = pprof.New(d.Log, conf.Pprof)
		if err != nil {
			return
		}
		d.configWatcher.OnConfigUpdate(
			func(cfg config.Config) { d.pproffhandlr.ReloadConf(cfg.Pprof) },
		)
	}

	preLog.Info("Starting Vega Datanode",
		logging.String("version", d.Version),
		logging.String("version-hash", d.VersionHash))

	if d.conf.SQLStore.UseEmbedded {
		logDir := d.vegaPaths.StatePathFor(paths.DataNodeLogsHome)
		postgresLogger := &lumberjack.Logger{
			Filename: filepath.Join(logDir, "embedded-postgres.log"),
			MaxSize:  d.conf.SQLStore.LogRotationConfig.MaxSize,
			MaxAge:   d.conf.SQLStore.LogRotationConfig.MaxAge,
			Compress: true,
		}

		runtimeDir := d.vegaPaths.StatePathFor(paths.DataNodeEmbeddedPostgresRuntimeDir)
		d.embeddedPostgres, err = sqlstore.StartEmbeddedPostgres(d.Log, d.conf.SQLStore,
			runtimeDir, postgresLogger)

		if err != nil {
			return fmt.Errorf("failed to start embedded postgres: %w", err)
		}

		go func() {
			for range d.ctx.Done() {
				d.embeddedPostgres.Stop()
			}
		}()
	}

	if d.conf.SQLStore.WipeOnStartup {
		if err = sqlstore.WipeDatabaseAndMigrateSchemaToLatestVersion(preLog, d.conf.SQLStore.ConnectionConfig, sqlstore.EmbedMigrations,
			bool(d.conf.SQLStore.VerboseMigration)); err != nil {
			return fmt.Errorf("failed to wiped database:%w", err)
		}
		preLog.Info("Wiped all existing data from the datanode")
	}

	initialisedFromNetworkHistory := false
	if d.conf.NetworkHistory.Enabled {
		preLog.Info("Initializing Network History")

		if d.conf.AutoInitialiseFromNetworkHistory {
			if err := networkhistory.KillAllConnectionsToDatabase(context.Background(), d.conf.SQLStore.ConnectionConfig); err != nil {
				return fmt.Errorf("failed to kill all connections to database: %w", err)
			}
		}

		err = d.initialiseNetworkHistory(preLog, d.conf.SQLStore.ConnectionConfig)
		if err != nil {
			return fmt.Errorf("failed to initialise network history:%w", err)
		}

		if d.conf.AutoInitialiseFromNetworkHistory {
			preLog.Info("Auto Initialising Datanode From Network History")
			apiPorts := []int{d.conf.API.Port}
			apiPorts = append(apiPorts, d.conf.NetworkHistory.Initialise.GrpcAPIPorts...)

			if err = networkhistory.InitialiseDatanodeFromNetworkHistory(d.ctx, d.conf.NetworkHistory.Initialise,
				preLog, d.conf.SQLStore.ConnectionConfig, d.networkHistoryService, apiPorts,
				bool(d.conf.SQLStore.VerboseMigration)); err != nil {
				return fmt.Errorf("failed to initialize datanode from network history: %w", err)
			}

			initialisedFromNetworkHistory = true
			preLog.Info("Initialized from network history")
		}
	}

	if !initialisedFromNetworkHistory {
		operation := func() (opErr error) {
			preLog.Info("Attempting to initialise database...")
			opErr = d.initialiseDatabase(preLog)
			if opErr != nil {
				preLog.Error("Failed to initialise database, retrying...", logging.Error(opErr))
			}
			preLog.Info("Database initialised")
			return opErr
		}

		retryConfig := d.conf.SQLStore.ConnectionRetryConfig

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

	err = sqlstore.ApplyDataRetentionPolicies(d.conf.SQLStore, preLog)
	if err != nil {
		return fmt.Errorf("failed to apply data retention policies:%w", err)
	}

	preLog.Info("Enabling SQL stores")

	d.transactionalConnectionSource, err = sqlstore.NewTransactionalConnectionSource(preLog, d.conf.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to create transactional connection source: %w", err)
	}

	logSqlstore := d.Log.Named("sqlstore")
	d.CreateAllStores(d.ctx, logSqlstore, d.transactionalConnectionSource, d.conf.CandlesV2.CandleStore)

	logService := d.Log.Named("service")
	logService.SetLevel(d.conf.Service.Level.Get())
	if err := d.SetupServices(d.ctx, logService, d.conf.CandlesV2); err != nil {
		return err
	}

	err = networkhistory.VerifyChainID(d.conf.ChainID, d.chainService)
	if err != nil {
		return fmt.Errorf("failed to verify chain id:%w", err)
	}

	d.SetupSQLSubscribers()

	return nil
}

func (d *DN) initialiseDatabase(preLog *logging.Logger) error {
	var err error
	conf := d.conf.SQLStore.ConnectionConfig
	conf.MaxConnPoolSize = 1
	pool, err := sqlstore.CreateConnectionPool(conf)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}
	defer pool.Close()

	hasVegaSchema, err := sqlstore.HasVegaSchema(d.ctx, pool)
	if err != nil {
		return fmt.Errorf("failed to check if database has schema: %w", err)
	}

	// If it's an empty database, recreate it with correct locale settings
	if !hasVegaSchema {
		err = sqlstore.RecreateVegaDatabase(d.ctx, preLog, d.conf.SQLStore.ConnectionConfig)
		if err != nil {
			return fmt.Errorf("failed to recreate vega schema: %w", err)
		}
	}

	err = sqlstore.MigrateToLatestSchema(preLog, d.conf.SQLStore)
	if err != nil {
		return fmt.Errorf("failed to migrate to latest schema:%w", err)
	}

	return nil
}

// we've already set everything up WRT arguments etc... just bootstrap the node.
func (d *DN) preRun([]string) (err error) {
	// ensure that context is cancelled if we return an error here
	defer func() {
		if err != nil {
			d.cancel()
		}
	}()

	preLog := d.Log.Named("start.preRun")
	brokerLog := d.Log.Named("broker")
	eventSourceLog := brokerLog.Named("eventsource")

	eventReceiverSender, err := broker.NewEventReceiverSender(d.conf.Broker, eventSourceLog, d.conf.ChainID)
	if err != nil {
		preLog.Error("unable to initialise event source", logging.Error(err))
		return err
	}

	var rawEventSource broker.RawEventReceiver = eventReceiverSender

	if d.conf.Broker.UseBufferedEventSource {
		bufferFilePath, err := d.vegaPaths.CreateStatePathFor(paths.DataNodeEventBufferHome)
		if err != nil {
			preLog.Error("failed to create path for buffered event source", logging.Error(err))
			return err
		}

		archiveFilesPath, err := d.vegaPaths.CreateStatePathFor(paths.DataNodeArchivedEventBufferHome)
		if err != nil {
			d.Log.Error("failed to create archive path for buffered event source", logging.Error(err))
			return err
		}

		rawEventSource, err = broker.NewBufferedEventSource(d.ctx, d.Log, d.conf.Broker.BufferedEventSourceConfig, eventReceiverSender,
			bufferFilePath, archiveFilesPath)
		if err != nil {
			preLog.Error("unable to initialise file buffered event source", logging.Error(err))
			return err
		}
	}

	var eventSource broker.EventReceiver
	eventSource = broker.NewDeserializer(rawEventSource)
	eventSource = broker.NewFanOutEventSource(eventSource, d.conf.SQLStore.FanOutBufferSize, 2)

	var onBlockCommittedHandler func(ctx context.Context, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool)
	var protocolUpgradeHandler broker.ProtocolUpgradeHandler

	if d.conf.NetworkHistory.Enabled {
		blockCommitHandler := networkhistory.NewBlockCommitHandler(d.Log, d.conf.NetworkHistory, d.snapshotService.SnapshotData,
			bool(d.conf.Broker.UseEventFile), d.conf.Broker.FileEventSourceConfig.TimeBetweenBlocks.Duration,
			5*time.Second, 6)
		onBlockCommittedHandler = blockCommitHandler.OnBlockCommitted
		protocolUpgradeHandler = networkhistory.NewProtocolUpgradeHandler(d.Log, d.protocolUpgradeService, eventReceiverSender,
			d.networkHistoryService.CreateAndPublishSegment)
	} else {
		onBlockCommittedHandler = func(ctx context.Context, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool) {}
		protocolUpgradeHandler = networkhistory.NewProtocolUpgradeHandler(d.Log, d.protocolUpgradeService, eventReceiverSender,
			func(ctx context.Context, chainID string, toHeight int64) error { return nil })
	}

	d.sqlBroker = broker.NewSQLStoreBroker(d.Log, d.conf.Broker, d.conf.ChainID, eventSource,
		d.transactionalConnectionSource,
		d.blockStore,
		onBlockCommittedHandler,
		protocolUpgradeHandler,
		d.GetSQLSubscribers(),
	)

	d.broker, err = broker.New(d.ctx, brokerLog, d.conf.Broker, d.conf.ChainID, eventSource)
	if err != nil {
		preLog.Error("unable to initialise broker", logging.Error(err))
		return err
	}

	// Event service as used by old and new world
	d.eventService = subscribers.NewService(preLog, d.broker, d.conf.Broker.EventBusClientBufferSize)

	nodeAddr := fmt.Sprintf("%v:%v", d.conf.API.CoreNodeIP, d.conf.API.CoreNodeGRPCPort)
	conn, err := grpc.Dial(nodeAddr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	d.vegaCoreServiceClient = vegaprotoapi.NewCoreServiceClient(conn)
	return nil
}

func (d *DN) initialiseNetworkHistory(preLog *logging.Logger, connConfig sqlstore.ConnectionConfig) error {
	// Want to pre-allocate some connections to ensure a connection is always available,
	// 3 is chosen to allow for the fact that pool size can temporarily drop below the min pool size.
	connConfig.MaxConnPoolSize = 3
	connConfig.MinConnPoolSize = 3

	networkHistoryPool, err := sqlstore.CreateConnectionPool(connConfig)
	if err != nil {
		return fmt.Errorf("failed to create network history connection pool: %w", err)
	}

	preNetworkHistoryLog := preLog.Named("networkHistory")
	networkHistoryLog := d.Log.Named("networkHistory")
	networkHistoryLog.SetLevel(d.conf.NetworkHistory.Level.Get())

	snapshotServiceLog := networkHistoryLog.Named("snapshot")
	networkHistoryServiceLog := networkHistoryLog.Named("service")

	d.snapshotService, err = snapshot.NewSnapshotService(snapshotServiceLog, d.conf.NetworkHistory.Snapshot,
		networkHistoryPool,
		d.vegaPaths.StatePathFor(paths.DataNodeNetworkHistorySnapshotCopyTo), func(version int64) error {
			if err = sqlstore.MigrateUpToSchemaVersion(preNetworkHistoryLog, d.conf.SQLStore, version, sqlstore.EmbedMigrations); err != nil {
				return fmt.Errorf("failed to migrate up to schema version %d: %w", version, err)
			}
			return nil
		},
		func(version int64) error {
			if err = sqlstore.MigrateDownToSchemaVersion(preNetworkHistoryLog, d.conf.SQLStore, version, sqlstore.EmbedMigrations); err != nil {
				return fmt.Errorf("failed to migrate down to schema version %d: %w", version, err)
			}
			return nil
		})
	if err != nil {
		return fmt.Errorf("failed to create snapshot service:%w", err)
	}
	d.networkHistoryService, err = networkhistory.New(d.ctx, networkHistoryServiceLog, d.conf.NetworkHistory, d.vegaPaths.StatePathFor(paths.DataNodeNetworkHistoryHome),
		networkHistoryPool,
		d.conf.ChainID, d.snapshotService, d.conf.API.Port, d.vegaPaths.StatePathFor(paths.DataNodeNetworkHistorySnapshotCopyFrom),
		d.vegaPaths.StatePathFor(paths.DataNodeNetworkHistorySnapshotCopyTo), d.conf.MaxMemoryPercent)

	if err != nil {
		return fmt.Errorf("failed to create networkHistory service:%w", err)
	}

	return nil
}
