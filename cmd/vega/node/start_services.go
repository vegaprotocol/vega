package node

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/banking"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/blockchain/nullchain"
	"code.vegaprotocol.io/vega/blockchain/recorder"
	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/checkpoint"
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
	oracleAdaptors "code.vegaprotocol.io/vega/oracles/adaptors"
	"code.vegaprotocol.io/vega/plugins"
	"code.vegaprotocol.io/vega/processor"
	"code.vegaprotocol.io/vega/rewards"
	"code.vegaprotocol.io/vega/snapshot"
	"code.vegaprotocol.io/vega/spam"
	"code.vegaprotocol.io/vega/staking"
	"code.vegaprotocol.io/vega/statevar"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/validators"
	"code.vegaprotocol.io/vega/vegatime"
	"github.com/spf13/afero"
	tmtypes "github.com/tendermint/tendermint/abci/types"
)

var (
	ErrUnknownChainProvider    = errors.New("unknown chain provider")
	ErrERC20AssetWithNullChain = errors.New("cannot use ERC20 asset with nullchain")
)

func (n *NodeCommand) startServices(_ []string) (err error) {
	// ensure that context is cancelled if we return an error here
	defer func() {
		if err != nil {
			n.cancel()
		}
	}()

	// this doesn't fail
	n.timeService = vegatime.New(n.conf.Time)
	n.stats = stats.New(n.Log, n.conf.Stats, n.Version, n.VersionHash)

	// plugins
	n.settlePlugin = plugins.NewPositions(n.ctx)
	n.notaryPlugin = plugins.NewNotary(n.ctx)
	n.assetPlugin = plugins.NewAsset(n.ctx)
	n.withdrawalPlugin = plugins.NewWithdrawal(n.ctx)
	n.depositPlugin = plugins.NewDeposit(n.ctx)

	n.genesisHandler = genesis.New(n.Log, n.conf.Genesis)
	n.genesisHandler.OnGenesisTimeLoaded(n.timeService.SetTimeNow)

	n.broker, err = broker.New(n.ctx, n.Log, n.conf.Broker)
	if err != nil {
		n.Log.Error("unable to initialise broker", logging.Error(err))
		return err
	}

	n.eventService = subscribers.NewService(n.broker)

	now := n.timeService.GetTimeNow()
	n.assets = assets.New(n.Log, n.conf.Assets, n.nodeWallets, n.ethClient, n.timeService, n.conf.IsValidator())
	n.collateral = collateral.New(n.Log, n.conf.Collateral, n.broker, now)
	n.oracle = oracles.NewEngine(n.Log, n.conf.Oracles, now, n.broker, n.timeService)
	n.timeService.NotifyOnTick(n.oracle.UpdateCurrentTime)
	n.oracleAdaptors = oracleAdaptors.New()

	// if we are not a validator, no need to instantiate the commander
	if n.conf.IsValidator() {
		// we cannot pass the Chain dependency here (that's set by the blockchain)
		n.commander, err = nodewallets.NewCommander(n.conf.NodeWallet, n.Log, nil, n.nodeWallets.Vega, n.stats)
		if err != nil {
			return err
		}
	}

	n.limits = limits.New(n.Log, n.conf.Limits, n.broker)
	n.timeService.NotifyOnTick(n.limits.OnTick)

	if n.conf.IsValidator() {
		n.topology = validators.NewTopology(
			n.Log, n.conf.Validators, validators.WrapNodeWallets(n.nodeWallets), n.broker, n.conf.IsValidator())
	} else {
		n.topology = validators.NewTopology(n.Log, n.conf.Validators, nil, n.broker, n.conf.IsValidator())
	}

	n.witness = validators.NewWitness(n.Log, n.conf.Validators, n.topology, n.commander, n.timeService)
	n.netParams = netparams.New(n.Log, n.conf.NetworkParameters, n.broker)
	n.timeService.NotifyOnTick(n.netParams.OnChainTimeUpdate)
	n.evtfwd = evtforward.New(
		n.Log, n.conf.EvtForward, n.commander, n.timeService, n.topology)

	n.stakingAccounts, n.stakeVerifier = staking.New(
		n.Log, n.conf.Staking, n.broker, n.timeService, n.witness, n.ethClient, n.netParams, n.evtfwd, n.conf.IsValidator(),
	)
	n.epochService = epochtime.NewService(n.Log, n.conf.Epoch, n.timeService, n.broker)

	n.statevar = statevar.New(n.Log, n.conf.StateVar, n.broker, n.topology, n.commander, n.timeService)
	n.feesTracker = execution.NewFeesTracker(n.epochService)
	marketTracker := execution.NewMarketTracker()

	// instantiate the execution engine
	n.executionEngine = execution.NewEngine(
		n.Log, n.conf.Execution, n.timeService, n.collateral, n.oracle, n.broker, n.statevar, n.feesTracker, marketTracker,
	)

	if n.conf.Blockchain.ChainProvider == blockchain.ProviderNullChain {
		// Use staking-loop to pretend a dummy builtin asssets deposited with the faucet was staked
		stakingLoop := nullchain.NewStakingLoop(n.collateral, n.assets)
		n.governance = governance.NewEngine(n.Log, n.conf.Governance, stakingLoop, n.broker, n.assets, n.witness, n.netParams, now)
		n.delegation = delegation.New(n.Log, delegation.NewDefaultConfig(), n.broker, n.topology, stakingLoop, n.epochService, n.timeService)
	} else {
		n.governance = governance.NewEngine(n.Log, n.conf.Governance, n.stakingAccounts, n.broker, n.assets, n.witness, n.netParams, now)
		n.delegation = delegation.New(n.Log, delegation.NewDefaultConfig(), n.broker, n.topology, n.stakingAccounts, n.epochService, n.timeService)
	}

	// setup rewards engine
	n.rewards = rewards.New(n.Log, n.conf.Rewards, n.broker, n.delegation, n.epochService, n.collateral, n.timeService, n.topology, n.feesTracker, marketTracker)

	n.notary = notary.NewWithSnapshot(
		n.Log, n.conf.Notary, n.topology, n.broker, n.commander, n.timeService)
	n.banking = banking.New(n.Log, n.conf.Banking, n.collateral, n.witness, n.timeService, n.assets, n.notary, n.broker, n.topology, n.epochService)

	// checkpoint engine
	n.checkpoint, err = checkpoint.New(n.Log, n.conf.Checkpoint, n.assets, n.collateral, n.governance, n.netParams, n.delegation, n.epochService, n.topology, n.banking)
	if err != nil {
		panic(err)
	}

	n.genesisHandler.OnGenesisAppStateLoaded(
		// be sure to keep this in order.
		// the node upon genesis will load all asset first in the node
		// state. This is important to happened first as we will load the
		// asset which will be considered as the governance token.
		n.UponGenesis,
		// This needs to happen always after, as it defined the network
		// parameters, one of them is  the Governance Token asset ID.
		// which if not loaded in the previous state, then will make the node
		// panic at startup.
		n.netParams.UponGenesis,
		n.topology.LoadValidatorsOnGenesis,
		n.limits.UponGenesis,
		n.checkpoint.UponGenesis,
	)

	n.spam = spam.New(n.Log, n.conf.Spam, n.epochService, n.stakingAccounts)
	n.snapshot, err = snapshot.New(n.ctx, n.vegaPaths, n.conf.Snapshot, n.Log, n.timeService)
	if err != nil {
		panic(err)
	}
	// notify delegation, rewards, and accounting on changes in the validator pub key
	n.topology.NotifyOnKeyChange(n.delegation.ValidatorKeyChanged, n.stakingAccounts.ValidatorKeyChanged, n.governance.ValidatorKeyChanged)

	n.snapshot.AddProviders(n.checkpoint, n.collateral, n.governance, n.delegation, n.netParams, n.epochService, n.assets, n.banking,
		n.notary, n.spam, n.stakingAccounts, n.stakeVerifier, n.limits, n.topology, n.evtfwd, n.executionEngine, n.feesTracker, marketTracker)

	// setup config reloads for all engines / services /etc
	n.setupConfigWatchers()
	n.timeService.NotifyOnTick(n.confWatcher.OnTimeUpdate)

	// setup some network parameters runtime validations and network parameters updates dispatches
	// this must come before we try to load from a snapshot, which happens in startBlockchain
	if err := n.setupNetParameters(); err != nil {
		return err
	}

	// now instantiate the blockchain layer
	if n.app, err = n.startBlockchain(); err != nil {
		return err
	}

	return nil
}

