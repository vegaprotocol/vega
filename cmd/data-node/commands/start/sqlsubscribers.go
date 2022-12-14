package start

import (
	"context"

	"code.vegaprotocol.io/vega/datanode/broker"
	"code.vegaprotocol.io/vega/datanode/candlesv2"
	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlsubscribers"
	"code.vegaprotocol.io/vega/logging"
)

type SQLSubscribers struct {
	// Stores
	assetStore                *sqlstore.Assets
	blockStore                *sqlstore.Blocks
	accountStore              *sqlstore.Accounts
	balanceStore              *sqlstore.Balances
	ledger                    *sqlstore.Ledger
	partyStore                *sqlstore.Parties
	orderStore                *sqlstore.Orders
	tradeStore                *sqlstore.Trades
	networkLimitsStore        *sqlstore.NetworkLimits
	marketDataStore           *sqlstore.MarketData
	rewardStore               *sqlstore.Rewards
	delegationStore           *sqlstore.Delegations
	marketsStore              *sqlstore.Markets
	epochStore                *sqlstore.Epochs
	depositStore              *sqlstore.Deposits
	withdrawalsStore          *sqlstore.Withdrawals
	proposalStore             *sqlstore.Proposals
	voteStore                 *sqlstore.Votes
	marginLevelsStore         *sqlstore.MarginLevels
	riskFactorStore           *sqlstore.RiskFactors
	netParamStore             *sqlstore.NetworkParameters
	checkpointStore           *sqlstore.Checkpoints
	oracleSpecStore           *sqlstore.OracleSpec
	oracleDataStore           *sqlstore.OracleData
	liquidityProvisionStore   *sqlstore.LiquidityProvision
	positionStore             *sqlstore.Positions
	transfersStore            *sqlstore.Transfers
	stakeLinkingStore         *sqlstore.StakeLinking
	notaryStore               *sqlstore.Notary
	multiSigSignerAddedStore  *sqlstore.ERC20MultiSigSignerEvent
	keyRotationsStore         *sqlstore.KeyRotations
	ethereumKeyRotationsStore *sqlstore.EthereumKeyRotations
	nodeStore                 *sqlstore.Node
	candleStore               *sqlstore.Candles
	chainStore                *sqlstore.Chain
	pupStore                  *sqlstore.ProtocolUpgradeProposals
	snapStore                 *sqlstore.CoreSnapshotData

	// Services
	candleService               *candlesv2.Svc
	marketDepthService          *service.MarketDepth
	riskService                 *service.Risk
	marketDataService           *service.MarketData
	positionService             *service.Position
	tradeService                *service.Trade
	ledgerService               *service.Ledger
	rewardService               *service.Reward
	delegationService           *service.Delegation
	assetService                *service.Asset
	blockService                *service.Block
	partyService                *service.Party
	accountService              *service.Account
	orderService                *service.Order
	networkLimitsService        *service.NetworkLimits
	marketsService              *service.Markets
	epochService                *service.Epoch
	depositService              *service.Deposit
	withdrawalService           *service.Withdrawal
	governanceService           *service.Governance
	riskFactorService           *service.RiskFactor
	networkParameterService     *service.NetworkParameter
	checkpointService           *service.Checkpoint
	oracleSpecService           *service.OracleSpec
	oracleDataService           *service.OracleData
	liquidityProvisionService   *service.LiquidityProvision
	transferService             *service.Transfer
	stakeLinkingService         *service.StakeLinking
	notaryService               *service.Notary
	multiSigService             *service.MultiSig
	keyRotationsService         *service.KeyRotations
	ethereumKeyRotationsService *service.EthereumKeyRotation
	nodeService                 *service.Node
	chainService                *service.Chain
	protocolUpgradeService      *service.ProtocolUpgrade
	coreSnapshotService         *service.SnapshotData

	// Subscribers
	accountSub              *sqlsubscribers.Account
	assetSub                *sqlsubscribers.Asset
	partySub                *sqlsubscribers.Party
	transferResponseSub     *sqlsubscribers.TransferResponse
	orderSub                *sqlsubscribers.Order
	networkLimitsSub        *sqlsubscribers.NetworkLimits
	marketDataSub           *sqlsubscribers.MarketData
	tradesSub               *sqlsubscribers.TradeSubscriber
	rewardsSub              *sqlsubscribers.Reward
	delegationsSub          *sqlsubscribers.Delegation
	marketCreatedSub        *sqlsubscribers.MarketCreated
	marketUpdatedSub        *sqlsubscribers.MarketUpdated
	epochSub                *sqlsubscribers.Epoch
	depositSub              *sqlsubscribers.Deposit
	withdrawalSub           *sqlsubscribers.Withdrawal
	proposalsSub            *sqlsubscribers.Proposal
	votesSub                *sqlsubscribers.Vote
	marginLevelsSub         *sqlsubscribers.MarginLevels
	riskFactorSub           *sqlsubscribers.RiskFactor
	netParamSub             *sqlsubscribers.NetworkParameter
	checkpointSub           *sqlsubscribers.Checkpoint
	oracleSpecSub           *sqlsubscribers.OracleSpec
	oracleDataSub           *sqlsubscribers.OracleData
	liquidityProvisionSub   *sqlsubscribers.LiquidityProvision
	positionsSub            *sqlsubscribers.Position
	transferSub             *sqlsubscribers.Transfer
	stakeLinkingSub         *sqlsubscribers.StakeLinking
	notarySub               *sqlsubscribers.Notary
	multiSigSignerEventSub  *sqlsubscribers.ERC20MultiSigSignerEvent
	keyRotationsSub         *sqlsubscribers.KeyRotation
	ethereumKeyRotationsSub *sqlsubscribers.EthereumKeyRotation
	nodeSub                 *sqlsubscribers.Node
	marketDepthSub          *sqlsubscribers.MarketDepth
	pupSub                  *sqlsubscribers.ProtocolUpgrade
	snapSub                 *sqlsubscribers.SnapshotData
}

