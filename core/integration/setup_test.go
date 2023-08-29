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

package core_test

import (
	"context"
	"fmt"
	"os"

	"code.vegaprotocol.io/vega/core/banking"
	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/delegation"
	"code.vegaprotocol.io/vega/core/epochtime"
	"code.vegaprotocol.io/vega/core/evtforward"
	"code.vegaprotocol.io/vega/core/execution"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/notary"
	"code.vegaprotocol.io/vega/core/rewards"
	"code.vegaprotocol.io/vega/core/teams"
	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/core/vesting"
	"code.vegaprotocol.io/vega/libs/num"

	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/integration/helpers"
	"code.vegaprotocol.io/vega/core/integration/steps/market"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/plugins"
	"code.vegaprotocol.io/vega/core/types"
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

func (DummyASVM) Get(_ string) num.Decimal {
	return num.MustDecimalFromString("0.01")
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

	ntry           *notary.SnapshotNotary
	stateVarEngine *stubs.StateVarStub
	witness        *validators.Witness
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
	execsetup.cfg.Position.StreamPositionVerbose = true
	execsetup.cfg.Risk.StreamMarginLevelsVerbose = true
	execsetup.log = logging.NewTestLogger()
	execsetup.timeService = stubs.NewTimeStub()
	execsetup.broker = stubs.NewBrokerStub()
	execsetup.collateralEngine = collateral.New(
		execsetup.log, collateral.NewDefaultConfig(), execsetup.timeService, execsetup.broker,
	)

	vegaAsset := types.Asset{
		ID: "VEGA",
		Details: &types.AssetDetails{
			Name:   "VEGA",
			Symbol: "VEGA",
		},
	}
	execsetup.collateralEngine.EnableAsset(context.Background(), vegaAsset)

	usdt := types.Asset{
		ID: "USDT",
		Details: &types.AssetDetails{
			Name:   "USDT",
			Symbol: "USDT",
		},
	}
	execsetup.collateralEngine.EnableAsset(context.Background(), usdt)

	usdc := types.Asset{
		ID: "USDC",
		Details: &types.AssetDetails{
			Name:   "USDC",
			Symbol: "USDC",
		},
	}

	execsetup.collateralEngine.EnableAsset(context.Background(), usdc)

	execsetup.epochEngine = epochtime.NewService(execsetup.log, epochtime.NewDefaultConfig(), execsetup.broker)

	execsetup.topology = stubs.NewTopologyStub("nodeID", execsetup.broker)

	execsetup.stakingAccount = stubs.NewStakingAccountStub()
	execsetup.epochEngine.NotifyOnEpoch(execsetup.stakingAccount.OnEpochEvent, execsetup.stakingAccount.OnEpochRestore)

	teams := teams.NewEngine(execsetup.epochEngine, execsetup.broker, execsetup.timeService)
	marketActivityTracker := common.NewMarketActivityTracker(execsetup.log, execsetup.epochEngine, teams, execsetup.stakingAccount)
	commander := stubs.NewCommanderStub()
	execsetup.netDeposits = num.UintZero()
	execsetup.witness = validators.NewWitness(context.Background(), execsetup.log, validators.NewDefaultConfig(), execsetup.topology, commander, execsetup.timeService)

	execsetup.ntry = notary.NewWithSnapshot(execsetup.log, notary.NewDefaultConfig(), execsetup.topology, execsetup.broker, commander)
	execsetup.assetsEngine = stubs.NewAssetStub()
	ethSourceNoop := evtforward.NewNoopEngine(execsetup.log, evtforward.NewDefaultConfig())
	execsetup.banking = banking.New(execsetup.log, banking.NewDefaultConfig(), execsetup.collateralEngine, execsetup.witness, execsetup.timeService, execsetup.assetsEngine, execsetup.ntry, execsetup.broker, execsetup.topology, execsetup.epochEngine, marketActivityTracker, stubs.NewBridgeViewStub(), ethSourceNoop)

	execsetup.delegationEngine = delegation.New(execsetup.log, delegation.NewDefaultConfig(), execsetup.broker, execsetup.topology, execsetup.stakingAccount, execsetup.epochEngine, execsetup.timeService)

	vesting := vesting.New(execsetup.log, execsetup.collateralEngine, DummyASVM{}, execsetup.broker, execsetup.assetsEngine)
	// TODO fix activity streak
	activityStreak := &DummyActivityStreak{}
	execsetup.rewardsEngine = rewards.New(execsetup.log, rewards.NewDefaultConfig(), execsetup.broker, execsetup.delegationEngine, execsetup.epochEngine, execsetup.collateralEngine, execsetup.timeService, marketActivityTracker, execsetup.topology, vesting, execsetup.banking, activityStreak)

	execsetup.oracleEngine = spec.NewEngine(
		execsetup.log, spec.NewDefaultConfig(), execsetup.timeService, execsetup.broker)

	execsetup.builtinOracle = spec.NewBuiltin(execsetup.oracleEngine, execsetup.timeService)

	execsetup.stateVarEngine = stubs.NewStateVar()
	// @TODO stub assets engine and pass it in

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
		),
		execsetup.broker,
	)
	execsetup.epochEngine.NotifyOnEpoch(execsetup.executionEngine.OnEpochEvent, execsetup.executionEngine.OnEpochRestore)
	execsetup.positionPlugin = plugins.NewPositions(context.Background())
	execsetup.broker.Subscribe(execsetup.positionPlugin)

	execsetup.block = helpers.NewBlock()

	execsetup.registerTimeServiceCallbacks()

	execsetup.netParams = netparams.New(execsetup.log, netparams.NewDefaultConfig(), execsetup.broker)
	if err := execsetup.registerNetParamsCallbacks(); err != nil {
		panic(err)
	}

	execsetup.netParams.Watch(
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
			Param:   netparams.MaxPeggedOrders,
			Watcher: execsetup.executionEngine.OnMaxPeggedOrderUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarkPriceUpdateMaximumFrequency,
			Watcher: execsetup.executionEngine.OnMarkPriceUpdateMaximumFrequency,
		},
		netparams.WatchParam{
			Param:   netparams.SpamProtectionMaxStopOrdersPerMarket,
			Watcher: execsetup.executionEngine.OnMarketPartiesMaximumStopOrdersUpdate,
		},

		netparams.WatchParam{
			Param:   netparams.MarketLiquidityV2EarlyExitPenalty,
			Watcher: execsetup.executionEngine.OnMarketLiquidityV2EarlyExitPenaltyUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityV2SLANonPerformanceBondPenaltyMax,
			Watcher: execsetup.executionEngine.OnMarketLiquidityV2SLANonPerformanceBondPenaltyMaxUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityV2SLANonPerformanceBondPenaltySlope,
			Watcher: execsetup.executionEngine.OnMarketLiquidityV2SLANonPerformanceBondPenaltySlopeUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityV2BondPenaltyParameter,
			Watcher: execsetup.executionEngine.OnMarketLiquidityV2BondPenaltyUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityV2MaximumLiquidityFeeFactorLevel,
			Watcher: execsetup.executionEngine.OnMarketLiquidityV2MaximumLiquidityFeeFactorLevelUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityV2StakeToCCYVolume,
			Watcher: execsetup.executionEngine.OnMarketLiquidityV2StakeToCCYVolumeUpdate,
		},
	)
	return execsetup
}

