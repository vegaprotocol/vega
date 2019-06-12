// This file contains the pre-run hooks for the command. It's where all the stuff gets bootstrapped, basically
package main

import (
	"context"
	"os"

	"code.vegaprotocol.io/vega/internal"
	"code.vegaprotocol.io/vega/internal/accounts"
	"code.vegaprotocol.io/vega/internal/blockchain"
	"code.vegaprotocol.io/vega/internal/candles"
	"code.vegaprotocol.io/vega/internal/config"
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

func envConfigPath() string {
	return os.Getenv("VEGA_CONFIG")
}

func (l *NodeCommand) persistentPre(_ *cobra.Command, args []string) (err error) {
	// this shouldn't happen...
	if l.cancel != nil {
		l.cancel()
	}
	// ensure we cancel the context on error
	defer func() {
		if err != nil {
			l.cancel()
		}
	}()
	l.ctx, l.cancel = context.WithCancel(context.Background())
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
	l.configPath = configPath

	// VEGA config (holds all package level configs)
	cfgwatchr, err := config.NewFromFile(l.ctx, l.Log, configPath, configPath)
	if err != nil {
		l.Log.Error("unable to start config watcher", logging.Error(err))
		return
	}
	conf := cfgwatchr.Get()
	l.cfgwatchr = cfgwatchr

	// reload logger with the setup from configuration
	l.Log = logging.NewLoggerFromConfig(conf.Logging)

	if flagProvided("--with-pprof") || conf.Pprof.Enabled {
		l.Log.Info("vega is starting with pprof profile, this is not a recommended setting for production")
		l.pproffhandlr, err = pprof.New(l.Log, conf.Pprof)
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
	if l.candleStore, err = storage.NewCandles(l.Log, l.conf.Candles); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.candleStore.ReloadConf(cfg.Candles) })

	if l.orderStore, err = storage.NewOrders(l.Log, l.conf.Orders, l.cancel); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.orderStore.ReloadConf(cfg.Orders) })

	if l.tradeStore, err = storage.NewTrades(l.Log, l.conf.Trades, l.cancel); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.tradeStore.ReloadConf(cfg.Trades) })

	if l.riskStore, err = storage.NewRisks(l.conf.Risk); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.riskStore.ReloadConf(cfg.Risk) })

	if l.marketStore, err = storage.NewMarkets(l.Log, l.conf.Markets); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.marketStore.ReloadConf(cfg.Markets) })

	if l.partyStore, err = storage.NewParties(l.conf.Parties); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.partyStore.ReloadConf(cfg.Parties) })

	if l.accounts, err = storage.NewAccounts(l.Log, l.conf.Accounts); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.accounts.ReloadConf(cfg.Accounts) })

	return nil
}

// we've already set everything up WRT arguments etc... just bootstrap the node
func (l *NodeCommand) preRun(_ *cobra.Command, _ []string) (err error) {
	// ensure that context is cancelled if we return an error here
	defer func() {
		if err != nil {
			l.cancel()
		}
	}()
	// this doesn't fail
	l.timeService = vegatime.NewService(l.conf.Time)
	if l.blockchainClient, err = blockchain.NewClient(&l.conf.Blockchain); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.timeService.ReloadConf(cfg.Time) })

	// start services
	if l.candleService, err = candles.NewService(l.Log, l.conf.Candles, l.candleStore); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.candleService.ReloadConf(cfg.Candles) })
	if l.orderService, err = orders.NewService(l.Log, l.conf.Orders, l.orderStore, l.timeService, l.blockchainClient); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.orderService.ReloadConf(cfg.Orders) })

	if l.tradeService, err = trades.NewService(l.Log, l.conf.Trades, l.tradeStore, l.riskStore); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.tradeService.ReloadConf(cfg.Trades) })

	if l.marketService, err = markets.NewService(l.Log, l.conf.Markets, l.marketStore, l.orderStore); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.marketService.ReloadConf(cfg.Markets) })

	// last assignment to err, no need to check here, if something went wrong, we'll know about it
	l.partyService, err = parties.NewService(l.Log, l.conf.Parties, l.partyStore)
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.partyService.ReloadConf(cfg.Parties) })
	l.accountsService = accounts.NewService(l.Log, l.conf.Accounts, l.accounts)
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.accountsService.ReloadConf(cfg.Accounts) })
	return
}
