package protocol

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/banking"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/blockchain/nullchain"
	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/checkpoint"
	ethclient "code.vegaprotocol.io/vega/client/eth"
	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/delegation"
	"code.vegaprotocol.io/vega/epochtime"
	"code.vegaprotocol.io/vega/evtforward"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/limits"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/netparams/checks"
	"code.vegaprotocol.io/vega/netparams/dispatch"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/notary"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/oracles/adaptors"
	oracleAdaptors "code.vegaprotocol.io/vega/oracles/adaptors"
	"code.vegaprotocol.io/vega/plugins"
	"code.vegaprotocol.io/vega/pow"
	"code.vegaprotocol.io/vega/processor"
	"code.vegaprotocol.io/vega/rewards"
	"code.vegaprotocol.io/vega/snapshot"
	"code.vegaprotocol.io/vega/spam"
	"code.vegaprotocol.io/vega/staking"
	"code.vegaprotocol.io/vega/statevar"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/validators"
	"code.vegaprotocol.io/vega/validators/erc20multisig"
	"code.vegaprotocol.io/vega/vegatime"
)

type allServices struct {
	ctx             context.Context
	log             *logging.Logger
	confWatcher     *config.Watcher
	confListenerIDs []int
	conf            config.Config

	broker *broker.Broker

	timeService  *vegatime.Svc
	epochService *epochtime.Svc
	eventService *subscribers.Service

	blockchainClient *blockchain.Client

	stats *stats.Stats

	vegaPaths paths.Paths

	marketActivityTracker *execution.MarketActivityTracker
	statevar              *statevar.Engine
	snapshot              *snapshot.Engine
	executionEngine       *execution.Engine
	governance            *governance.Engine
	collateral            *collateral.Engine
	oracle                *oracles.Engine
	oracleAdaptors        *adaptors.Adaptors
	netParams             *netparams.Store
	delegation            *delegation.Engine
	limits                *limits.Engine
	rewards               *rewards.Engine
	checkpoint            *checkpoint.Engine
	spam                  *spam.Engine
	pow                   processor.PoWEngine
	builtinOracle         *oracles.Builtin
	codec                 abci.Codec

	assets               *assets.Service
	topology             *validators.Topology
	notary               *notary.SnapshotNotary
	eventForwarder       *evtforward.Forwarder
	eventForwarderEngine EventForwarderEngine
	witness              *validators.Witness
	banking              *banking.Engine
	genesisHandler       *genesis.Handler

	// plugins
	settlePlugin     *plugins.Positions
	notaryPlugin     *plugins.Notary
	assetPlugin      *plugins.Asset
	withdrawalPlugin *plugins.Withdrawal
	depositPlugin    *plugins.Deposit

	// staking
	ethClient             *ethclient.Client
	ethConfirmations      *ethclient.EthereumConfirmations
	stakingAccounts       *staking.Accounting
	stakeVerifier         *staking.StakeVerifier
	stakeCheckpoint       *staking.Checkpoint
	erc20MultiSigTopology *erc20multisig.Topology

	commander *nodewallets.Commander
}

