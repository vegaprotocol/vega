package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"code.vegaprotocol.io/vega/accounts"
	"code.vegaprotocol.io/vega/api"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/buffer"
	"code.vegaprotocol.io/vega/candles"
	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/markets"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitoring"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/orders"
	"code.vegaprotocol.io/vega/parties"
	"code.vegaprotocol.io/vega/plugins"
	"code.vegaprotocol.io/vega/pprof"
	"code.vegaprotocol.io/vega/processor"
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/storage"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/trades"
	"code.vegaprotocol.io/vega/transfers"
	"code.vegaprotocol.io/vega/validators"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/spf13/cobra"
)

type AccountStore interface {
	buffer.AccountStore
	accounts.AccountStore
	Close() error
	ReloadConf(storage.Config)
}

type CandleStore interface {
	buffer.CandleStore
	candles.CandleStore
	Close() error
	ReloadConf(storage.Config)
}

type OrderStore interface {
	buffer.OrderStore
	orders.OrderStore
	GetMarketDepth(context.Context, string, uint64) (*proto.MarketDepth, error)
	Close() error
	ReloadConf(storage.Config)
}

type TradeStore interface {
	buffer.TradeStore
	trades.TradeStore
	Close() error
	ReloadConf(storage.Config)
}

// NodeCommand use to implement 'node' command.
type NodeCommand struct {
	command

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

	transferSub    *subscribers.TransferResponse
	marketEventSub *subscribers.MarketEvent
	orderSub       *subscribers.OrderEvent
	accountSub     *subscribers.AccountSub
	partySub       *subscribers.PartySub
	tradeSub       *subscribers.TradeSub
	marginLevelSub *subscribers.MarginLevelSub

	orderBuf        *buffer.Order
	tradeBuf        *buffer.Trade
	partyBuf        *buffer.Party
	marketBuf       *buffer.Market
	accountBuf      *buffer.Account
	candleBuf       *buffer.Candle
	marketDataBuf   *buffer.MarketData
	marginLevelsBuf *buffer.MarginLevels
	settleBuf       *buffer.Settlement
	lossSocBuf      *buffer.LossSocialization
	proposalBuf     *buffer.Proposal
	voteBuf         *buffer.Vote

	candleService     *candles.Svc
	tradeService      *trades.Svc
	marketService     *markets.Svc
	orderService      *orders.Svc
	partyService      *parties.Svc
	timeService       *vegatime.Svc
	accountsService   *accounts.Svc
	transfersService  *transfers.Svc
	riskService       *risk.Svc
	governanceService *governance.Svc

	blockchain       *blockchain.Blockchain
	blockchainClient *blockchain.Client

	pproffhandlr *pprof.Pprofhandler
	configPath   string
	conf         config.Config
	stats        *stats.Stats
	withPPROF    bool
	noChain      bool
	noStores     bool
	Log          *logging.Logger
	cfgwatchr    *config.Watcher

	executionEngine *execution.Engine
	processor       *processor.Processor
	governance      *governance.Engine
	collateral      *collateral.Engine

	mktscfg []proto.Market

	nodeWallet           *nodewallet.Service
	nodeWalletPassphrase string

	assets   *assets.Service
	topology *validators.Topology

	// plugins
	settlePlugin     *plugins.Positions
	governancePlugin *plugins.Governance
}

// Init initialises the node command.
func (l *NodeCommand) Init(c *Cli) {
	l.cli = c
	l.cmd = &cobra.Command{
		Use:               "node",
		Short:             "Run a new Vega node",
		Long:              "Run a new Vega node as defined by config files",
		Args:              cobra.MaximumNArgs(1),
		PersistentPreRunE: l.persistentPre,
		PreRunE:           l.preRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			return l.runNode(args)
		},
		PostRunE:          l.postRun,
		PersistentPostRun: l.persistentPost,
		Example:           nodeExample(),
	}
	l.addFlags()
}

// addFlags adds flags for specific command.
func (l *NodeCommand) addFlags() {
	flagSet := l.cmd.Flags()
	flagSet.StringVarP(&l.configPath, "config", "C", "", "file path to search for vega config file(s)")
	flagSet.StringVarP(&l.nodeWalletPassphrase, "nodewallet-passphrase", "p", "", "The path to a file containg the passphrase used to unlock the vega nodewallet, if not provided, prompt a password input")
	flagSet.BoolVarP(&l.withPPROF, "with-pprof", "", false, "start the node with pprof support")
	flagSet.BoolVarP(&l.noChain, "no-chain", "", false, "start the node using the noop chain")
	flagSet.BoolVarP(&l.noStores, "no-stores", "", false, "start the node without stores support")
}

// runNode is the entry of node command.
func (l *NodeCommand) runNode(args []string) error {
	defer l.cancel()

	statusChecker := monitoring.New(l.Log, l.conf.Monitoring, l.blockchainClient)
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { statusChecker.ReloadConf(cfg.Monitoring) })
	statusChecker.OnChainDisconnect(l.cancel)
	statusChecker.OnChainVersionObtained(func(v string) {
		l.stats.SetChainVersion(v)
	})

	var err error

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
		l.tradeService,
		l.candleService,
		l.accountsService,
		l.transfersService,
		l.riskService,
		l.governanceService,
		statusChecker,
	)
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { grpcServer.ReloadConf(cfg.API) })
	go grpcServer.Start()
	metrics.Start(l.conf.Metrics)

	// start gateway
	var gty *Gateway

	if l.conf.GatewayEnabled {
		gty, err = startGateway(l.Log, l.conf.Gateway)
		if err != nil {
			return err
		}
	}

	l.Log.Info("Vega startup complete")

	waitSig(l.ctx, l.Log)

	// Clean up and close resources
	grpcServer.Stop()
	l.blockchain.Stop()
	statusChecker.Stop()

	// cleanup gateway
	if l.conf.GatewayEnabled {
		if gty != nil {
			gty.stop()
		}
	}

	return nil
}

// nodeExample shows examples for node command, and is used in auto-generated cli docs.
func nodeExample() string {
	return `$ vega node
VEGA started successfully`
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
