// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package core_test

import (
	"context"
	"fmt"
	"os"

	"code.vegaprotocol.io/vega/core/activitystreak"
	"code.vegaprotocol.io/vega/core/banking"
	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/delegation"
	"code.vegaprotocol.io/vega/core/epochtime"
	"code.vegaprotocol.io/vega/core/evtforward"
	"code.vegaprotocol.io/vega/core/execution"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/integration/helpers"
	"code.vegaprotocol.io/vega/core/integration/steps/market"
	referralcfg "code.vegaprotocol.io/vega/core/integration/steps/referral"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/notary"
	"code.vegaprotocol.io/vega/core/parties"
	"code.vegaprotocol.io/vega/core/plugins"
	"code.vegaprotocol.io/vega/core/referral"
	"code.vegaprotocol.io/vega/core/rewards"
	"code.vegaprotocol.io/vega/core/teams"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/core/vesting"
	"code.vegaprotocol.io/vega/core/volumediscount"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	protos "code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/mock/gomock"
)

var (
	execsetup *executionTestSetup
	reporter  tstReporter
)

type tstReporter struct {
	scenario string
}

func (t tstReporter) Errorf(format string, args ...interface{}) {
	fmt.Printf("%s ERROR: %s", t.scenario, fmt.Sprintf(format, args...))
}

func (t tstReporter) Fatalf(format string, args ...interface{}) {
	fmt.Printf("%s FATAL: %s", t.scenario, fmt.Sprintf(format, args...))
	os.Exit(1)
}

type DummyASVM struct{}

func (DummyASVM) GetRewardsVestingMultiplier(_ string) num.Decimal {
	return num.MustDecimalFromString("0.01")
}

var (
	marketConfig          = market.NewMarketConfig()
	referralProgramConfig = referralcfg.NewReferralProgramConfig()
	volumeDiscountTiers   = map[string][]*types.VolumeBenefitTier{}
)

type executionTestSetup struct {
	cfg              execution.Config
	log              *logging.Logger
	ctrl             *gomock.Controller
	timeService      *stubs.TimeStub
	broker           *stubs.BrokerStub
	executionEngine  *exEng
	collateralEngine *collateral.Engine
	oracleEngine     *spec.Engine
	builtinOracle    *spec.Builtin
	epochEngine      *epochtime.Svc
	delegationEngine *delegation.Engine
	positionPlugin   *plugins.Positions
	topology         *stubs.TopologyStub
	stakingAccount   *stubs.StakingAccountStub
	rewardsEngine    *rewards.Engine
	assetsEngine     *stubs.AssetStub
	banking          *banking.Engine

	// save party accounts state
	markets []types.Market

	block *helpers.Block

	netParams *netparams.Store

	// keep track of net deposits/withdrawals (ignores asset type)
	netDeposits *num.Uint

	// record parts of state before each step
	accountsBefore                []protos.Account
	ledgerMovementsBefore         int
	insurancePoolDepositsOverStep map[string]*num.Int
	eventsBefore                  int

	notary                *notary.SnapshotNotary
	stateVarEngine        *stubs.StateVarStub
	witness               *validators.Witness
	teamsEngine           *teams.Engine
	profilesEngine        *parties.Engine
	referralProgram       *referral.Engine
	activityStreak        *activitystreak.Engine
	vesting               *vesting.Engine
	volumeDiscountProgram *volumediscount.Engine
}

