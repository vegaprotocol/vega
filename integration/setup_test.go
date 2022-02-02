package core_test

import (
	"context"
	"fmt"
	"os"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/delegation"
	"code.vegaprotocol.io/vega/epochtime"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/rewards"
	"code.vegaprotocol.io/vega/types/num"

	"code.vegaprotocol.io/vega/integration/helpers"
	"code.vegaprotocol.io/vega/integration/steps/market"
	"code.vegaprotocol.io/vega/integration/stubs"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/plugins"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/mock/gomock"
)

var (
	execsetup *executionTestSetup
	reporter  tstReporter
)

type tstReporter struct {
	err  error
	step string
}

func (t tstReporter) Errorf(format string, args ...interface{}) {
	fmt.Printf("%s ERROR: %s", t.step, fmt.Sprintf(format, args...))
}

func (t tstReporter) Fatalf(format string, args ...interface{}) {
	fmt.Printf("%s FATAL: %s", t.step, fmt.Sprintf(format, args...))
	os.Exit(1)
}

var marketConfig = market.NewMarketConfig()

type executionTestSetup struct {
	cfg              execution.Config
	log              *logging.Logger
	ctrl             *gomock.Controller
	timeService      *stubs.TimeStub
	broker           *stubs.BrokerStub
	executionEngine  *exEng
	collateralEngine *collateral.Engine
	oracleEngine     *oracles.Engine
	epochEngine      *epochtime.Svc
	delegationEngine *delegation.Engine
	positionPlugin   *plugins.Positions
	topology         *stubs.TopologyStub
	stakingAccount   *stubs.StakingAccountStub
	rewardsEngine    *rewards.Engine
	assetsEngine     *stubs.AssetStub

	// save party accounts state
	markets []types.Market

	block *helpers.Block

	netParams *netparams.Store

	// keep track of net deposits/withdrawals (ignores asset type)
	netDeposits *num.Uint
}

func newExecutionTestSetup() *executionTestSetup {
	if execsetup != nil && execsetup.ctrl != nil {
		execsetup.ctrl.Finish()
	} else if execsetup == nil {
		execsetup = &executionTestSetup{}
	}

	ctrl := gomock.NewController(&reporter)
	execsetup.ctrl = ctrl
	execsetup.cfg = execution.NewDefaultConfig()
	execsetup.log = logging.NewTestLogger()
	execsetup.timeService = stubs.NewTimeStub()
	execsetup.broker = stubs.NewBrokerStub()
	currentTime := execsetup.timeService.GetTimeNow()
	execsetup.collateralEngine = collateral.New(
		execsetup.log, collateral.NewDefaultConfig(), execsetup.broker, currentTime,
	)

	vegaAsset := types.Asset{
		ID: "VEGA",
		Details: &types.AssetDetails{
			Name:   "VEGA",
			Symbol: "VEGA",
		},
	}
	execsetup.collateralEngine.EnableAsset(context.Background(), vegaAsset)

	execsetup.epochEngine = epochtime.NewService(execsetup.log, epochtime.NewDefaultConfig(), execsetup.timeService, execsetup.broker)
	execsetup.topology = stubs.NewTopologyStub("nodeID")

	execsetup.stakingAccount = stubs.NewStakingAccountStub()
	execsetup.epochEngine.NotifyOnEpoch(execsetup.stakingAccount.OnEpochEvent)

	feesTracker := execution.NewFeesTracker(execsetup.epochEngine)

	execsetup.delegationEngine = delegation.New(execsetup.log, delegation.NewDefaultConfig(), execsetup.broker, execsetup.topology, execsetup.stakingAccount, execsetup.epochEngine, execsetup.timeService)
	marketTracker := execution.NewMarketTracker()
	execsetup.rewardsEngine = rewards.New(execsetup.log, rewards.NewDefaultConfig(), execsetup.broker, execsetup.delegationEngine, execsetup.epochEngine, execsetup.collateralEngine, execsetup.timeService, execsetup.topology, feesTracker, marketTracker)
	execsetup.oracleEngine = oracles.NewEngine(
		execsetup.log, oracles.NewDefaultConfig(), currentTime, execsetup.broker, execsetup.timeService,
	)
	execsetup.assetsEngine = stubs.NewAssetStub()

	stateVarEngine := stubs.NewStateVar()
	execsetup.timeService.NotifyOnTick(stateVarEngine.OnTimeTick)
	// @TODO stub assets engine and pass it in

	execsetup.executionEngine = newExEng(
		execution.NewEngine(
			execsetup.log,
			execsetup.cfg,
			execsetup.timeService,
			execsetup.collateralEngine,
			execsetup.oracleEngine,
			execsetup.broker,
			stateVarEngine,
			feesTracker,
			marketTracker,
			execsetup.assetsEngine, // assets
		),
		execsetup.broker,
	)

	execsetup.positionPlugin = plugins.NewPositions(context.Background())
	execsetup.broker.Subscribe(execsetup.positionPlugin)

	execsetup.block = helpers.NewBlock()

	execsetup.netParams = netparams.New(execsetup.log, netparams.NewDefaultConfig(), execsetup.broker)
	if err := execsetup.registerNetParamsCallbacks(); err != nil {
		panic(err)
	}

	execsetup.netParams.Watch(
		netparams.WatchParam{
			Param:   netparams.FloatingPointUpdatesDuration,
			Watcher: stateVarEngine.OnFloatingPointUpdatesDurationUpdate,
		},
	)

	execsetup.netDeposits = num.Zero()

	return execsetup
}

