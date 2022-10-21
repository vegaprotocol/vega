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
	"fmt"

	"code.vegaprotocol.io/vega/datanode/api"
	"code.vegaprotocol.io/vega/datanode/broker"
	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/datanode/dehistory"
	"code.vegaprotocol.io/vega/datanode/dehistory/initialise"
	"code.vegaprotocol.io/vega/datanode/dehistory/snapshot"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/subscribers"
	"code.vegaprotocol.io/vega/libs/pprof"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	vegaprotoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"

	"github.com/cenkalti/backoff"
	"google.golang.org/grpc"
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
		l.embeddedPostgres, err = sqlstore.StartEmbeddedPostgres(l.Log, l.conf.SQLStore,
			paths.DataNodeEmbeddedPostgresRuntimeDir.String(), EmbeddedPostgresLog{})

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

	dataNodeHasData, err := initialise.DataNodeHasData(l.ctx, l.conf.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to check if data node has scheme or is empty: %w", err)
	}

	if !dataNodeHasData && bool(l.conf.DeHistory.Enabled) && bool(l.conf.AutoInitialiseFromDeHistory) {
		l.Log.Info("Auto Initialising Datanode From Decentralized History")
		if err = initialise.DatanodeFromDeHistory(l.ctx, l.conf.DeHistory.Initialise,
			l.Log, l.deHistoryService, l.conf.API.Port); err != nil {
			return fmt.Errorf("failed to initialise datanode from decentralized history:%w", err)
		}
		l.Log.Info("Finished Auto Initialising Datanode From Decentralized History")
	} else {
		operation := func() (opErr error) {
			l.Log.Info("Attempting to connect to SQL stores...")
			opErr = l.initialiseDatabase()
			if opErr != nil {
				l.Log.Error("Failed to connect to SQL stores, retrying...", logging.Error(opErr))
			}
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

	err = initialise.VerifyChainID(l.conf.ChainID, l.chainService)
	if err != nil {
		return fmt.Errorf("failed to verify chain id:%w", err)
	}

	l.SetupSQLSubscribers(l.ctx, l.Log)

	return nil
}

func (l *NodeCommand) initialiseDatabase() error {
	var err error

	hasVegaSchema, err := initialise.HasVegaSchema(l.ctx, l.conf.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to check if database is empty: %w", err)
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

	eventSource, err := broker.NewEventSource(l.conf.Broker, l.Log, l.conf.ChainID)
	if err != nil {
		l.Log.Error("unable to initialise event source", logging.Error(err))
		return err
	}

	eventSource = broker.NewFanOutEventSource(eventSource, l.conf.SQLStore.FanOutBufferSize, 2)

	var onBlockCommittedFn func(ctx context.Context, chainId string, lastCommittedBlockHeight int64)

	if l.snapshotService != nil {
		blockCommitHandler := snapshot.NewBlockCommitHandler(l.Log, l.snapshotService.SnapshotData, l.networkParameterService.GetByKey,
			l.conf.Broker)
		onBlockCommittedFn = blockCommitHandler.OnBlockCommitted
	} else {
		onBlockCommittedFn = func(ctx context.Context, chainId string, lastCommittedBlockHeight int64) {}
	}

	l.sqlBroker = broker.NewSQLStoreBroker(l.Log, l.conf.Broker, l.conf.ChainID, eventSource,
		l.transactionalConnectionSource,
		l.blockStore,
		l.protocolUpgradeService,
		onBlockCommittedFn,
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
	useEmbedded := bool(l.conf.SQLStore.UseEmbedded)
	snapshotsCopyFromDir, snapshotsCopyToDir := initialise.GetSnapshotPaths(useEmbedded, l.conf.DeHistory.Snapshot, l.vegaPaths)
	if useEmbedded {
		l.conf.DeHistory.Snapshot.DatabaseSnapshotsCopyFromPath = snapshotsCopyFromDir
		l.conf.DeHistory.Snapshot.DatabaseSnapshotsCopyToPath = snapshotsCopyToDir
	}

	deHistoryLog := l.Log.Named("deHistory")
	deHistoryLog.SetLevel(l.conf.DeHistory.Level.Get())

	snapshotServiceLog := deHistoryLog.Named("snapshot")
	deHistoryServiceLog := deHistoryLog.Named("service")

	var err error
	l.snapshotService, err = snapshot.NewSnapshotService(snapshotServiceLog, l.conf.DeHistory.Snapshot, l.conf.SQLStore.ConnectionConfig, snapshotsCopyToDir)
	if err != nil {
		return fmt.Errorf("failed to create snapshot service:%w", err)
	}

	l.deHistoryService, err = dehistory.New(l.ctx, deHistoryServiceLog, l.conf.DeHistory, l.vegaPaths.StatePathFor(paths.DataNodeDeHistoryHome),
		l.conf.SQLStore.ConnectionConfig, l.conf.ChainID, l.snapshotService, l.conf.API.Port, snapshotsCopyFromDir, snapshotsCopyToDir)

	if err != nil {
		return fmt.Errorf("failed to create deHistory service:%w", err)
	}

	return nil
}

// Todo should be able to configure this to send to a file.
type EmbeddedPostgresLog struct{}

func (n2 EmbeddedPostgresLog) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (n2 EmbeddedPostgresLog) String() string {
	return ""
}
