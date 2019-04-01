package main

import (
	"context"
	"strings"
	"time"

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
	Log        *logging.Logger
}

type errStack []error

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
}

func (l *NodeCommand) persistentPre(_ *cobra.Command, args []string) error {
	// this shouldn't happen...
	if l.cfunc != nil {
		l.cfunc()
	}
	l.ctx, l.cfunc = context.WithCancel(context.Background())
	// Use configPath from args
	configPath := l.configPath
	if configPath == "" {
		// Use configPath from ENV
		configPath = envConfigPath()
		if configPath == "" {
			// Default directory ($HOME/.vega)
			configPath = defaultVegaDir()
		}
	}

	l.Log.Info("Config path", logging.String("config-path", configPath))

	// VEGA config (holds all package level configs)
	conf, err := internal.NewConfigFromFile(l.Log, configPath)
	if err != nil {
		// We revert to default configs if there are any errors in read/parse process
		l.Log.Error("Error reading config from file, using defaults", logging.Error(err))
		if conf, err = internal.NewDefaultConfig(l.Log, defaultVegaDir()); err != nil {
			// cancel context here
			l.cfunc()
			return err
		}
	} else {
		conf.ListenForChanges()
	}
	// assign config vars
	l.configPath, l.conf = configPath, conf
	l.stats = internal.NewStats(l.Log, l.cli.version, l.cli.versionHash)
	return nil
}

func (l *NodeCommand) postRun(_ *cobra.Command, _ []string) error {
	var werr errStack
	if l.candleStore != nil {
		if err := l.candleStore.Close(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing candle store in command."))
		}
	}
	if l.riskStore != nil {
		if err := l.riskStore.Close(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing risk store in command."))
		}
	}
	if l.tradeStore != nil {
		if err := l.tradeStore.Close(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing trade store in command."))
		}
	}
	if l.orderStore != nil {
		if err := l.orderStore.Close(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing order store in command."))
		}
	}
	if l.marketStore != nil {
		if err := l.marketStore.Close(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing market store in command."))
		}
	}
	if l.partyStore != nil {
		if err := l.partyStore.Close(); err != nil {
			werr = append(werr, errors.Wrap(err, "error closing party store in command."))
		}
	}
	return werr
}

func (l *NodeCommand) persistentPost(_ *cobra.Command, _ []string) {
	l.cfunc()
}

// we've already set everything up WRT arguments etc... just bootstrap the node
func (l *NodeCommand) preRun(_ *cobra.Command, _ []string) (err error) {
	// ensure that context is cancelled if we return an error here
	defer func() {
		if err != nil {
			l.cfunc()
		}
	}()
	// set up storage
	if l.candleStore, err = storage.NewCandles(l.conf.Storage); err != nil {
		return
	}
	if l.orderStore, err = storage.NewOrders(l.conf.Storage, l.cfunc); err != nil {
		return
	}
	if l.tradeStore, err = storage.NewTrades(l.conf.Storage, l.cfunc); err != nil {
		return
	}
	if l.riskStore, err = storage.NewRisks(l.conf.Storage); err != nil {
		return
	}
	if l.marketStore, err = storage.NewMarkets(l.conf.Storage); err != nil {
		return
	}
	if l.partyStore, err = storage.NewParties(l.conf.Storage); err != nil {
		return
	}
	// this doesn't fail
	l.timeService = vegatime.NewService(l.conf.Time)
	if l.blockchainClient, err = blockchain.NewClient(l.conf.Blockchain); err != nil {
		return
	}
	// start services
	if l.candleService, err = candles.NewService(l.conf.Candles, l.candleStore); err != nil {
		return
	}
	if l.orderService, err = orders.NewService(l.conf.Orders, l.orderStore, l.timeService, l.blockchainClient); err != nil {
		return
	}
	if l.tradeService, err = trades.NewService(l.conf.Trades, l.tradeStore, l.riskStore); err != nil {
		return
	}
	if l.marketService, err = markets.NewService(l.conf.Markets, l.marketStore, l.orderStore); err != nil {
		return
	}
	// last assignment to err, no need to check here, if something went wrong, we'll know about it
	l.partyService, err = parties.NewService(l.conf.Parties, l.partyStore)
	return
}

// runNode is the entry of node command.
func (l *NodeCommand) runNode(args []string) error {
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

	statusChecker := monitoring.NewStatusChecker(l.Log, l.blockchainClient, 500*time.Millisecond)
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

	waitSig(l.ctx)
	l.cfunc()

	// Clean up and close resources
	l.Log.Info("Closing REST proxy server", logging.Error(restServer.Stop()))
	l.Log.Info("Closing GRPC server", logging.Error(grpcServer.Stop()))
	l.Log.Info("Closing GraphQL server", logging.Error(graphServer.Stop()))
	l.Log.Info("Closing blockchain server", logging.Error(socketServer.Stop()))
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

// Error - implement the error interface on the errStack type
func (e errStack) Error() string {
	s := make([]string, 0, len(e))
	for _, err := range e {
		s = append(s, err.Error())
	}
	return strings.Join(s, "\n")
}
