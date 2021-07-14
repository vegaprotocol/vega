package core_test

import (
	"context"
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/integration/steps"
	"code.vegaprotocol.io/vega/types/num"
)

var (
	gdOpts = godog.Options{
		Output: colors.Colored(os.Stdout),
		Format: "progress",
	}

	features string
)

func init() {
	godog.BindFlags("godog.", flag.CommandLine, &gdOpts)
	flag.StringVar(&features, "features", "", "a coma separated list of paths to the feature files")
}

func TestMain(m *testing.M) {
	flag.Parse()
	gdOpts.Paths = flag.Args()

	if features != "" {
		paths := strings.Split(features, ",")
		gdOpts.Paths = paths
	}

	status := godog.RunWithOptions("godogs", func(s *godog.Suite) {
		FeatureContext(s)
	}, gdOpts)

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}

func FeatureContext(s *godog.Suite) {
	s.BeforeScenario(func(_ interface{}) {
		execsetup = newExecutionTestSetup()
	})
	// each step changes the output from the reporter
	// so we know where a mock failed
	s.BeforeStep(func(step *gherkin.Step) {
		// rm any errors from previous step (if applies)
		reporter.err = nil
		reporter.step = step.Text
	})
	// if a mock assert failed, we're just setting an error here and crash out of the test here
	s.AfterStep(func(step *gherkin.Step, err error) {
		if err != nil && reporter.err == nil {
			reporter.err = err
		}
		if reporter.err != nil {
			reporter.Fatalf("some mock assertion failed: %v", reporter.err)
		}
	})

	// Market steps
	s.Step(`the simple risk model named "([^"]*)":$`, func(name string, table *gherkin.DataTable) error {
		return steps.TheSimpleRiskModel(marketConfig, name, table)
	})
	s.Step(`the log normal risk model named "([^"]*)":$`, func(name string, table *gherkin.DataTable) error {
		return steps.TheLogNormalRiskModel(marketConfig, name, table)
	})
	s.Step(`the fees configuration named "([^"]*)":$`, func(name string, table *gherkin.DataTable) error {
		return steps.TheFeesConfiguration(marketConfig, name, table)
	})
	s.Step(`the oracle spec filtering data from "([^"]*)" named "([^"]*)":$`, func(pubKeys, name string, table *gherkin.DataTable) error {
		return steps.TheOracleSpec(marketConfig, name, pubKeys, table)
	})
	s.Step(`the price monitoring updated every "([^"]*)" seconds named "([^"]*)":$`, func(updateFrequency, name string, table *gherkin.DataTable) error {
		return steps.ThePriceMonitoring(marketConfig, name, updateFrequency, table)
	})
	s.Step(`the margin calculator named "([^"]*)":$`, func(name string, table *gherkin.DataTable) error {
		return steps.TheMarginCalculator(marketConfig, name, table)
	})
	s.Step(`^the markets:$`, func(table *gherkin.DataTable) error {
		markets, err := steps.TheMarkets(marketConfig, execsetup.executionEngine, execsetup.collateralEngine, table)
		execsetup.markets = markets
		return err
	})

	// Other steps
	s.Step(`^the initial insurance pool balance is "([^"]*)" for the markets:$`, func(amountstr string) error {
		//		amount, _ := strconv.ParseUint(amountstr, 10, 0)
		amount, _ := num.UintFromString(amountstr, 10)
		for _, mkt := range execsetup.markets {
			asset, _ := mkt.GetAsset()
			marketInsuranceAccount, err := execsetup.collateralEngine.GetMarketInsurancePoolAccount(mkt.Id, asset)
			if err != nil {
				return err
			}
			if err := execsetup.collateralEngine.IncrementBalance(context.Background(), marketInsuranceAccount.ID, amount); err != nil {
				return err
			}
		}
		return nil
	})
	s.Step(`^the initial insurance pool balance is "([^"]*)" for the asset:$`, func(amountstr string) error {
		amount, _ := num.UintFromString(amountstr, 10)
		for _, mkt := range execsetup.markets {
			asset, _ := mkt.GetAsset()
			assetInsuranceAccount := execsetup.collateralEngine.GetAssetInsurancePoolAccount(asset)
			if err := execsetup.collateralEngine.IncrementBalance(context.Background(), assetInsuranceAccount.ID, amount); err != nil {
				return err
			}
		}
		return nil
	})

	s.Step(`^the following network parameters are set:$`, func(table *gherkin.DataTable) error {
		return steps.TheFollowingNetworkParametersAreSet(execsetup.netParams, table)
	})
	s.Step(`^time is updated to "([^"]*)"$`, func(rawTime string) error {
		steps.TimeIsUpdatedTo(execsetup.timeService, rawTime)
		return nil
	})
	s.Step(`^the parties cancel the following orders:$`, func(table *gherkin.DataTable) error {
		return steps.PartiesCancelTheFollowingOrders(execsetup.broker, execsetup.executionEngine, table)
	})
	s.Step(`^the parties cancel all their orders for the markets:$`, func(table *gherkin.DataTable) error {
		return steps.PartiesCancelAllTheirOrdersForTheMarkets(execsetup.broker, execsetup.executionEngine, table)
	})
	s.Step(`^the parties amend the following orders:$`, func(table *gherkin.DataTable) error {
		return steps.PartiesAmendTheFollowingOrders(execsetup.broker, execsetup.executionEngine, table)
	})
	s.Step(`^the parties place the following pegged orders:$`, func(table *gherkin.DataTable) error {
		return steps.PartiesPlaceTheFollowingPeggedOrders(execsetup.executionEngine, table)
	})
	s.Step(`^the parties deposit on asset's general account the following amount:$`, func(table *gherkin.DataTable) error {
		return steps.PartiesDepositTheFollowingAssets(execsetup.collateralEngine, execsetup.broker, table)
	})
	s.Step(`^the parties withdraw the following assets:$`, func(table *gherkin.DataTable) error {
		return steps.PartiesWithdrawTheFollowingAssets(execsetup.collateralEngine, table)
	})
	s.Step(`^the parties place the following orders:$`, func(table *gherkin.DataTable) error {
		return steps.PartiesPlaceTheFollowingOrders(execsetup.executionEngine, table)
	})
	s.Step(`^the parties submit the following liquidity provision:$`, func(table *gherkin.DataTable) error {
		return steps.PartiesSubmitLiquidityProvision(execsetup.executionEngine, table)
	})
	s.Step(`^the opening auction period ends for market "([^"]+)"$`, func(marketID string) error {
		return steps.MarketOpeningAuctionPeriodEnds(execsetup.timeService, execsetup.markets, marketID)
	})
	s.Step(`^the oracles broadcast data signed with "([^"]*)":$`, func(pubKeys string, properties *gherkin.DataTable) error {
		return steps.OraclesBroadcastDataSignedWithKeys(execsetup.oracleEngine, pubKeys, properties)
	})

	// block time stuff
	s.Step(`^the average block duration is "([^"]+)" with variance "([^"]+)"$`, func(block, variance string) error {
		return steps.TheAverageBlockDurationWithVariance(execsetup.block, block, variance)
	})
	s.Step(`^the average block duration is "([^"]+)"$`, func(blockTime string) error {
		return steps.TheAverageBlockDurationIs(execsetup.block, blockTime)
	})

	s.Step(`the network moves ahead "([^"]+)" blocks`, func(blocks string) error {
		return steps.TheNetworkMovesAheadNBlocks(execsetup.block, execsetup.timeService, blocks)
	})
	s.Step(`the network moves ahead "([^"]+)" with block duration of "([^"]+)"`, func(total, block string) error {
		return steps.TheNetworkMovesAheadDurationWithBlocks(execsetup.block, execsetup.timeService, total, block)
	})

	// Assertion steps
	s.Step(`^the parties should have the following account balances:$`, func(table *gherkin.DataTable) error {
		return steps.PartiesShouldHaveTheFollowingAccountBalances(execsetup.broker, table)
	})
	s.Step(`^the parties should have the following margin levels:$`, func(table *gherkin.DataTable) error {
		return steps.ThePartiesShouldHaveTheFollowingMarginLevels(execsetup.broker, table)
	})
	s.Step(`^the parties should have the following profit and loss:$`, func(table *gherkin.DataTable) error {
		return steps.PartiesHaveTheFollowingProfitAndLoss(execsetup.positionPlugin, table)
	})
	s.Step(`^the order book should have the following volumes for market "([^"]*)":$`, func(marketID string, table *gherkin.DataTable) error {
		return steps.TheOrderBookOfMarketShouldHaveTheFollowingVolumes(execsetup.broker, marketID, table)
	})
	s.Step(`^the orders should have the following status:$`, func(table *gherkin.DataTable) error {
		return steps.TheOrdersShouldHaveTheFollowingStatus(execsetup.broker, table)
	})
	s.Step(`^the orders should have the following states:$`, func(table *gherkin.DataTable) error {
		return steps.TheOrdersShouldHaveTheFollowingStates(execsetup.broker, table)
	})
	s.Step(`^the pegged orders should have the following states:$`, func(table *gherkin.DataTable) error {
		return steps.ThePeggedOrdersShouldHaveTheFollowingStates(execsetup.broker, table)
	})
	s.Step(`^the following orders should be rejected:$`, func(table *gherkin.DataTable) error {
		return steps.TheFollowingOrdersShouldBeRejected(execsetup.broker, table)
	})
	s.Step(`^the following orders should be stopped:$`, func(table *gherkin.DataTable) error {
		return steps.TheFollowingOrdersShouldBeStopped(execsetup.broker, table)
	})
	s.Step(`^"([^"]*)" should have general account balance of "([^"]*)" for asset "([^"]*)"$`, func(party, balance, asset string) error {
		return steps.PartyShouldHaveGeneralAccountBalanceForAsset(execsetup.broker, party, asset, balance)
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
	s.Step(`^the following network trades should be executed:$`, func(table *gherkin.DataTable) error {
		return steps.TheFollowingNetworkTradesShouldBeExecuted(execsetup.broker, table)
	})
	s.Step(`^the following trades should be executed:$`, func(table *gherkin.DataTable) error {
		return steps.TheFollowingTradesShouldBeExecuted(execsetup.broker, table)
	})
	s.Step(`^the trading mode should be "([^"]*)" for the market "([^"]*)"$`, func(tradingMode, marketID string) error {
		return steps.TheTradingModeShouldBeForMarket(execsetup.executionEngine, marketID, tradingMode)
	})
	s.Step(`^the insurance pool balance should be "([^"]*)" for the market "([^"]*)"$`, func(rawAmount, marketID string) error {
		return steps.TheInsurancePoolBalanceShouldBeForTheMarket(execsetup.broker, rawAmount, marketID)
	})
	s.Step(`^the insurance pool balance should be "([^"]*)" for the asset "([^"]*)"$`, func(rawAmount, asset string) error {
		return steps.TheInsurancePoolBalanceShouldBeForTheAsset(execsetup.broker, rawAmount, asset)
	})
	s.Step(`^the following transfers should happen:$`, func(table *gherkin.DataTable) error {
		return steps.TheFollowingTransfersShouldHappen(execsetup.broker, table)
	})
	s.Step(`^the mark price should be "([^"]*)" for the market "([^"]*)"$`, func(rawMarkPrice, marketID string) error {
		return steps.TheMarkPriceForTheMarketIs(execsetup.executionEngine, marketID, rawMarkPrice)
	})
	s.Step(`^the liquidity provisions should have the following states:$`, func(table *gherkin.DataTable) error {
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
	s.Step(`^the liquidity provider fee shares for the market "([^"]*)" should be:$`, func(marketID string, table *gherkin.DataTable) error {
		return steps.TheLiquidityProviderFeeSharesForTheMarketShouldBe(execsetup.executionEngine, marketID, table)
	})
	s.Step(`^the price monitoring bounds for the market "([^"]*)" should be:$`, func(marketID string, table *gherkin.DataTable) error {
		return steps.ThePriceMonitoringBoundsForTheMarketShouldBe(execsetup.executionEngine, marketID, table)
	})
	s.Step(`^the accumulated liquidity fees should be "([^"]*)" for the market "([^"]*)"$`, func(amount, marketID string) error {
		return steps.TheAccumulatedLiquidityFeesShouldBeForTheMarket(execsetup.broker, amount, marketID)
	})
	s.Step(`^the liquidity fee factor should "([^"]*)" for the market "([^"]*)"$`, func(fee, marketID string) error {
		return steps.TheLiquidityFeeFactorShouldForTheMarket(execsetup.broker, fee, marketID)
	})
	s.Step(`^the market data for the market "([^"]+)" should be:$`, func(marketID string, table *gherkin.DataTable) error {
		return steps.TheMarketDataShouldBe(execsetup.executionEngine, marketID, table)
	})
	s.Step(`the auction ends with a traded volume of "([^"]+)" at a price of "([^"]+)"`, func(vol, price string) error {
		now := execsetup.timeService.GetTimeNow()
		return steps.TheAuctionTradedVolumeAndPriceShouldBe(execsetup.broker, vol, price, now)
	})

	// Debug steps
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

	// Event steps
	s.Step(`^clear order events by reference:$`, func(table *gherkin.DataTable) error {
		return steps.ClearOrdersByReference(execsetup.broker, table)
	})
	s.Step(`^clear transfer events$`, func() error {
		steps.ClearTransferEvents(execsetup.broker)
		return nil
	})
	s.Step(`^clear order events$`, func() error {
		steps.ClearOrderEvents(execsetup.broker)
		return nil
	})
}