func (e *executionTestSetup) registerNetParamsCallbacks() error {
	return e.netParams.Watch(
		netparams.WatchParam{
			Param:   netparams.MarketMarginScalingFactors,
			Watcher: e.executionEngine.OnMarketMarginScalingFactorsUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketFeeFactorsMakerFee,
			Watcher: e.executionEngine.OnMarketFeeFactorsMakerFeeUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketFeeFactorsInfrastructureFee,
			Watcher: e.executionEngine.OnMarketFeeFactorsInfrastructureFeeUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityStakeToCCYSiskas,
			Watcher: e.executionEngine.OnSuppliedStakeToObligationFactorUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketValueWindowLength,
			Watcher: e.executionEngine.OnMarketValueWindowLengthUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketTargetStakeScalingFactor,
			Watcher: e.executionEngine.OnMarketTargetStakeScalingFactorUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketTargetStakeTimeWindow,
			Watcher: e.executionEngine.OnMarketTargetStakeTimeWindowUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityProvidersFeeDistribitionTimeStep,
			Watcher: e.executionEngine.OnMarketLiquidityProvidersFeeDistributionTimeStep,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityProvisionShapesMaxSize,
			Watcher: e.executionEngine.OnMarketLiquidityProvisionShapesMaxSizeUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityMaximumLiquidityFeeFactorLevel,
			Watcher: e.executionEngine.OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityBondPenaltyParameter,
			Watcher: e.executionEngine.OnMarketLiquidityBondPenaltyUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityTargetStakeTriggeringRatio,
			Watcher: e.executionEngine.OnMarketLiquidityTargetStakeTriggeringRatio,
		},
		netparams.WatchParam{
			Param:   netparams.MarketAuctionMinimumDuration,
			Watcher: e.executionEngine.OnMarketAuctionMinimumDurationUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketProbabilityOfTradingTauScaling,
			Watcher: e.executionEngine.OnMarketProbabilityOfTradingTauScalingUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.DelegationMinAmount,
			Watcher: e.delegationEngine.OnMinAmountChanged,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardMaxPayoutPerParticipant,
			Watcher: e.rewardsEngine.UpdateMaxPayoutPerParticipantForStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardDelegatorShare,
			Watcher: e.rewardsEngine.UpdateDelegatorShareForStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardMinimumValidatorStake,
			Watcher: e.rewardsEngine.UpdateMinimumValidatorStakeForStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.RewardAsset,
			Watcher: e.rewardsEngine.UpdateAssetForStakingAndDelegation,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardCompetitionLevel,
			Watcher: e.rewardsEngine.UpdateCompetitionLevelForStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardsMinValidators,
			Watcher: e.rewardsEngine.UpdateMinValidatorsStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardOptimalStakeMultiplier,
			Watcher: e.rewardsEngine.UpdateOptimalStakeMultiplierStakingRewardScheme,
		},
		netparams.WatchParam{
			Param:   netparams.ValidatorsEpochLength,
			Watcher: e.epochEngine.OnEpochLengthUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketMinLpStakeQuantumMultiple,
			Watcher: e.executionEngine.OnMinLpStakeQuantumMultipleUpdate,
		},
	)
}