func newExecutionTestSetup() *executionTestSetup {
	if execsetup != nil && execsetup.ctrl != nil {
		execsetup.ctrl.Finish()
	} else if execsetup == nil {
		execsetup = &executionTestSetup{}
	}

	ctx := context.Background()

	ctrl := gomock.NewController(&reporter)
	execsetup.ctrl = ctrl
	execsetup.cfg = execution.NewDefaultConfig()
	execsetup.cfg.Position.StreamPositionVerbose = true
	execsetup.cfg.Risk.StreamMarginLevelsVerbose = true

	execsetup.netDeposits = num.UintZero()
	execsetup.block = helpers.NewBlock()

	execsetup.log = logging.NewTestLogger()

	execsetup.broker = stubs.NewBrokerStub()
	execsetup.positionPlugin = plugins.NewPositions(ctx)
	execsetup.broker.Subscribe(execsetup.positionPlugin)

	execsetup.timeService = stubs.NewTimeStub()
	execsetup.epochEngine = epochtime.NewService(execsetup.log, epochtime.NewDefaultConfig(), execsetup.broker)

	commander := stubs.NewCommanderStub()

	execsetup.collateralEngine = collateral.New(execsetup.log, collateral.NewDefaultConfig(), execsetup.timeService, execsetup.broker)
	enableAssets(ctx, execsetup.collateralEngine)

	execsetup.netParams = netparams.New(execsetup.log, netparams.NewDefaultConfig(), execsetup.broker)

	execsetup.topology = stubs.NewTopologyStub("nodeID", execsetup.broker)

	execsetup.witness = validators.NewWitness(ctx, execsetup.log, validators.NewDefaultConfig(), execsetup.topology, commander, execsetup.timeService)

	eventForwarder := evtforward.NewNoopEngine(execsetup.log, evtforward.NewDefaultConfig())

	execsetup.oracleEngine = spec.NewEngine(execsetup.log, spec.NewDefaultConfig(), execsetup.timeService, execsetup.broker)
	execsetup.builtinOracle = spec.NewBuiltin(execsetup.oracleEngine, execsetup.timeService)

	execsetup.stakingAccount = stubs.NewStakingAccountStub()
	execsetup.epochEngine.NotifyOnEpoch(execsetup.stakingAccount.OnEpochEvent, execsetup.stakingAccount.OnEpochRestore)

	execsetup.teamsEngine = teams.NewEngine(execsetup.broker, execsetup.timeService)
	execsetup.profilesEngine = parties.NewEngine(execsetup.broker)

	execsetup.stateVarEngine = stubs.NewStateVar()
	marketActivityTracker := common.NewMarketActivityTracker(execsetup.log, execsetup.teamsEngine, execsetup.stakingAccount)

	execsetup.notary = notary.NewWithSnapshot(execsetup.log, notary.NewDefaultConfig(), execsetup.topology, execsetup.broker, commander)

	execsetup.assetsEngine = stubs.NewAssetStub()

	execsetup.referralProgram = referral.NewEngine(execsetup.broker, execsetup.timeService, marketActivityTracker, execsetup.stakingAccount)
	execsetup.epochEngine.NotifyOnEpoch(execsetup.referralProgram.OnEpoch, execsetup.referralProgram.OnEpochRestore)

	execsetup.volumeDiscountProgram = volumediscount.New(execsetup.broker, marketActivityTracker)
	execsetup.epochEngine.NotifyOnEpoch(execsetup.volumeDiscountProgram.OnEpoch, execsetup.volumeDiscountProgram.OnEpochRestore)

	execsetup.banking = banking.New(execsetup.log, banking.NewDefaultConfig(), execsetup.collateralEngine, execsetup.witness, execsetup.timeService, execsetup.assetsEngine, execsetup.notary, execsetup.broker, execsetup.topology, marketActivityTracker, stubs.NewBridgeViewStub(), eventForwarder)

	execsetup.executionEngine = newExEng(
		execution.NewEngine(
			execsetup.log,
			execsetup.cfg,
			execsetup.timeService,
			execsetup.collateralEngine,
			execsetup.oracleEngine,
			execsetup.broker,
			execsetup.stateVarEngine,
			marketActivityTracker,
			execsetup.assetsEngine, // assets
			execsetup.referralProgram,
			execsetup.volumeDiscountProgram,
			execsetup.banking,
		),
		execsetup.broker,
	)
	execsetup.epochEngine.NotifyOnEpoch(execsetup.executionEngine.OnEpochEvent, execsetup.executionEngine.OnEpochRestore)
	execsetup.epochEngine.NotifyOnEpoch(marketActivityTracker.OnEpochEvent, marketActivityTracker.OnEpochRestore)
	execsetup.epochEngine.NotifyOnEpoch(execsetup.banking.OnEpoch, execsetup.banking.OnEpochRestore)

	execsetup.delegationEngine = delegation.New(execsetup.log, delegation.NewDefaultConfig(), execsetup.broker, execsetup.topology, execsetup.stakingAccount, execsetup.epochEngine, execsetup.timeService)

	execsetup.activityStreak = activitystreak.New(execsetup.log, execsetup.executionEngine, execsetup.broker)
	execsetup.epochEngine.NotifyOnEpoch(execsetup.activityStreak.OnEpochEvent, execsetup.activityStreak.OnEpochRestore)

	execsetup.vesting = vesting.New(execsetup.log, execsetup.collateralEngine, execsetup.activityStreak, execsetup.broker, execsetup.assetsEngine)
	execsetup.rewardsEngine = rewards.New(execsetup.log, rewards.NewDefaultConfig(), execsetup.broker, execsetup.delegationEngine, execsetup.epochEngine, execsetup.collateralEngine, execsetup.timeService, marketActivityTracker, execsetup.topology, execsetup.vesting, execsetup.banking, execsetup.activityStreak)

	// register this after the rewards engine is created to make sure the on epoch is called in the right order.
	execsetup.epochEngine.NotifyOnEpoch(execsetup.vesting.OnEpochEvent, execsetup.vesting.OnEpochRestore)

	// The team engine is used to know the team a party belongs to. The computation
	// of the referral program rewards requires this information. Since the team
	// switches happen when the end of epoch is reached, it needs to be one of the
	// last services to register on epoch update, so the computation is made based
	// on the team the parties belonged to during the epoch and not the new one.
	execsetup.epochEngine.NotifyOnEpoch(execsetup.teamsEngine.OnEpoch, execsetup.teamsEngine.OnEpochRestore)

	execsetup.registerTimeServiceCallbacks()

	if err := execsetup.registerNetParamsCallbacks(); err != nil {
		panic(fmt.Errorf("failed to register network parameters: %w", err))
	}

	return execsetup
}

