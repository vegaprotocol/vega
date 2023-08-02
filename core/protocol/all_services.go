// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package protocol

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/datasource"
	"code.vegaprotocol.io/vega/core/datasource/external/ethcall"
	"code.vegaprotocol.io/vega/core/datasource/external/ethverifier"

	"code.vegaprotocol.io/vega/libs/subscribers"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/banking"
	"code.vegaprotocol.io/vega/core/blockchain"
	"code.vegaprotocol.io/vega/core/blockchain/abci"
	"code.vegaprotocol.io/vega/core/blockchain/nullchain"
	"code.vegaprotocol.io/vega/core/bridges"
	"code.vegaprotocol.io/vega/core/broker"
	"code.vegaprotocol.io/vega/core/checkpoint"
	ethclient "code.vegaprotocol.io/vega/core/client/eth"
	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/datasource/spec/adaptors"
	oracleAdaptors "code.vegaprotocol.io/vega/core/datasource/spec/adaptors"
	"code.vegaprotocol.io/vega/core/delegation"
	"code.vegaprotocol.io/vega/core/epochtime"
	"code.vegaprotocol.io/vega/core/evtforward"
	"code.vegaprotocol.io/vega/core/execution"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/genesis"
	"code.vegaprotocol.io/vega/core/governance"
	"code.vegaprotocol.io/vega/core/limits"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/netparams/checks"
	"code.vegaprotocol.io/vega/core/netparams/dispatch"
	"code.vegaprotocol.io/vega/core/nodewallets"
	"code.vegaprotocol.io/vega/core/notary"
	"code.vegaprotocol.io/vega/core/pow"
	"code.vegaprotocol.io/vega/core/processor"
	"code.vegaprotocol.io/vega/core/protocolupgrade"
	"code.vegaprotocol.io/vega/core/rewards"
	"code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/spam"
	"code.vegaprotocol.io/vega/core/staking"
	"code.vegaprotocol.io/vega/core/statevar"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/core/validators/erc20multisig"
	"code.vegaprotocol.io/vega/core/vegatime"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/version"
)