func (n *NodeCommand) startBlockchain() (*processor.App, error) {
	// if tm chain, setup the client
	switch n.conf.Blockchain.ChainProvider {
	case blockchain.ProviderTendermint:
		a, err := abci.NewClient(n.conf.Blockchain.Tendermint.ClientAddr)
		if err != nil {
			return nil, err
		}
		n.blockchainClient = blockchain.NewClient(a)
	}

	app := processor.NewApp(
		n.Log,
		n.vegaPaths,
		n.conf.Processor,
		n.cancel,
		n.assets,
		n.banking,
		n.broker,
		n.witness,
		n.evtfwd,
		n.executionEngine,
		n.genesisHandler,
		n.governance,
		n.notary,
		n.stats.Blockchain,
		n.timeService,
		n.epochService,
		n.topology,
		n.netParams,
		&processor.Oracle{
			Engine:   n.oracle,
			Adaptors: n.oracleAdaptors,
		},
		n.delegation,
		n.limits,
		n.stakeVerifier,
		n.checkpoint,
		n.spam,
		n.stakingAccounts,
		n.rewards,
		n.snapshot,
		n.statevar,
		n.blockchainClient,
		n.Version,
	)

	// Load from a snapshot if that has been requested
	// This has to happen after creating the application since that is where we add the
	// replay-protector to the provider list
	if n.conf.Snapshot.StartHeight != 0 {
		err := n.snapshot.LoadHeight(n.ctx, n.conf.Snapshot.StartHeight)
		if err != nil {
			return nil, err
		}

		// Replace the restore replay-protector, the tolerance value here doesn't matter so is set to zero
		app.Abci().ReplaceReplayProtector(0)
	}

	switch n.conf.Blockchain.ChainProvider {
	case blockchain.ProviderTendermint:
		srv, err := n.startABCI(n.ctx, app)
		n.blockchainServer = blockchain.NewServer(srv)
		if err != nil {
			return nil, err
		}
	case blockchain.ProviderNullChain:
		abciApp := app.Abci()
		null := nullchain.NewClient(n.Log, n.conf.Blockchain.Null, abciApp)

		// nullchain acts as both the client and the server because its does everything
		n.blockchainServer = blockchain.NewServer(null)
		n.blockchainClient = blockchain.NewClient(null)
	default:
		return nil, ErrUnknownChainProvider
	}

	// setup the commander only if we are a validator node
	if n.conf.IsValidator() {
		n.commander.SetChain(n.blockchainClient)
	}
	return app, nil
}