func (s *SQLSubscribers) GetSQLSubscribers() []broker.SQLBrokerSubscriber {
	return []broker.SQLBrokerSubscriber{
		s.accountSub,
		s.assetSub,
		s.partySub,
		s.transferResponseSub,
		s.orderSub,
		s.networkLimitsSub,
		s.marketDataSub,
		s.tradesSub,
		s.rewardsSub,
		s.delegationsSub,
		s.marketCreatedSub,
		s.marketUpdatedSub,
		s.epochSub,
		s.marketUpdatedSub,
		s.depositSub,
		s.withdrawalSub,
		s.proposalsSub,
		s.votesSub,
		s.depositSub,
		s.marginLevelsSub,
		s.riskFactorSub,
		s.netParamSub,
		s.checkpointSub,
		s.positionsSub,
		s.oracleSpecSub,
		s.oracleDataSub,
		s.liquidityProvisionSub,
		s.transferSub,
		s.stakeLinkingSub,
		s.notarySub,
		s.multiSigSignerEventSub,
		s.keyRotationsSub,
		s.nodeSub,
		s.marketDepthSub,
		s.ethereumKeyRotationsSub,
		s.pupSub,
		s.snapSub,
	}
}

func (s *SQLSubscribers) CreateAllStores(ctx context.Context, Log *logging.Logger, transactionalConnectionSource *sqlstore.ConnectionSource,
	candleV2Config candlesv2.CandleStoreConfig,
) {
	s.assetStore = sqlstore.NewAssets(transactionalConnectionSource)
	s.blockStore = sqlstore.NewBlocks(transactionalConnectionSource)
	s.partyStore = sqlstore.NewParties(transactionalConnectionSource)
	s.partyStore.Initialise(ctx)
	s.accountStore = sqlstore.NewAccounts(transactionalConnectionSource)
	s.balanceStore = sqlstore.NewBalances(transactionalConnectionSource)
	s.ledger = sqlstore.NewLedger(transactionalConnectionSource)
	s.orderStore = sqlstore.NewOrders(transactionalConnectionSource, Log)
	s.tradeStore = sqlstore.NewTrades(transactionalConnectionSource)
	s.networkLimitsStore = sqlstore.NewNetworkLimits(transactionalConnectionSource)
	s.marketDataStore = sqlstore.NewMarketData(transactionalConnectionSource)
	s.rewardStore = sqlstore.NewRewards(transactionalConnectionSource)
	s.marketsStore = sqlstore.NewMarkets(transactionalConnectionSource)
	s.delegationStore = sqlstore.NewDelegations(transactionalConnectionSource)
	s.epochStore = sqlstore.NewEpochs(transactionalConnectionSource)
	s.depositStore = sqlstore.NewDeposits(transactionalConnectionSource)
	s.withdrawalsStore = sqlstore.NewWithdrawals(transactionalConnectionSource)
	s.proposalStore = sqlstore.NewProposals(transactionalConnectionSource)
	s.voteStore = sqlstore.NewVotes(transactionalConnectionSource)
	s.marginLevelsStore = sqlstore.NewMarginLevels(transactionalConnectionSource)
	s.riskFactorStore = sqlstore.NewRiskFactors(transactionalConnectionSource)
	s.netParamStore = sqlstore.NewNetworkParameters(transactionalConnectionSource)
	s.checkpointStore = sqlstore.NewCheckpoints(transactionalConnectionSource)
	s.positionStore = sqlstore.NewPositions(transactionalConnectionSource)
	s.oracleSpecStore = sqlstore.NewOracleSpec(transactionalConnectionSource)
	s.oracleDataStore = sqlstore.NewOracleData(transactionalConnectionSource)
	s.liquidityProvisionStore = sqlstore.NewLiquidityProvision(transactionalConnectionSource, Log)
	s.transfersStore = sqlstore.NewTransfers(transactionalConnectionSource)
	s.stakeLinkingStore = sqlstore.NewStakeLinking(transactionalConnectionSource)
	s.notaryStore = sqlstore.NewNotary(transactionalConnectionSource)
	s.multiSigSignerAddedStore = sqlstore.NewERC20MultiSigSignerEvent(transactionalConnectionSource)
	s.keyRotationsStore = sqlstore.NewKeyRotations(transactionalConnectionSource)
	s.ethereumKeyRotationsStore = sqlstore.NewEthereumKeyRotations(transactionalConnectionSource)
	s.nodeStore = sqlstore.NewNode(transactionalConnectionSource)
	s.candleStore = sqlstore.NewCandles(ctx, transactionalConnectionSource, candleV2Config)
	s.chainStore = sqlstore.NewChain(transactionalConnectionSource)
	s.pupStore = sqlstore.NewProtocolUpgradeProposals(transactionalConnectionSource)
	s.snapStore = sqlstore.NewCoreSnapshotData(transactionalConnectionSource)
}

