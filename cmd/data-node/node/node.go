package node

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"code.vegaprotocol.io/data-node/api"

	"code.vegaprotocol.io/data-node/accounts"
	"code.vegaprotocol.io/data-node/assets"
	"code.vegaprotocol.io/data-node/broker"
	"code.vegaprotocol.io/data-node/candles"
	"code.vegaprotocol.io/data-node/checkpoint"
	"code.vegaprotocol.io/data-node/config"
	"code.vegaprotocol.io/data-node/delegations"
	"code.vegaprotocol.io/data-node/epochs"
	"code.vegaprotocol.io/data-node/fee"
	"code.vegaprotocol.io/data-node/gateway/server"
	"code.vegaprotocol.io/data-node/governance"
	"code.vegaprotocol.io/data-node/liquidity"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/markets"
	"code.vegaprotocol.io/data-node/metrics"
	"code.vegaprotocol.io/data-node/netparams"
	"code.vegaprotocol.io/data-node/nodes"
	"code.vegaprotocol.io/data-node/notary"
	"code.vegaprotocol.io/data-node/oracles"
	"code.vegaprotocol.io/data-node/orders"
	"code.vegaprotocol.io/data-node/parties"
	"code.vegaprotocol.io/data-node/plugins"
	"code.vegaprotocol.io/data-node/pprof"
	"code.vegaprotocol.io/data-node/risk"
	"code.vegaprotocol.io/data-node/sqlstore"
	"code.vegaprotocol.io/data-node/sqlsubscribers"
	"code.vegaprotocol.io/data-node/staking"
	"code.vegaprotocol.io/data-node/storage"
	"code.vegaprotocol.io/data-node/subscribers"
	"code.vegaprotocol.io/data-node/trades"
	"code.vegaprotocol.io/data-node/transfers"
	"code.vegaprotocol.io/data-node/vegatime"
	types "code.vegaprotocol.io/protos/vega"
	vegaprotoapi "code.vegaprotocol.io/protos/vega/api/v1"
	"code.vegaprotocol.io/shared/paths"

	"golang.org/x/sync/errgroup"
)

type AccountStore interface {
	accounts.AccountStore
	SaveBatch([]*types.Account) error
	Close() error
	ReloadConf(storage.Config)
}

type CandleStore interface {
	FetchLastCandle(marketID string, interval types.Interval) (*types.Candle, error)
	GenerateCandlesFromBuffer(marketID string, previousCandlesBuf map[string]types.Candle) error
	candles.CandleStore
	Close() error
	ReloadConf(storage.Config)
}

type OrderStore interface {
	orders.OrderStore
	SaveBatch([]types.Order) error
	Close() error
	ReloadConf(storage.Config)
}

type TradeStore interface {
	trades.TradeStore
	SaveBatch([]types.Trade) error
	Close() error
	ReloadConf(storage.Config)
}