func (n *NodeCommand) setupNetParameters() error {
	// now we are going to setup some network parameters which can be done
	// through runtime checks
	// e.g: changing the governance asset require the Assets and Collateral engines, so we can ensure any changes there are made for a valid asset

	if err := n.netParams.AddRules(
		netparams.ParamStringRules(
			netparams.RewardAsset,
			checks.RewardAssetUpdate(n.Log, n.assets, n.collateral),
		),
	); err != nil {
		return err
	}

	// now add some watcher for our netparams
	return n.netParams.Watch(
		netparams.WatchParam{
			Param:   netparams.DelegationMinAmount,
			Watcher: n.delegation.OnMinAmountChanged,
		},
		netparams.WatchParam{
			Param:   netparams.RewardAsset,
			Watcher: dispatch.RewardAssetUpdate(n.Log, n.assets),
		},
		netparams.WatchParam{
			Param:   netparams.MarketMarginScalingFactors,
			Watcher: n.executionEngine.OnMarketMarginScalingFactorsUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketFeeFactorsMakerFee,
			Watcher: n.executionEngine.OnMarketFeeFactorsMakerFeeUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketFeeFactorsInfrastructureFee,
			Watcher: n.executionEngine.OnMarketFeeFactorsInfrastructureFeeUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityStakeToCCYSiskas,
			Watcher: n.executionEngine.OnSuppliedStakeToObligationFactorUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketValueWindowLength,
			Watcher: n.executionEngine.OnMarketValueWindowLengthUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketTargetStakeScalingFactor,
			Watcher: n.executionEngine.OnMarketTargetStakeScalingFactorUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketTargetStakeTimeWindow,
			Watcher: n.executionEngine.OnMarketTargetStakeTimeWindowUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.BlockchainsEthereumConfig,
			Watcher: n.ethClient.OnEthereumConfigUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityProvidersFeeDistribitionTimeStep,
			Watcher: n.executionEngine.OnMarketLiquidityProvidersFeeDistributionTimeStep,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityProvisionShapesMaxSize,
			Watcher: n.executionEngine.OnMarketLiquidityProvisionShapesMaxSizeUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketMinLpStakeQuantumMultiple,
			Watcher: n.executionEngine.OnMinLpStakeQuantumMultipleUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityMaximumLiquidityFeeFactorLevel,
			Watcher: n.executionEngine.OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityBondPenaltyParameter,
			Watcher: n.executionEngine.OnMarketLiquidityBondPenaltyUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityTargetStakeTriggeringRatio,
			Watcher: n.executionEngine.OnMarketLiquidityTargetStakeTriggeringRatio,
		},
		netparams.WatchParam{
			Param:   netparams.MarketAuctionMinimumDuration,
			Watcher: n.executionEngine.OnMarketAuctionMinimumDurationUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketProbabilityOfTradingTauScaling,
			Watcher: n.executionEngine.OnMarketProbabilityOfTradingTauScalingUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketMinProbabilityOfTradingForLPOrders,
			Watcher: n.executionEngine.OnMarketMinProbabilityOfTradingForLPOrdersUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.ValidatorsEpochLength,
			Watcher: n.epochService.OnEpochLengthUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardMaxPayoutPerParticipant,
			Watcher: n.rewards.UpdateMaxPayoutPerParticipantForStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardDelegatorShare,
			Watcher: n.rewards.UpdateDelegatorShareForStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardMinimumValidatorStake,
			Watcher: n.rewards.UpdateMinimumValidatorStakeForStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.RewardAsset,
			Watcher: n.rewards.UpdateAssetForStakingAndDelegation,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardCompetitionLevel,
			Watcher: n.rewards.UpdateCompetitionLevelForStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardsMinValidators,
			Watcher: n.rewards.UpdateMinValidatorsStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardOptimalStakeMultiplier,
			Watcher: n.rewards.UpdateOptimalStakeMultiplierStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.ValidatorsVoteRequired,
			Watcher: n.witness.OnDefaultValidatorsVoteRequiredUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.ValidatorsVoteRequired,
			Watcher: n.notary.OnDefaultValidatorsVoteRequiredUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.NetworkCheckpointTimeElapsedBetweenCheckpoints,
			Watcher: n.checkpoint.OnTimeElapsedUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.SpamProtectionMaxVotes,
			Watcher: n.spam.OnMaxVotesChanged,
		},
		netparams.WatchParam{
			Param:   netparams.SpamProtectionMaxProposals,
			Watcher: n.spam.OnMaxProposalsChanged,
		},
		netparams.WatchParam{
			Param:   netparams.SpamProtectionMaxDelegations,
			Watcher: n.spam.OnMaxDelegationsChanged,
		},
		netparams.WatchParam{
			Param:   netparams.SpamProtectionMinTokensForProposal,
			Watcher: n.spam.OnMinTokensForProposalChanged,
		},
		netparams.WatchParam{
			Param:   netparams.SpamProtectionMinTokensForVoting,
			Watcher: n.spam.OnMinTokensForVotingChanged,
		},
		netparams.WatchParam{
			Param:   netparams.SpamProtectionMinTokensForDelegation,
			Watcher: n.spam.OnMinTokensForDelegationChanged,
		},
		netparams.WatchParam{
			Param:   netparams.SnapshotIntervalLength,
			Watcher: n.snapshot.OnSnapshotIntervalUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.ValidatorsVoteRequired,
			Watcher: n.statevar.OnDefaultValidatorsVoteRequiredUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.FloatingPointUpdatesDuration,
			Watcher: n.statevar.OnFloatingPointUpdatesDurationUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.TransferFeeFactor,
			Watcher: n.banking.OnTransferFeeFactorUpdate,
		},
	)
}

