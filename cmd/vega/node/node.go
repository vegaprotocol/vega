package node

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"code.vegaprotocol.io/vega/accounts"
	"code.vegaprotocol.io/vega/api"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/banking"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/candles"
	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/evtforward"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/fee"
	"code.vegaprotocol.io/vega/gateway/server"
	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/liquidity"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/markets"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitoring"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/notary"
	"code.vegaprotocol.io/vega/orders"
	"code.vegaprotocol.io/vega/parties"
	"code.vegaprotocol.io/vega/plugins"
	"code.vegaprotocol.io/vega/pprof"
	"code.vegaprotocol.io/vega/processor"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/storage"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/trades"
	"code.vegaprotocol.io/vega/transfers"
	"code.vegaprotocol.io/vega/validators"
	"code.vegaprotocol.io/vega/vegatime"
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

	broker *broker.Broker

	transferSub      *subscribers.TransferResponse
	marketEventSub   *subscribers.MarketEvent
	orderSub         *subscribers.OrderEvent
	accountSub       *subscribers.AccountSub
	partySub         *subscribers.PartySub
	tradeSub         *subscribers.TradeSub
	marginLevelSub   *subscribers.MarginLevelSub
	governanceSub    *subscribers.GovernanceDataSub
	voteSub          *subscribers.VoteSub
	marketDataSub    *subscribers.MarketDataSub
	newMarketSub     *subscribers.Market
	marketUpdatedSub *subscribers.MarketUpdated
	candleSub        *subscribers.CandleSub
	riskFactorSub    *subscribers.RiskFactorSub
	marketDepthSub   *subscribers.MarketDepthBuilder

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

	abciServer       *abci.Server
	blockchainClient *blockchain.Client

	pproffhandlr *pprof.Pprofhandler
	configPath   string
	conf         config.Config
	stats        *stats.Stats
	Log          *logging.Logger
	cfgwatchr    *config.Watcher

	executionEngine *execution.Engine
	governance      *governance.Engine
	collateral      *collateral.Engine
	oracles         *processor.Oracles
	netParams       *netparams.Store

	mktscfg []types.Market

	nodeWallet           *nodewallet.Service
	nodeWalletPassphrase string

	assets         *assets.Service
	topology       *validators.Topology
	notary         *notary.Notary
	evtfwd         *evtforward.EvtForwarder
	erc            *validators.ExtResChecker
	banking        *banking.Engine
	genesisHandler *genesis.Handler

	// plugins
	settlePlugin     *plugins.Positions
	notaryPlugin     *plugins.Notary
	assetPlugin      *plugins.Asset
	withdrawalPlugin *plugins.Withdrawal
	depositPlugin    *plugins.Deposit

	app *processor.App

	Version     string
	VersionHash string
}

func (l *NodeCommand) Run(cfgwatchr *config.Watcher, rootPath string, nodeWalletPassphrase string, args []string) error {
	l.cfgwatchr = cfgwatchr
	l.nodeWalletPassphrase = nodeWalletPassphrase

	l.conf, l.configPath = cfgwatchr.Get(), rootPath

	tmCfg := l.conf.Blockchain.Tendermint
	if tmCfg.ABCIRecordDir != "" && tmCfg.ABCIReplayFile != "" {
		return errors.New("you can't specify both abci-record and abci-replay flags")
	}

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
	defer func() {
		if err := l.nodeWallet.Cleanup(); err != nil {
			l.Log.Error("error cleaning up nodewallet", logging.Error(err))
		}
	}()

	statusChecker := monitoring.New(l.Log, l.conf.Monitoring, l.blockchainClient)
	statusChecker.OnChainDisconnect(l.cancel)
	statusChecker.OnChainVersionObtained(
		func(v string) { l.stats.SetChainVersion(v) },
	)

	// gRPC server
	grpcServer := api.NewGRPCServer(
		l.Log,
		l.conf.API,
		l.stats,
		l.blockchainClient,
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
		l.evtfwd,
		l.assetService,
		l.feeService,
		l.eventService,
		l.withdrawalPlugin,
		l.depositPlugin,
		l.marketDepthSub,
		l.netParamsService,
		statusChecker,
	)

	// watch configs
	l.cfgwatchr.OnConfigUpdate(
		func(cfg config.Config) { grpcServer.ReloadConf(cfg.API) },
		func(cfg config.Config) { statusChecker.ReloadConf(cfg.Monitoring) },
	)

	// start the grpc server
	go grpcServer.Start()
	metrics.Start(l.conf.Metrics)

	// start gateway
	var (
		gty *server.Server
	)
	if l.conf.GatewayEnabled {
		gty = server.New(l.conf.Gateway, l.Log)
		if err := gty.Start(); err != nil {
			return err
		}
	}

	l.Log.Info("Vega startup complete")
	waitSig(l.ctx, l.Log)

	// Clean up and close resources
	grpcServer.Stop()
	l.abciServer.Stop()
	statusChecker.Stop()

	// cleanup gateway
	if l.conf.GatewayEnabled {
		if gty != nil {
			gty.Stop()
		}
	}

	return nil
}

// waitSig will wait for a sigterm or sigint interrupt.
func waitSig(ctx context.Context, log *logging.Logger) {
	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	select {
	case sig := <-gracefulStop:
		log.Info("Caught signal", logging.String("name", fmt.Sprintf("%+v", sig)))
	case <-ctx.Done():
		// nothing to do
	}
}

func flagProvided(flag string) bool {
	for _, v := range os.Args[1:] {
		if v == flag {
			return true
		}
	}

	return false
}
