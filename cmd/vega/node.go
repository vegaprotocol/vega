package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"code.vegaprotocol.io/vega/accounts"
	"code.vegaprotocol.io/vega/api"
	"code.vegaprotocol.io/vega/auth"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/buffer"
	"code.vegaprotocol.io/vega/candles"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/markets"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitoring"
	"code.vegaprotocol.io/vega/orders"
	"code.vegaprotocol.io/vega/parties"
	"code.vegaprotocol.io/vega/pprof"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/storage"
	"code.vegaprotocol.io/vega/trades"
	"code.vegaprotocol.io/vega/transfers"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NodeCommand use to implement 'node' command.
type NodeCommand struct {
	command

	ctx    context.Context
	cancel context.CancelFunc

	accounts              *storage.Account
	candleStore           *storage.Candle
	orderStore            *storage.Order
	marketStore           *storage.Market
	tradeStore            *storage.Trade
	partyStore            *storage.Party
	riskStore             *storage.Risk
	transferResponseStore *storage.TransferResponse

	orderBuf *buffer.Order
	tradeBuf *buffer.Trade

	candleService    *candles.Svc
	tradeService     *trades.Svc
	marketService    *markets.Svc
	orderService     *orders.Svc
	partyService     *parties.Svc
	timeService      *vegatime.Svc
	auth             *auth.Svc
	accountsService  *accounts.Svc
	transfersService *transfers.Svc

	blockchain       *blockchain.Blockchain
	blockchainClient *blockchain.Client

	pproffhandlr *pprof.Pprofhandler
	configPath   string
	conf         config.Config
	stats        *stats.Stats
	withPPROF    bool
	noChain      bool
	Log          *logging.Logger
	cfgwatchr    *config.Watcher

	executionEngine *execution.Engine
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
	flagSet.BoolVarP(&l.withPPROF, "with-pprof", "", false, "start the node with pprof support")
	flagSet.BoolVarP(&l.noChain, "no-chain", "", false, "start the node using the noop chain")
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
	if l.conf.Auth.Enabled {
		l.auth, err = auth.New(l.ctx, l.Log, l.conf.Auth)
		if err != nil {
			return errors.Wrap(err, "unable to start auth service")
		}
		l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.auth.ReloadConf(cfg.Auth) })
	}

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
		statusChecker,
	)
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { grpcServer.ReloadConf(cfg.API) })
	go grpcServer.Start()
	if l.conf.Auth.Enabled {
		l.auth.OnPartiesUpdated(grpcServer.OnPartiesUpdated)
	}
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
