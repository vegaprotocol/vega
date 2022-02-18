package node

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/accounts"
	"code.vegaprotocol.io/data-node/assets"
	"code.vegaprotocol.io/data-node/broker"
	"code.vegaprotocol.io/data-node/candles"
	"code.vegaprotocol.io/data-node/checkpoint"
	"code.vegaprotocol.io/data-node/config"
	"code.vegaprotocol.io/data-node/delegations"
	"code.vegaprotocol.io/data-node/epochs"
	"code.vegaprotocol.io/data-node/fee"
	"code.vegaprotocol.io/data-node/governance"
	"code.vegaprotocol.io/data-node/liquidity"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/markets"
	"code.vegaprotocol.io/data-node/netparams"
	"code.vegaprotocol.io/data-node/nodes"
	"code.vegaprotocol.io/data-node/notary"
	"code.vegaprotocol.io/data-node/oracles"
	"code.vegaprotocol.io/data-node/orders"
	"code.vegaprotocol.io/data-node/parties"
	"code.vegaprotocol.io/data-node/plugins"
	"code.vegaprotocol.io/data-node/pprof"
	"code.vegaprotocol.io/data-node/risk"
	"code.vegaprotocol.io/data-node/sqlstore"
	"code.vegaprotocol.io/data-node/sqlsubscribers"
	"code.vegaprotocol.io/data-node/staking"
	"code.vegaprotocol.io/data-node/storage"
	"code.vegaprotocol.io/data-node/subscribers"
	"code.vegaprotocol.io/data-node/trades"
	"code.vegaprotocol.io/data-node/transfers"
	"code.vegaprotocol.io/data-node/vegatime"
	vegaprotoapi "code.vegaprotocol.io/protos/vega/api/v1"

	"google.golang.org/grpc"
)

func (l *NodeCommand) persistentPre(args []string) (err error) {
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

	conf := l.configWatcher.Get()

	// reload logger with the setup from configuration
	l.Log = logging.NewLoggerFromConfig(conf.Logging)

	if conf.Pprof.Enabled {
		l.Log.Info("vega is starting with pprof profile, this is not a recommended setting for production")
		l.pproffhandlr, err = pprof.New(l.Log, conf.Pprof)
		if err != nil {
			return
		}
		l.configWatcher.OnConfigUpdate(
			func(cfg config.Config) { l.pproffhandlr.ReloadConf(cfg.Pprof) },
		)
	}

	l.Log.Info("Starting Vega",
		logging.String("version", l.Version),
		logging.String("version-hash", l.VersionHash))

	// this doesn't fail
	l.timeService = vegatime.New(l.conf.Time)

	// Set ulimits
	if err = l.SetUlimits(); err != nil {
		l.Log.Warn("Unable to set ulimits",
			logging.Error(err))
	} else {
		l.Log.Debug("Set ulimits",
			logging.Uint64("nofile", l.conf.UlimitNOFile))
	}

	// set up storage, this should be persistent
	if err := l.setupStorages(); err != nil {
		return err
	}
	l.setupSubscribers()
	l.setupSQLSubscribers()
	return nil
}

