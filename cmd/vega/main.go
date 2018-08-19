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
	"vega/datastore"
	"vega/log"
	"vega/api/endpoints/gql"
)

func main() {
	// Configuration and logging
	config := core.GetConfig()

	if err := initLogger(); err != nil {
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
	storage := &datastore.MemoryStoreProvider{}
	storage.Init([]string{"BTC/DEC18"}, []string{})

	// VEGA core
	vega := core.New(config, storage)
	vega.InitialiseMarkets()

	// Initialise concrete consumer services
	orderService := api.NewOrderService()
	tradeService := api.NewTradeService()
	orderService.Init(vega, storage.OrderStore())
	tradeService.Init(vega, storage.TradeStore())

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
}

func initLogger() error {
	// Load the os executable file location
	ex, err := os.Executable()
	if err != nil {
		return err
	}
	logFileName := filepath.Dir(ex) + "/vega-out.log"
	// delete log file each time ./vega is spun up
	// --> (this file gets HUGE quickly) when DEBUG level
	// todo: create new one for timestamp
	// t := time.Now()
	// fmt.Println(t.Format("20060102150405"))
	_, err = os.Stat(logFileName)
	if err == nil {
		err = os.Remove(logFileName)
		if err != nil {
			return err
		}
	}
	log.InitFileLogger(logFileName, log.InfoLevel)
	// todo: config set log level and console logger ON/OFF
	//log.InitConsoleLogger(log.InfoLevel)
	return nil
}
