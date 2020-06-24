package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"code.vegaprotocol.io/vega/accounts"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/buffer"
	"code.vegaprotocol.io/vega/candles"
	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/fsutil"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/markets"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/orders"
	"code.vegaprotocol.io/vega/parties"
	"code.vegaprotocol.io/vega/plugins"
	"code.vegaprotocol.io/vega/pprof"
	"code.vegaprotocol.io/vega/processor"
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/storage"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/trades"
	"code.vegaprotocol.io/vega/transfers"
	"code.vegaprotocol.io/vega/validators"
	"code.vegaprotocol.io/vega/vegatime"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/cenkalti/backoff/v4"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	uuid "github.com/satori/go.uuid"
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

	if flagProvided("--no-stores") {
		conf.StoresEnabled = false
	}

	// if theses is not specified, we then trigger a prompt
	// for the user to type his password
	var nodeWalletPassphrase string
	if len(l.nodeWalletPassphrase) <= 0 {
		nodeWalletPassphrase, err = getTerminalPassphrase()
	} else {
		nodeWalletPassphrase, err = getFilePassphrase(l.nodeWalletPassphrase)
	}
	if err != nil {
		return fmt.Errorf("cannot start the node, passphrase error: %v", err)
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

	// this doesn't fail
	l.timeService = vegatime.New(l.conf.Time)

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
	l.setupSubscibers()

	if !l.conf.StoresEnabled {
		l.Log.Info("node setted up without badger store support")
	} else {
		l.Log.Info("node setted up with badger store support")
	}

	// instanciate the ETHClient
	ethclt, err := ethclient.Dial(l.conf.NodeWallet.ETH.Address)
	if err != nil {
		return err
	}

	// nodewallet
	l.nodeWallet, err = nodewallet.New(l.Log, l.conf.NodeWallet, nodeWalletPassphrase, ethclt)
	if err != nil {
		return err
	}

	// ensure all require wallet are available
	err = l.nodeWallet.EnsureRequireWallets()
	if err != nil {
		return err
	}

	// initialize the assets service now
	l.assets, err = assets.New(l.Log, l.conf.Assets, l.nodeWallet, l.timeService)
	if err != nil {
		return err
	}

	// TODO: remove that once we have infrastructure to use token through governance
	assetSrcs, err := assets.LoadDevAssets(l.conf.Assets)
	if err != nil {
		return err
	}

	for _, v := range assetSrcs {
		v := v
		aid, err := l.assets.NewAsset(uuid.NewV4().String(), v)
		if err != nil {
			return fmt.Errorf("error instanciating asset %v\n", err)
		}

		asset, err := l.assets.Get(aid)
		if err != nil {
			return fmt.Errorf("unable to get asset %v\n", err)
		}

		// just a simple backoff here
		err = backoff.Retry(
			func() error {
				err := asset.Validate()
				if !asset.IsValid() {
					return err
				}
				return nil
			},
			backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5),
		)
		if err != nil {
			return fmt.Errorf("unable to instanciate new asset %v", err)
		}
		if err := l.assets.Enable(aid); err != nil {
			return fmt.Errorf("unable to enable asset: %v", err)
		}
		l.Log.Info("new asset added successfully",
			logging.String("asset", asset.String()))
	}

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
	l.mktscfg = pmkts

	return nil
}

func (l *NodeCommand) setupSubscibers() {
	l.transferSub = subscribers.NewTransferResponse(l.ctx, l.transferResponseStore)
	l.marketEventSub = subscribers.NewMarketEvent(l.ctx, l.Log)
	l.orderSub = subscribers.NewOrderEvent(l.ctx, l.Log, l.orderStore)
	l.accountSub = subscribers.NewAccountSub(l.ctx, l.accounts)
	l.partySub = subscribers.NewPartySub(l.ctx, l.partyStore)
	l.tradeSub = subscribers.NewTradeSub(l.ctx, l.tradeStore)
	l.marginLevelSub = subscribers.NewMarginLevelSub(l.ctx, l.riskStore)
	l.governanceSub = subscribers.NewGovernanceDataSub(l.ctx)
	l.voteSub = subscribers.NewVoteSub(l.ctx, false)
}

func (l *NodeCommand) setupBuffers() {
	l.orderBuf = buffer.NewOrder(l.orderStore)
	l.tradeBuf = buffer.NewTrade(l.tradeStore)
	l.partyBuf = buffer.NewParty(l.partyStore)
	l.accountBuf = buffer.NewAccount(l.accounts)
	l.candleBuf = buffer.NewCandle(l.candleStore)
	l.marketBuf = buffer.NewMarket(l.marketStore)

	l.marketDataBuf = buffer.NewMarketData()
	l.marketDataBuf.Register(l.marketDataStore)

	l.marginLevelsBuf = buffer.NewMarginLevels()
	l.marginLevelsBuf.Register(l.riskStore)
	l.settleBuf = buffer.NewSettlement()
	l.lossSocBuf = buffer.NewLossSocialization()
	l.proposalBuf = buffer.NewProposal()
	l.voteBuf = buffer.NewVote()
}