func (e *executionTestSetup) registerTimeServiceCallbacks() {
	e.timeService.NotifyOnTick(
		e.epochEngine.OnTick,
		e.witness.OnTick,
		e.ntry.OnTick,
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
			Param:   netparams.MarketLiquidityV2BondPenaltyParameter,
			Watcher: e.executionEngine.OnMarketLiquidityV2BondPenaltyUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityV2EarlyExitPenalty,
			Watcher: e.executionEngine.OnMarketLiquidityV2EarlyExitPenaltyUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityV2MaximumLiquidityFeeFactorLevel,
			Watcher: e.executionEngine.OnMarketLiquidityV2MaximumLiquidityFeeFactorLevelUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityV2SLANonPerformanceBondPenaltySlope,
			Watcher: e.executionEngine.OnMarketLiquidityV2SLANonPerformanceBondPenaltySlopeUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityV2SLANonPerformanceBondPenaltyMax,
			Watcher: e.executionEngine.OnMarketLiquidityV2SLANonPerformanceBondPenaltyMaxUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityV2StakeToCCYVolume,
			Watcher: e.executionEngine.OnMarketLiquidityV2StakeToCCYVolumeUpdate,
		},
		// End of liquidity version 2.
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
		netparams.WatchParam{
			Param:   netparams.MarketMinProbabilityOfTradingForLPOrders,
			Watcher: e.executionEngine.OnMarketMinProbabilityOfTradingForLPOrdersUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketSuccessorLaunchWindow,
			Watcher: execsetup.executionEngine.OnSuccessorMarketTimeWindowUpdate,
		},
	)
}

type DummyActivityStreak struct{}

func (*DummyActivityStreak) GetRewardsDistributionMultiplier(party string) num.Decimal {
	return num.DecimalOne()
}
