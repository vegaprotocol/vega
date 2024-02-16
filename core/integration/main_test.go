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
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"testing"

	"code.vegaprotocol.io/vega/core/integration/helpers"
	"code.vegaprotocol.io/vega/core/integration/steps"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
)

var (
	gdOpts = godog.Options{
		Output: colors.Colored(os.Stdout),
		Format: "progress",
	}

	perpsSwap bool
	features  string

	expectingEventsOverStepText = `there were "([0-9]+)" events emitted over the last step`
)

func init() {
	godog.BindFlags("godog.", flag.CommandLine, &gdOpts)
	flag.StringVar(&features, "features", "", "a coma separated list of paths to the feature files")
	flag.BoolVar(&perpsSwap, "perps", false, "Runs all tests swapping out the default futures oracles for their corresponding perps oracle")
}

func TestMain(m *testing.M) {
	flag.Parse()
	gdOpts.Paths = flag.Args()

	if testing.Short() {
		log.Print("Skipping core integration tests, go test run with -short")
		return
	}
	if perpsSwap {
		marketConfig.OracleConfigs.SwapToPerps()
		gdOpts.Tags += " ~NoPerp"
	}

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
		// set default netparams
		execsetup.netParams.Update(ctx, netparams.MarketSuccessorLaunchWindow, "1h")

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

	s.Step(`^the stop orders should have the following states$`, func(table *godog.Table) error {
		return steps.TheStopOrdersShouldHaveTheFollowingStates(execsetup.broker, table)
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
	s.Step(`^the current epoch is "([^"]+)"$`, func(epoch string) error {
		return steps.TheCurrentEpochIs(execsetup.broker, epoch)
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
	s.Step(`^the perpetual oracles from "([^"]+)":`, func(signers string, table *godog.Table) error {
		return steps.ThePerpsOracleSpec(marketConfig, signers, table)
	})
	s.Step(`^the composite price oracles from "([^"]+)":`, func(signers string, table *godog.Table) error {
		return steps.TheCompositePriceOracleSpec(marketConfig, signers, table)
	})
	s.Step(`the price monitoring named "([^"]*)":$`, func(name string, table *godog.Table) error {
		return steps.ThePriceMonitoring(marketConfig, name, table)
	})
	s.Step(`the liquidity sla params named "([^"]*)":$`, func(name string, table *godog.Table) error {
		return steps.TheLiquiditySLAPArams(marketConfig, name, table)
	})
	s.Step(`the liquidity monitoring parameters:$`, func(table *godog.Table) error {
		return steps.TheLiquidityMonitoring(marketConfig, table)
	})
	s.Step(`the margin calculator named "([^"]*)":$`, func(name string, table *godog.Table) error {
		return steps.TheMarginCalculator(marketConfig, name, table)
	})
	s.Step(`^the parties submit update margin mode:$`, func(table *godog.Table) error {
		return steps.ThePartiesUpdateMarginMode(execsetup.executionEngine, table)
	})

	s.Step(`^the markets:$`, func(table *godog.Table) error {
		markets, err := steps.TheMarkets(marketConfig, execsetup.executionEngine, execsetup.collateralEngine, execsetup.netParams, execsetup.timeService.GetTimeNow(), table)
		execsetup.markets = markets
		return err
	})
	s.Step(`^the spot markets:$`, func(table *godog.Table) error {
		markets, err := steps.TheSpotMarkets(marketConfig, execsetup.executionEngine, execsetup.collateralEngine, execsetup.netParams, execsetup.timeService.GetTimeNow(), table)
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

	s.Step(`^the spot markets are updated:$`, func(table *godog.Table) error {
		markets, err := steps.TheSpotMarketsUpdated(marketConfig, execsetup.executionEngine, execsetup.markets, execsetup.netParams, table)
		if err != nil {
			return err
		}
		execsetup.markets = markets
		return nil
	})

	s.Step(`the successor market "([^"]+)" is enacted$`, func(successor string) error {
		if err := steps.TheSuccesorMarketIsEnacted(successor, execsetup.markets, execsetup.executionEngine); err != nil {
			return err
		}
		return nil
	})

	// Other steps
	s.Step(`^the initial insurance pool balance is "([^"]*)" for all the markets$`, func(amountstr string) error {
		amount, _ := num.UintFromString(amountstr, 10)
		for _, mkt := range execsetup.markets {
			assets, _ := mkt.GetAssets()
			marketInsuranceAccount, err := execsetup.collateralEngine.GetMarketInsurancePoolAccount(mkt.ID, assets[0])
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

	s.Step(`^the mark price algo should be "([^"]+)" for the market "([^"]+)"$`, func(mpAlgo, marketID string) error {
		return steps.TheMarkPriceAlgoShouldBeForMarket(execsetup.broker, marketID, mpAlgo)
	})

	s.Step(`^the last market state should be "([^"]+)" for the market "([^"]+)"$`, func(mState, marketID string) error {
		return steps.TheLastStateUpdateShouldBeForMarket(execsetup.broker, marketID, mState)
	})
	s.Step(`^the market state should be "([^"]*)" for the market "([^"]*)"$`, func(marketState, marketID string) error {
		return steps.TheMarketStateShouldBeForMarket(execsetup.executionEngine, marketID, marketState)
	})
	s.Step(`^the following network parameters are set:$`, func(table *godog.Table) error {
		return steps.TheFollowingNetworkParametersAreSet(execsetup.netParams, table)
	})
	s.Step(`^the market states are updated through governance:`, func(data *godog.Table) error {
		return steps.TheMarketStateIsUpdatedTo(execsetup.executionEngine, data)
	})
	s.Step(`^time is updated to "([^"]*)"$`, func(rawTime string) error {
		steps.TimeIsUpdatedTo(execsetup.executionEngine, execsetup.timeService, rawTime)
		return nil
	})
	s.Step(`^the parties cancel the following orders:$`, func(table *godog.Table) error {
		return steps.PartiesCancelTheFollowingOrders(execsetup.broker, execsetup.executionEngine, table)
	})
	s.Step(`^the parties cancel the following stop orders:$`, func(table *godog.Table) error {
		return steps.PartiesCancelTheFollowingStopOrders(execsetup.broker, execsetup.executionEngine, table)
	})
	s.Step(`^the party "([^"]*)" cancels all their stop orders for the market "([^"]*)"$`, func(partyId, marketId string) error {
		return steps.PartyCancelsAllTheirStopOrdersForTheMarket(execsetup.executionEngine, partyId, marketId)
	})
	s.Step(`^the party "([^"]*)" cancels all their stop orders`, func(partyId string) error {
		return steps.PartyCancelsAllTheirStopOrders(execsetup.executionEngine, partyId)
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
	s.Step(`^the party "([^"]+)" adds the following orders to a batch:$`, func(party string, table *godog.Table) error {
		return steps.PartyAddsTheFollowingOrdersToABatch(party, execsetup.executionEngine, execsetup.timeService, table)
	})

	s.Step(`^the party "([^"]+)" adds the following iceberg orders to a batch:$`, func(party string, table *godog.Table) error {
		return steps.PartyAddsTheFollowingIcebergOrdersToABatch(party, execsetup.executionEngine, execsetup.timeService, table)
	})

	s.Step(`^the party "([^"]+)" starts a batch instruction$`, func(party string) error {
		return steps.PartyStartsABatchInstruction(party, execsetup.executionEngine)
	})

	s.Step(`^the party "([^"]+)" submits their batch instruction$`, func(party string) error {
		return steps.PartySubmitsTheirBatchInstruction(party, execsetup.executionEngine)
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
	s.Step(`^the parties have the following transfer fee discounts`, func(table *godog.Table) error {
		return steps.PartiesAvailableFeeDiscounts(execsetup.banking, table)
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
		return steps.OraclesBroadcastDataSignedWithKeys(execsetup.oracleEngine, execsetup.timeService, pubKeys, properties)
	})
	s.Step(`^the oracles broadcast data with block time signed with "([^"]*)":$`, func(pubKeys string, properties *godog.Table) error {
		return steps.OraclesBroadcastDataWithBlockTimeSignedWithKeys(execsetup.oracleEngine, execsetup.timeService, pubKeys, properties)
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

	s.Step(`^the parties place the following iceberg orders:$`, func(table *godog.Table) error {
		return steps.PartiesPlaceTheFollowingIcebergOrders(execsetup.executionEngine, execsetup.timeService, table)
	})

	s.Step(`^the parties place the following pegged iceberg orders:$`, func(table *godog.Table) error {
		return steps.PartiesPlaceTheFollowingPeggedIcebergOrders(execsetup.executionEngine, execsetup.timeService, table)
	})

	s.Step(`^the parties amend the following pegged iceberg orders:$`, func(table *godog.Table) error {
		return steps.PartiesAmendTheFollowingPeggedIcebergOrders(execsetup.broker, execsetup.executionEngine, execsetup.timeService, table)
	})

	s.Step(`^the iceberg orders should have the following states:$`, func(table *godog.Table) error {
		return steps.TheIcebergOrdersShouldHaveTheFollowingStates(execsetup.broker, table)
	})

	s.Step(`the network moves ahead "([^"]+)" blocks`, func(blocks string) error {
		return steps.TheNetworkMovesAheadNBlocks(execsetup.executionEngine, execsetup.block, execsetup.timeService, blocks, execsetup.epochEngine)
	})
	s.Step(`the network moves ahead "([^"]+)" with block duration of "([^"]+)"`, func(total, block string) error {
		return steps.TheNetworkMovesAheadDurationWithBlocks(execsetup.executionEngine, execsetup.block, execsetup.timeService, total, block)
	})
	s.Step(`^the network moves ahead "([^"]+)" epochs$`, func(epochs string) error {
		return steps.TheNetworkMovesAheadNEpochs(execsetup.broker, execsetup.block, execsetup.executionEngine, execsetup.epochEngine, execsetup.timeService, epochs)
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
	s.Step(`^"([^"]*)" should have vesting account balance of "([^"]*)" for asset "([^"]*)"$`, func(party, balance, asset string) error {
		return steps.PartyShouldHaveVestingAccountBalanceForAsset(execsetup.broker, party, asset, balance)
	})
	s.Step(`^parties should have the following vesting account balances:$`, func(table *godog.Table) error {
		return steps.PartiesShouldHaveVestingAccountBalances(execsetup.broker, table)
	})
	s.Step(`^parties should have the following vested account balances:$`, func(table *godog.Table) error {
		return steps.PartiesShouldHaveVestedAccountBalances(execsetup.broker, table)
	})
	s.Step(`^"([^"]*)" should have vested account balance of "([^"]*)" for asset "([^"]*)"$`, func(party, balance, asset string) error {
		return steps.PartyShouldHaveVestedAccountBalanceForAsset(execsetup.broker, party, asset, balance)
	})
	s.Step(`^"([^"]*)" should have holding account balance of "([^"]*)" for asset "([^"]*)"$`, func(party, balance, asset string) error {
		return steps.PartyShouldHaveHoldingAccountBalanceForAsset(execsetup.broker, party, asset, balance)
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
	s.Step(`^the global insurance pool balance should be "([^"]*)" for the asset "([^"]*)"$`, func(rawAmount, asset string) error {
		return steps.TheGlobalInsuranceBalanceShouldBeForTheAsset(execsetup.broker, rawAmount, asset)
	})
	s.Step(`^the party "([^"]*)" lp liquidity fee account balance should be "([^"]*)" for the market "([^"]*)"$`, func(party, rawAmount, market string) error {
		return steps.TheLPLiquidityFeeBalanceShouldBeForTheMarket(execsetup.broker, party, rawAmount, market)
	})
	s.Step(`^the party "([^"]*)" lp liquidity bond account balance should be "([^"]*)" for the market "([^"]*)"$`, func(party, rawAmount, market string) error {
		return steps.TheLPLiquidityBondBalanceShouldBeForTheMarket(execsetup.broker, party, rawAmount, market)
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
	s.Step(`^the product data for the market "([^"]+)" should be:$`, func(marketID string, table *godog.Table) error {
		return steps.TheProductDataShouldBe(execsetup.executionEngine, marketID, table)
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

	// Referral program steps.
	s.Step(`^the referral program:$`, func(table *godog.Table) error {
		return steps.TheReferralProgram(referralProgramConfig, execsetup.referralProgram, table)
	})
	s.Step(`^the referral benefit tiers "([^"]+)":$`, func(name string, table *godog.Table) error {
		return steps.TheReferralBenefitTiersConfiguration(referralProgramConfig, name, table)
	})
	s.Step(`^the referral staking tiers "([^"]+)":$`, func(name string, table *godog.Table) error {
		return steps.TheReferralStakingTiersConfiguration(referralProgramConfig, name, table)
	})
	s.Step(`^the parties create the following referral codes:$`, func(table *godog.Table) error {
		return steps.PartiesCreateTheFollowingReferralCode(execsetup.referralProgram, execsetup.teamsEngine, table)
	})
	s.Step(`^the parties apply the following referral codes:$`, func(table *godog.Table) error {
		return steps.PartiesApplyTheFollowingReferralCode(execsetup.referralProgram, execsetup.teamsEngine, table)
	})
	s.Step(`^the team "([^"]*)" has the following members:$`, func(team string, table *godog.Table) error {
		return steps.TheTeamHasTheFollowingMembers(execsetup.teamsEngine, team, table)
	})
	s.Step(`^the following teams with referees are created:$`, func(table *godog.Table) error {
		return steps.TheFollowingTeamsWithRefereesAreCreated(execsetup.collateralEngine, execsetup.broker, execsetup.netDeposits, execsetup.referralProgram, execsetup.teamsEngine, table)
	})

	s.Step(`the referral set stats for code "([^"]+)" at epoch "([^"]+)" should have a running volume of (\d+):`, func(code, epoch, volume string, table *godog.Table) error {
		return steps.TheReferralSetStatsShouldBe(execsetup.broker, code, epoch, volume, table)
	})
	s.Step(`the activity streaks at epoch "([^"]+)" should be:`, func(epoch string, table *godog.Table) error {
		return steps.TheActivityStreaksShouldBe(execsetup.broker, epoch, table)
	})
	s.Step(`the vesting stats at epoch "([^"]+)" should be:`, func(epoch string, table *godog.Table) error {
		return steps.TheVestingStatsShouldBe(execsetup.broker, epoch, table)
	})
	s.Step(`the volume discount stats at epoch "([^"]+)" should be:`, func(epoch string, table *godog.Table) error {
		return steps.TheVolumeDiscountStatsShouldBe(execsetup.broker, epoch, table)
	})
	// AMM steps
	s.Step(`^the parties submit the following AMM:$`, func(table *godog.Table) error {
		return steps.PartiesSubmitTheFollowingAMMs(execsetup.executionEngine, table)
	})
	s.Step(`^the parties amend the following AMM:$`, func(table *godog.Table) error {
		return steps.PartiesAmendTheFollowingAMMs(execsetup.executionEngine, table)
	})
	s.Step(`^the parties cancel the following AMM:$`, func(table *godog.Table) error {
		return steps.PartiesCancelTheFollowingAMMs(execsetup.executionEngine, table)
	})
	s.Step(`^the AMM pool status should be:$`, func(table *godog.Table) error {
		return steps.AMMPoolStatusShouldBe(execsetup.broker, table)
	})
	s.Step(`^the following AMM pool events should be emitted:$`, func(table *godog.Table) error {
		return steps.ExpectToSeeAMMEvents(execsetup.broker, table)
	})
	// AMM specific debugging
	s.Step(`^debug all AMM pool events$`, func() error {
		return steps.DebugAMMPoolEvents(execsetup.broker, execsetup.log)
	})
	s.Step(`^debug AMM pool events for party "([^"]+)"$`, func(party string) error {
		return steps.DebugAMMPoolEventsForPartyMarket(execsetup.broker, execsetup.log, ptr.From(party), nil)
	})
	s.Step(`^debug all AMM pool events for market "([^"]+)"$`, func(market string) error {
		return steps.DebugAMMPoolEventsForPartyMarket(execsetup.broker, execsetup.log, nil, ptr.From(market))
	})
	s.Step(`^debug all AMM pool events for market "([^"]+)" and party "([^"]+)"$`, func(market, party string) error {
		return steps.DebugAMMPoolEventsForPartyMarket(execsetup.broker, execsetup.log, ptr.From(party), ptr.From(market))
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
	s.Step(`^debug all events as JSON file "([^"]+)"$`, func(fname string) error {
		return steps.DebugAllEventsJSONFile(execsetup.broker, execsetup.log, fname)
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
	s.Step(`^debug network parameter "([^"]*)"$`, func(name string) error {
		return steps.DebugNetworkParameter(execsetup.log, execsetup.netParams, name)
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
	s.Step(`^the loss socialisation amounts are:$`, func(table *godog.Table) error {
		return steps.TheLossSocialisationAmountsAre(execsetup.broker, table)
	})
	s.Step(`^debug loss socialisation events$`, func() error {
		return steps.DebugLossSocialisationEvents(execsetup.broker, execsetup.log)
	})

	// Decimal places steps
	s.Step(`^the following assets are registered:$`, func(table *godog.Table) error {
		return steps.RegisterAsset(table, execsetup.assetsEngine, execsetup.collateralEngine)
	})
	s.Step(`^the following assets are updated:$`, func(table *godog.Table) error {
		return steps.UpdateAsset(table, execsetup.assetsEngine, execsetup.collateralEngine)
	})
	s.Step(`^set assets to strict$`, func() error {
		execsetup.assetsEngine.SetStrict()
		return nil
	})
	s.Step(`^set assets to permissive$`, func() error {
		execsetup.assetsEngine.SetPermissive()
		return nil
	})

	s.Step(`^the parties should have the following position changes for market "([^)]+)":$`, func(mkt string, table *godog.Table) error {
		return steps.PartiesShouldHaveTheFollowingPositionStatus(execsetup.broker, mkt, table)
	})

	s.Step(`^the parties should have the following aggregated position changes for market "([^)]+)":$`, func(mkt string, table *godog.Table) error {
		return steps.PartiesShouldHaveTheFollowingPositionStatusAgg(execsetup.broker, mkt, table)
	})

	s.Step(`^the volume discount program tiers named "([^"]*)":$`, func(vdp string, table *godog.Table) error {
		return steps.VolumeDiscountProgramTiers(volumeDiscountTiers, vdp, table)
	})

	s.Step(`^the volume discount program:$`, func(table *godog.Table) error {
		return steps.VolumeDiscountProgram(execsetup.volumeDiscountProgram, volumeDiscountTiers, table)
	})

	s.Step(`^the party "([^"]*)" has the following discount factor "([^"]*)"$`, func(party, discountFactor string) error {
		return steps.PartyHasTheFollowingDiscountFactor(party, discountFactor, execsetup.volumeDiscountProgram)
	})

	s.Step(`^the party "([^"]*)" has the following taker notional "([^"]*)"$`, func(party, notional string) error {
		return steps.PartyHasTheFollowingTakerNotional(party, notional, execsetup.volumeDiscountProgram)
	})

	s.Step(`^create the network treasury account for asset "([^"]*)"$`, func(asset string) error {
		return steps.CreateNetworkTreasuryAccount(execsetup.collateralEngine, asset)
	})

	s.Step(`the liquidation strategies:$`, func(table *godog.Table) error {
		return steps.TheLiquidationStrategies(marketConfig, table)
	})

	s.Step(`^clear trade events$`, func() error {
		return steps.ClearTradeEvents(execsetup.broker)
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
