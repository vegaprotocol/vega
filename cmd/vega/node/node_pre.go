package node

import (
	"context"
	"errors"
	"fmt"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/banking"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/blockchain/abci"
	"code.vegaprotocol.io/vega/blockchain/nullchain"
	"code.vegaprotocol.io/vega/blockchain/recorder"
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
	"code.vegaprotocol.io/vega/libs/pprof"
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
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/validators"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/cenkalti/backoff"
	"github.com/prometheus/common/log"
	"github.com/spf13/afero"
	tmtypes "github.com/tendermint/tendermint/abci/types"
)

var ErrUnknownChainProvider = errors.New("unknown chain provider")

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

	conf := l.confWatcher.Get()

	if flagProvided("--no-chain") {
		conf.Blockchain.ChainProvider = "noop"
	}

	// reload logger with the setup from configuration
	l.Log = logging.NewLoggerFromConfig(conf.Logging)

	if conf.Pprof.Enabled {
		l.Log.Info("vega is starting with pprof profile, this is not a recommended setting for production")
		l.pproffhandlr, err = pprof.New(l.Log, conf.Pprof)
		if err != nil {
			return
		}
		l.confWatcher.OnConfigUpdate(
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

	l.stats = stats.New(l.Log, l.conf.Stats, l.Version, l.VersionHash)

	l.ethClient, err = ethclient.Dial(l.ctx, l.conf.NodeWallet.ETH.Address)
	if err != nil {
		return fmt.Errorf("could not instantiate ethereum client: %w", err)
	}

	l.nodeWallets, err = nodewallets.GetNodeWallets(l.conf.NodeWallet, l.vegaPaths, l.nodeWalletPassphrase)
	if err != nil {
		return fmt.Errorf("couldn't get node wallets: %w", err)
	}

	return l.nodeWallets.Verify()
}

// UponGenesis loads all asset from genesis state.
func (l *NodeCommand) UponGenesis(ctx context.Context, rawstate []byte) (err error) {
	l.Log.Debug("Entering node.NodeCommand.UponGenesis")
	defer func() {
		if err != nil {
			l.Log.Debug("Failure in node.NodeCommand.UponGenesis", logging.Error(err))
		} else {
			l.Log.Debug("Leaving node.NodeCommand.UponGenesis without error")
		}
	}()

	state, err := assets.LoadGenesisState(rawstate)
	if err != nil {
		return err
	}
	if state == nil {
		return nil
	}

	for k, v := range state {
		err := l.loadAsset(ctx, k, v)
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *NodeCommand) loadAsset(ctx context.Context, id string, v *proto.AssetDetails) error {
	aid, err := l.assets.NewAsset(id, types.AssetDetailsFromProto(v))
	if err != nil {
		return fmt.Errorf("error instanciating asset %v", err)
	}

	asset, err := l.assets.Get(aid)
	if err != nil {
		return fmt.Errorf("unable to get asset %v", err)
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
		return fmt.Errorf("unable to instantiate new asset err=%v, asset-source=%s", err, v.String())
	}
	if err := l.assets.Enable(aid); err != nil {
		l.Log.Error("invalid genesis asset",
			logging.String("asset-details", v.String()),
			logging.Error(err))
		return fmt.Errorf("unable to enable asset: %v", err)
	}

	assetD := asset.Type()
	if err := l.collateral.EnableAsset(ctx, *assetD); err != nil {
		return fmt.Errorf("unable to enable asset in collateral: %v", err)
	}

	l.Log.Info("new asset added successfully",
		logging.String("asset", asset.String()))

	return nil
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
				log.Fatalf("replay: %v", err)
			}
		}()
	}

	return srv, nil
}

