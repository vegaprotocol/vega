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
	"regexp"
	"testing"

	"code.vegaprotocol.io/vega/core/integration/helpers"
	"code.vegaprotocol.io/vega/core/integration/steps"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/spf13/pflag"
)

var (
	gdOpts = godog.Options{
		Output: colors.Colored(os.Stdout),
		Format: "progress",
	}

	features string

	expectingEventsOverStepText = `there were "([0-9]+)" events emitted over the last step`
)

func init() {
	godog.BindCommandLineFlags("godog.", &gdOpts)
	pflag.StringVar(&features, "features", "", "a coma separated list of paths to the feature files")
}

func TestMain(m *testing.M) {
	pflag.Parse()
	gdOpts.Paths = pflag.Args()

	status := godog.TestSuite{
		Name:                 "godogs",
		TestSuiteInitializer: InitializeTestSuite,
		ScenarioInitializer:  InitializeScenario,
		Options:              &gdOpts,
	}.Run()

	os.Exit(status)
}

func InitializeTestSuite(ctx *godog.TestSuiteContext) {}

func InitializeScenario(s *godog.ScenarioContext) {
	s.BeforeScenario(func(*godog.Scenario) {
		execsetup = newExecutionTestSetup()
	})
	s.StepContext().Before(func(ctx context.Context, st *godog.Step) (context.Context, error) {
		// record accounts before step
		execsetup.accountsBefore = execsetup.broker.GetAccounts()
		execsetup.ledgerMovementsBefore = len(execsetup.broker.GetTransfers(false))
		execsetup.insurancePoolDepositsOverStep = make(map[string]*num.Int)

		// don't record events before step if it's the step that's meant to assess number of events over a regular step
		if b, _ := regexp.MatchString(expectingEventsOverStepText, st.Text); !b {
			execsetup.eventsBefore = len(execsetup.broker.GetAllEvents())
		}

		return ctx, nil
	})
	s.StepContext().After(func(ctx context.Context, st *godog.Step, status godog.StepResultStatus, err error) (context.Context, error) {
		aerr := reconcileAccounts()
		if aerr != nil {
			aerr = fmt.Errorf("failed to reconcile account balance changes over the last step from emitted events: %v", aerr)
		}
		return ctx, aerr
	})
	s.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		berr := steps.TheCumulatedBalanceForAllAccountsShouldBeWorth(execsetup.broker, execsetup.netDeposits.String())
		if berr != nil {
			berr = fmt.Errorf("error at scenario end (testing net deposits/withdrawals against cumulated balance for all accounts): %v", berr)
		}
		return ctx, berr
	})

	// delegation/validator steps
	s.Step(`the validators:$`, func(table *godog.Table) error {
		return steps.TheValidators(execsetup.topology, execsetup.stakingAccount, execsetup.delegationEngine, table)
	})
	s.Step(`^the parties should have the following delegation balances for epoch (\d+):$`, func(epoch string, table *godog.Table) error {
		return steps.PartiesShouldHaveTheFollowingDelegationBalances(execsetup.broker, table, epoch)
	})

	s.Step(`^the validators should have the following val scores for epoch (\d+):$`, func(epoch string, table *godog.Table) error {
		return steps.ValidatorsShouldHaveTheFollowingScores(execsetup.broker, table, epoch)
	})
	s.Step(`^the parties receive the following reward for epoch (\d+):$`, func(epoch string, table *godog.Table) error {
		return steps.PartiesShouldReceiveTheFollowingReward(execsetup.broker, table, epoch)
	})

	// Market steps
	s.Step(`the simple risk model named "([^"]*)":$`, func(name string, table *godog.Table) error {
		return steps.TheSimpleRiskModel(marketConfig, name, table)
	})
	s.Step(`the log normal risk model named "([^"]*)":$`, func(name string, table *godog.Table) error {
		return steps.TheLogNormalRiskModel(marketConfig, name, table)
	})
	s.Step(`the fees configuration named "([^"]*)":$`, func(name string, table *godog.Table) error {
		return steps.TheFeesConfiguration(marketConfig, name, table)
	})
	s.Step(`^the oracle spec for settlement data filtering data from "([^"]*)" named "([^"]*)":$`, func(signers string, name string, table *godog.Table) error {
		return steps.TheOracleSpec(marketConfig, name, "settlement data", signers, table)
	})
	s.Step(`^the oracle spec for trading termination filtering data from "([^"]*)" named "([^"]*)":$`, func(signers string, name string, table *godog.Table) error {
		return steps.TheOracleSpec(marketConfig, name, "trading termination", signers, table)
	})
	s.Step(`^the settlement data decimals for the oracle named "([^"]*)" is given in "([^"]*)" decimal places$`, func(name, decimals string) error {
		return steps.OracleSpecSettlementDataDecimals(marketConfig, name, decimals)
	})
	s.Step(`the price monitoring named "([^"]*)":$`, func(name string, table *godog.Table) error {
		return steps.ThePriceMonitoring(marketConfig, name, table)
	})
	s.Step(`the liquidity monitoring parameters:$`, func(table *godog.Table) error {
		return steps.TheLiquidityMonitoring(marketConfig, table)
	})
	s.Step(`the margin calculator named "([^"]*)":$`, func(name string, table *godog.Table) error {
		return steps.TheMarginCalculator(marketConfig, name, table)
	})
	s.Step(`^the markets:$`, func(table *godog.Table) error {
		markets, err := steps.TheMarkets(marketConfig, execsetup.executionEngine, execsetup.collateralEngine, execsetup.netParams, table)
		execsetup.markets = markets
		return err
	})
	s.Step(`^the markets are updated:$`, func(table *godog.Table) error {
		markets, err := steps.TheMarketsUpdated(marketConfig, execsetup.executionEngine, execsetup.markets, execsetup.netParams, table)
		if err != nil {
			return err
		}
		execsetup.markets = markets
		return nil
	})

	// Other steps
	s.Step(`^the initial insurance pool balance is "([^"]*)" for all the markets$`, func(amountstr string) error {
		amount, _ := num.UintFromString(amountstr, 10)
		for _, mkt := range execsetup.markets {
			asset, _ := mkt.GetAsset()
			marketInsuranceAccount, err := execsetup.collateralEngine.GetMarketInsurancePoolAccount(mkt.ID, asset)
			if err != nil {
				return err
			}
			if err := execsetup.collateralEngine.IncrementBalance(context.Background(), marketInsuranceAccount.ID, amount); err != nil {
				return err
			}
			execsetup.insurancePoolDepositsOverStep[marketInsuranceAccount.ID] = num.IntFromUint(amount, true)
			// add to the net deposits
			execsetup.netDeposits.Add(execsetup.netDeposits, amount)
		}
		return nil
	})

	s.Step(`^the market state should be "([^"]*)" for the market "([^"]*)"$`, func(marketState, marketID string) error {
		return steps.TheMarketStateShouldBeForMarket(execsetup.executionEngine, marketID, marketState)
	})
	s.Step(`^the following network parameters are set:$`, func(table *godog.Table) error {
		return steps.TheFollowingNetworkParametersAreSet(execsetup.netParams, table)
	})
	s.Step(`^time is updated to "([^"]*)"$`, func(rawTime string) error {
		steps.TimeIsUpdatedTo(execsetup.executionEngine, execsetup.timeService, rawTime)
		return nil
	})
	s.Step(`^the parties cancel the following orders:$`, func(table *godog.Table) error {
		return steps.PartiesCancelTheFollowingOrders(execsetup.broker, execsetup.executionEngine, table)
	})
	s.Step(`^the parties cancel all their orders for the markets:$`, func(table *godog.Table) error {
		return steps.PartiesCancelAllTheirOrdersForTheMarkets(execsetup.broker, execsetup.executionEngine, table)
	})
	s.Step(`^the parties amend the following orders:$`, func(table *godog.Table) error {
		return steps.PartiesAmendTheFollowingOrders(execsetup.broker, execsetup.executionEngine, table)
	})
	s.Step(`^the parties place the following pegged orders:$`, func(table *godog.Table) error {
		return steps.PartiesPlaceTheFollowingPeggedOrders(execsetup.executionEngine, table)
	})
	s.Step(`^the parties deposit on asset's general account the following amount:$`, func(table *godog.Table) error {
		return steps.PartiesDepositTheFollowingAssets(execsetup.collateralEngine, execsetup.broker, execsetup.netDeposits, table)
	})
	s.Step(`^the parties deposit on staking account the following amount:$`, func(table *godog.Table) error {
		return steps.PartiesTransferToStakingAccount(execsetup.stakingAccount, execsetup.broker, table, "")
	})
	s.Step(`^the parties withdraw from staking account the following amount:$`, func(table *godog.Table) error {
		return steps.PartiesWithdrawFromStakingAccount(execsetup.stakingAccount, execsetup.broker, table)
	})

	s.Step(`^the parties withdraw the following assets:$`, func(table *godog.Table) error {
		return steps.PartiesWithdrawTheFollowingAssets(execsetup.collateralEngine, execsetup.broker, execsetup.netDeposits, table)
	})
	s.Step(`^the parties place the following orders:$`, func(table *godog.Table) error {
		return steps.PartiesPlaceTheFollowingOrders(execsetup.executionEngine, execsetup.timeService, table)
	})

	s.Step(`^the parties place the following orders "([^"]+)" blocks apart:$`, func(blockCount string, table *godog.Table) error {
		return steps.PartiesPlaceTheFollowingOrdersBlocksApart(execsetup.executionEngine, execsetup.timeService, execsetup.block, execsetup.epochEngine, table, blockCount)
	})
	s.Step(`^the parties place the following orders with ticks:$`, func(table *godog.Table) error {
		return steps.PartiesPlaceTheFollowingOrdersWithTicks(execsetup.executionEngine, execsetup.timeService, execsetup.epochEngine, table)
	})

	s.Step(`^the parties submit the following liquidity provision:$`, func(table *godog.Table) error {
		return steps.PartiesSubmitLiquidityProvision(execsetup.executionEngine, table)
	})
	s.Step(`^party "([^"]+)" cancels their liquidity provision for market "([^"]+)"$`, func(party, marketID string) error {
		return steps.PartyCancelsTheirLiquidityProvision(execsetup.executionEngine, marketID, party)
	})
	s.Step(`^the parties submit the following one off transfers:$`, func(table *godog.Table) error {
		return steps.PartiesSubmitTransfers(execsetup.banking, table)
	})
	s.Step(`^the parties submit the following recurring transfers:$`, func(table *godog.Table) error {
		return steps.PartiesSubmitRecurringTransfers(execsetup.banking, table)
	})
	s.Step(`^the parties submit the following transfer cancellations:$`, func(table *godog.Table) error {
		return steps.PartiesCancelTransfers(execsetup.banking, table)
	})
	s.Step(`^the parties submit the following delegations:$`, func(table *godog.Table) error {
		return steps.PartiesDelegateTheFollowingStake(execsetup.delegationEngine, table)
	})
	s.Step(`^the parties submit the following undelegations:$`, func(table *godog.Table) error {
		return steps.PartiesUndelegateTheFollowingStake(execsetup.delegationEngine, table)
	})

	s.Step(`^the opening auction period ends for market "([^"]+)"$`, func(marketID string) error {
		return steps.MarketOpeningAuctionPeriodEnds(execsetup.executionEngine, execsetup.timeService, execsetup.markets, marketID)
	})
	s.Step(`^the oracles broadcast data signed with "([^"]*)":$`, func(pubKeys string, properties *godog.Table) error {
		return steps.OraclesBroadcastDataSignedWithKeys(execsetup.oracleEngine, pubKeys, properties)
	})
	s.Step(`^the following LP events should be emitted:$`, func(table *godog.Table) error {
		return steps.TheFollowingLPEventsShouldBeEmitted(execsetup.broker, table)
	})

	// block time stuff
	s.Step(`^the average block duration is "([^"]+)" with variance "([^"]+)"$`, func(block, variance string) error {
		return steps.TheAverageBlockDurationWithVariance(execsetup.block, block, variance)
	})
	s.Step(`^the average block duration is "([^"]+)"$`, func(blockTime string) error {
		return steps.TheAverageBlockDurationIs(execsetup.block, blockTime)
	})

	s.Step(`the network moves ahead "([^"]+)" blocks`, func(blocks string) error {
		return steps.TheNetworkMovesAheadNBlocks(execsetup.executionEngine, execsetup.block, execsetup.timeService, blocks, execsetup.epochEngine)
	})
	s.Step(`the network moves ahead "([^"]+)" with block duration of "([^"]+)"`, func(total, block string) error {
		return steps.TheNetworkMovesAheadDurationWithBlocks(execsetup.executionEngine, execsetup.block, execsetup.timeService, total, block)
	})

	// Assertion steps
	s.Step(`^the parties should have the following staking account balances:$`, func(table *godog.Table) error {
		return steps.PartiesShouldHaveTheFollowingStakingAccountBalances(execsetup.stakingAccount, table)
	})
	s.Step(`^the parties should have the following account balances:$`, func(table *godog.Table) error {
		return steps.PartiesShouldHaveTheFollowingAccountBalances(execsetup.broker, table)
	})
	s.Step(`^the parties should have the following margin levels:$`, func(table *godog.Table) error {
		return steps.ThePartiesShouldHaveTheFollowingMarginLevels(execsetup.broker, table)
	})
	s.Step(`^the parties should have the following profit and loss:$`, func(table *godog.Table) error {
		return steps.PartiesHaveTheFollowingProfitAndLoss(execsetup.positionPlugin, table)
	})
	s.Step(`^the order book should have the following volumes for market "([^"]*)":$`, func(marketID string, table *godog.Table) error {
		return steps.TheOrderBookOfMarketShouldHaveTheFollowingVolumes(execsetup.broker, marketID, table)
	})
	s.Step(`^the orders should have the following status:$`, func(table *godog.Table) error {
		return steps.TheOrdersShouldHaveTheFollowingStatus(execsetup.broker, table)
	})
	s.Step(`^the orders should have the following states:$`, func(table *godog.Table) error {
		return steps.TheOrdersShouldHaveTheFollowingStates(execsetup.broker, table)
	})
	s.Step(`^the pegged orders should have the following states:$`, func(table *godog.Table) error {
		return steps.ThePeggedOrdersShouldHaveTheFollowingStates(execsetup.broker, table)
	})
	s.Step(`^the following orders should be rejected:$`, func(table *godog.Table) error {
		return steps.TheFollowingOrdersShouldBeRejected(execsetup.broker, table)
	})
	s.Step(`^the following orders should be stopped:$`, func(table *godog.Table) error {
		return steps.TheFollowingOrdersShouldBeStopped(execsetup.broker, table)
	})
	s.Step(`^"([^"]*)" should have general account balance of "([^"]*)" for asset "([^"]*)"$`, func(party, balance, asset string) error {
		return steps.PartyShouldHaveGeneralAccountBalanceForAsset(execsetup.broker, party, asset, balance)
	})
	s.Step(`^the reward account of type "([^"]*)" should have balance of "([^"]*)" for asset "([^"]*)"$`, func(accountType, balance, asset string) error {
		return steps.RewardAccountBalanceForAssetShouldMatch(execsetup.broker, accountType, asset, balance)
	})
	s.Step(`^"([^"]*)" should have one account per asset$`, func(owner string) error {
		return steps.PartyShouldHaveOneAccountPerAsset(execsetup.broker, owner)
	})
	s.Step(`^"([^"]*)" should have one margin account per market$`, func(owner string) error {
		return steps.PartyShouldHaveOneMarginAccountPerMarket(execsetup.broker, owner)
	})
	s.Step(`^the cumulated balance for all accounts should be worth "([^"]*)"$`, func(rawAmount string) error {
		return steps.TheCumulatedBalanceForAllAccountsShouldBeWorth(execsetup.broker, rawAmount)
	})
	s.Step(`^the settlement account should have a balance of "([^"]*)" for the market "([^"]*)"$`, func(rawAmount, marketID string) error {
		return steps.TheSettlementAccountShouldHaveBalanceForMarket(execsetup.broker, rawAmount, marketID)
	})
	s.Step(`^the following network trades should be executed:$`, func(table *godog.Table) error {
		return steps.TheFollowingNetworkTradesShouldBeExecuted(execsetup.broker, table)
	})
	s.Step(`^the following trades should be executed:$`, func(table *godog.Table) error {
		return steps.TheFollowingTradesShouldBeExecuted(execsetup.broker, table)
	})
	s.Step(`^the trading mode should be "([^"]*)" for the market "([^"]*)"$`, func(tradingMode, marketID string) error {
		return steps.TheTradingModeShouldBeForMarket(execsetup.executionEngine, marketID, tradingMode)
	})
	s.Step(`^the insurance pool balance should be "([^"]*)" for the market "([^"]*)"$`, func(rawAmount, marketID string) error {
		return steps.TheInsurancePoolBalanceShouldBeForTheMarket(execsetup.broker, rawAmount, marketID)
	})
	s.Step(`^the network treasury balance should be "([^"]*)" for the asset "([^"]*)"$`, func(rawAmount, asset string) error {
		return steps.TheNetworkTreasuryBalanceShouldBeForTheAsset(execsetup.broker, rawAmount, asset)
	})
	s.Step(`^the following transfers should happen:$`, func(table *godog.Table) error {
		return steps.TheFollowingTransfersShouldHappen(execsetup.broker, table)
	})
	s.Step(`^the mark price should be "([^"]*)" for the market "([^"]*)"$`, func(rawMarkPrice, marketID string) error {
		return steps.TheMarkPriceForTheMarketIs(execsetup.executionEngine, marketID, rawMarkPrice)
	})
	s.Step(`^the liquidity provisions should have the following states:$`, func(table *godog.Table) error {
		return steps.TheLiquidityProvisionsShouldHaveTheFollowingStates(execsetup.broker, table)
	})
	s.Step(`^the target stake should be "([^"]*)" for the market "([^"]*)"$`, func(stake, marketID string) error {
		return steps.TheTargetStakeShouldBeForMarket(execsetup.executionEngine, marketID, stake)
	})
	s.Step(`^the supplied stake should be "([^"]*)" for the market "([^"]*)"$`, func(stake, marketID string) error {
		return steps.TheSuppliedStakeShouldBeForTheMarket(execsetup.executionEngine, marketID, stake)
	})
	s.Step(`^the open interest should be "([^"]*)" for the market "([^"]*)"$`, func(stake, marketID string) error {
		return steps.TheOpenInterestShouldBeForTheMarket(execsetup.executionEngine, marketID, stake)
	})
	s.Step(`^the liquidity provider fee shares for the market "([^"]*)" should be:$`, func(marketID string, table *godog.Table) error {
		return steps.TheLiquidityProviderFeeSharesForTheMarketShouldBe(execsetup.executionEngine, marketID, table)
	})
	s.Step(`^the price monitoring bounds for the market "([^"]*)" should be:$`, func(marketID string, table *godog.Table) error {
		return steps.ThePriceMonitoringBoundsForTheMarketShouldBe(execsetup.executionEngine, marketID, table)
	})
	s.Step(`^the accumulated liquidity fees should be "([^"]*)" for the market "([^"]*)"$`, func(amount, marketID string) error {
		return steps.TheAccumulatedLiquidityFeesShouldBeForTheMarket(execsetup.broker, amount, marketID)
	})
	s.Step(`^the accumulated infrastructure fees should be "([^"]*)" for the asset "([^"]*)"$`, func(amount, asset string) error {
		return steps.TheAccumulatedInfrastructureFeesShouldBeForTheMarket(execsetup.broker, amount, asset)
	})
	s.Step(`^the liquidity fee factor should be "([^"]*)" for the market "([^"]*)"$`, func(fee, marketID string) error {
		return steps.TheLiquidityFeeFactorShouldForTheMarket(execsetup.broker, fee, marketID)
	})
	s.Step(`^the market data for the market "([^"]+)" should be:$`, func(marketID string, table *godog.Table) error {
		return steps.TheMarketDataShouldBe(execsetup.executionEngine, marketID, table)
	})
	s.Step(`the auction ends with a traded volume of "([^"]+)" at a price of "([^"]+)"`, func(vol, price string) error {
		now := execsetup.timeService.GetTimeNow()
		return steps.TheAuctionTradedVolumeAndPriceShouldBe(execsetup.broker, vol, price, now)
	})
	s.Step(expectingEventsOverStepText, func(eventCounter int) error {
		return steps.ExpectingEventsOverStep(execsetup.broker, execsetup.eventsBefore, eventCounter)
	})
	s.Step(`there were "([0-9]+)" events emitted in this scenario so far`, func(eventCounter int) error {
		return steps.ExpectingEventsInTheSecenarioSoFar(execsetup.broker, eventCounter)
	})
	s.Step(`fail`, func() {
		reporter.Fatalf("fail step invoked")
	})

	// Debug steps
	s.Step(`^debug accounts$`, func() error {
		steps.DebugAccounts(execsetup.broker, execsetup.log)
		return nil
	})
	s.Step(`^debug transfers$`, func() error {
		steps.DebugTransfers(execsetup.broker, execsetup.log)
		return nil
	})
	s.Step(`^debug trades$`, func() error {
		steps.DebugTrades(execsetup.broker, execsetup.log)
		return nil
	})
	s.Step(`^debug orders$`, func() error {
		steps.DebugOrders(execsetup.broker, execsetup.log)
		return nil
	})
	s.Step(`^debug market data for "([^"]*)"$`, func(mkt string) error {
		return steps.DebugMarketData(execsetup.executionEngine, execsetup.log, mkt)
	})
	s.Step(`^debug all events$`, func() error {
		steps.DebugAllEvents(execsetup.broker, execsetup.log)
		return nil
	})
	s.Step(`^debug auction events$`, func() error {
		steps.DebugAuctionEvents(execsetup.broker, execsetup.log)
		return nil
	})
	s.Step(`^debug transaction errors$`, func() error {
		steps.DebugTxErrors(execsetup.broker, execsetup.log)
		return nil
	})
	s.Step(`^debug liquidity submission errors$`, func() error {
		steps.DebugLPSTxErrors(execsetup.broker, execsetup.log)
		return nil
	})
	s.Step(`^debug liquidity provision events$`, func() error {
		steps.DebugLPs(execsetup.broker, execsetup.log)
		return nil
	})
	s.Step(`^debug detailed liquidity provision events$`, func() error {
		steps.DebugLPDetail(execsetup.log, execsetup.broker)
		return nil
	})
	s.Step(`^debug orderbook volumes for market "([^"]*)"$`, func(mkt string) error {
		return steps.DebugVolumesForMarket(execsetup.log, execsetup.broker, mkt)
	})
	s.Step(`^debug detailed orderbook volumes for market "([^"]*)"$`, func(mkt string) error {
		return steps.DebugVolumesForMarketDetail(execsetup.log, execsetup.broker, mkt)
	})
	s.Step(`^debug last "([0-9]+)" events$`, func(eventCounter int) error {
		steps.DebugLastNEvents(eventCounter, execsetup.broker, execsetup.log)
		return nil
	})

	// Event steps
	s.Step(`^clear all events$`, func() error {
		steps.ClearAllEvents(execsetup.broker)
		return nil
	})
	s.Step(`^clear transfer response events$`, func() error {
		steps.ClearTransferResponseEvents(execsetup.broker)
		return nil
	})
	s.Step(`^the following events should be emitted:$`, func(table *godog.Table) error {
		return steps.TheFollowingEventsShouldBeEmitted(execsetup.broker, table)
	})
	s.Step(`^the following events should NOT be emitted:$`, func(table *godog.Table) error {
		return steps.TheFollowingEventsShouldNotBeEmitted(execsetup.broker, table)
	})
	s.Step(`^a total of "([0-9]+)" events should be emitted$`, func(eventCounter int) error {
		return steps.TotalOfEventsShouldBeEmitted(execsetup.broker, eventCounter)
	})

	// Decimal places steps
	s.Step(`^the following assets are registered:$`, func(table *godog.Table) error {
		return steps.RegisterAsset(table, execsetup.assetsEngine)
	})
	s.Step(`^set assets to strict$`, func() error {
		execsetup.assetsEngine.SetStrict()
		return nil
	})
	s.Step(`^set assets to permissive$`, func() error {
		execsetup.assetsEngine.SetPermissive()
		return nil
	})
}

func reconcileAccounts() error {
	return helpers.ReconcileAccountChanges(execsetup.collateralEngine, execsetup.accountsBefore, execsetup.broker.GetAccounts(), execsetup.insurancePoolDepositsOverStep, extractLedgerEntriesOverStep())
}

func extractLedgerEntriesOverStep() []*vega.LedgerEntry {
	transfers := execsetup.broker.GetTransfers(false)
	n := len(transfers) - execsetup.ledgerMovementsBefore
	ret := make([]*vega.LedgerEntry, 0, n)
	if n > 0 {
		for i := execsetup.ledgerMovementsBefore; i < len(transfers); i++ {
			ret = append(ret, transfers[i])
		}
	}
	return ret
}
