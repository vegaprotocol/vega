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
	"os"
	"os/signal"
	"syscall"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"golang.org/x/sync/errgroup"

	"code.vegaprotocol.io/vega/datanode/api"
	"code.vegaprotocol.io/vega/datanode/broker"
	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/datanode/dehistory"
	"code.vegaprotocol.io/vega/datanode/dehistory/snapshot"
	"code.vegaprotocol.io/vega/datanode/gateway/server"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/subscribers"
	"code.vegaprotocol.io/vega/libs/pprof"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
)

// NodeCommand use to implement 'node' command.
type NodeCommand struct {
	SQLSubscribers
	ctx    context.Context
	cancel context.CancelFunc

	embeddedPostgres              *embeddedpostgres.EmbeddedPostgres
	transactionalConnectionSource *sqlstore.ConnectionSource

	deHistoryService *dehistory.Service
	snapshotService  *snapshot.Service

	vegaCoreServiceClient api.CoreServiceClient

	broker    *broker.Broker
	sqlBroker broker.SQLStoreEventBroker

	eventService *subscribers.Service

	pproffhandlr  *pprof.Pprofhandler
	Log           *logging.Logger
	vegaPaths     paths.Paths
	configWatcher *config.Watcher
	conf          config.Config

	Version     string
	VersionHash string
}

func (l *NodeCommand) Run(cfgwatchr *config.Watcher, vegaPaths paths.Paths, args []string) error {
	l.configWatcher = cfgwatchr

	l.conf = cfgwatchr.Get()
	l.vegaPaths = vegaPaths

	stages := []func([]string) error{
		l.persistentPre,
		l.preRun,
		l.runNode,
		l.postRun,
		l.persistentPost,
	}
	for _, fn := range stages {
		if err := fn(args); err != nil {
			return err
		}
	}

	return nil
}

// Stop is for graceful shutdown.
func (l *NodeCommand) Stop() {
	l.cancel()
}

// runNode is the entry of node command.
func (l *NodeCommand) runNode([]string) error {
	defer l.cancel()

	ctx, cancel := context.WithCancel(l.ctx)
	eg, ctx := errgroup.WithContext(ctx)

	// gRPC server
	grpcServer := l.createGRPCServer(l.conf.API)

	// watch configs
	l.configWatcher.OnConfigUpdate(
		func(cfg config.Config) { grpcServer.ReloadConf(cfg.API) },
	)

	// start the grpc server
	eg.Go(func() error { return grpcServer.Start(ctx, nil) })

	// start gateway
	if l.conf.GatewayEnabled {
		gty := server.New(l.conf.Gateway, l.Log, l.vegaPaths)
		eg.Go(func() error { return gty.Start(ctx) })
	}

	eg.Go(func() error {
		return l.broker.Receive(ctx)
	})

	eg.Go(func() error {
		return l.sqlBroker.Receive(ctx)
	})

	// waitSig will wait for a sigterm or sigint interrupt.
	eg.Go(func() error {
		gracefulStop := make(chan os.Signal, 1)
		signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT)

		select {
		case sig := <-gracefulStop:
			l.Log.Info("Caught signal", logging.String("name", fmt.Sprintf("%+v", sig)))
			cancel()
		case <-ctx.Done():
			return ctx.Err()
		}

		return nil
	})

	metrics.Start(l.conf.Metrics)

	l.Log.Info("Vega data node startup complete")

	err := eg.Wait()
	if errors.Is(err, context.Canceled) {
		return nil
	}

	return err
}

func (l *NodeCommand) createGRPCServer(config api.Config) *api.GRPCServer {
	grpcServer := api.NewGRPCServer(
		l.Log,
		config,
		l.vegaCoreServiceClient,
		l.eventService,
		l.orderService,
		l.networkLimitsService,
		l.marketDataService,
		l.tradeService,
		l.assetService,
		l.accountService,
		l.rewardService,
		l.marketsService,
		l.delegationService,
		l.epochService,
		l.depositService,
		l.withdrawalService,
		l.governanceService,
		l.riskFactorService,
		l.riskService,
		l.networkParameterService,
		l.blockService,
		l.checkpointService,
		l.partyService,
		l.candleService,
		l.oracleSpecService,
		l.oracleDataService,
		l.liquidityProvisionService,
		l.positionService,
		l.transferService,
		l.stakeLinkingService,
		l.notaryService,
		l.multiSigService,
		l.keyRotationsService,
		l.ethereumKeyRotationsService,
		l.nodeService,
		l.marketDepthService,
		l.ledgerService,
		l.protocolUpgradeService,
		l.deHistoryService,
		l.coreSnapshotService,
	)
	return grpcServer
}
