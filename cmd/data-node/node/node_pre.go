// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package node

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/candlesv2"
	"code.vegaprotocol.io/data-node/service"
	"code.vegaprotocol.io/shared/paths"

	"code.vegaprotocol.io/data-node/broker"
	"code.vegaprotocol.io/data-node/config"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/pprof"
	"code.vegaprotocol.io/data-node/sqlstore"
	"code.vegaprotocol.io/data-node/sqlsubscribers"
	"code.vegaprotocol.io/data-node/subscribers"
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

	// Set ulimits
	if err = l.SetUlimits(); err != nil {
		l.Log.Warn("Unable to set ulimits",
			logging.Error(err))
	} else {
		l.Log.Debug("Set ulimits",
			logging.Uint64("nofile", l.conf.UlimitNOFile))
	}

	l.Log.Info("Enabling SQL stores")
	if err := l.setupStoresSQL(); err != nil {
		return err
	}
	if err := l.setupServices(); err != nil {
		return err
	}
	l.setupSQLSubscribers()

	return nil
}

func (l *NodeCommand) setupSQLSubscribers() {
	l.accountSub = sqlsubscribers.NewAccount(l.accountService, l.Log)
	l.assetSub = sqlsubscribers.NewAsset(l.assetService, l.Log)
	l.partySub = sqlsubscribers.NewParty(l.partyService, l.Log)
	l.transferResponseSub = sqlsubscribers.NewTransferResponse(l.ledgerService, l.accountService, l.Log)
	l.orderSub = sqlsubscribers.NewOrder(l.orderService, l.Log)
	l.networkLimitsSub = sqlsubscribers.NewNetworkLimitSub(l.ctx, l.networkLimitsService, l.Log)
	l.marketDataSub = sqlsubscribers.NewMarketData(l.marketDataService, l.Log)
	l.tradesSub = sqlsubscribers.NewTradesSubscriber(l.tradeService, l.Log)
	l.rewardsSub = sqlsubscribers.NewReward(l.rewardService, l.Log)
	l.marketCreatedSub = sqlsubscribers.NewMarketCreated(l.marketsService, l.Log)
	l.marketUpdatedSub = sqlsubscribers.NewMarketUpdated(l.marketsService, l.Log)
	l.delegationsSub = sqlsubscribers.NewDelegation(l.delegationService, l.Log)
	l.epochSub = sqlsubscribers.NewEpoch(l.epochService, l.Log)
	l.depositSub = sqlsubscribers.NewDeposit(l.depositService, l.Log)
	l.withdrawalSub = sqlsubscribers.NewWithdrawal(l.withdrawalService, l.Log)
	l.proposalsSub = sqlsubscribers.NewProposal(l.governanceService, l.Log)
	l.votesSub = sqlsubscribers.NewVote(l.governanceService, l.Log)
	l.marginLevelsSub = sqlsubscribers.NewMarginLevels(l.riskService, l.accountStore, l.Log)
	l.riskFactorSub = sqlsubscribers.NewRiskFactor(l.riskFactorService, l.Log)
	l.netParamSub = sqlsubscribers.NewNetworkParameter(l.networkParameterService, l.Log)
	l.checkpointSub = sqlsubscribers.NewCheckpoint(l.checkpointService, l.Log)
	l.positionsSub = sqlsubscribers.NewPosition(l.positionService, l.Log)
	l.oracleSpecSub = sqlsubscribers.NewOracleSpec(l.oracleSpecService, l.Log)
	l.oracleDataSub = sqlsubscribers.NewOracleData(l.oracleDataService, l.Log)
	l.liquidityProvisionSub = sqlsubscribers.NewLiquidityProvision(l.liquidityProvisionService, l.Log)
	l.transferSub = sqlsubscribers.NewTransfer(l.transfersStore, l.accountService, l.Log)
	l.stakeLinkingSub = sqlsubscribers.NewStakeLinking(l.stakeLinkingService, l.Log)
	l.notarySub = sqlsubscribers.NewNotary(l.notaryService, l.Log)
	l.multiSigSignerEventSub = sqlsubscribers.NewERC20MultiSigSignerEvent(l.multiSigService, l.Log)
	l.keyRotationsSub = sqlsubscribers.NewKeyRotation(l.keyRotationsService, l.Log)
	l.nodeSub = sqlsubscribers.NewNode(l.nodeService, l.Log)
	l.marketDepthSub = sqlsubscribers.NewMarketDepth(l.marketDepthService)
}