func enableAssets(ctx context.Context, collateralEngine *collateral.Engine) {
	vegaAsset := types.Asset{
		ID: "VEGA",
		Details: &types.AssetDetails{
			Name:    "VEGA",
			Symbol:  "VEGA",
			Quantum: num.MustDecimalFromString("1"),
		},
	}
	if err := collateralEngine.EnableAsset(ctx, vegaAsset); err != nil {
		panic(fmt.Errorf("could not enable asset %q: %w", vegaAsset, err))
	}

	usdt := types.Asset{
		ID: "USDT",
		Details: &types.AssetDetails{
			Name:    "USDT",
			Symbol:  "USDT",
			Quantum: num.MustDecimalFromString("1"),
		},
	}
	if err := collateralEngine.EnableAsset(ctx, usdt); err != nil {
		panic(fmt.Errorf("could not enable asset %q: %w", usdt, err))
	}

	usdc := types.Asset{
		ID: "USDC",
		Details: &types.AssetDetails{
			Name:    "USDC",
			Symbol:  "USDC",
			Quantum: num.MustDecimalFromString("1"),
		},
	}
	if err := collateralEngine.EnableAsset(ctx, usdc); err != nil {
		panic(fmt.Errorf("could not enable asset %q: %w", usdc, err))
	}
}

func (e *executionTestSetup) registerTimeServiceCallbacks() {
	e.timeService.NotifyOnTick(
		e.epochEngine.OnTick,
		e.witness.OnTick,
		e.notary.OnTick,
		e.banking.OnTick,
		e.delegationEngine.OnTick,
		e.builtinOracle.OnTick,
		e.stateVarEngine.OnTick,
		e.executionEngine.OnTick,
	)
}

