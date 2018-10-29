// cmd/vega/main.go
package main

import (
	"os"
	"path/filepath"

	"vega/api"
	"vega/api/endpoints/grpc"
	"vega/api/endpoints/restproxy"
	"vega/blockchain"
	"vega/core"
	"vega/log"
	"vega/api/endpoints/gql"
	"time"
	"fmt"
	"flag"
)

func main() {
	// Configuration and logging
	config := core.GetConfig()

	// flags
	var logLevelFlag string
	flag.StringVar(&logLevelFlag, "log", "info", "pass log level: debug, info, error, fatal")
	flag.BoolVar(&config.RemoveExpiredGTTOrders, "remove_expired_gtt", false, "if true expired GTT orders are saved to data stores")
	flag.BoolVar(&config.LogPriceLevels, "log_price_levels", false, "if true log price levels")
	flag.Parse()

	if err := initLogger(logLevelFlag); err != nil {
		log.Fatalf("%s", err)
	}

	// todo read from something like gitlab
	config.AppVersion = "0.1.927"
	config.AppVersionHash = "d6cd1e2bd19e03a81132a23b2025920577f84e37"
	appVersion := os.Getenv("APP_VERSION")
	appVersionHash := os.Getenv("APP_VERSION_HASH")
	if appVersion != "" && appVersionHash != "" {
		config.AppVersion = appVersion
		config.AppVersionHash = appVersionHash
	}

	// Storage Service provides read stores for consumer VEGA API
	// Uses in memory storage (maps/slices etc), configurable in future
	//storage := &datastore.MemoryStoreProvider{}
	//storage.Init([]string{"BTC/DEC18"}, []string{})

	// VEGA core
	vega := core.New(config)
	vega.InitialiseMarkets()

	// Initialise concrete consumer services
	orderService := api.NewOrderService()
	tradeService := api.NewTradeService()
	orderService.Init(vega)
	tradeService.Init(vega)

	// GRPC server
	// Port 3002
	grpcServer := grpc.NewGRPCServer(orderService, tradeService)
	go grpcServer.Start()

	// REST<>GRPC (gRPC proxy) server
	// Port 3003
	restServer := restproxy.NewRestProxyServer()
	go restServer.Start()

	// GraphQL server (using new production quality gQL)
	// Port 3004
	graphServer := gql.NewGraphQLServer(orderService, tradeService)
	go graphServer.Start()

	// ABCI socket server
	// Port 46658
	if err := blockchain.Start(vega); err != nil {
		log.Fatalf("%s", err)
	}

	orderService.Stop()
}

func initLogger(levelStr string) error {
	level := parseLogLevel(levelStr)

	// Load the os executable file location
	ex, err := os.Executable()
	if err != nil {
		return err
	}
	t := time.Now()
	logFileName := filepath.Dir(ex) + "/vega-" + t.Format("20060102150405") + ".log"
	fmt.Println(logFileName)

	_, err = os.Stat(logFileName)
	if err == nil {
		err = os.Remove(logFileName)
		if err != nil {
			return err
		}
	}

	log.InitFileLogger(logFileName, level)
	// todo: config set log level and console logger ON/OFF
	//log.InitConsoleLogger(log.DebugLevel)
	return nil
}

func parseLogLevel(level string) log.Level {
	switch(level) {
	case "debug":
		return log.DebugLevel
	case "info":
		return log.InfoLevel
	case "error":
		return log.ErrorLevel
	case "fatal":
		return log.FatalLevel
	}
	return log.InfoLevel
}
