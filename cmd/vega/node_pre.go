// This file contains the pre-run hooks for the command. It's where all the stuff gets bootstrapped, basically
package main

import (
	"context"

	"code.vegaprotocol.io/vega/internal"
	"code.vegaprotocol.io/vega/internal/blockchain"
	"code.vegaprotocol.io/vega/internal/candles"
	"code.vegaprotocol.io/vega/internal/fsutil"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/markets"
	"code.vegaprotocol.io/vega/internal/orders"
	"code.vegaprotocol.io/vega/internal/parties"
	"code.vegaprotocol.io/vega/internal/pprof"
	"code.vegaprotocol.io/vega/internal/storage"
	"code.vegaprotocol.io/vega/internal/trades"
	"code.vegaprotocol.io/vega/internal/vegatime"

	"github.com/spf13/cobra"
)

func (l *NodeCommand) persistentPre(_ *cobra.Command, args []string) (err error) {
	// this shouldn't happen...
	if l.cfunc != nil {
		l.cfunc()
	}
	// ensure we cancel the context on error
	defer func() {
		if err != nil {
			l.cfunc()
		}
	}()
	l.ctx, l.cfunc = context.WithCancel(context.Background())
	// Use configPath from args
	configPath := l.configPath
	if configPath == "" {
		// Use configPath from ENV
		configPath = envConfigPath()
		if configPath == "" {
			// Default directory ($HOME/.vega)
			configPath = fsutil.DefaultVegaDir()
		}
	}

	// VEGA config (holds all package level configs)
	conf, err := internal.NewConfigFromFile(l.Log, configPath)
	if err != nil {
		// We revert to default configs if there are any errors in read/parse process
		l.Log.Error("Error reading config from file, using defaults", logging.Error(err))
		if conf, err = internal.NewDefaultConfig(l.Log, fsutil.DefaultVegaDir()); err != nil {
			// cancel context here
			return err
		}
	} else {
		conf.ListenForChanges()
	}
	l.Log = conf.GetLogger()

	if flagProvided("--with-pprof") || conf.Pprof.Enabled {
		l.Log.Info("vega is starting with pprof profile, this is not a recommended setting for production")
		l.pproffhandlr, err = pprof.New(conf.Pprof)
		if err != nil {
			return
		}
	}

	l.Log.Info("Starting Vega",
		logging.String("config-path", configPath),
		logging.String("version", Version),
		logging.String("version-hash", VersionHash))

	// assign config vars
	l.configPath, l.conf = configPath, conf
	l.stats = internal.NewStats(l.Log, l.cli.version, l.cli.versionHash)
	// set up storage, this should be persistent
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
	return nil
}

// we've already set everything up WRT arguments etc... just bootstrap the node
func (l *NodeCommand) preRun(_ *cobra.Command, _ []string) (err error) {
	// ensure that context is cancelled if we return an error here
	defer func() {
		if err != nil {
			l.cfunc()
		}
	}()
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