func (l *NodeCommand) startBlockchain(ctx context.Context, commander *nodewallets.Commander) (*processor.App, error) {
	app := processor.NewApp(
		l.Log,
		l.vegaPaths,
		l.conf.Processor,
		l.cancel,
		l.assets,
		l.banking,
		l.broker,
		l.witness,
		l.evtfwd,
		l.executionEngine,
		commander,
		l.genesisHandler,
		l.governance,
		l.notary,
		l.stats.Blockchain,
		l.timeService,
		l.epochService,
		l.topology,
		l.netParams,
		&processor.Oracle{
			Engine:   l.oracle,
			Adaptors: l.oracleAdaptors,
		},
		l.delegation,
		l.limits,
		l.stakeVerifier,
		l.checkpoint,
		l.spam,
		l.stakingAccounts,
		l.snapshot,
		l.Version,
	)

	switch l.conf.Blockchain.ChainProvider {
	case "tendermint":
		srv, err := l.startABCI(ctx, app)
		l.blockchainServer = blockchain.NewServer(srv)
		if err != nil {
			return nil, err
		}

		a, err := abci.NewClient(l.conf.Blockchain.Tendermint.ClientAddr)
		if err != nil {
			return nil, err
		}
		l.blockchainClient = blockchain.NewClient(a)
	case "nullchain":
		abciApp := app.Abci()
		n := nullchain.NewClient(l.Log, l.conf.Blockchain.Null, abciApp)

		// nullchain acts as both the client and the server because its does everything
		l.blockchainServer = blockchain.NewServer(n)
		l.blockchainClient = blockchain.NewClient(n)
	default:
		return nil, ErrUnknownChainProvider
	}

	commander.SetChain(l.blockchainClient)
	return app, nil
}

// we've already set everything up WRT arguments etc... just bootstrap the node.
func (l *NodeCommand) preRun(_ []string) (err error) {
	// ensure that context is cancelled if we return an error here
	defer func() {
		if err != nil {
			l.cancel()
		}
	}()

	// plugins
	l.settlePlugin = plugins.NewPositions(l.ctx)
	l.notaryPlugin = plugins.NewNotary(l.ctx)
	l.assetPlugin = plugins.NewAsset(l.ctx)
	l.withdrawalPlugin = plugins.NewWithdrawal(l.ctx)
	l.depositPlugin = plugins.NewDeposit(l.ctx)

	l.genesisHandler = genesis.New(l.Log, l.conf.Genesis)
	l.genesisHandler.OnGenesisTimeLoaded(l.timeService.SetTimeNow)

	l.broker, err = broker.New(l.ctx, l.Log, l.conf.Broker)
	if err != nil {
		log.Error("unable to initialise broker", logging.Error(err))
		return err
	}

	l.eventService = subscribers.NewService(l.broker)

	now := l.timeService.GetTimeNow()
	l.assets = assets.New(l.Log, l.conf.Assets, l.nodeWallets, l.ethClient, l.timeService)
	l.collateral = collateral.New(l.Log, l.conf.Collateral, l.broker, now)
	l.oracle = oracles.NewEngine(l.Log, l.conf.Oracles, now, l.broker, l.timeService)
	l.timeService.NotifyOnTick(l.oracle.UpdateCurrentTime)
	l.oracleAdaptors = oracleAdaptors.New()

	// instantiate the execution engine
	l.executionEngine = execution.NewEngine(
		l.Log, l.conf.Execution, l.timeService, l.collateral, l.oracle, l.broker,
	)

	// we cannot pass the Chain dependency here (that's set by the blockchain)
	commander, err := nodewallets.NewCommander(l.conf.NodeWallet, l.Log, nil, l.nodeWallets.Vega, l.stats)
	if err != nil {
		return err
	}

	l.limits = limits.New(l.Log, l.conf.Limits)
	l.timeService.NotifyOnTick(l.limits.OnTick)
	l.topology = validators.NewTopology(l.Log, l.conf.Validators, l.nodeWallets.Vega, l.broker)
	l.witness = validators.NewWitness(l.Log, l.conf.Validators, l.topology, commander, l.timeService)
	l.netParams = netparams.New(l.Log, l.conf.NetworkParameters, l.broker)

	l.stakingAccounts, l.stakeVerifier = staking.New(
		l.Log, l.conf.Staking, l.broker, l.timeService, l.witness, l.ethClient, l.netParams,
	)

	l.governance = governance.NewEngine(l.Log, l.conf.Governance, l.stakingAccounts, l.broker, l.assets, l.witness, l.netParams, now)

	l.epochService = epochtime.NewService(l.Log, l.conf.Epoch, l.timeService, l.broker)
	l.delegation = delegation.New(l.Log, delegation.NewDefaultConfig(), l.broker, l.topology, l.stakingAccounts, l.epochService, l.timeService)
	l.netParams.Watch(
		netparams.WatchParam{
			Param:   netparams.DelegationMinAmount,
			Watcher: l.delegation.OnMinAmountChanged,
		})

	// checkpoint engine
	l.checkpoint, err = checkpoint.New(l.Log, l.conf.Checkpoint, l.assets, l.collateral, l.governance, l.netParams, l.delegation, l.epochService)
	if err != nil {
		panic(err)
	}

	l.genesisHandler.OnGenesisAppStateLoaded(
		// be sure to keep this in order.
		// the node upon genesis will load all asset first in the node
		// state. This is important to happened first as we will load the
		// asset which will be considered as the governance token.
		l.UponGenesis,
		// This needs to happen always after, as it defined the network
		// parameters, one of them is  the Governance Token asset ID.
		// which if not loaded in the previous state, then will make the node
		// panic at startup.
		l.netParams.UponGenesis,
		l.topology.LoadValidatorsOnGenesis,
		l.limits.UponGenesis,
		l.checkpoint.UponGenesis,
	)

	l.notary = notary.NewWithSnapshot(l.Log, l.conf.Notary, l.topology, l.broker, commander, l.timeService)
	l.evtfwd = evtforward.New(l.Log, l.conf.EvtForward, commander, l.timeService, l.topology)
	l.banking = banking.New(l.Log, l.conf.Banking, l.collateral, l.witness, l.timeService, l.assets, l.notary, l.broker, l.topology)
	l.spam = spam.New(l.Log, l.conf.Spam, l.epochService, l.stakingAccounts)
	l.snapshot, err = snapshot.New(l.ctx, l.vegaPaths, l.conf.Snapshot, l.Log, l.timeService)
	if err != nil {
		panic(err)
	}

	// setup rewards engine
	l.rewards = rewards.New(l.Log, l.conf.Rewards, l.broker, l.delegation, l.epochService, l.collateral, l.timeService)

	l.snapshot.AddProviders(l.checkpoint, l.collateral, l.governance, l.delegation, l.netParams, l.epochService, l.assets, l.banking,
		l.notary, l.spam, l.rewards, l.stakingAccounts, l.stakeVerifier, l.limits, l.topology, l.evtfwd, l.executionEngine)

	// now instantiate the blockchain layer
	if l.app, err = l.startBlockchain(l.ctx, commander); err != nil {
		return err
	}

	// setup config reloads for all engines / services /etc
	l.setupConfigWatchers()
	l.timeService.NotifyOnTick(l.confWatcher.OnTimeUpdate)

	// setup some network parameters runtime validations
	// and network parameters updates dispatches
	return l.setupNetParameters()
}

