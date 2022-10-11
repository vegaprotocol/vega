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
	"code.vegaprotocol.io/vega/datanode/snapshot"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/subscribers"
	"code.vegaprotocol.io/vega/libs/pprof"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	vegaprotoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"
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

	l.Log.Info("Enabling SQL stores")

	transactionalConnectionSource, err := l.initialiseDatabase()
	if err != nil {
		return err
	}

	l.CreateAllStores(l.ctx, l.Log, transactionalConnectionSource, l.conf.CandlesV2.CandleStore)

	log := l.Log.Named("service")
	log.SetLevel(l.conf.Service.Level.Get())
	if err := l.SetupServices(l.ctx, log, l.conf.CandlesV2); err != nil {
		return err
	}

	l.snapshotService, err = snapshot.NewSnapshotService(l.Log, l.conf.Snapshot, l.conf.Broker, l.blockStore,
		l.networkParameterService.GetByKey,
		l.chainService, l.conf.SQLStore.ConnectionConfig,
		l.vegaPaths.StatePathFor(paths.DataNodeSnapshotHome))
	if err != nil {
		l.Log.Error("failed to create snapshot service", logging.Error(err))
		return err
	}

	l.SetupSQLSubscribers(l.ctx, l.Log)

	return nil
}

func (l *NodeCommand) initialiseDatabase() (*sqlstore.ConnectionSource, error) {
	var err error
	if l.conf.SQLStore.UseEmbedded {
		l.embeddedPostgres, _, _, err = sqlstore.StartEmbeddedPostgres(l.Log, l.conf.SQLStore,
			l.vegaPaths.StatePathFor(paths.DataNodeStorageHome))
		if err != nil {
			return nil, fmt.Errorf("failed to start embedded postgres: %w", err)
		}
	}

	hasVegaSchema, err := snapshot.HasVegaSchema(l.ctx, l.conf.SQLStore.ConnectionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to check if database is empty: %w", err)
	}

	// If it's an empty database, recreate it with correct locale settings
	if !hasVegaSchema {
		err = sqlstore.RecreateVegaDatabase(l.ctx, l.Log, l.conf.SQLStore.ConnectionConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to recreate vega schema: %w", err)
		}
	}

	err = sqlstore.MigrateToLatestSchema(l.Log, l.conf.SQLStore)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate to latest schema:%w", err)
	}

	err = sqlstore.ApplyDataRetentionPolicies(l.conf.SQLStore)
	if err != nil {
		return nil, fmt.Errorf("failed to apply data retention policies:%w", err)
	}

	transactionalConnectionSource, err := sqlstore.NewTransactionalConnectionSource(l.Log, l.conf.SQLStore.ConnectionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection source:%w", err)
	}

	l.transactionalConnectionSource = transactionalConnectionSource

	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot service: %w", err)
	}

	return transactionalConnectionSource, nil
}

// we've already set everything up WRT arguments etc... just bootstrap the node.
func (l *NodeCommand) preRun([]string) (err error) {
	// ensure that context is cancelled if we return an error here
	defer func() {
		if err != nil {
			l.cancel()
		}
	}()

	eventSource, err := broker.NewEventSource(l.conf.Broker, l.Log)
	if err != nil {
		l.Log.Error("unable to initialise event source", logging.Error(err))
		return err
	}

	eventSource = broker.NewFanOutEventSource(eventSource, l.conf.SQLStore.FanOutBufferSize, 2)

	l.snapshotPublisher, err = snapshot.NewPublisher(l.ctx, l.Log, l.conf.Snapshot, l.vegaPaths.StatePathFor(paths.DataNodeSnapshotHome))
	if err != nil {
		l.Log.Error("failed to create snapshot publisher", logging.Error(err))
		return err
	}

	l.sqlBroker = broker.NewSQLStoreBroker(l.Log, l.conf.Broker, l.chainService, eventSource,
		l.transactionalConnectionSource,
		l.blockStore,
		l.snapshotService.OnBlockCommitted,
		l.GetSQLSubscribers(),
	)

	l.broker, err = broker.New(l.ctx, l.Log, l.conf.Broker, l.chainService, eventSource)
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
