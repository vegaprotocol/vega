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
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"code.vegaprotocol.io/vega/datanode/admin"
	"code.vegaprotocol.io/vega/datanode/api"
	"code.vegaprotocol.io/vega/datanode/broker"
	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/datanode/gateway/server"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/datanode/networkhistory"
	"code.vegaprotocol.io/vega/datanode/networkhistory/snapshot"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/pprof"
	"code.vegaprotocol.io/vega/libs/subscribers"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"golang.org/x/sync/errgroup"
)

// NodeCommand use to implement 'node' command.
type NodeCommand struct {
	SQLSubscribers
	ctx    context.Context
	cancel context.CancelFunc

	embeddedPostgres              *embeddedpostgres.EmbeddedPostgres
	transactionalConnectionSource *sqlstore.ConnectionSource

	networkHistoryService *networkhistory.Service
	snapshotService       *snapshot.Service

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

func (l *NodeCommand) Run(ctx context.Context, cfgwatchr *config.Watcher, vegaPaths paths.Paths, args []string) error {
	l.configWatcher = cfgwatchr

	l.conf = cfgwatchr.Get()
	l.vegaPaths = vegaPaths
	if l.cancel != nil {
		l.cancel()
	}
	l.ctx, l.cancel = context.WithCancel(ctx)

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
	nodeLog := l.Log.Named("start.runNode")
	var eg *errgroup.Group
	eg, l.ctx = errgroup.WithContext(l.ctx)

	// gRPC server
	grpcServer := l.createGRPCServer(l.conf.API)

	// Admin server
	adminServer := admin.NewServer(l.Log, l.conf.Admin, l.vegaPaths, admin.NewNetworkHistoryAdminService(l.networkHistoryService))

	// watch configs
	l.configWatcher.OnConfigUpdate(
		func(cfg config.Config) {
			grpcServer.ReloadConf(cfg.API)
			adminServer.ReloadConf(cfg.Admin)
		},
	)

	// start the grpc server
	eg.Go(func() error { return grpcServer.Start(l.ctx, nil) })

	// start the admin server
	eg.Go(func() error {
		if err := adminServer.Start(l.ctx); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})

	// start gateway
	if l.conf.GatewayEnabled {
		gty := server.New(l.conf.Gateway, l.Log, l.vegaPaths)
		eg.Go(func() error { return gty.Start(l.ctx) })
	}

	eg.Go(func() error {
		return l.broker.Receive(l.ctx)
	})

	eg.Go(func() error {
		defer func() {
			if l.conf.NetworkHistory.Enabled {
				l.networkHistoryService.Stop()
			}
		}()

		return l.sqlBroker.Receive(l.ctx)
	})

	// waitSig will wait for a sigterm or sigint interrupt.
	eg.Go(func() error {
		gracefulStop := make(chan os.Signal, 1)
		signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT)

		select {
		case sig := <-gracefulStop:
			nodeLog.Info("Caught signal", logging.String("name", fmt.Sprintf("%+v", sig)))
			l.cancel()
		case <-l.ctx.Done():
			return l.ctx.Err()
		}
		return nil
	})

	metrics.Start(l.conf.Metrics)

	nodeLog.Info("Vega data node startup complete")

	if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		nodeLog.Error("Vega data node stopped with error", logging.Error(err))
		return fmt.Errorf("vega data node stopped with error: %w", err)
	}

	return nil
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
		l.networkHistoryService,
		l.coreSnapshotService,
		l.stopOrderService,
		l.fundingPeriodService,
		l.partyActivityStreakService,
		l.referralProgramService,
		l.referralSetsService,
		l.teamsService,
		l.vestingStatsService,
		l.feesStatsService,
		l.fundingPaymentService,
		l.volumeDiscountStatsService,
		l.volumeDiscountProgramService,
		l.paidLiquidityFeesStatsService,
		l.partyLockedBalancesService,
		l.partyVestingBalancesService,
		l.transactionResultsService,
		l.gamesService,
		l.marginModesService,
		l.ammPoolsService,
	)
	return grpcServer
}
