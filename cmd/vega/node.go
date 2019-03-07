package main

import (
	"code.vegaprotocol.io/vega/internal"
	"code.vegaprotocol.io/vega/internal/api/endpoints/gql"
	"code.vegaprotocol.io/vega/internal/api/endpoints/grpc"
	"code.vegaprotocol.io/vega/internal/api/endpoints/restproxy"
	"code.vegaprotocol.io/vega/internal/appstatus"
	"code.vegaprotocol.io/vega/internal/blockchain"
	"code.vegaprotocol.io/vega/internal/execution"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/matching"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	flagSet.StringVarP(&l.configPath, "config", "C", "", "file path to search for vega config file(s)")
}

// runNode is the entry of node command.
func (l *NodeCommand) runNode(args []string) error {

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
		defaultConf, err := internal.NewDefaultConfig(l.Log, defaultVegaDir())
		if err != nil {
			return err
		}
		conf = defaultConf
	} else {
		conf.ListenForChanges()
	}

	resolver, err := internal.NewResolver(conf)

	// Statistics provider
	stats := internal.NewStats(l.Log, l.cli.version, l.cli.versionHash)

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
	partyService, err := resolver.ResolvePartyService()
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

	appst := appstatus.New(l.Log, client)

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
		appst,
	)
	go grpcServer.Start()

	// REST<>gRPC (gRPC proxy) server
	restServer := restproxy.NewRestProxyServer(conf.API)
	go restServer.Start()

	// GraphQL server
	graphServer := gql.NewGraphQLServer(
		conf.API, orderService, tradeService, candleService, marketService,
		partyService, timeService, appst)
	go graphServer.Start()

	waitSig()

	// Clean up and close resources
	l.Log.Info("Closing REST proxy server", logging.Error(restServer.Stop()))
	l.Log.Info("Closing GRPC server", logging.Error(grpcServer.Stop()))
	l.Log.Info("Closing GraphQL server", logging.Error(graphServer.Stop()))
	l.Log.Info("Closing blockchain server", logging.Error(socketServer.Stop()))
	l.Log.Info("Closing stores", logging.Error(resolver.CloseStores()))
	appst.Stop()

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