func newServices(
	ctx context.Context,
	log *logging.Logger,
	conf *config.Watcher,
	// this is a parameter as not reloaded as part of the protocol
	nodeWallets *nodewallets.NodeWallets,
	ethClient *ethclient.Client,
	ethConfirmations *ethclient.EthereumConfirmations,
	blockchainClient *blockchain.Client,
	vegaPaths paths.Paths,
	stats *stats.Stats,
) (_ *allServices, err error) {
	svcs := &allServices{
		ctx:              ctx,
		log:              log,
		confWatcher:      conf,
		conf:             conf.Get(),
		ethClient:        ethClient,
		ethConfirmations: ethConfirmations,
		blockchainClient: blockchainClient,
		stats:            stats,
		vegaPaths:        vegaPaths,
	}

	svcs.broker, err = broker.New(svcs.ctx, svcs.log, svcs.conf.Broker)
	if err != nil {
		svcs.log.Error("unable to initialise broker", logging.Error(err))
		return nil, err
	}

	svcs.timeService = vegatime.New(svcs.conf.Time, svcs.broker)
	svcs.epochService = epochtime.NewService(svcs.log, svcs.conf.Epoch, svcs.timeService, svcs.broker)
	// if we are not a validator, no need to instantiate the commander
	if svcs.conf.IsValidator() {
		// we cannot pass the Chain dependency here (that's set by the blockchain)
		svcs.commander, err = nodewallets.NewCommander(
			svcs.conf.NodeWallet, svcs.log, blockchainClient, nodeWallets.Vega, svcs.stats)
		if err != nil {
			return nil, err
		}
	}

	// plugins
	svcs.settlePlugin = plugins.NewPositions(svcs.ctx)
	svcs.notaryPlugin = plugins.NewNotary(svcs.ctx)
	svcs.assetPlugin = plugins.NewAsset(svcs.ctx)
	svcs.withdrawalPlugin = plugins.NewWithdrawal(svcs.ctx)
	svcs.depositPlugin = plugins.NewDeposit(svcs.ctx)

	svcs.genesisHandler = genesis.New(svcs.log, svcs.conf.Genesis)

	svcs.genesisHandler.OnGenesisTimeLoaded(svcs.timeService.SetTimeNow)
	svcs.eventService = subscribers.NewService(svcs.broker)

	now := svcs.timeService.GetTimeNow()
	svcs.assets = assets.New(svcs.log, svcs.conf.Assets, nodeWallets, svcs.ethClient, svcs.timeService, svcs.conf.HaveEthClient())
	svcs.collateral = collateral.New(svcs.log, svcs.conf.Collateral, svcs.broker, now)
	svcs.oracle = oracles.NewEngine(svcs.log, svcs.conf.Oracles, now, svcs.broker, svcs.timeService)
	svcs.builtinOracle = oracles.NewBuiltinOracle(svcs.oracle, svcs.timeService)
	svcs.oracleAdaptors = oracleAdaptors.New()

	svcs.limits = limits.New(svcs.log, svcs.conf.Limits, svcs.broker)
	svcs.timeService.NotifyOnTick(svcs.limits.OnTick)
	svcs.netParams = netparams.New(svcs.log, svcs.conf.NetworkParameters, svcs.broker)

	svcs.erc20MultiSigTopology = erc20multisig.NewERC20MultisigTopology(
		svcs.conf.ERC20MultiSig, svcs.log, nil, svcs.broker, svcs.ethClient, svcs.ethConfirmations, svcs.netParams,
	)

	if svcs.conf.IsValidator() {
		svcs.topology = validators.NewTopology(
			svcs.log, svcs.conf.Validators, validators.WrapNodeWallets(nodeWallets), svcs.broker, svcs.conf.IsValidator(), svcs.commander, svcs.erc20MultiSigTopology)
	} else {
		svcs.topology = validators.NewTopology(svcs.log, svcs.conf.Validators, nil, svcs.broker, svcs.conf.IsValidator(), nil, svcs.erc20MultiSigTopology)
	}

	svcs.witness = validators.NewWitness(svcs.log, svcs.conf.Validators, svcs.topology, svcs.commander, svcs.timeService)

	// this is done to go around circular deps...
	svcs.erc20MultiSigTopology.SetWitness(svcs.witness)

	svcs.timeService.NotifyOnTick(svcs.erc20MultiSigTopology.OnTick)

	svcs.timeService.NotifyOnTick(svcs.netParams.OnChainTimeUpdate)
	svcs.eventForwarder = evtforward.New(svcs.log, svcs.conf.EvtForward, svcs.commander, svcs.timeService, svcs.topology)

	if svcs.conf.HaveEthClient() {
		svcs.eventForwarderEngine = evtforward.NewEngine(svcs.log, svcs.conf.EvtForward)
	} else {
		svcs.eventForwarderEngine = evtforward.NewNoopEngine(svcs.log, svcs.conf.EvtForward)
	}

	// this is done to go around circular deps again...
	svcs.erc20MultiSigTopology.SetEthereumEventSource(svcs.eventForwarderEngine)

	svcs.stakingAccounts, svcs.stakeVerifier, svcs.stakeCheckpoint = staking.New(
		svcs.log, svcs.conf.Staking, svcs.broker, svcs.timeService, svcs.witness, svcs.ethClient, svcs.netParams, svcs.eventForwarder, svcs.conf.HaveEthClient(), svcs.ethConfirmations, svcs.eventForwarderEngine,
	)
	svcs.epochService.NotifyOnEpoch(svcs.topology.OnEpochEvent, svcs.topology.OnEpochRestore)

	svcs.statevar = statevar.New(svcs.log, svcs.conf.StateVar, svcs.broker, svcs.topology, svcs.commander, svcs.timeService)
	svcs.marketActivityTracker = execution.NewMarketActivityTracker(svcs.log, svcs.epochService)

	// instantiate the execution engine
	svcs.executionEngine = execution.NewEngine(
		svcs.log, svcs.conf.Execution, svcs.timeService, svcs.collateral, svcs.oracle, svcs.broker, svcs.statevar, svcs.marketActivityTracker, svcs.assets,
	)

	if svcs.conf.Blockchain.ChainProvider == blockchain.ProviderNullChain {
		// Use staking-loop to pretend a dummy builtin assets deposited with the faucet was staked
		svcs.codec = &processor.NullBlockchainTxCodec{}
		stakingLoop := nullchain.NewStakingLoop(svcs.collateral, svcs.assets)
		svcs.governance = governance.NewEngine(svcs.log, svcs.conf.Governance, stakingLoop, svcs.broker, svcs.assets, svcs.witness, svcs.executionEngine, svcs.netParams, now)
		svcs.delegation = delegation.New(svcs.log, svcs.conf.Delegation, svcs.broker, svcs.topology, stakingLoop, svcs.epochService, svcs.timeService)
	} else {
		svcs.codec = &processor.TxCodec{}
		svcs.spam = spam.New(svcs.log, svcs.conf.Spam, svcs.epochService, svcs.stakingAccounts)
		svcs.governance = governance.NewEngine(svcs.log, svcs.conf.Governance, svcs.stakingAccounts, svcs.broker, svcs.assets, svcs.witness, svcs.executionEngine, svcs.netParams, now)
		svcs.delegation = delegation.New(svcs.log, svcs.conf.Delegation, svcs.broker, svcs.topology, svcs.stakingAccounts, svcs.epochService, svcs.timeService)
	}

	svcs.notary = notary.NewWithSnapshot(svcs.log, svcs.conf.Notary, svcs.topology, svcs.broker, svcs.commander, svcs.timeService)
	// TODO(): this is not pretty
	svcs.topology.SetNotary(svcs.notary)

	svcs.banking = banking.New(svcs.log, svcs.conf.Banking, svcs.collateral, svcs.witness, svcs.timeService, svcs.assets, svcs.notary, svcs.broker, svcs.topology, svcs.epochService, svcs.marketActivityTracker)

	svcs.rewards = rewards.New(svcs.log, svcs.conf.Rewards, svcs.broker, svcs.delegation, svcs.epochService, svcs.collateral, svcs.timeService, svcs.marketActivityTracker, svcs.topology)

	// checkpoint engine
	svcs.checkpoint, err = checkpoint.New(svcs.log, svcs.conf.Checkpoint, svcs.assets, svcs.collateral, svcs.governance, svcs.netParams, svcs.delegation, svcs.epochService, svcs.topology, svcs.banking, svcs.stakeCheckpoint, svcs.erc20MultiSigTopology, svcs.marketActivityTracker)
	if err != nil {
		return nil, err
	}

	// register the callback to startup stuff when checkpoint is loaded
	svcs.checkpoint.RegisterOnCheckpointLoaded(func(_ context.Context) {
		// checkpoint have been loaded
		// which means that genesis has been loaded as well
		// we should be fully ready to start the event sourcing from ethereum
		svcs.eventForwarderEngine.Start()
	})

	svcs.genesisHandler.OnGenesisAppStateLoaded(
		// be sure to keep this in order.
		// the node upon genesis will load all asset first in the node
		// state. This is important to happened first as we will load the
		// asset which will be considered as the governance tokesvcs.
		svcs.UponGenesis,
		// This needs to happen always after, as it defined the network
		// parameters, one of them is  the Governance Token asset ID.
		// which if not loaded in the previous state, then will make the node
		// panic at startup.
		svcs.netParams.UponGenesis,
		svcs.topology.LoadValidatorsOnGenesis,
		svcs.limits.UponGenesis,
		svcs.checkpoint.UponGenesis,
	)

	svcs.snapshot, err = snapshot.New(svcs.ctx, svcs.vegaPaths, svcs.conf.Snapshot, svcs.log, svcs.timeService, svcs.stats.Blockchain)
	if err != nil {
		return nil, fmt.Errorf("failed to start snapshot engine: %w", err)
	}

	// notify delegation, rewards, and accounting on changes in the validator pub key
	svcs.topology.NotifyOnKeyChange(svcs.delegation.ValidatorKeyChanged, svcs.stakingAccounts.ValidatorKeyChanged, svcs.governance.ValidatorKeyChanged)

	svcs.snapshot.AddProviders(svcs.checkpoint, svcs.collateral, svcs.governance, svcs.delegation, svcs.netParams, svcs.epochService, svcs.assets, svcs.banking, svcs.witness,
		svcs.notary, svcs.stakingAccounts, svcs.stakeVerifier, svcs.limits, svcs.topology, svcs.eventForwarder, svcs.executionEngine, svcs.marketActivityTracker, svcs.statevar, svcs.erc20MultiSigTopology)

	if svcs.spam != nil {
		svcs.snapshot.AddProviders(svcs.spam)
	}
	powWatchers := []netparams.WatchParam{}

	if svcs.conf.Blockchain.ChainProvider == blockchain.ProviderNullChain {
		svcs.pow = pow.NewNoop()
	} else {
		pow := pow.New(svcs.log, svcs.conf.PoW, svcs.epochService)
		svcs.pow = pow
		svcs.snapshot.AddProviders(pow)
		powWatchers = []netparams.WatchParam{
			{
				Param:   netparams.SpamPoWNumberOfPastBlocks,
				Watcher: pow.UpdateSpamPoWNumberOfPastBlocks,
			},
			{
				Param:   netparams.SpamPoWDifficulty,
				Watcher: pow.UpdateSpamPoWDifficulty,
			},
			{
				Param:   netparams.SpamPoWHashFunction,
				Watcher: pow.UpdateSpamPoWHashFunction,
			},
			{
				Param:   netparams.SpamPoWIncreasingDifficulty,
				Watcher: pow.UpdateSpamPoWIncreasingDifficulty,
			},
			{
				Param:   netparams.SpamPoWNumberOfTxPerBlock,
				Watcher: pow.UpdateSpamPoWNumberOfTxPerBlock,
			},
		}
	}

	// setup config reloads for all engines / services /etc
	svcs.registerConfigWatchers()

	// setup some network parameters runtime validations and network parameters
	// updates dispatches this must come before we try to load from a snapshot,
	// which happens in startBlockchain
	if err := svcs.setupNetParameters(powWatchers); err != nil {
		return nil, err
	}

	return svcs, nil
}