func (l *NodeCommand) setupSubscribers() {
	l.timeUpdateSub = subscribers.NewTimeSub(l.ctx, l.timeService, l.Log, true)
	l.transferRespSub = subscribers.NewTransferResponse(l.ctx, l.transferResponseStore, l.Log, true)
	l.marketEventSub = subscribers.NewMarketEvent(l.ctx, l.conf.Subscribers, l.Log, false)
	l.orderSub = subscribers.NewOrderEvent(l.ctx, l.conf.Subscribers, l.Log, l.orderStore, true)
	l.accountSub = subscribers.NewAccountSub(l.ctx, l.accounts, l.Log, true)
	l.partySub = subscribers.NewPartySub(l.ctx, l.partyStore, l.Log, true)
	l.tradeSub = subscribers.NewTradeSub(l.ctx, l.tradeStore, l.Log, true)
	l.marginLevelSub = subscribers.NewMarginLevelSub(l.ctx, l.riskStore, l.Log, true)
	l.governanceSub = subscribers.NewGovernanceDataSub(l.ctx, l.Log, true)
	l.voteSub = subscribers.NewVoteSub(l.ctx, false, true, l.Log)
	l.marketDataSub = subscribers.NewMarketDataSub(l.ctx, l.marketDataStore, l.Log, true)
	l.newMarketSub = subscribers.NewMarketSub(l.ctx, l.marketStore, l.Log, true)
	l.marketUpdatedSub = subscribers.NewMarketUpdatedSub(l.ctx, l.marketStore, l.Log, true)
	l.candleSub = subscribers.NewCandleSub(l.ctx, l.candleStore, l.Log, true)
	l.marketDepthSub = subscribers.NewMarketDepthBuilder(l.ctx, l.Log, true)
	l.riskFactorSub = subscribers.NewRiskFactorSub(l.ctx, l.riskStore, l.Log, true)
	l.nodesSub = subscribers.NewNodesSub(l.ctx, l.nodeStore, l.Log, true)
	l.delegationBalanceSub = subscribers.NewDelegationBalanceSub(l.ctx, l.nodeStore, l.epochStore, l.delegationStore, l.Log, true)
	l.epochUpdateSub = subscribers.NewEpochUpdateSub(l.ctx, l.epochStore, l.Log, true)
	l.rewardsSub = subscribers.NewRewards(l.ctx, l.Log, true)
	l.checkpointSub = subscribers.NewCheckpointSub(l.ctx, l.Log, l.checkpointStore, true)
	l.transferSub = subscribers.NewTransferSub(l.ctx, l.transferStore, l.Log, true)
}

func (l *NodeCommand) setupSQLSubscribers() {
	if !l.conf.SQLStore.Enabled {
		return
	}

	l.assetSubSQL = sqlsubscribers.NewAsset(l.ctx, l.assetStoreSQL, l.blockStoreSQL, l.Log)
	l.timeSubSQL = sqlsubscribers.NewTimeSub(l.ctx, l.blockStoreSQL, l.Log)
	l.transferResponseSubSQL = sqlsubscribers.NewTransferResponse(l.ctx, l.ledgerSQL, l.accountStoreSQL, l.balanceStoreSQL, l.partyStoreSQL, l.blockStoreSQL, l.Log)
}

