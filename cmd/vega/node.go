package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"code.vegaprotocol.io/vega/internal"
	"code.vegaprotocol.io/vega/internal/api/endpoints/gql"
	"code.vegaprotocol.io/vega/internal/api/endpoints/grpc"
	"code.vegaprotocol.io/vega/internal/api/endpoints/restproxy"
	"code.vegaprotocol.io/vega/internal/blockchain"
	"code.vegaprotocol.io/vega/internal/candles"
	"code.vegaprotocol.io/vega/internal/execution"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/markets"
	"code.vegaprotocol.io/vega/internal/monitoring"
	"code.vegaprotocol.io/vega/internal/orders"
	"code.vegaprotocol.io/vega/internal/parties"
	"code.vegaprotocol.io/vega/internal/storage"
	"code.vegaprotocol.io/vega/internal/trades"
	"code.vegaprotocol.io/vega/internal/vegatime"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NodeCommand use to implement 'node' command.
type NodeCommand struct {
	command

	ctx   context.Context
	cfunc context.CancelFunc

	candleStore *storage.Candle
	orderStore  *storage.Order
	marketStore *storage.Market
	tradeStore  *storage.Trade
	partyStore  *storage.Party
	riskStore   *storage.Risk

	candleService *candles.Svc
	tradeService  *trades.Svc
	marketService *markets.Svc
	orderService  *orders.Svc
	partyService  *parties.Svc
	timeService   *vegatime.Svc

	blockchainClient *blockchain.Client

	configPath string
	conf       *internal.Config
	stats      *internal.Stats
	withPPROF  bool
	Log        *logging.Logger
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
}

// runNode is the entry of node command.
func (l *NodeCommand) runNode(args []string) error {
	defer l.cfunc()
	// check node_pre.go, that's where everything gets bootstrapped
	// Execution engine (broker operation at runtime etc)
	executionEngine := execution.NewEngine(
		l.conf.Execution,
		l.timeService,
		l.orderStore,
		l.tradeStore,
		l.candleStore,
		l.marketStore,
		l.partyStore,
	)

	// ABCI<>blockchain server
	bcService := blockchain.NewService(l.conf.Blockchain, l.stats.Blockchain, executionEngine, l.timeService)
	bcProcessor := blockchain.NewProcessor(l.conf.Blockchain, bcService)
	bcApp := blockchain.NewApplication(
		l.conf.Blockchain,
		l.stats.Blockchain,
		bcProcessor,
		bcService,
		l.timeService,
		l.cfunc,
	)
	socketServer := blockchain.NewServer(l.conf.Blockchain, l.stats.Blockchain, bcApp)
	if err := socketServer.Start(); err != nil {
		return errors.Wrap(err, "ABCI socket server error")
	}

	statusChecker := monitoring.New(l.conf.Monitoring, l.blockchainClient)
	statusChecker.OnChainDisconnect(l.cfunc)

	// gRPC server
	grpcServer := grpc.NewGRPCServer(
		l.conf.API,
		l.stats,
		l.blockchainClient,
		l.timeService,
		l.marketService,
		l.partyService,
		l.orderService,
		l.tradeService,
		l.candleService,
		statusChecker,
	)
	go grpcServer.Start()

	// REST<>gRPC (gRPC proxy) server
	restServer := restproxy.NewRestProxyServer(l.conf.API)
	go restServer.Start()

	// GraphQL server
	graphServer := gql.NewGraphQLServer(
		l.conf.API,
		l.orderService,
		l.tradeService,
		l.candleService,
		l.marketService,
		l.partyService,
		l.timeService,
		statusChecker,
	)
	go graphServer.Start()

	l.Log.Info("Vega startup complete")

	waitSig(l.ctx, l.Log)

	// Clean up and close resources
	restServer.Stop()
	grpcServer.Stop()
	graphServer.Stop()
	socketServer.Stop()
	statusChecker.Stop()

	return nil
}

// nodeExample shows examples for node command, and is used in auto-generated cli docs.
func nodeExample() string {
	return `$ vega node
VEGA started successfully`
}

// envConfigPath attempts to look at ENV variable VEGA_CONFIG for the config.toml path
func envConfigPath() string {
	err := viper.BindEnv("config")
	if err == nil {
		return viper.GetString("config")
	}
	return ""
}

// waitSig will wait for a sigterm or sigint interrupt.
func waitSig(ctx context.Context, log *logging.Logger) {
	var gracefulStop = make(chan os.Signal)
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