func (svcs *allServices) Stop() {
	svcs.confWatcher.Unregister(svcs.confListenerIDs)
	svcs.eventForwarderEngine.Stop()
	svcs.snapshot.Close()
}

func (svcs *allServices) registerConfigWatchers() {
	svcs.confListenerIDs = svcs.confWatcher.OnConfigUpdateWithID(
		func(cfg config.Config) { svcs.executionEngine.ReloadConf(cfg.Execution) },
		func(cfg config.Config) { svcs.notary.ReloadConf(cfg.Notary) },
		func(cfg config.Config) { svcs.eventForwarderEngine.ReloadConf(cfg.EvtForward) },
		func(cfg config.Config) { svcs.eventForwarder.ReloadConf(cfg.EvtForward) },
		func(cfg config.Config) { svcs.topology.ReloadConf(cfg.Validators) },
		func(cfg config.Config) { svcs.witness.ReloadConf(cfg.Validators) },
		func(cfg config.Config) { svcs.assets.ReloadConf(cfg.Assets) },
		func(cfg config.Config) { svcs.banking.ReloadConf(cfg.Banking) },
		func(cfg config.Config) { svcs.governance.ReloadConf(cfg.Governance) },
		func(cfg config.Config) { svcs.stats.ReloadConf(cfg.Stats) },
	)
	svcs.timeService.NotifyOnTick(svcs.confWatcher.OnTimeUpdate)
}

