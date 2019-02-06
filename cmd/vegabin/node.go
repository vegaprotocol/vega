package main

import (
	"vega/api"
	"vega/api/endpoints/gql"
	"vega/api/endpoints/grpc"
	"vega/api/endpoints/restproxy"
	
	"vega/internal/blockchain"
	"vega/internal/candles"
	"vega/internal/execution"
	"vega/internal/logging"
	"vega/internal/matching"
	"vega/internal/orders"
	"vega/internal/storage"
	"vega/internal/trades"
	"vega/internal/vegatime"

	"github.com/spf13/cobra"
)

// NodeCommand use to implement 'node' command.
type NodeCommand struct {
	command

	//username string
	//password string
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
	//	l.addFlags()
}

//// addFlags adds flags for specific command.
//func (l *NodeCommand) addFlags() {
//	flagSet := l.cmd.Flags()
//
//	flagSet.StringVarP(&l.username, "username", "u", "", "username for vega")
//	flagSet.StringVarP(&l.password, "password", "p", "", "password for vega")
//}

// runNode is the entry of node command.
func (l *NodeCommand) runNode(args []string) error {




	
	logger := logging.NewLogger()

	if l.cli.Option.Debug {
		level := logging.DebugLevel
		logger.InitConsoleLogger(level)
		logger.Infof("Starting up VEGA node with logging at DEBUG level")
	} else {
		level := logging.InfoLevel
		logger.InitConsoleLogger(level)
		logger.Infof("Starting up VEGA node with logging at INFO level")
	}

	logger.AddExitHandler()

	//var logLevelFlag string
	//flag.StringVar(&logLevelFlag, "log", "info", "pass log level: debug, info, error, fatal")
	////flag.BoolVar(&config.LogPriceLevels, "log_price_levels", false, "if true log price levels")
	//flag.Parse()

	storeConfig := storage.NewConfig()

	orderStore, err := storage.NewOrderStore(storeConfig)
	if err != nil {
		// todo log fatal?
		return err
	}
	defer orderStore.Close()

	tradeStore, err := storage.NewTradeStore(storeConfig)
	if err != nil {
		// todo log fatal?
		return err
	}
	defer tradeStore.Close()

	candleStore, err := storage.NewCandleStore(storeConfig)
	if err != nil {
		// todo log fatal?
		return err
	}
	defer candleStore.Close()

	partyStore, err := storage.NewPartyStore(storeConfig)
	if err != nil {
		// todo log fatal?
		return err
	}
	defer partyStore.Close()

	marketStore, err := storage.NewMarketStore(storeConfig)
	if err != nil {
		// todo log fatal?
		return err
	}
	defer marketStore.Close()

	riskStore, err := storage.NewRiskStore(storeConfig)
	if err != nil {
		// todo log fatal?
		return err
	}
	defer riskStore.Close()

	vtc := vegatime.NewConfig()
	timeService := vegatime.NewTimeService(vtc)

	orderService := orders.NewOrderService(orderStore, timeService)
	tradeService := trades.NewTradeService(tradeStore, riskStore)
	candleService := candles.NewCandleService(candleStore)
	//partyService := parties.NewService(partyStore)
	//marketService := markets.NewService(marketStore)

	apiConfig := api.NewConfig()

	// gRPC server
	// Port 3002
	grpcServer := grpc.NewGRPCServer(apiConfig, orderService, tradeService, candleService)
	go grpcServer.Start()

	// REST<>gRPC (gRPC proxy) server
	// Port 3003
	restServer := restproxy.NewRestProxyServer(apiConfig)
	go restServer.Start()

	// GraphQL server
	// Port 3004
	graphServer := gql.NewGraphQLServer(apiConfig, orderService, tradeService, candleService)
	go graphServer.Start()

	// Matching engine (todo) create these inside execution engine will be coupled to vega commands
	matchingEngine := matching.NewMatchingEngine(false)
	matchingEngine.CreateMarket("BTC/DEC19")

	// Execution engine (broker operation of markets at runtime etc)
	executionEngine := execution.NewExecutionEngine(matchingEngine, timeService, orderStore, tradeStore)

	// ABCI socket server
	// Port 46658
	socketServer := blockchain.NewServer(executionEngine, timeService)
	if err := socketServer.Start(); err != nil {
		logger.Fatalf("ABCI socket server fatal error: %s", err)
	}

	return nil
}

// nodeExample shows examples for node command, and is used in auto-generated cli docs.
func nodeExample() string {
	return `$ vega node
VEGA started successfully`
}