type EthCallEngine interface {
	Start()
	Stop()
	MakeResult(specID string, bytes []byte) (ethcall.Result, error)
	CallSpec(ctx context.Context, id string, atBlock uint64) (ethcall.Result, error)
	GetRequiredConfirmations(id string) (uint64, error)
	OnSpecActivated(ctx context.Context, spec datasource.Spec) error
	OnSpecDeactivated(ctx context.Context, spec datasource.Spec)
	UpdatePreviousEthBlock(height uint64, timestamp uint64)
}

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

	marketActivityTracker   *common.MarketActivityTracker
	statevar                *statevar.Engine
	snapshotEngine          *snapshot.Engine
	executionEngine         *execution.Engine
	governance              *governance.Engine
	collateral              *collateral.Engine
	oracle                  *spec.Engine
	oracleAdaptors          *adaptors.Adaptors
	netParams               *netparams.Store
	delegation              *delegation.Engine
	limits                  *limits.Engine
	rewards                 *rewards.Engine
	checkpoint              *checkpoint.Engine
	spam                    *spam.Engine
	pow                     processor.PoWEngine
	builtinOracle           *spec.Builtin
	codec                   abci.Codec
	ethereumOraclesVerifier *ethverifier.Verifier

	assets                *assets.Service
	topology              *validators.Topology
	notary                *notary.SnapshotNotary
	eventForwarder        *evtforward.Forwarder
	eventForwarderEngine  EventForwarderEngine
	ethCallEngine         EthCallEngine
	witness               *validators.Witness
	banking               *banking.Engine
	genesisHandler        *genesis.Handler
	protocolUpgradeEngine *protocolupgrade.Engine

	// staking
	ethClient             *ethclient.Client
	ethConfirmations      *ethclient.EthereumConfirmations
	stakingAccounts       *staking.Accounting
	stakeVerifier         *staking.StakeVerifier
	stakeCheckpoint       *staking.Checkpoint
	erc20MultiSigTopology *erc20multisig.Topology

	erc20BridgeView *bridges.ERC20LogicView

	commander  *nodewallets.Commander
	gastimator *processor.Gastimator
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

	svcs.broker, err = broker.New(svcs.ctx, svcs.log, svcs.conf.Broker, stats.Blockchain)
	if err != nil {
		svcs.log.Error("unable to initialise broker", logging.Error(err))
		return nil, err
	}

	// this will be needed very soon, instantiate straight away
	svcs.erc20BridgeView = bridges.NewERC20LogicView(ethClient, ethConfirmations)

	svcs.timeService = vegatime.New(svcs.conf.Time, svcs.broker)
	svcs.epochService = epochtime.NewService(svcs.log, svcs.conf.Epoch, svcs.broker)

	// if we are not a validator, no need to instantiate the commander
	if svcs.conf.IsValidator() {
		// we cannot pass the Chain dependency here (that's set by the blockchain)
		svcs.commander, err = nodewallets.NewCommander(
			svcs.conf.NodeWallet, svcs.log, blockchainClient, nodeWallets.Vega, svcs.stats)
		if err != nil {
			return nil, err
		}
	}

	svcs.genesisHandler = genesis.New(svcs.log, svcs.conf.Genesis)
	svcs.genesisHandler.OnGenesisTimeLoaded(svcs.timeService.SetTimeNow)

	svcs.eventService = subscribers.NewService(svcs.log, svcs.broker, svcs.conf.Broker.EventBusClientBufferSize)
	svcs.collateral = collateral.New(svcs.log, svcs.conf.Collateral, svcs.timeService, svcs.broker)

	svcs.limits = limits.New(svcs.log, svcs.conf.Limits, svcs.timeService, svcs.broker)

	svcs.netParams = netparams.New(svcs.log, svcs.conf.NetworkParameters, svcs.broker)

	svcs.erc20MultiSigTopology = erc20multisig.NewERC20MultisigTopology(
		svcs.conf.ERC20MultiSig, svcs.log, nil, svcs.broker, svcs.ethClient, svcs.ethConfirmations, svcs.netParams,
	)

	if svcs.conf.IsValidator() {
		svcs.topology = validators.NewTopology(
			svcs.log, svcs.conf.Validators, validators.WrapNodeWallets(nodeWallets), svcs.broker, svcs.conf.IsValidator(), svcs.commander, svcs.erc20MultiSigTopology, svcs.timeService)
	} else {
		svcs.topology = validators.NewTopology(svcs.log, svcs.conf.Validators, nil, svcs.broker, svcs.conf.IsValidator(), nil, svcs.erc20MultiSigTopology, svcs.timeService)
	}

	svcs.protocolUpgradeEngine = protocolupgrade.New(svcs.log, svcs.conf.ProtocolUpgrade, svcs.broker, svcs.topology, version.Get())
	svcs.witness = validators.NewWitness(svcs.ctx, svcs.log, svcs.conf.Validators, svcs.topology, svcs.commander, svcs.timeService)

	// this is done to go around circular deps...
	svcs.erc20MultiSigTopology.SetWitness(svcs.witness)
	svcs.eventForwarder = evtforward.New(svcs.log, svcs.conf.EvtForward, svcs.commander, svcs.timeService, svcs.topology)

	if svcs.conf.HaveEthClient() {
		svcs.eventForwarderEngine = evtforward.NewEngine(svcs.log, svcs.conf.EvtForward)
	} else {
		svcs.eventForwarderEngine = evtforward.NewNoopEngine(svcs.log, svcs.conf.EvtForward)
	}

	svcs.oracle = spec.NewEngine(svcs.log, svcs.conf.Oracles, svcs.timeService, svcs.broker)

	svcs.ethCallEngine = ethcall.NewEngine(svcs.log, svcs.conf.EvtForward.EthCall, svcs.ethClient, svcs.eventForwarder)

	if svcs.conf.IsValidator() && ethClient != nil {
		go svcs.ethCallEngine.Start()
	}

	svcs.ethereumOraclesVerifier = ethverifier.New(svcs.log, svcs.witness, svcs.timeService, svcs.broker,
		svcs.oracle, svcs.ethCallEngine, svcs.ethConfirmations)

	// Not using the activation event bus event here as on recovery the ethCallEngine needs to have all specs - is this necessary?
	svcs.oracle.AddSpecActivationListener(svcs.ethCallEngine)

	svcs.builtinOracle = spec.NewBuiltin(svcs.oracle, svcs.timeService)
	svcs.oracleAdaptors = oracleAdaptors.New()

	// this is done to go around circular deps again..s
	svcs.erc20MultiSigTopology.SetEthereumEventSource(svcs.eventForwarderEngine)

	svcs.stakingAccounts, svcs.stakeVerifier, svcs.stakeCheckpoint = staking.New(
		svcs.log, svcs.conf.Staking, svcs.timeService, svcs.broker, svcs.witness, svcs.ethClient, svcs.netParams, svcs.eventForwarder, svcs.conf.HaveEthClient(), svcs.ethConfirmations, svcs.eventForwarderEngine,
	)
	svcs.epochService.NotifyOnEpoch(svcs.topology.OnEpochEvent, svcs.topology.OnEpochRestore)
	svcs.epochService.NotifyOnEpoch(stats.OnEpochEvent, stats.OnEpochRestore)

	svcs.statevar = statevar.New(svcs.log, svcs.conf.StateVar, svcs.broker, svcs.topology, svcs.commander)
	svcs.marketActivityTracker = common.NewMarketActivityTracker(svcs.log, svcs.epochService)

	svcs.notary = notary.NewWithSnapshot(svcs.log, svcs.conf.Notary, svcs.topology, svcs.broker, svcs.commander)

	if svcs.conf.IsValidator() {
		svcs.assets = assets.New(svcs.log, svcs.conf.Assets, nodeWallets.Ethereum, svcs.ethClient, svcs.broker, svcs.erc20BridgeView, svcs.notary, svcs.conf.HaveEthClient())
	} else {
		svcs.assets = assets.New(svcs.log, svcs.conf.Assets, nil, svcs.ethClient, svcs.broker, svcs.erc20BridgeView, svcs.notary, svcs.conf.HaveEthClient())
	}

	// TODO(): this is not pretty
	svcs.topology.SetNotary(svcs.notary)

	// instantiate the execution engine
	svcs.executionEngine = execution.NewEngine(
		svcs.log, svcs.conf.Execution, svcs.timeService, svcs.collateral, svcs.oracle, svcs.broker, svcs.statevar, svcs.marketActivityTracker, svcs.assets,
	)
	svcs.epochService.NotifyOnEpoch(svcs.executionEngine.OnEpochEvent, svcs.executionEngine.OnEpochRestore)

	svcs.gastimator = processor.NewGastimator(svcs.executionEngine)

	svcs.banking = banking.New(svcs.log, svcs.conf.Banking, svcs.collateral, svcs.witness, svcs.timeService, svcs.assets, svcs.notary, svcs.broker, svcs.topology, svcs.epochService, svcs.marketActivityTracker, svcs.erc20BridgeView, svcs.eventForwarderEngine)
	svcs.spam = spam.New(svcs.log, svcs.conf.Spam, svcs.epochService, svcs.stakingAccounts)

	if svcs.conf.Blockchain.ChainProvider == blockchain.ProviderNullChain {
		// Use staking-loop to pretend a dummy builtin assets deposited with the faucet was staked
		svcs.codec = &processor.NullBlockchainTxCodec{}
		stakingLoop := nullchain.NewStakingLoop(svcs.collateral, svcs.assets)
		svcs.governance = governance.NewEngine(svcs.log, svcs.conf.Governance, stakingLoop, svcs.timeService, svcs.broker, svcs.assets, svcs.witness, svcs.executionEngine, svcs.netParams, svcs.banking)
		svcs.delegation = delegation.New(svcs.log, svcs.conf.Delegation, svcs.broker, svcs.topology, stakingLoop, svcs.epochService, svcs.timeService)

		svcs.spam.DisableSpamProtection() // Disable evaluation for the spam policies by the Spam Engine
	} else {
		svcs.codec = &processor.TxCodec{}
		svcs.spam = spam.New(svcs.log, svcs.conf.Spam, svcs.epochService, svcs.stakingAccounts)
		svcs.governance = governance.NewEngine(svcs.log, svcs.conf.Governance, svcs.stakingAccounts, svcs.timeService, svcs.broker, svcs.assets, svcs.witness, svcs.executionEngine, svcs.netParams, svcs.banking)
		svcs.delegation = delegation.New(svcs.log, svcs.conf.Delegation, svcs.broker, svcs.topology, svcs.stakingAccounts, svcs.epochService, svcs.timeService)
	}

	svcs.rewards = rewards.New(svcs.log, svcs.conf.Rewards, svcs.broker, svcs.delegation, svcs.epochService, svcs.collateral, svcs.timeService, svcs.marketActivityTracker, svcs.topology)

	svcs.registerTimeServiceCallbacks()

	// checkpoint engine
	svcs.checkpoint, err = checkpoint.New(svcs.log, svcs.conf.Checkpoint, svcs.assets, svcs.collateral, svcs.governance, svcs.netParams, svcs.delegation, svcs.epochService, svcs.topology, svcs.banking, svcs.stakeCheckpoint, svcs.erc20MultiSigTopology, svcs.marketActivityTracker, svcs.executionEngine)
	if err != nil {
		return nil, err
	}

	// register the callback to startup stuff when checkpoint is loaded
	svcs.checkpoint.RegisterOnCheckpointLoaded(func(_ context.Context) {
		// checkpoint have been loaded
		// which means that genesis has been loaded as well
		// we should be fully ready to start the event sourcing from ethereum
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

	svcs.snapshotEngine, err = snapshot.NewEngine(svcs.vegaPaths, svcs.conf.Snapshot, svcs.log, svcs.timeService, svcs.stats.Blockchain)
	if err != nil {
		return nil, fmt.Errorf("could not initialize the snapshot engine: %w", err)
	}

	// notify delegation, rewards, and accounting on changes in the validator pub key
	svcs.topology.NotifyOnKeyChange(svcs.governance.ValidatorKeyChanged)

	svcs.snapshotEngine.AddProviders(svcs.checkpoint, svcs.collateral, svcs.governance, svcs.delegation, svcs.netParams, svcs.epochService, svcs.assets, svcs.banking, svcs.witness,
		svcs.notary, svcs.stakingAccounts, svcs.stakeVerifier, svcs.limits, svcs.topology, svcs.eventForwarder, svcs.executionEngine, svcs.marketActivityTracker, svcs.statevar,
		svcs.erc20MultiSigTopology, svcs.protocolUpgradeEngine, svcs.ethereumOraclesVerifier)

	svcs.snapshotEngine.AddProviders(svcs.spam)

	pow := pow.New(svcs.log, svcs.conf.PoW, svcs.timeService)
	if svcs.conf.Blockchain.ChainProvider == blockchain.ProviderNullChain {
		pow.DisableVerification()
	}

	svcs.pow = pow
	svcs.snapshotEngine.AddProviders(pow)
	powWatchers := []netparams.WatchParam{
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
		{
			Param:   netparams.ValidatorsEpochLength,
			Watcher: pow.OnEpochDurationChanged,
		},
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

func (svcs *allServices) registerTimeServiceCallbacks() {
	svcs.timeService.NotifyOnTick(
		svcs.epochService.OnTick,
		svcs.builtinOracle.OnTick,

		svcs.netParams.OnTick,
		svcs.erc20MultiSigTopology.OnTick,
		svcs.witness.OnTick,

		svcs.eventForwarder.OnTick,
		svcs.stakeVerifier.OnTick,
		svcs.statevar.OnTick,
		svcs.executionEngine.OnTick,
		svcs.delegation.OnTick,
		svcs.notary.OnTick,
		svcs.banking.OnTick,
		svcs.assets.OnTick,
		svcs.limits.OnTick,

		svcs.ethereumOraclesVerifier.OnTick,
	)
}

func (svcs *allServices) Stop() {
	svcs.confWatcher.Unregister(svcs.confListenerIDs)
	svcs.eventForwarderEngine.Stop()
	svcs.snapshotEngine.Close()
	svcs.ethCallEngine.Stop()
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
				Param:   netparams.ValidatorsEpochLength,
				Watcher: svcs.spam.OnEpochDurationChanged,
			},
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
			{
				Param:   netparams.SpamProtectionMinMultisigUpdates,
				Watcher: svcs.spam.OnMinTokensForMultisigUpdatesChanged,
			},
		}
	}

	watchers := []netparams.WatchParam{
		{
			Param:   netparams.MinBlockCapacity,
			Watcher: svcs.gastimator.OnMinBlockCapacityUpdate,
		},
		{
			Param:   netparams.MaxGasPerBlock,
			Watcher: svcs.gastimator.OnMaxGasUpdate,
		},
		{
			Param:   netparams.DefaultGas,
			Watcher: svcs.gastimator.OnDefaultGasUpdate,
		},
		{
			Param:   netparams.ValidatorsVoteRequired,
			Watcher: svcs.protocolUpgradeEngine.OnRequiredMajorityChanged,
		},
		{
			Param:   netparams.ValidatorPerformanceScalingFactor,
			Watcher: svcs.topology.OnPerformanceScalingChanged,
		},
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
			Param:   netparams.MarketLiquidityStakeToCCYVolume,
			Watcher: svcs.executionEngine.OnSuppliedStakeToObligationFactorUpdate,
		},
		{
			Param:   netparams.MarketValueWindowLength,
			Watcher: svcs.executionEngine.OnMarketValueWindowLengthUpdate,
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
			Param:   netparams.MaxPeggedOrders,
			Watcher: svcs.executionEngine.OnMaxPeggedOrderUpdate,
		},
		{
			Param:   netparams.MarketLiquidityProvidersFeeDistributionTimeStep,
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
			Param:   netparams.RewardMarketCreationQuantumMultiple,
			Watcher: svcs.executionEngine.OnMarketCreationQuantumMultipleUpdate,
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
		// Liquidity version 2.
		{
			Param:   netparams.MarketLiquidityV2BondPenaltyParameter,
			Watcher: svcs.executionEngine.OnMarketLiquidityV2BondPenaltyUpdate,
		},
		{
			Param:   netparams.MarketLiquidityV2EarlyExitPenalty,
			Watcher: svcs.executionEngine.OnMarketLiquidityV2EarlyExitPenaltyUpdate,
		},
		{
			Param:   netparams.MarketLiquidityV2MaximumLiquidityFeeFactorLevel,
			Watcher: svcs.executionEngine.OnMarketLiquidityV2MaximumLiquidityFeeFactorLevelUpdate,
		},
		{
			Param:   netparams.MarketLiquidityV2SLANonPerformanceBondPenaltySlope,
			Watcher: svcs.executionEngine.OnMarketLiquidityV2SLANonPerformanceBondPenaltySlopeUpdate,
		},
		{
			Param:   netparams.MarketLiquidityV2SLANonPerformanceBondPenaltyMax,
			Watcher: svcs.executionEngine.OnMarketLiquidityV2SLANonPerformanceBondPenaltyMaxUpdate,
		},
		{
			Param:   netparams.MarketLiquidityV2StakeToCCYVolume,
			Watcher: svcs.executionEngine.OnMarketLiquidityV2SuppliedStakeToObligationFactorUpdate,
		},
		// End of liquidity version 2.
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
			Watcher: svcs.snapshotEngine.OnSnapshotIntervalUpdate,
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
			Param:   netparams.GovernanceTransferMaxFraction,
			Watcher: svcs.banking.OnMaxFractionChanged,
		},
		{
			Param:   netparams.GovernanceTransferMaxAmount,
			Watcher: svcs.banking.OnMaxAmountChanged,
		},
		{
			Param:   netparams.TransferMinTransferQuantumMultiple,
			Watcher: svcs.banking.OnMinTransferQuantumMultiple,
		},
		{
			Param:   netparams.SpamProtectionMinimumWithdrawalQuantumMultiple,
			Watcher: svcs.banking.OnMinWithdrawQuantumMultiple,
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
			Param:   netparams.SpotMarketTradingEnabled,
			Watcher: svcs.limits.OnLimitsProposeSpotMarketEnabledFromUpdate,
		},
		{
			Param:   netparams.PerpsMarketTradingEnabled,
			Watcher: svcs.limits.OnLimitsProposePerpsMarketEnabledFromUpdate,
		},
		{
			Param:   netparams.LimitsProposeAssetEnabledFrom,
			Watcher: svcs.limits.OnLimitsProposeAssetEnabledFromUpdate,
		},
		{
			Param:   netparams.MarkPriceUpdateMaximumFrequency,
			Watcher: svcs.executionEngine.OnMarkPriceUpdateMaximumFrequency,
		},
		{
			Param:   netparams.MarketSuccessorLaunchWindow,
			Watcher: svcs.executionEngine.OnSuccessorMarketTimeWindowUpdate,
		},
		{
			Param:   netparams.SpamProtectionMaxStopOrdersPerMarket,
			Watcher: svcs.executionEngine.OnMarketPartiesMaximumStopOrdersUpdate,
		},
	}

	watchers = append(watchers, powWatchers...)
	watchers = append(watchers, spamWatchers...)

	// now add some watcher for our netparams
	return svcs.netParams.Watch(watchers...)
}