func (svcs *allServices) setupNetParameters(powWatchers []netparams.WatchParam) error {
	// now we are going to setup some network parameters which can be done
	// through runtime checks
	// e.g: changing the governance asset require the Assets and Collateral engines, so we can ensure any changes there are made for a valid asset

	if err := svcs.netParams.AddRules(
		netparams.ParamStringRules(
			netparams.RewardAsset,
			checks.RewardAssetUpdate(svcs.log, svcs.assets, svcs.collateral),
		),
	); err != nil {
		return err
	}

	spamWatchers := []netparams.WatchParam{}
	if svcs.spam != nil {
		spamWatchers = []netparams.WatchParam{
			{
				Param:   netparams.SpamProtectionMaxVotes,
				Watcher: svcs.spam.OnMaxVotesChanged,
			},
			{
				Param:   netparams.StakingAndDelegationRewardMinimumValidatorStake,
				Watcher: svcs.spam.OnMinValidatorTokensChanged,
			},
			{
				Param:   netparams.SpamProtectionMaxProposals,
				Watcher: svcs.spam.OnMaxProposalsChanged,
			},
			{
				Param:   netparams.SpamProtectionMaxDelegations,
				Watcher: svcs.spam.OnMaxDelegationsChanged,
			},
			{
				Param:   netparams.SpamProtectionMinTokensForProposal,
				Watcher: svcs.spam.OnMinTokensForProposalChanged,
			},
			{
				Param:   netparams.SpamProtectionMinTokensForVoting,
				Watcher: svcs.spam.OnMinTokensForVotingChanged,
			},
			{
				Param:   netparams.SpamProtectionMinTokensForDelegation,
				Watcher: svcs.spam.OnMinTokensForDelegationChanged,
			},
			{
				Param:   netparams.TransferMaxCommandsPerEpoch,
				Watcher: svcs.spam.OnMaxTransfersChanged,
			},
		}
	}

	watchers := []netparams.WatchParam{
		{
			Param:   netparams.ValidatorsEpochLength,
			Watcher: svcs.topology.OnEpochLengthUpdate,
		},
		{
			Param:   netparams.NumberOfTendermintValidators,
			Watcher: svcs.topology.UpdateNumberOfTendermintValidators,
		},
		{
			Param:   netparams.ValidatorIncumbentBonus,
			Watcher: svcs.topology.UpdateValidatorIncumbentBonusFactor,
		},
		{
			Param:   netparams.NumberEthMultisigSigners,
			Watcher: svcs.topology.UpdateNumberEthMultisigSigners,
		},
		{
			Param:   netparams.MultipleOfTendermintValidatorsForEtsatzSet,
			Watcher: svcs.topology.UpdateErsatzValidatorsFactor,
		},
		{
			Param:   netparams.MinimumEthereumEventsForNewValidator,
			Watcher: svcs.topology.UpdateMinimumEthereumEventsForNewValidator,
		},
		{
			Param:   netparams.StakingAndDelegationRewardMinimumValidatorStake,
			Watcher: svcs.topology.UpdateMinimumRequireSelfStake,
		},
		{
			Param:   netparams.DelegationMinAmount,
			Watcher: svcs.delegation.OnMinAmountChanged,
		},
		{
			Param:   netparams.RewardAsset,
			Watcher: dispatch.RewardAssetUpdate(svcs.log, svcs.assets),
		},
		{
			Param:   netparams.MarketMarginScalingFactors,
			Watcher: svcs.executionEngine.OnMarketMarginScalingFactorsUpdate,
		},
		{
			Param:   netparams.MarketFeeFactorsMakerFee,
			Watcher: svcs.executionEngine.OnMarketFeeFactorsMakerFeeUpdate,
		},
		{
			Param:   netparams.MarketFeeFactorsInfrastructureFee,
			Watcher: svcs.executionEngine.OnMarketFeeFactorsInfrastructureFeeUpdate,
		},
		{
			Param:   netparams.MarketLiquidityStakeToCCYSiskas,
			Watcher: svcs.executionEngine.OnSuppliedStakeToObligationFactorUpdate,
		},
		{
			Param:   netparams.MarketValueWindowLength,
			Watcher: svcs.executionEngine.OnMarketValueWindowLengthUpdate,
		},
		{
			Param:   netparams.MarketTargetStakeScalingFactor,
			Watcher: svcs.executionEngine.OnMarketTargetStakeScalingFactorUpdate,
		},
		{
			Param:   netparams.MarketTargetStakeTimeWindow,
			Watcher: svcs.executionEngine.OnMarketTargetStakeTimeWindowUpdate,
		},
		{
			Param: netparams.BlockchainsEthereumConfig,
			Watcher: func(ctx context.Context, cfg interface{}) error {
				ethCfg, err := types.EthereumConfigFromUntypedProto(cfg)
				if err != nil {
					return fmt.Errorf("invalid Ethereum configuration: %w", err)
				}

				if err := svcs.ethClient.UpdateEthereumConfig(ethCfg); err != nil {
					return err
				}

				return svcs.eventForwarderEngine.SetupEthereumEngine(svcs.ethClient, svcs.eventForwarder, svcs.conf.EvtForward.Ethereum, ethCfg, svcs.assets)
			},
		},
		{
			Param:   netparams.MarketLiquidityProvidersFeeDistribitionTimeStep,
			Watcher: svcs.executionEngine.OnMarketLiquidityProvidersFeeDistributionTimeStep,
		},
		{
			Param:   netparams.MarketLiquidityProvisionShapesMaxSize,
			Watcher: svcs.executionEngine.OnMarketLiquidityProvisionShapesMaxSizeUpdate,
		},
		{
			Param:   netparams.MarketMinLpStakeQuantumMultiple,
			Watcher: svcs.executionEngine.OnMinLpStakeQuantumMultipleUpdate,
		},
		{
			Param:   netparams.MarketLiquidityMaximumLiquidityFeeFactorLevel,
			Watcher: svcs.executionEngine.OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate,
		},
		{
			Param:   netparams.MarketLiquidityBondPenaltyParameter,
			Watcher: svcs.executionEngine.OnMarketLiquidityBondPenaltyUpdate,
		},
		{
			Param:   netparams.MarketLiquidityTargetStakeTriggeringRatio,
			Watcher: svcs.executionEngine.OnMarketLiquidityTargetStakeTriggeringRatio,
		},
		{
			Param:   netparams.MarketAuctionMinimumDuration,
			Watcher: svcs.executionEngine.OnMarketAuctionMinimumDurationUpdate,
		},
		{
			Param:   netparams.MarketProbabilityOfTradingTauScaling,
			Watcher: svcs.executionEngine.OnMarketProbabilityOfTradingTauScalingUpdate,
		},
		{
			Param:   netparams.MarketMinProbabilityOfTradingForLPOrders,
			Watcher: svcs.executionEngine.OnMarketMinProbabilityOfTradingForLPOrdersUpdate,
		},
		{
			Param:   netparams.ValidatorsEpochLength,
			Watcher: svcs.epochService.OnEpochLengthUpdate,
		},
		{
			Param:   netparams.StakingAndDelegationRewardMaxPayoutPerParticipant,
			Watcher: svcs.rewards.UpdateMaxPayoutPerParticipantForStakingRewardScheme,
		},
		{
			Param:   netparams.StakingAndDelegationRewardDelegatorShare,
			Watcher: svcs.rewards.UpdateDelegatorShareForStakingRewardScheme,
		},
		{
			Param:   netparams.StakingAndDelegationRewardMinimumValidatorStake,
			Watcher: svcs.rewards.UpdateMinimumValidatorStakeForStakingRewardScheme,
		},
		{
			Param:   netparams.RewardAsset,
			Watcher: svcs.rewards.UpdateAssetForStakingAndDelegation,
		},
		{
			Param:   netparams.StakingAndDelegationRewardCompetitionLevel,
			Watcher: svcs.rewards.UpdateCompetitionLevelForStakingRewardScheme,
		},
		{
			Param:   netparams.StakingAndDelegationRewardsMinValidators,
			Watcher: svcs.rewards.UpdateMinValidatorsStakingRewardScheme,
		},
		{
			Param:   netparams.StakingAndDelegationRewardOptimalStakeMultiplier,
			Watcher: svcs.rewards.UpdateOptimalStakeMultiplierStakingRewardScheme,
		},
		{
			Param:   netparams.ErsatzvalidatorsRewardFactor,
			Watcher: svcs.rewards.UpdateErsatzRewardFactor,
		},
		{
			Param:   netparams.ValidatorsVoteRequired,
			Watcher: svcs.witness.OnDefaultValidatorsVoteRequiredUpdate,
		},
		{
			Param:   netparams.ValidatorsVoteRequired,
			Watcher: svcs.notary.OnDefaultValidatorsVoteRequiredUpdate,
		},
		{
			Param:   netparams.NetworkCheckpointTimeElapsedBetweenCheckpoints,
			Watcher: svcs.checkpoint.OnTimeElapsedUpdate,
		},

		{
			Param:   netparams.SnapshotIntervalLength,
			Watcher: svcs.snapshot.OnSnapshotIntervalUpdate,
		},
		{
			Param:   netparams.ValidatorsVoteRequired,
			Watcher: svcs.statevar.OnDefaultValidatorsVoteRequiredUpdate,
		},
		{
			Param:   netparams.FloatingPointUpdatesDuration,
			Watcher: svcs.statevar.OnFloatingPointUpdatesDurationUpdate,
		},
		{
			Param:   netparams.TransferFeeFactor,
			Watcher: svcs.banking.OnTransferFeeFactorUpdate,
		},
		{
			Param:   netparams.TransferMinTransferQuantumMultiple,
			Watcher: svcs.banking.OnMinTransferQuantumMultiple,
		},
		{
			Param: netparams.BlockchainsEthereumConfig,
			Watcher: func(_ context.Context, cfg interface{}) error {
				// nothing to do if not a validator
				if !svcs.conf.HaveEthClient() {
					return nil
				}
				ethCfg, err := types.EthereumConfigFromUntypedProto(cfg)
				if err != nil {
					return fmt.Errorf("invalid ethereum configuration: %w", err)
				}

				svcs.ethConfirmations.UpdateConfirmations(ethCfg.Confirmations())
				return nil
			},
		},
		{
			Param:   netparams.LimitsProposeMarketEnabledFrom,
			Watcher: svcs.limits.OnLimitsProposeMarketEnabledFromUpdate,
		},
		{
			Param:   netparams.LimitsProposeAssetEnabledFrom,
			Watcher: svcs.limits.OnLimitsProposeAssetEnabledFromUpdate,
		},
	}

	watchers = append(watchers, powWatchers...)
	watchers = append(watchers, spamWatchers...)

	// now add some watcher for our netparams
	return svcs.netParams.Watch(watchers...)
}