func (l *NodeCommand) setupStorages() (err error) {
	// always enabled market,parties etc stores as they are in memory or boths use them
	if l.marketStore, err = storage.NewMarkets(l.Log, l.conf.Storage, l.cancel); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.marketStore.ReloadConf(cfg.Storage) })

	l.marketDataStore = storage.NewMarketData(l.Log, l.conf.Storage)
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.marketDataStore.ReloadConf(cfg.Storage) })

	l.riskStore = storage.NewRisks(l.Log, l.conf.Storage)
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.riskStore.ReloadConf(cfg.Storage) })

	if l.partyStore, err = storage.NewParties(l.conf.Storage); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.partyStore.ReloadConf(cfg.Storage) })
	if l.transferResponseStore, err = storage.NewTransferResponses(l.Log, l.conf.Storage); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.transferResponseStore.ReloadConf(cfg.Storage) })

	// if stores are not enabled, initialise the noop stores and do nothing else
	if !l.conf.StoresEnabled {
		l.orderStore = storage.NewNoopOrders(l.Log, l.conf.Storage)
		l.tradeStore = storage.NewNoopTrades(l.Log, l.conf.Storage)
		l.accounts = storage.NewNoopAccounts(l.Log, l.conf.Storage)
		l.candleStore = storage.NewNoopCandles(l.Log, l.conf.Storage)
		return
	}

	if l.candleStore, err = storage.NewCandles(l.Log, l.conf.Storage, l.cancel); err != nil {
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

	if l.accounts, err = storage.NewAccounts(l.Log, l.conf.Storage, l.cancel); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.accounts.ReloadConf(cfg.Storage) })

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

	l.broker = broker.New(l.ctx)
	_ = l.broker.Subscribe(l.marketEventSub, false) // not required, use channel
	l.broker.SubscribeBatch(true, l.transferSub, l.orderSub, l.accountSub, l.partySub, l.tradeSub, l.marginLevelSub, l.governanceSub, l.voteSub)

	now, _ := l.timeService.GetTimeNow()

	//  create collateral
	l.collateral, err = collateral.New(l.Log, l.conf.Collateral, l.broker, l.lossSocBuf, now)
	if err != nil {
		log.Error("unable to initialise collateral", logging.Error(err))
		return err
	}

	// instantiate the execution engine
	l.executionEngine = execution.NewEngine(
		l.Log,
		l.conf.Execution,
		l.timeService,
		l.candleBuf,
		l.marketBuf,
		l.marketDataBuf,
		l.settleBuf,
		l.lossSocBuf,
		l.mktscfg,
		l.collateral,
		l.broker,
	)
	// we cannot pass the Chain dependency here (that's set by the blockchain)
	commander := nodewallet.NewCommander(l.ctx, nil)
	l.topology = validators.NewTopology(l.Log, nil)

	netParams := governance.DefaultNetworkParameters(l.Log)
	l.governance, err = governance.NewEngine(l.Log, l.conf.Governance, netParams, l.collateral, l.proposalBuf, l.voteBuf, l.broker, l.topology, l.nodeWallet, commander, l.assets, now, !l.noStores)
	if err != nil {
		log.Error("unable to initialise governance", logging.Error(err))
		return err
	}

	// TODO(jeremy): for now we assume a node started without the stores support
	// is a validator, this will need to be changed later on.
	l.processor, err = processor.New(l.Log, l.conf.Processor, l.executionEngine, l.timeService, l.stats.Blockchain, commander, l.nodeWallet, l.assets, l.topology, l.governance, l.proposalBuf, !l.noStores)
	if err != nil {
		return err
	}

	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.executionEngine.ReloadConf(cfg.Execution) })

	// plugins
	l.settlePlugin = plugins.NewPositions(l.settleBuf, l.lossSocBuf)
	l.settlePlugin.Start(l.ctx) // open channel from the start

	// now instanciate the blockchain layer
	l.blockchain, err = blockchain.New(l.Log, l.conf.Blockchain, l.processor, l.timeService, l.stats.Blockchain, commander, l.cancel)
	if err != nil {
		return errors.Wrap(err, "unable to start the blockchain")
	}

	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.blockchain.ReloadConf(cfg.Blockchain) })

	// get the chain client as well.
	l.blockchainClient = l.blockchain.Client()
	l.topology.SetChain(l.blockchain.Client())

	// start services
	if l.candleService, err = candles.NewService(l.Log, l.conf.Candles, l.candleStore); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.candleService.ReloadConf(cfg.Candles) })
	if l.orderService, err = orders.NewService(l.Log, l.conf.Orders, l.orderStore, l.timeService, l.blockchainClient); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.orderService.ReloadConf(cfg.Orders) })

	if l.tradeService, err = trades.NewService(l.Log, l.conf.Trades, l.tradeStore, l.riskStore, l.settlePlugin); err != nil {
		return
	}
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.tradeService.ReloadConf(cfg.Trades) })

	if l.marketService, err = markets.NewService(l.Log, l.conf.Markets, l.marketStore, l.orderStore, l.marketDataStore); err != nil {
		return
	}

	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.marketService.ReloadConf(cfg.Markets) })

	l.riskService = risk.NewService(l.Log, l.conf.Risk, l.riskStore)
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.riskService.ReloadConf(cfg.Risk) })

	l.governanceService = governance.NewService(l.Log, l.conf.Governance, l.broker, l.governanceSub, l.voteSub)
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.governanceService.ReloadConf(cfg.Governance) })

	// last assignment to err, no need to check here, if something went wrong, we'll know about it
	l.partyService, err = parties.NewService(l.Log, l.conf.Parties, l.partyStore)
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.partyService.ReloadConf(cfg.Parties) })
	l.accountsService = accounts.NewService(l.Log, l.conf.Accounts, l.accounts, l.blockchainClient)
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.accountsService.ReloadConf(cfg.Accounts) })
	l.transfersService = transfers.NewService(l.Log, l.conf.Transfers, l.transferResponseStore)
	l.cfgwatchr.OnConfigUpdate(func(cfg config.Config) { l.transfersService.ReloadConf(cfg.Transfers) })
	return
}

func getTerminalPassphrase() (string, error) {
	fmt.Printf("please enter nodewallet passphrase:")
	password, err := terminal.ReadPassword(0)
	if err != nil {
		return "", err
	}

	fmt.Println("")
	return string(password), nil
}

func getFilePassphrase(path string) (string, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}