func (e *executionTestSetup) registerNetParamsCallbacks() error {
	return e.netParams.Watch(
		netparams.WatchParam{
			Param:   netparams.StakingAndDelegationRewardMinimumValidatorStake,
			Watcher: e.topology.OnMinDelegationUpdated,
		},
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
			Param:   netparams.MarketValueWindowLength,
			Watcher: e.executionEngine.OnMarketValueWindowLengthUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityMaximumLiquidityFeeFactorLevel,
			Watcher: e.executionEngine.OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate,
		},
		// Liquidity version 2.
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityBondPenaltyParameter,
			Watcher: e.executionEngine.OnMarketLiquidityV2BondPenaltyUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityEarlyExitPenalty,
			Watcher: e.executionEngine.OnMarketLiquidityV2EarlyExitPenaltyUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityMaximumLiquidityFeeFactorLevel,
			Watcher: e.executionEngine.OnMarketLiquidityV2MaximumLiquidityFeeFactorLevelUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquiditySLANonPerformanceBondPenaltySlope,
			Watcher: e.executionEngine.OnMarketLiquidityV2SLANonPerformanceBondPenaltySlopeUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquiditySLANonPerformanceBondPenaltyMax,
			Watcher: e.executionEngine.OnMarketLiquidityV2SLANonPerformanceBondPenaltyMaxUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityStakeToCCYVolume,
			Watcher: e.executionEngine.OnMarketLiquidityV2StakeToCCYVolumeUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityProvidersFeeCalculationTimeStep,
			Watcher: execsetup.executionEngine.OnMarketLiquidityV2ProvidersFeeCalculationTimeStep,
		},
		// End of liquidity version 2.
		netparams.WatchParam{
			Param:   netparams.MarketAuctionMinimumDuration,
			Watcher: e.executionEngine.OnMarketAuctionMinimumDurationUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketAuctionMaximumDuration,
			Watcher: e.executionEngine.OnMarketAuctionMaximumDurationUpdate,
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
		netparams.WatchParam{
			Param:   netparams.MarketMinProbabilityOfTradingForLPOrders,
			Watcher: e.executionEngine.OnMarketMinProbabilityOfTradingForLPOrdersUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketSuccessorLaunchWindow,
			Watcher: execsetup.executionEngine.OnSuccessorMarketTimeWindowUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.FloatingPointUpdatesDuration,
			Watcher: execsetup.stateVarEngine.OnFloatingPointUpdatesDurationUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.TransferFeeFactor,
			Watcher: execsetup.banking.OnTransferFeeFactorUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.TransferMinTransferQuantumMultiple,
			Watcher: execsetup.banking.OnMinTransferQuantumMultiple,
		},
		netparams.WatchParam{
			Param:   netparams.TransferFeeMaxQuantumAmount,
			Watcher: execsetup.banking.OnMaxQuantumAmountUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.TransferFeeDiscountDecayFraction,
			Watcher: execsetup.banking.OnTransferFeeDiscountDecayFractionUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.TransferFeeDiscountMinimumTrackedAmount,
			Watcher: execsetup.banking.OnTransferFeeDiscountMinimumTrackedAmountUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MaxPeggedOrders,
			Watcher: execsetup.executionEngine.OnMaxPeggedOrderUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarkPriceUpdateMaximumFrequency,
			Watcher: execsetup.executionEngine.OnMarkPriceUpdateMaximumFrequency,
		},
		netparams.WatchParam{
			Param:   netparams.InternalCompositePriceUpdateFrequency,
			Watcher: execsetup.executionEngine.OnInternalCompositePriceUpdateFrequency,
		},
		netparams.WatchParam{
			Param:   netparams.SpamProtectionMaxStopOrdersPerMarket,
			Watcher: execsetup.executionEngine.OnMarketPartiesMaximumStopOrdersUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityEarlyExitPenalty,
			Watcher: execsetup.executionEngine.OnMarketLiquidityV2EarlyExitPenaltyUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquiditySLANonPerformanceBondPenaltyMax,
			Watcher: execsetup.executionEngine.OnMarketLiquidityV2SLANonPerformanceBondPenaltyMaxUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquiditySLANonPerformanceBondPenaltySlope,
			Watcher: execsetup.executionEngine.OnMarketLiquidityV2SLANonPerformanceBondPenaltySlopeUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityBondPenaltyParameter,
			Watcher: execsetup.executionEngine.OnMarketLiquidityV2BondPenaltyUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityMaximumLiquidityFeeFactorLevel,
			Watcher: execsetup.executionEngine.OnMarketLiquidityV2MaximumLiquidityFeeFactorLevelUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityStakeToCCYVolume,
			Watcher: execsetup.executionEngine.OnMarketLiquidityV2StakeToCCYVolumeUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityProvidersFeeCalculationTimeStep,
			Watcher: execsetup.executionEngine.OnMarketLiquidityV2ProvidersFeeCalculationTimeStep,
		},
		netparams.WatchParam{
			Param:   netparams.ReferralProgramMinStakedVegaTokens,
			Watcher: execsetup.referralProgram.OnReferralProgramMinStakedVegaTokensUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.ReferralProgramMaxPartyNotionalVolumeByQuantumPerEpoch,
			Watcher: execsetup.referralProgram.OnReferralProgramMaxPartyNotionalVolumeByQuantumPerEpochUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.ReferralProgramMaxReferralRewardProportion,
			Watcher: execsetup.referralProgram.OnReferralProgramMaxReferralRewardProportionUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.ReferralProgramMinStakedVegaTokens,
			Watcher: execsetup.teamsEngine.OnReferralProgramMinStakedVegaTokensUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.RewardsActivityStreakBenefitTiers,
			Watcher: execsetup.activityStreak.OnBenefitTiersUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.RewardsActivityStreakMinQuantumOpenVolume,
			Watcher: execsetup.activityStreak.OnMinQuantumOpenNationalVolumeUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.RewardsActivityStreakMinQuantumTradeVolume,
			Watcher: execsetup.activityStreak.OnMinQuantumTradeVolumeUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.RewardsActivityStreakInactivityLimit,
			Watcher: execsetup.activityStreak.OnRewardsActivityStreakInactivityLimit,
		},
		netparams.WatchParam{
			Param:   netparams.RewardsVestingBaseRate,
			Watcher: execsetup.vesting.OnRewardVestingBaseRateUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.RewardsVestingMinimumTransfer,
			Watcher: execsetup.vesting.OnRewardVestingMinimumTransferUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.RewardsVestingBenefitTiers,
			Watcher: execsetup.vesting.OnBenefitTiersUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityEquityLikeShareFeeFraction,
			Watcher: execsetup.executionEngine.OnMarketLiquidityEquityLikeShareFeeFractionUpdate,
		},
	)
}
