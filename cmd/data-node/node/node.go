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

package node

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"code.vegaprotocol.io/data-node/datanode/candlesv2"
	"code.vegaprotocol.io/data-node/datanode/service"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"

	"code.vegaprotocol.io/data-node/datanode/api"

	"code.vegaprotocol.io/data-node/datanode/broker"
	"code.vegaprotocol.io/data-node/datanode/config"
	"code.vegaprotocol.io/data-node/datanode/gateway/server"
	"code.vegaprotocol.io/data-node/datanode/metrics"
	"code.vegaprotocol.io/data-node/datanode/pprof"
	"code.vegaprotocol.io/data-node/datanode/sqlstore"
	"code.vegaprotocol.io/data-node/datanode/sqlsubscribers"
	"code.vegaprotocol.io/data-node/datanode/subscribers"
	"code.vegaprotocol.io/data-node/logging"
	vegaprotoapi "code.vegaprotocol.io/protos/vega/api/v1"
	"code.vegaprotocol.io/shared/paths"

	"golang.org/x/sync/errgroup"
)

// NodeCommand use to implement 'node' command.
type NodeCommand struct {
	ctx    context.Context
	cancel context.CancelFunc

	embeddedPostgres              *embeddedpostgres.EmbeddedPostgres
	transactionalConnectionSource *sqlstore.ConnectionSource

	// Stores
	assetStore               *sqlstore.Assets
	blockStore               *sqlstore.Blocks
	accountStore             *sqlstore.Accounts
	balanceStore             *sqlstore.Balances
	ledger                   *sqlstore.Ledger
	partyStore               *sqlstore.Parties
	orderStore               *sqlstore.Orders
	tradeStore               *sqlstore.Trades
	networkLimitsStore       *sqlstore.NetworkLimits
	marketDataStore          *sqlstore.MarketData
	rewardStore              *sqlstore.Rewards
	delegationStore          *sqlstore.Delegations
	marketsStore             *sqlstore.Markets
	epochStore               *sqlstore.Epochs
	depositStore             *sqlstore.Deposits
	withdrawalsStore         *sqlstore.Withdrawals
	proposalStore            *sqlstore.Proposals
	voteStore                *sqlstore.Votes
	marginLevelsStore        *sqlstore.MarginLevels
	riskFactorStore          *sqlstore.RiskFactors
	netParamStore            *sqlstore.NetworkParameters
	checkpointStore          *sqlstore.Checkpoints
	oracleSpecStore          *sqlstore.OracleSpec
	oracleDataStore          *sqlstore.OracleData
	liquidityProvisionStore  *sqlstore.LiquidityProvision
	positionStore            *sqlstore.Positions
	transfersStore           *sqlstore.Transfers
	stakeLinkingStore        *sqlstore.StakeLinking
	notaryStore              *sqlstore.Notary
	multiSigSignerAddedStore *sqlstore.ERC20MultiSigSignerEvent
	keyRotationsStore        *sqlstore.KeyRotations
	nodeStore                *sqlstore.Node
	candleStore              *sqlstore.Candles
	chainStore               *sqlstore.Chain

	// Services
	candleService             *candlesv2.Svc
	marketDepthService        *service.MarketDepth
	riskService               *service.Risk
	marketDataService         *service.MarketData
	positionService           *service.Position
	tradeService              *service.Trade
	ledgerService             *service.Ledger
	rewardService             *service.Reward
	delegationService         *service.Delegation
	assetService              *service.Asset
	blockService              *service.Block
	partyService              *service.Party
	accountService            *service.Account
	orderService              *service.Order
	networkLimitsService      *service.NetworkLimits
	marketsService            *service.Markets
	epochService              *service.Epoch
	depositService            *service.Deposit
	withdrawalService         *service.Withdrawal
	governanceService         *service.Governance
	riskFactorService         *service.RiskFactor
	networkParameterService   *service.NetworkParameter
	checkpointService         *service.Checkpoint
	oracleSpecService         *service.OracleSpec
	oracleDataService         *service.OracleData
	liquidityProvisionService *service.LiquidityProvision
	transferService           *service.Transfer
	stakeLinkingService       *service.StakeLinking
	notaryService             *service.Notary
	multiSigService           *service.MultiSig
	keyRotationsService       *service.KeyRotations
	nodeService               *service.Node
	chainService              *service.Chain

	vegaCoreServiceClient vegaprotoapi.CoreServiceClient

	broker    *broker.Broker
	sqlBroker broker.SqlStoreEventBroker

	accountSub             *sqlsubscribers.Account
	assetSub               *sqlsubscribers.Asset
	partySub               *sqlsubscribers.Party
	transferResponseSub    *sqlsubscribers.TransferResponse
	orderSub               *sqlsubscribers.Order
	networkLimitsSub       *sqlsubscribers.NetworkLimits
	marketDataSub          *sqlsubscribers.MarketData
	tradesSub              *sqlsubscribers.TradeSubscriber
	rewardsSub             *sqlsubscribers.Reward
	delegationsSub         *sqlsubscribers.Delegation
	marketCreatedSub       *sqlsubscribers.MarketCreated
	marketUpdatedSub       *sqlsubscribers.MarketUpdated
	epochSub               *sqlsubscribers.Epoch
	depositSub             *sqlsubscribers.Deposit
	withdrawalSub          *sqlsubscribers.Withdrawal
	proposalsSub           *sqlsubscribers.Proposal
	votesSub               *sqlsubscribers.Vote
	marginLevelsSub        *sqlsubscribers.MarginLevels
	riskFactorSub          *sqlsubscribers.RiskFactor
	netParamSub            *sqlsubscribers.NetworkParameter
	checkpointSub          *sqlsubscribers.Checkpoint
	oracleSpecSub          *sqlsubscribers.OracleSpec
	oracleDataSub          *sqlsubscribers.OracleData
	liquidityProvisionSub  *sqlsubscribers.LiquidityProvision
	positionsSub           *sqlsubscribers.Position
	transferSub            *sqlsubscribers.Transfer
	stakeLinkingSub        *sqlsubscribers.StakeLinking
	notarySub              *sqlsubscribers.Notary
	multiSigSignerEventSub *sqlsubscribers.ERC20MultiSigSignerEvent
	keyRotationsSub        *sqlsubscribers.KeyRotation
	nodeSub                *sqlsubscribers.Node
	marketDepthSub         *sqlsubscribers.MarketDepth

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

// runNode is the entry of node command.
func (l *NodeCommand) runNode(args []string) error {
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
		l.nodeService,
		l.marketDepthService,
		l.ledgerService,
	)
	return grpcServer
}