func (l *NodeCommand) setupStorages() error {
	var err error

	l.marketDataStore = storage.NewMarketData(l.Log, l.conf.Storage)
	l.riskStore = storage.NewRisks(l.Log, l.conf.Storage)
	l.nodeStore = storage.NewNode(l.Log, l.conf.Storage)
	l.epochStore = storage.NewEpoch(l.Log, l.nodeStore, l.conf.Storage)
	l.delegationStore = storage.NewDelegations(l.Log, l.conf.Storage)
	l.transferStore = storage.NewTransfers(l.Log, l.conf.Storage)

	if l.partyStore, err = storage.NewParties(l.conf.Storage); err != nil {
		return err
	}
	if l.transferResponseStore, err = storage.NewTransferResponses(l.Log, l.conf.Storage); err != nil {
		return err
	}

	if l.conf.SQLStore.Enabled {
		sqlStore, err := sqlstore.InitialiseStorage(l.Log, l.conf.SQLStore, l.vegaPaths)
		if err != nil {
			return fmt.Errorf("couldn't initialise sql storage: %w", err)
		}

		l.assetStoreSQL = sqlstore.NewAssets(sqlStore)
		l.blockStoreSQL = sqlstore.NewBlocks(sqlStore)
		l.partyStoreSQL = sqlstore.NewParties(sqlStore)
		l.accountStoreSQL = sqlstore.NewAccounts(sqlStore)
		l.balanceStoreSQL = sqlstore.NewBalances(sqlStore)
		l.ledgerSQL = sqlstore.NewLedger(sqlStore)
		l.sqlStore = sqlStore
	}

	st, err := storage.InitialiseStorage(l.vegaPaths)
	if err != nil {
		return fmt.Errorf("couldn't initialise storage: %w", err)
	}

	if l.marketStore, err = storage.NewMarkets(l.Log, st.MarketsHome, l.conf.Storage, l.cancel); err != nil {
		return err
	}
	if l.candleStore, err = storage.NewCandles(l.Log, st.CandlesHome, l.conf.Storage, l.cancel); err != nil {
		return err
	}
	if l.orderStore, err = storage.NewOrders(l.Log, st.OrdersHome, l.conf.Storage, l.cancel); err != nil {
		return err
	}
	if l.tradeStore, err = storage.NewTrades(l.Log, st.TradesHome, l.conf.Storage, l.cancel); err != nil {
		return err
	}
	if l.accounts, err = storage.NewAccounts(l.Log, st.AccountsHome, l.conf.Storage, l.cancel); err != nil {
		return err
	}
	if l.checkpointStore, err = storage.NewCheckpoints(l.Log, st.CheckpointsHome, l.conf.Storage, l.cancel); err != nil {
		return err
	}
	if l.chainInfoStore, err = storage.NewChainInfo(l.Log, st.ChainInfoHome, l.conf.Storage, l.cancel); err != nil {
		return err
	}
	l.configWatcher.OnConfigUpdate(
		func(cfg config.Config) { l.accounts.ReloadConf(cfg.Storage) },
		func(cfg config.Config) { l.tradeStore.ReloadConf(cfg.Storage) },
		func(cfg config.Config) { l.orderStore.ReloadConf(cfg.Storage) },
		func(cfg config.Config) { l.candleStore.ReloadConf(cfg.Storage) },
		func(cfg config.Config) { l.transferResponseStore.ReloadConf(cfg.Storage) },
		func(cfg config.Config) { l.partyStore.ReloadConf(cfg.Storage) },
		func(cfg config.Config) { l.riskStore.ReloadConf(cfg.Storage) },
		func(cfg config.Config) { l.marketDataStore.ReloadConf(cfg.Storage) },
		func(cfg config.Config) { l.marketStore.ReloadConf(cfg.Storage) },
		func(cfg config.Config) { l.nodeStore.ReloadConf(cfg.Storage) },
		func(cfg config.Config) { l.epochStore.ReloadConf(cfg.Storage) },
		func(cfg config.Config) { l.delegationStore.ReloadConf(cfg.Storage) },
		func(cfg config.Config) { l.chainInfoStore.ReloadConf(cfg.Storage) },
		func(cfg config.Config) { l.transferStore.ReloadConf(cfg.Storage) },
	)

	return nil
}