func (l *NodeCommand) setupNetParameters() error {
	// now we are going to setup some network parameters which can be done
	// through runtime checks
	// e.g: changing the governance asset require the Assets and Collateral engines, so we can ensure any changes there are made for a valid asset

	if err := l.netParams.AddRules(
		netparams.ParamStringRules(
			netparams.RewardAsset,
			checks.RewardAssetUpdate(l.Log, l.assets, l.collateral),
		),
	); err != nil {
		return err
	}

	// now add some watcher for our netparams
	return l.netParams.Watch(
		netparams.WatchParam{
			Param:   netparams.RewardAsset,
			Watcher: dispatch.RewardAssetUpdate(l.Log, l.assets),
		},
		netparams.WatchParam{
			Param:   netparams.MarketMarginScalingFactors,
			Watcher: l.executionEngine.OnMarketMarginScalingFactorsUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketFeeFactorsMakerFee,
			Watcher: l.executionEngine.OnMarketFeeFactorsMakerFeeUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketFeeFactorsInfrastructureFee,
			Watcher: l.executionEngine.OnMarketFeeFactorsInfrastructureFeeUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityStakeToCCYSiskas,
			Watcher: l.executionEngine.OnSuppliedStakeToObligationFactorUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketValueWindowLength,
			Watcher: l.executionEngine.OnMarketValueWindowLengthUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketTargetStakeScalingFactor,
			Watcher: l.executionEngine.OnMarketTargetStakeScalingFactorUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketTargetStakeTimeWindow,
			Watcher: l.executionEngine.OnMarketTargetStakeTimeWindowUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.BlockchainsEthereumConfig,
			Watcher: l.ethClient.OnEthereumConfigUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityProvidersFeeDistribitionTimeStep,
			Watcher: l.executionEngine.OnMarketLiquidityProvidersFeeDistributionTimeStep,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityProvisionShapesMaxSize,
			Watcher: l.executionEngine.OnMarketLiquidityProvisionShapesMaxSizeUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityMaximumLiquidityFeeFactorLevel,
			Watcher: l.executionEngine.OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityBondPenaltyParameter,
			Watcher: l.executionEngine.OnMarketLiquidityBondPenaltyUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityTargetStakeTriggeringRatio,
			Watcher: l.executionEngine.OnMarketLiquidityTargetStakeTriggeringRatio,
		},
		netparams.WatchParam{
			Param:   netparams.MarketAuctionMinimumDuration,
			Watcher: l.executionEngine.OnMarketAuctionMinimumDurationUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketProbabilityOfTradingTauScaling,
			Watcher: l.executionEngine.OnMarketProbabilityOfTradingTauScalingUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketMinProbabilityOfTradingForLPOrders,
			Watcher: l.executionEngine.OnMarketMinProbabilityOfTradingForLPOrdersUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.ValidatorsEpochLength,
			Watcher: l.epochService.OnEpochLengthUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.RewardAsset,
			Watcher: l.rewards.UpdateAssetForStakingAndDelegationRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardPayoutFraction,
			Watcher: l.rewards.UpdatePayoutFractionForStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardPayoutDelay,
			Watcher: l.rewards.UpdatePayoutDelayForStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardMaxPayoutPerParticipant,
			Watcher: l.rewards.UpdateMaxPayoutPerParticipantForStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardDelegatorShare,
			Watcher: l.rewards.UpdateDelegatorShareForStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardMinimumValidatorStake,
			Watcher: l.rewards.UpdateMinimumValidatorStakeForStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardMaxPayoutPerEpoch,
			Watcher: l.rewards.UpdateMaxPayoutPerEpochStakeForStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardCompetitionLevel,
			Watcher: l.rewards.UpdateCompetitionLevelForStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardsMinValidators,
			Watcher: l.rewards.UpdateMinValidatorsStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardOptimalStakeMultiplier,
			Watcher: l.rewards.UpdateOptimalStakeMultiplierStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.ValidatorsVoteRequired,
			Watcher: l.witness.OnDefaultValidatorsVoteRequiredUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.ValidatorsVoteRequired,
			Watcher: l.notary.OnDefaultValidatorsVoteRequiredUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.NetworkCheckpointTimeElapsedBetweenCheckpoints,
			Watcher: l.checkpoint.OnTimeElapsedUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.SpamProtectionMaxVotes,
			Watcher: l.spam.OnMaxVotesChanged,
		},
		netparams.WatchParam{
			Param:   netparams.SpamProtectionMaxProposals,
			Watcher: l.spam.OnMaxProposalsChanged,
		},
		netparams.WatchParam{
			Param:   netparams.SpamProtectionMaxDelegations,
			Watcher: l.spam.OnMaxDelegationsChanged,
		},
		netparams.WatchParam{
			Param:   netparams.SpamProtectionMinTokensForProposal,
			Watcher: l.spam.OnMinTokensForProposalChanged,
		},
		netparams.WatchParam{
			Param:   netparams.SpamProtectionMinTokensForVoting,
			Watcher: l.spam.OnMinTokensForVotingChanged,
		},
		netparams.WatchParam{
			Param:   netparams.SpamProtectionMinTokensForDelegation,
			Watcher: l.spam.OnMinTokensForDelegationChanged,
		},
		netparams.WatchParam{
			Param:   netparams.SnapshotIntervalLength,
			Watcher: l.snapshot.OnSnapshotIntervalUpdate,
		},
	)
}

func (l *NodeCommand) setupConfigWatchers() {
	l.confWatcher.OnConfigUpdate(
		func(cfg config.Config) { l.executionEngine.ReloadConf(cfg.Execution) },
		func(cfg config.Config) { l.notary.ReloadConf(cfg.Notary) },
		func(cfg config.Config) { l.evtfwd.ReloadConf(cfg.EvtForward) },
		func(cfg config.Config) { l.blockchainServer.ReloadConf(cfg.Blockchain) },
		func(cfg config.Config) { l.topology.ReloadConf(cfg.Validators) },
		func(cfg config.Config) { l.witness.ReloadConf(cfg.Validators) },
		func(cfg config.Config) { l.assets.ReloadConf(cfg.Assets) },
		func(cfg config.Config) { l.banking.ReloadConf(cfg.Banking) },
		func(cfg config.Config) { l.governance.ReloadConf(cfg.Governance) },
		func(cfg config.Config) { l.app.ReloadConf(cfg.Processor) },
		func(cfg config.Config) { l.stats.ReloadConf(cfg.Stats) },
	)
}
