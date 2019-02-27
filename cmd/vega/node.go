package main

import (
	"vega/api/endpoints/gql"
	"vega/api/endpoints/grpc"
	"vega/api/endpoints/restproxy"
	"vega/internal"
	"vega/internal/blockchain"
	"vega/internal/execution"
	"vega/internal/logging"
	"vega/internal/matching"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// NodeCommand use to implement 'node' command.
type NodeCommand struct {
	command

	configPath string
	Log        *logging.Logger
}

// Init initialises the node command.
func (l *NodeCommand) Init(c *Cli) {
	l.cli = c
	l.cmd = &cobra.Command{
		Use:   "node",
		Short: "Run a new Vega node",
		Long:  "Run a new Vega node as defined by config files",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return l.runNode(args)
		},
		Example: nodeExample(),
	}
	l.addFlags()
}

// addFlags adds flags for specific command.
func (l *NodeCommand) addFlags() {
	flagSet := l.cmd.Flags()
	flagSet.StringVarP(&l.configPath, "configPath", "C", "", "file path to search for vega config file(s)")
}

// runNode is the entry of node command.
func (l *NodeCommand) runNode(args []string) error {

	// Set up configuration and create a resolver
	configPath := l.configPath
	if configPath == "" {
		configPath = DefaultVegaDir() // the root directory for vega config and data files
	}

	// VEGA config (holds all package level configs)
	conf, err := internal.ConfigFromFile(l.Log, configPath)
	if err != nil {
		// We revert to default configs if there are any errors in read/parse process
		l.Log.Error("Error reading config from file, using defaults", zap.Error(err))
		conf, err = internal.DefaultConfig(l.Log, DefaultVegaDir())
		if err != nil {
			return err
		}
	}
	conf.ListenForChanges()

	resolver, err := internal.NewResolver(conf)
	defer l.Log.Info("closing stores", zap.Error(resolver.CloseStores()))

	// Statistics provider
	stats := internal.NewStats(l.Log)

	// Resolve services for injection to servers/execution engine
	orderService, err := resolver.ResolveOrderService()
	if err != nil {
		return err
	}
	tradeService, err := resolver.ResolveTradeService()
	if err != nil {
		return err
	}
	candleService, err := resolver.ResolveCandleService()
	if err != nil {
		return err
	}
	marketService, err := resolver.ResolveMarketService()
	if err != nil {
		return err
	}
	timeService, err := resolver.ResolveTimeService()
	if err != nil {
		return err
	}
	orderStore, err := resolver.ResolveOrderStore()
	if err != nil {
		return err
	}
	tradeStore, err := resolver.ResolveTradeStore()
	if err != nil {
		return err
	}
	candleStore, err := resolver.ResolveCandleStore()
	if err != nil {
		return err
	}
	marketStore, err := resolver.ResolveMarketStore()
	if err != nil {
		return err
	}

	client, err := resolver.ResolveBlockchainClient()
	if err != nil {
		return err
	}

	// gRPC server
	grpcServer := grpc.NewGRPCServer(
		conf.API,
		stats,
		client,
		timeService,
		marketService,
		orderService,
		tradeService,
		candleService,
	)
	go grpcServer.Start()

	// REST<>gRPC (gRPC proxy) server
	restServer := restproxy.NewRestProxyServer(conf.API)
	go restServer.Start()

	// GraphQL server
	graphServer := gql.NewGraphQLServer(conf.API, orderService, tradeService, candleService, marketService, timeService)
	go graphServer.Start()

	// Execution engine (broker operation at runtime etc)
	matchingEngine := matching.NewMatchingEngine(conf.Matching)
	executionEngine := execution.NewExecutionEngine(
		conf.Execution,
		matchingEngine,
		timeService,
		orderStore,
		tradeStore,
		candleStore,
		marketStore,
	)

	// ABCI<>blockchain server
	socketServer := blockchain.NewServer(conf.Blockchain, stats.Blockchain, executionEngine, timeService)
	err = socketServer.Start()
	if err != nil {
		return errors.Wrap(err, "ABCI socket server error")
	}
	defer l.Log.Info("closing blockchain server", zap.Error(socketServer.Stop()))

	waitsig()

	return nil
}

// nodeExample shows examples for node command, and is used in auto-generated cli docs.
func nodeExample() string {
	return `$ vega node
VEGA started successfully`
}