func (l *NodeCommand) setupStoresSQL() error {
	var err error
	if l.conf.SQLStore.UseEmbedded {
		l.embeddedPostgres, err = sqlstore.StartEmbeddedPostgres(l.Log, l.conf.SQLStore,
			l.vegaPaths.StatePathFor(paths.DataNodeStorageHome))
		if err != nil {
			return fmt.Errorf("failed to start embedded postgres: %w", err)
		}
	}

	err = sqlstore.MigrateToLatestSchema(l.Log, l.conf.SQLStore)
	if err != nil {
		return fmt.Errorf("failed to migrate to latest schema:%w", err)
	}

	err = sqlstore.ApplyDataRetentionPolicies(l.conf.SQLStore)
	if err != nil {
		return fmt.Errorf("failed to apply data retention policies:%w", err)
	}

	transactionalConnectionSource, err := sqlstore.NewTransactionalConnectionSource(l.Log, l.conf.SQLStore.ConnectionConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection source:%w", err)
	}

	l.transactionalConnectionSource = transactionalConnectionSource

	l.assetStore = sqlstore.NewAssets(transactionalConnectionSource)
	l.blockStore = sqlstore.NewBlocks(transactionalConnectionSource)
	l.partyStore = sqlstore.NewParties(transactionalConnectionSource)
	l.partyStore.Initialise()
	l.accountStore = sqlstore.NewAccounts(transactionalConnectionSource)
	l.balanceStore = sqlstore.NewBalances(transactionalConnectionSource)
	l.ledger = sqlstore.NewLedger(transactionalConnectionSource)
	l.orderStore = sqlstore.NewOrders(transactionalConnectionSource, l.Log)
	l.tradeStore = sqlstore.NewTrades(transactionalConnectionSource)
	l.networkLimitsStore = sqlstore.NewNetworkLimits(transactionalConnectionSource)
	l.marketDataStore = sqlstore.NewMarketData(transactionalConnectionSource)
	l.rewardStore = sqlstore.NewRewards(transactionalConnectionSource)
	l.marketsStore = sqlstore.NewMarkets(transactionalConnectionSource)
	l.delegationStore = sqlstore.NewDelegations(transactionalConnectionSource)
	l.epochStore = sqlstore.NewEpochs(transactionalConnectionSource)
	l.depositStore = sqlstore.NewDeposits(transactionalConnectionSource)
	l.withdrawalsStore = sqlstore.NewWithdrawals(transactionalConnectionSource)
	l.proposalStore = sqlstore.NewProposals(transactionalConnectionSource)
	l.voteStore = sqlstore.NewVotes(transactionalConnectionSource)
	l.marginLevelsStore = sqlstore.NewMarginLevels(transactionalConnectionSource)
	l.riskFactorStore = sqlstore.NewRiskFactors(transactionalConnectionSource)
	l.netParamStore = sqlstore.NewNetworkParameters(transactionalConnectionSource)
	l.checkpointStore = sqlstore.NewCheckpoints(transactionalConnectionSource)
	l.positionStore = sqlstore.NewPositions(transactionalConnectionSource)
	l.oracleSpecStore = sqlstore.NewOracleSpec(transactionalConnectionSource)
	l.oracleDataStore = sqlstore.NewOracleData(transactionalConnectionSource)
	l.liquidityProvisionStore = sqlstore.NewLiquidityProvision(transactionalConnectionSource)
	l.transfersStore = sqlstore.NewTransfers(transactionalConnectionSource)
	l.stakeLinkingStore = sqlstore.NewStakeLinking(transactionalConnectionSource)
	l.notaryStore = sqlstore.NewNotary(transactionalConnectionSource)
	l.multiSigSignerAddedStore = sqlstore.NewERC20MultiSigSignerEvent(transactionalConnectionSource)
	l.keyRotationsStore = sqlstore.NewKeyRotations(transactionalConnectionSource)
	l.nodeStore = sqlstore.NewNode(transactionalConnectionSource)
	l.candleStore = sqlstore.NewCandles(l.ctx, transactionalConnectionSource, l.conf.CandlesV2.CandleStore)
	l.chainStore = sqlstore.NewChain(transactionalConnectionSource)
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

	eventSource, err := broker.NewEventSource(l.conf.Broker, l.Log)
	if err != nil {
		l.Log.Error("unable to initialise event source", logging.Error(err))
		return err
	}

	eventSource = broker.NewFanOutEventSource(eventSource, l.conf.SQLStore.FanOutBufferSize, 2)

	l.sqlBroker = broker.NewSqlStoreBroker(l.Log, l.conf.Broker, l.chainService, eventSource,
		l.transactionalConnectionSource,
		l.blockStore,
		l.accountSub,
		l.assetSub,
		l.partySub,
		l.transferResponseSub,
		l.orderSub,
		l.networkLimitsSub,
		l.marketDataSub,
		l.tradesSub,
		l.rewardsSub,
		l.delegationsSub,
		l.marketCreatedSub,
		l.marketUpdatedSub,
		l.epochSub,
		l.marketUpdatedSub,
		l.depositSub,
		l.withdrawalSub,
		l.proposalsSub,
		l.votesSub,
		l.depositSub,
		l.marginLevelsSub,
		l.riskFactorSub,
		l.netParamSub,
		l.checkpointSub,
		l.positionsSub,
		l.oracleSpecSub,
		l.oracleDataSub,
		l.liquidityProvisionSub,
		l.transferSub,
		l.stakeLinkingSub,
		l.notarySub,
		l.multiSigSignerEventSub,
		l.keyRotationsSub,
		l.nodeSub,
		l.marketDepthSub,
	)

	l.broker, err = broker.New(l.ctx, l.Log, l.conf.Broker, l.chainService, eventSource)
	if err != nil {
		l.Log.Error("unable to initialise broker", logging.Error(err))
		return err
	}

	// Event service us used by old and new world
	l.eventService = subscribers.NewService(l.broker)

	nodeAddr := fmt.Sprintf("%v:%v", l.conf.API.CoreNodeIP, l.conf.API.CoreNodeGRPCPort)
	conn, err := grpc.Dial(nodeAddr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	l.vegaCoreServiceClient = vegaprotoapi.NewCoreServiceClient(conn)

	return nil
}

func (l *NodeCommand) setupServices() error {
	log := l.Log.Named("service")
	log.SetLevel(l.conf.Service.Level.Get())

	l.accountService = service.NewAccount(l.accountStore, l.balanceStore, log)
	l.assetService = service.NewAsset(l.assetStore, log)
	l.blockService = service.NewBlock(l.blockStore, log)
	l.candleService = candlesv2.NewService(l.ctx, log, l.conf.CandlesV2, l.candleStore)
	l.checkpointService = service.NewCheckpoint(l.checkpointStore, log)
	l.delegationService = service.NewDelegation(l.delegationStore, log)
	l.depositService = service.NewDeposit(l.depositStore, log)
	l.epochService = service.NewEpoch(l.epochStore, log)
	l.governanceService = service.NewGovernance(l.proposalStore, l.voteStore, log)
	l.keyRotationsService = service.NewKeyRotations(l.keyRotationsStore, log)
	l.ledgerService = service.NewLedger(l.ledger, log)
	l.liquidityProvisionService = service.NewLiquidityProvision(l.liquidityProvisionStore, log)
	l.marketDataService = service.NewMarketData(l.marketDataStore, log)
	l.marketDepthService = service.NewMarketDepth(l.orderStore, log)
	l.marketsService = service.NewMarkets(l.marketsStore, log)
	l.multiSigService = service.NewMultiSig(l.multiSigSignerAddedStore, log)
	l.networkLimitsService = service.NewNetworkLimits(l.networkLimitsStore, log)
	l.networkParameterService = service.NewNetworkParameter(l.netParamStore, log)
	l.nodeService = service.NewNode(l.nodeStore, log)
	l.notaryService = service.NewNotary(l.notaryStore, log)
	l.oracleDataService = service.NewOracleData(l.oracleDataStore, log)
	l.oracleSpecService = service.NewOracleSpec(l.oracleSpecStore, log)
	l.orderService = service.NewOrder(l.orderStore, log)
	l.partyService = service.NewParty(l.partyStore, log)
	l.positionService = service.NewPosition(l.positionStore, log)
	l.rewardService = service.NewReward(l.rewardStore, log)
	l.riskFactorService = service.NewRiskFactor(l.riskFactorStore, log)
	l.riskService = service.NewRisk(l.marginLevelsStore, l.accountStore, log)
	l.stakeLinkingService = service.NewStakeLinking(l.stakeLinkingStore, log)
	l.tradeService = service.NewTrade(l.tradeStore, log)
	l.transferService = service.NewTransfer(l.transfersStore, log)
	l.withdrawalService = service.NewWithdrawal(l.withdrawalsStore, log)
	l.chainService = service.NewChain(l.chainStore, log)

	toInit := []interface{ Initialise(context.Context) error }{
		l.marketDepthService,
		l.marketDataService,
		l.marketsService,
	}

	for _, svc := range toInit {
		if err := svc.Initialise(l.ctx); err != nil {
			return err
		}
	}

	return nil
}