func (s *SQLSubscribers) SetupServices(ctx context.Context, log *logging.Logger, candlesConfig candlesv2.Config) error {
	s.accountService = service.NewAccount(s.accountStore, s.balanceStore, log)
	s.assetService = service.NewAsset(s.assetStore, log)
	s.blockService = service.NewBlock(s.blockStore, log)
	s.candleService = candlesv2.NewService(ctx, log, candlesConfig, s.candleStore)
	s.checkpointService = service.NewCheckpoint(s.checkpointStore, log)
	s.delegationService = service.NewDelegation(s.delegationStore, log)
	s.depositService = service.NewDeposit(s.depositStore, log)
	s.epochService = service.NewEpoch(s.epochStore, log)
	s.governanceService = service.NewGovernance(s.proposalStore, s.voteStore, log)
	s.keyRotationsService = service.NewKeyRotations(s.keyRotationsStore, log)
	s.ethereumKeyRotationsService = service.NewEthereumKeyRotation(s.ethereumKeyRotationsStore, log)
	s.ledgerService = service.NewLedger(s.ledger, log)
	s.liquidityProvisionService = service.NewLiquidityProvision(s.liquidityProvisionStore, log)
	s.marketDataService = service.NewMarketData(s.marketDataStore, log)
	s.marketDepthService = service.NewMarketDepth(s.orderStore, log)
	s.marketsService = service.NewMarkets(s.marketsStore, log)
	s.multiSigService = service.NewMultiSig(s.multiSigSignerAddedStore, log)
	s.networkLimitsService = service.NewNetworkLimits(s.networkLimitsStore, log)
	s.networkParameterService = service.NewNetworkParameter(s.netParamStore, log)
	s.nodeService = service.NewNode(s.nodeStore, log)
	s.notaryService = service.NewNotary(s.notaryStore, log)
	s.oracleDataService = service.NewOracleData(s.oracleDataStore, log)
	s.oracleSpecService = service.NewOracleSpec(s.oracleSpecStore, log)
	s.orderService = service.NewOrder(s.orderStore, log)
	s.partyService = service.NewParty(s.partyStore, log)
	s.positionService = service.NewPosition(s.positionStore, log)
	s.rewardService = service.NewReward(s.rewardStore, log)
	s.riskFactorService = service.NewRiskFactor(s.riskFactorStore, log)
	s.riskService = service.NewRisk(s.marginLevelsStore, s.accountStore, log)
	s.stakeLinkingService = service.NewStakeLinking(s.stakeLinkingStore, log)
	s.tradeService = service.NewTrade(s.tradeStore, log)
	s.transferService = service.NewTransfer(s.transfersStore, log)
	s.withdrawalService = service.NewWithdrawal(s.withdrawalsStore, log)
	s.chainService = service.NewChain(s.chainStore, log)
	s.protocolUpgradeService = service.NewProtocolUpgrade(s.pupStore, log)
	s.coreSnapshotService = service.NewSnapshotData(s.snapStore, log)

	toInit := []interface{ Initialise(context.Context) error }{
		s.marketDepthService,
		s.marketDataService,
		s.marketsService,
	}

	for _, svc := range toInit {
		if err := svc.Initialise(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (s *SQLSubscribers) SetupSQLSubscribers(ctx context.Context, Log *logging.Logger) {
	s.accountSub = sqlsubscribers.NewAccount(s.accountService, Log)
	s.assetSub = sqlsubscribers.NewAsset(s.assetService, Log)
	s.partySub = sqlsubscribers.NewParty(s.partyService, Log)
	s.transferResponseSub = sqlsubscribers.NewTransferResponse(s.ledgerService, s.accountService, Log)
	s.orderSub = sqlsubscribers.NewOrder(s.orderService, Log)
	s.networkLimitsSub = sqlsubscribers.NewNetworkLimitSub(ctx, s.networkLimitsService, Log)
	s.marketDataSub = sqlsubscribers.NewMarketData(s.marketDataService, Log)
	s.tradesSub = sqlsubscribers.NewTradesSubscriber(s.tradeService, Log)
	s.rewardsSub = sqlsubscribers.NewReward(s.rewardService, Log)
	s.marketCreatedSub = sqlsubscribers.NewMarketCreated(s.marketsService, Log)
	s.marketUpdatedSub = sqlsubscribers.NewMarketUpdated(s.marketsService, Log)
	s.delegationsSub = sqlsubscribers.NewDelegation(s.delegationService, Log)
	s.epochSub = sqlsubscribers.NewEpoch(s.epochService, Log)
	s.depositSub = sqlsubscribers.NewDeposit(s.depositService, Log)
	s.withdrawalSub = sqlsubscribers.NewWithdrawal(s.withdrawalService, Log)
	s.proposalsSub = sqlsubscribers.NewProposal(s.governanceService, Log)
	s.votesSub = sqlsubscribers.NewVote(s.governanceService, Log)
	s.marginLevelsSub = sqlsubscribers.NewMarginLevels(s.riskService, s.accountStore, Log)
	s.riskFactorSub = sqlsubscribers.NewRiskFactor(s.riskFactorService, Log)
	s.netParamSub = sqlsubscribers.NewNetworkParameter(s.networkParameterService, Log)
	s.checkpointSub = sqlsubscribers.NewCheckpoint(s.checkpointService, Log)
	s.positionsSub = sqlsubscribers.NewPosition(s.positionService, Log)
	s.oracleSpecSub = sqlsubscribers.NewOracleSpec(s.oracleSpecService, Log)
	s.oracleDataSub = sqlsubscribers.NewOracleData(s.oracleDataService, Log)
	s.liquidityProvisionSub = sqlsubscribers.NewLiquidityProvision(s.liquidityProvisionService, Log)
	s.transferSub = sqlsubscribers.NewTransfer(s.transfersStore, s.accountService, Log)
	s.stakeLinkingSub = sqlsubscribers.NewStakeLinking(s.stakeLinkingService, Log)
	s.notarySub = sqlsubscribers.NewNotary(s.notaryService, Log)
	s.multiSigSignerEventSub = sqlsubscribers.NewERC20MultiSigSignerEvent(s.multiSigService, Log)
	s.keyRotationsSub = sqlsubscribers.NewKeyRotation(s.keyRotationsService, Log)
	s.ethereumKeyRotationsSub = sqlsubscribers.NewEthereumKeyRotation(s.ethereumKeyRotationsService, Log)
	s.nodeSub = sqlsubscribers.NewNode(s.nodeService, Log)
	s.marketDepthSub = sqlsubscribers.NewMarketDepth(s.marketDepthService)
	s.pupSub = sqlsubscribers.NewProtocolUpgrade(s.protocolUpgradeService, Log)
	s.snapSub = sqlsubscribers.NewSnapshotData(s.coreSnapshotService, Log)
}
