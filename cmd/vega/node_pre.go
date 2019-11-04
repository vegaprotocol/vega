package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"code.vegaprotocol.io/vega/accounts"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/buffer"
	"code.vegaprotocol.io/vega/candles"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/markets"
	"code.vegaprotocol.io/vega/orders"
	"code.vegaprotocol.io/vega/parties"
	"code.vegaprotocol.io/vega/pprof"
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/storage"
	"code.vegaprotocol.io/vega/trades"
	"code.vegaprotocol.io/vega/transfers"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/pkg/errors"
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

	if flagProvided("--no-chain") {
		conf.Blockchain.ChainProvider = "noop"
	}

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

	if err = l.loadMarketsConfig(); err != nil {
		return err
	}

	// Set ulimits
	if err = l.SetUlimits(); err != nil {
		l.Log.Warn("Unable to set ulimits",
			logging.Error(err))
	} else {
		l.Log.Debug("Set ulimits",
			logging.Uint64("nofile", l.conf.UlimitNOFile))
	}

	l.stats = stats.New(l.Log, l.cli.version, l.cli.versionHash)

	// set up storage, this should be persistent
	if err := l.setupStorages(); err != nil {
		return err
	}
	l.setupBuffers()

	return nil
}

func (l *NodeCommand) loadMarketsConfig() error {
	pmkts := []proto.Market{}
	mktsCfg := l.conf.Execution.Markets
	// loads markets from configuration
	for _, v := range mktsCfg.Configs {
		path := filepath.Join(mktsCfg.Path, v)
		buf, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to read market configuration at %s", path))
		}

		mkt := proto.Market{}
		err = jsonpb.Unmarshal(strings.NewReader(string(buf)), &mkt)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to unmarshal market configuration at %s", path))
		}

		l.Log.Info("New market loaded from configuation",
			logging.String("market-config", path),
			logging.String("market-id", mkt.Id))
		pmkts = append(pmkts, mkt)
	}

	return nil
}

func (l *NodeCommand) setupBuffers() {
	l.orderBuf = buffer.NewOrder(l.orderStore)
	l.tradeBuf = buffer.NewTrade(l.tradeStore)
	l.partyBuf = buffer.NewParty(l.partyStore)
	l.transferBuf = buffer.NewTransferResponse(l.transferResponseStore)
	l.marketBuf = buffer.NewMarket(l.marketStore)
	l.accountBuf = buffer.NewAccount(l.accounts)
	l.candleBuf = buffer.NewCandle(l.candleStore)
}

func (l *NodeCommand) setupStorages() (err error) {
	if l.candleStore, err = storage.NewCandles(l.Log, l.conf.Storage); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.candleStore.ReloadConf(cfg.Storage) })

	if l.orderStore, err = storage.NewOrders(l.Log, l.conf.Storage, l.cancel); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.orderStore.ReloadConf(cfg.Storage) })

	if l.tradeStore, err = storage.NewTrades(l.Log, l.conf.Storage, l.cancel); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.tradeStore.ReloadConf(cfg.Storage) })

	if l.riskStore, err = storage.NewRisks(l.conf.Storage); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.riskStore.ReloadConf(cfg.Storage) })

	if l.marketStore, err = storage.NewMarkets(l.Log, l.conf.Storage); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.marketStore.ReloadConf(cfg.Storage) })

	if l.partyStore, err = storage.NewParties(l.conf.Storage); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.partyStore.ReloadConf(cfg.Storage) })

	if l.accounts, err = storage.NewAccounts(l.Log, l.conf.Storage); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.accounts.ReloadConf(cfg.Storage) })

	if l.transferResponseStore, err = storage.NewTransferResponses(l.Log, l.conf.Storage); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.transferResponseStore.ReloadConf(cfg.Storage) })

	return
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
	l.timeService = vegatime.New(l.conf.Time)

	// instanciate the execution engine
	l.executionEngine = execution.NewEngine(
		l.Log,
		l.conf.Execution,
		l.timeService,
		l.orderBuf,
		l.tradeBuf,
		l.candleBuf,
		l.marketBuf,
		l.partyBuf,
		l.accountBuf,
		l.transferBuf,
		l.mktscfg,
	)
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.executionEngine.ReloadConf(cfg.Execution) })

	// now instanciate the blockchain layer
	l.blockchain, err = blockchain.New(l.Log, l.conf.Blockchain, l.executionEngine, l.timeService, l.stats.Blockchain, l.cancel)
	if err != nil {
		return errors.Wrap(err, "unable to start the blockchain")
	}

	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.blockchain.ReloadConf(cfg.Blockchain) })

	// get the chain client as well.
	l.blockchainClient = l.blockchain.Client()

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
	l.accountsService = accounts.NewService(l.Log, l.conf.Accounts, l.accounts, l.blockchainClient)
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.accountsService.ReloadConf(cfg.Accounts) })
	l.transfersService = transfers.NewService(l.Log, l.conf.Transfers, l.transferResponseStore)
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.transfersService.ReloadConf(cfg.Transfers) })
	return
}

// SetUlimits sets limits (within OS-specified limits):
// * nofile - max number of open files - for badger LSM tree
func (l *NodeCommand) SetUlimits() error {
	return syscall.Setrlimit(syscall.RLIMIT_NOFILE, &syscall.Rlimit{
		Max: l.conf.UlimitNOFile,
		Cur: l.conf.UlimitNOFile,
	})
}
