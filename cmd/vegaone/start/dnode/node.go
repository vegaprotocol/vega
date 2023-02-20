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
	"errors"
	"net/http"

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

// DN use to implement 'node' command.
type DN struct {
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

	eg *errgroup.Group
}

const namedLogger = "datanode"

func New(
	log *logging.Logger,
	vegaPaths paths.Paths,
) (*DN, error) {
	log = log.Named(namedLogger)

	confWatcher, err := config.NewWatcher(context.Background(), log, vegaPaths)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(ctx)

	dn := &DN{
		Log:           log,
		configWatcher: confWatcher,
		conf:          confWatcher.Get(),
		eg:            eg,
		ctx:           ctx,
		cancel:        cancel,
		vegaPaths:     vegaPaths,
	}

	if err := dn.persistentPre(nil); err != nil {
		return nil, err
	}

	if err := dn.preRun(nil); err != nil {
		return nil, err
	}

	return dn, nil
}

func (d *DN) Start() error {

	err := d.runNode(nil)
	if err != nil {
		d.cancel()
	}

	return nil
}

func (d *DN) Done() <-chan struct{} {
	return d.ctx.Done()
}

// Stop is for graceful shutdown.
func (d *DN) Stop() {
	d.cancel()
	err := d.eg.Wait()
	if !errors.Is(err, context.Canceled) {
		d.Log.Error("error with datanode shutdown", logging.Error(err))
	}

	if err := d.postRun([]string{}); err != nil {
		d.Log.Error("error with datanode shutdown", logging.Error(err))
	}

	if err := d.persistentPost([]string{}); err != nil {
		d.Log.Error("error with datanode shutdown", logging.Error(err))
	}
}

// runNode is the entry of node command.
func (l *DN) runNode([]string) error {
	nodeLog := l.Log.Named("start.runNode")

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
	l.eg.Go(func() error { return grpcServer.Start(l.ctx, nil) })

	// start the admin server
	l.eg.Go(func() error {
		if err := adminServer.Start(l.ctx); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})

	// start gateway
	if l.conf.GatewayEnabled {
		gty := server.New(l.conf.Gateway, l.Log, l.vegaPaths)
		l.eg.Go(func() error { return gty.Start(l.ctx) })
	}

	l.eg.Go(func() error {
		return l.broker.Receive(l.ctx)
	})

	l.eg.Go(func() error {
		defer func() {
			if l.conf.NetworkHistory.Enabled {
				l.networkHistoryService.Stop()
			}
		}()

		return l.sqlBroker.Receive(l.ctx)
	})

	metrics.Start(l.conf.Metrics)

	nodeLog.Info("Vega data node startup complete")

	return nil
}

func (l *DN) createGRPCServer(config api.Config) *api.GRPCServer {
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
	)
	return grpcServer
}