// NodeCommand use to implement 'node' command.
type NodeCommand struct {
	ctx    context.Context
	cancel context.CancelFunc

	accounts              AccountStore
	candleStore           CandleStore
	orderStore            OrderStore
	marketStore           *storage.Market
	marketDataStore       *storage.MarketData
	tradeStore            TradeStore
	partyStore            *storage.Party
	riskStore             *storage.Risk
	transferResponseStore *storage.TransferResponse
	nodeStore             *storage.Node
	epochStore            *storage.Epoch
	delegationStore       *storage.Delegations
	checkpointStore       *storage.Checkpoints
	chainInfoStore        *storage.ChainInfo
	transferStore         *storage.Transfers

	sqlStore              *sqlstore.SQLStore
	assetStoreSQL         *sqlstore.Assets
	blockStoreSQL         *sqlstore.Blocks
	accountStoreSQL       *sqlstore.Accounts
	balanceStoreSQL       *sqlstore.Balances
	ledgerSQL             *sqlstore.Ledger
	partyStoreSQL         *sqlstore.Parties
	orderStoreSQL         *sqlstore.Orders
	tradeStoreSQL         *sqlstore.Trades
	networkLimitsStoreSQL *sqlstore.NetworkLimits
	marketDataStoreSQL    *sqlstore.MarketData
	rewardStoreSQL        *sqlstore.Rewards
	delegationStoreSQL    *sqlstore.Delegations
	marketsStoreSQL       *sqlstore.Markets
	epochStoreSQL         *sqlstore.Epochs
	depositStoreSQL       *sqlstore.Deposits
	proposalStoreSQL      *sqlstore.Proposals
	voteStoreSQL          *sqlstore.Votes

	vegaCoreServiceClient vegaprotoapi.CoreServiceClient

	broker    *broker.Broker
	sqlBroker broker.SqlStoreEventBroker

	transferRespSub      *subscribers.TransferResponse
	marketEventSub       *subscribers.MarketEvent
	orderSub             *subscribers.OrderEvent
	accountSub           *subscribers.AccountSub
	partySub             *subscribers.PartySub
	tradeSub             *subscribers.TradeSub
	marginLevelSub       *subscribers.MarginLevelSub
	governanceSub        *subscribers.GovernanceDataSub
	voteSub              *subscribers.VoteSub
	marketDataSub        *subscribers.MarketDataSub
	newMarketSub         *subscribers.Market
	marketUpdatedSub     *subscribers.MarketUpdated
	candleSub            *subscribers.CandleSub
	riskFactorSub        *subscribers.RiskFactorSub
	marketDepthSub       *subscribers.MarketDepthBuilder
	nodesSub             *subscribers.NodesSub
	delegationBalanceSub *subscribers.DelegationBalanceSub
	epochUpdateSub       *subscribers.EpochUpdateSub
	timeUpdateSub        *subscribers.Time
	rewardsSub           *subscribers.RewardCounters
	checkpointSub        *subscribers.CheckpointSub
	transferSub          *subscribers.TransferSub

	assetSubSQL            *sqlsubscribers.Asset
	timeSubSQL             *sqlsubscribers.Time
	transferResponseSubSQL *sqlsubscribers.TransferResponse
	orderSubSQL            *sqlsubscribers.Order
	networkLimitsSubSQL    *sqlsubscribers.NetworkLimits
	marketDataSubSQL       *sqlsubscribers.MarketData
	tradesSubSQL           *sqlsubscribers.TradeSubscriber
	rewardsSubSQL          *sqlsubscribers.Reward
	delegationsSubSQL      *sqlsubscribers.Delegation
	marketCreatedSubSQL    *sqlsubscribers.MarketCreated
	marketUpdatedSubSQL    *sqlsubscribers.MarketUpdated
	epochSubSQL            *sqlsubscribers.Epoch
	depositSubSQL          *sqlsubscribers.Deposit
	proposalsSubSQL        *sqlsubscribers.Proposal
	votesSubSQL            *sqlsubscribers.Vote

	candleService     *candles.Svc
	tradeService      *trades.Svc
	marketService     *markets.Svc
	orderService      *orders.Svc
	liquidityService  *liquidity.Svc
	partyService      *parties.Svc
	timeService       *vegatime.Svc
	accountsService   *accounts.Svc
	transfersService  *transfers.Svc
	riskService       *risk.Svc
	governanceService *governance.Svc
	notaryService     *notary.Svc
	assetService      *assets.Svc
	feeService        *fee.Svc
	eventService      *subscribers.Service
	netParamsService  *netparams.Service
	oracleService     *oracles.Service
	nodeService       *nodes.Service
	epochService      *epochs.Service
	delegationService *delegations.Service
	stakingService    *staking.Service
	checkpointSvc     *checkpoint.Svc

	pproffhandlr  *pprof.Pprofhandler
	Log           *logging.Logger
	vegaPaths     paths.Paths
	configWatcher *config.Watcher
	conf          config.Config

	// plugins
	settlePlugin     *plugins.Positions
	notaryPlugin     *plugins.Notary
	assetPlugin      *plugins.Asset
	withdrawalPlugin *plugins.Withdrawal
	depositPlugin    *plugins.Deposit

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
	grpcServer := l.createGRPCServer(l.conf.API, bool(l.conf.SQLStore.Enabled))

	// watch configs
	l.configWatcher.OnConfigUpdate(
		func(cfg config.Config) { grpcServer.ReloadConf(cfg.API) },
	)

	// start the grpc server
	eg.Go(func() error { return grpcServer.Start(ctx, nil) })

	if l.conf.SQLStore.Enabled && l.conf.API.ExposeLegacyAPI {
		l.Log.Info("Running legacy APIs", logging.Int("port offset", l.conf.API.LegacyAPIPortOffset))

		apiConfig := addLegacyPortOffsetToAPIPorts(l.conf.API, l.conf.API.LegacyAPIPortOffset)
		legacyGRPCServer := l.createGRPCServer(apiConfig, false)

		l.configWatcher.OnConfigUpdate(
			func(cfg config.Config) {
				legacyGRPCServer.ReloadConf(addLegacyPortOffsetToAPIPorts(cfg.API, l.conf.API.LegacyAPIPortOffset))
			},
		)

		eg.Go(func() error { return legacyGRPCServer.Start(ctx, nil) })
	}

	// start gateway
	if l.conf.GatewayEnabled {
		gty := server.New(l.conf.Gateway, l.Log, l.vegaPaths)

		eg.Go(func() error { return gty.Start(ctx) })

		if l.conf.SQLStore.Enabled && l.conf.API.ExposeLegacyAPI {
			legacyAPIGatewayConf := l.conf.Gateway
			legacyAPIGatewayConf.Node.Port = legacyAPIGatewayConf.Node.Port + l.conf.API.LegacyAPIPortOffset
			legacyAPIGatewayConf.GraphQL.Port = legacyAPIGatewayConf.GraphQL.Port + l.conf.API.LegacyAPIPortOffset
			legacyAPIGatewayConf.REST.Port = legacyAPIGatewayConf.REST.Port + l.conf.API.LegacyAPIPortOffset
			legacyGty := server.New(legacyAPIGatewayConf, l.Log, l.vegaPaths)
			eg.Go(func() error { return legacyGty.Start(ctx) })
		}
	}

	eg.Go(func() error {
		return l.broker.Receive(ctx)
	})

	if l.conf.SQLStore.Enabled {
		eg.Go(func() error {
			return l.sqlBroker.Receive(ctx)
		})
	}

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

func (l *NodeCommand) createGRPCServer(config api.Config, useSQLStores bool) *api.GRPCServer {
	grpcServer := api.NewGRPCServer(
		l.Log,
		config,
		useSQLStores,
		l.vegaCoreServiceClient,
		l.timeService,
		l.marketService,
		l.partyService,
		l.orderService,
		l.liquidityService,
		l.tradeService,
		l.candleService,
		l.accountsService,
		l.transfersService,
		l.riskService,
		l.governanceService,
		l.notaryService,
		l.assetService,
		l.feeService,
		l.eventService,
		l.oracleService,
		l.withdrawalPlugin,
		l.depositPlugin,
		l.marketDepthSub,
		l.netParamsService,
		l.nodeService,
		l.epochService,
		l.delegationService,
		l.rewardsSub,
		l.stakingService,
		l.checkpointSvc,
		l.balanceStoreSQL,
		l.orderStoreSQL,
		l.networkLimitsStoreSQL,
		l.marketDataStoreSQL,
		l.tradeStoreSQL,
		l.assetStoreSQL,
		l.accountStoreSQL,
		l.rewardStoreSQL,
		l.marketsStoreSQL,
		l.delegationStoreSQL,
		l.epochStoreSQL,
		l.depositStoreSQL,
		l.proposalStoreSQL,
		l.voteStoreSQL,
	)
	return grpcServer
}

func addLegacyPortOffsetToAPIPorts(original api.Config, portOffset int) api.Config {
	apiConfig := original
	apiConfig.WebUIPort = apiConfig.WebUIPort + portOffset
	apiConfig.Port = apiConfig.Port + portOffset
	return apiConfig
}