// we've already set everything up WRT arguments etc... just bootstrap the node
func (l *NodeCommand) preRun(_ []string) (err error) {
	// ensure that context is cancelled if we return an error here
	defer func() {
		if err != nil {
			l.cancel()
		}
	}()

	l.broker, err = broker.New(l.ctx, l.Log, l.conf.Broker, l.chainInfoStore)
	if err != nil {
		l.Log.Error("unable to initialise broker", logging.Error(err))
		return err
	}

	// plugins
	l.settlePlugin = plugins.NewPositions(l.ctx)
	l.notaryPlugin = plugins.NewNotary(l.ctx)
	l.assetPlugin = plugins.NewAsset(l.ctx)
	l.withdrawalPlugin = plugins.NewWithdrawal(l.ctx)
	l.depositPlugin = plugins.NewDeposit(l.ctx)
	l.netParamsService = netparams.NewService(l.ctx)
	l.liquidityService = liquidity.NewService(l.ctx, l.Log, l.conf.Liquidity)
	l.oracleService = oracles.NewService(l.ctx)
	l.stakingService = staking.NewService(l.ctx, l.Log)

	// start services
	l.candleService = candles.NewService(l.Log, l.conf.Candles, l.candleStore)
	l.tradeService = trades.NewService(l.Log, l.conf.Trades, l.tradeStore, l.settlePlugin)
	l.marketService = markets.NewService(l.Log, l.conf.Markets, l.marketStore, l.orderStore, l.marketDataStore, l.marketDepthSub)
	l.riskService = risk.NewService(l.Log, l.conf.Risk, l.riskStore, l.marketStore, l.marketDataStore)
	l.governanceService = governance.NewService(l.Log, l.conf.Governance, l.broker, l.governanceSub, l.voteSub)
	l.orderService = orders.NewService(l.Log, l.conf.Orders, l.orderStore, l.timeService)
	l.feeService = fee.NewService(l.Log, l.conf.Fee, l.marketStore, l.marketDataStore)
	l.partyService, err = parties.NewService(l.Log, l.conf.Parties, l.partyStore)
	l.accountsService = accounts.NewService(l.Log, l.conf.Accounts, l.accounts)
	l.transfersService = transfers.NewService(l.Log, l.conf.Transfers, l.transferResponseStore, l.transferStore)
	l.notaryService = notary.NewService(l.Log, l.conf.Notary, l.notaryPlugin)
	l.assetService = assets.NewService(l.Log, l.conf.Assets, l.assetPlugin)
	l.eventService = subscribers.NewService(l.broker)
	l.epochService = epochs.NewService(l.Log, l.conf.Epochs, l.epochStore)
	l.delegationService = delegations.NewService(l.Log, l.conf.Delegations, l.delegationStore)
	l.nodeService = nodes.NewService(l.Log, l.conf.Nodes, l.nodeStore, l.epochStore)

	l.broker.SubscribeBatch(
		l.marketEventSub, l.transferRespSub, l.orderSub, l.accountSub,
		l.partySub, l.tradeSub, l.marginLevelSub, l.governanceSub,
		l.voteSub, l.marketDataSub, l.notaryPlugin, l.settlePlugin,
		l.newMarketSub, l.assetPlugin, l.candleSub, l.withdrawalPlugin,
		l.depositPlugin, l.marketDepthSub, l.riskFactorSub, l.netParamsService,
		l.liquidityService, l.marketUpdatedSub, l.oracleService, l.timeUpdateSub,
		l.nodesSub, l.delegationBalanceSub, l.epochUpdateSub, l.rewardsSub,
		l.stakingService, l.checkpointSub, l.transferSub,
	)

	if l.conf.SQLStore.Enabled {
		l.broker.SubscribeBatch(l.timeSubSQL, l.assetSubSQL, l.transferResponseSubSQL)
	}

	nodeAddr := fmt.Sprintf("%v:%v", l.conf.API.CoreNodeIP, l.conf.API.CoreNodeGRPCPort)
	conn, err := grpc.Dial(nodeAddr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	l.vegaCoreServiceClient = vegaprotoapi.NewCoreServiceClient(conn)

	l.checkpointSvc = checkpoint.NewService(l.Log, l.conf.Checkpoint, l.checkpointStore)

	// setup config reloads for all services /etc
	l.setupConfigWatchers()
	l.timeService.NotifyOnTick(l.configWatcher.OnTimeUpdate)

	return nil
}

func (l *NodeCommand) setupConfigWatchers() {
	l.configWatcher.OnConfigUpdate(
		func(cfg config.Config) { l.candleService.ReloadConf(cfg.Candles) },
		func(cfg config.Config) { l.orderService.ReloadConf(cfg.Orders) },
		func(cfg config.Config) { l.liquidityService.ReloadConf(cfg.Liquidity) },
		func(cfg config.Config) { l.tradeService.ReloadConf(cfg.Trades) },
		func(cfg config.Config) { l.marketService.ReloadConf(cfg.Markets) },
		func(cfg config.Config) { l.riskService.ReloadConf(cfg.Risk) },
		func(cfg config.Config) { l.governanceService.ReloadConf(cfg.Governance) },
		func(cfg config.Config) { l.assetService.ReloadConf(cfg.Assets) },
		func(cfg config.Config) { l.notaryService.ReloadConf(cfg.Notary) },
		func(cfg config.Config) { l.transfersService.ReloadConf(cfg.Transfers) },
		func(cfg config.Config) { l.accountsService.ReloadConf(cfg.Accounts) },
		func(cfg config.Config) { l.partyService.ReloadConf(cfg.Parties) },
		func(cfg config.Config) { l.nodeService.ReloadConf(cfg.Nodes) },
		func(cfg config.Config) { l.epochService.ReloadConf(cfg.Epochs) },
		func(cfg config.Config) { l.delegationService.ReloadConf(cfg.Delegations) },
		func(cfg config.Config) { l.checkpointSvc.ReloadConf(cfg.Checkpoint) },
	)
}