func (n *NodeCommand) setupConfigWatchers() {
	n.confWatcher.OnConfigUpdate(
		func(cfg config.Config) { n.executionEngine.ReloadConf(cfg.Execution) },
		func(cfg config.Config) { n.notary.ReloadConf(cfg.Notary) },
		func(cfg config.Config) { n.evtfwd.ReloadConf(cfg.EvtForward) },
		func(cfg config.Config) { n.blockchainServer.ReloadConf(cfg.Blockchain) },
		func(cfg config.Config) { n.topology.ReloadConf(cfg.Validators) },
		func(cfg config.Config) { n.witness.ReloadConf(cfg.Validators) },
		func(cfg config.Config) { n.assets.ReloadConf(cfg.Assets) },
		func(cfg config.Config) { n.banking.ReloadConf(cfg.Banking) },
		func(cfg config.Config) { n.governance.ReloadConf(cfg.Governance) },
		func(cfg config.Config) { n.app.ReloadConf(cfg.Processor) },
		func(cfg config.Config) { n.stats.ReloadConf(cfg.Stats) },
	)
}

func (l *NodeCommand) startABCI(ctx context.Context, app *processor.App) (*abci.Server, error) {
	var abciApp tmtypes.Application
	tmCfg := l.conf.Blockchain.Tendermint
	if path := tmCfg.ABCIRecordDir; path != "" {
		rec, err := recorder.NewRecord(path, afero.NewOsFs())
		if err != nil {
			return nil, err
		}

		// closer
		go func() {
			<-ctx.Done()
			rec.Stop()
		}()

		abciApp = recorder.NewApp(app.Abci(), rec)
	} else {
		abciApp = app.Abci()
	}

	srv := abci.NewServer(l.Log, l.conf.Blockchain, abciApp)
	if err := srv.Start(); err != nil {
		return nil, err
	}

	if path := tmCfg.ABCIReplayFile; path != "" {
		rec, err := recorder.NewReplay(path, afero.NewOsFs())
		if err != nil {
			return nil, err
		}

		// closer
		go func() {
			<-ctx.Done()
			rec.Stop()
		}()

		go func() {
			if err := rec.Replay(abciApp); err != nil {
				l.Log.Fatal("replay error", logging.Error(err))
			}
		}()
	}

	return srv, nil
}
