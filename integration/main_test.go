package core_test

import (
	"flag"
	"os"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/integration/steps"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/cucumber/godog/gherkin"
)

var (
	gdOpts = godog.Options{
		Output: colors.Colored(os.Stdout),
		Format: "progress",
	}
)

func init() {
	godog.BindFlags("godog.", flag.CommandLine, &gdOpts)
}

func TestMain(m *testing.M) {
	flag.Parse()
	gdOpts.Paths = flag.Args()

	status := godog.RunWithOptions("godogs", func(s *godog.Suite) {
		FeatureContext(s)
	}, gdOpts)

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}

func FeatureContext(s *godog.Suite) {
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

	s.Step(`^"([^"]*)" has only one margin account per market$`, func(owner string) error {
		return steps.TraderHasOnlyOneMarginAccountPerMarket(execsetup.broker, owner)
	})
	s.Step(`^"([^"]*)" withdraws "([^"]*)" from the "([^"]*)" account$`, func(owner, amountStr, asset string) error {
		return steps.TraderWithdrawsFromAccount(execsetup.collateral, owner, amountStr, asset)
	})
	s.Step(`^The "([^"]*)" makes a deposit of "([^"]*)" into the "([^"]*)" account$`, func(owner, amountstr, asset string) error {
		return steps.DepositIntoAccount(execsetup.collateral, owner, amountstr, asset)
	})
	s.Step(`^"([^"]*)" general account for asset "([^"]*)" balance is "([^"]*)"$`, generalAccountForAssetBalanceIs)
	s.Step(`^"([^"]*)" has only one account per asset$`, func(owner string) error {
		return steps.TraderHasOnlyOneAccountPerAsset(execsetup.broker, owner)
	})
	s.Step(`^the traders make the following deposits on asset's general account:$`, func(table *gherkin.DataTable) error {
		return steps.TradersDepositAssets(execsetup.collateral, execsetup.broker, table)
	})
	s.Step(`^the execution engine have these markets:$`, func(table *gherkin.DataTable) error {
		markets := steps.TheMarkets(marketExpiry, table)

		t, _ := time.Parse("2006-01-02T15:04:05Z", marketStart)
		execsetup = getExecutionTestSetup(t, markets)

		// reset market start time and expiry for next run
		marketExpiry = defaultMarketExpiry
		marketStart = defaultMarketStart

		return nil
	})
	s.Step(`^traders place following orders:$`, func(table *gherkin.DataTable) error {
		return steps.TradersPlaceFollowingOrders(execsetup.engine, table)
	})
	s.Step(`^traders have the following account balances:$`, func(table *gherkin.DataTable) error {
		return steps.TradersHaveTheFollowingAccountBalances(execsetup.broker, table)
	})
	s.Step(`^Cumulated balance for all accounts is worth "([^"]*)"$`, func(rawAmount string) error {
		return steps.CumulatedBalanceForAllAccountsIsWorth(execsetup.broker, rawAmount)
	})
	s.Step(`^the settlement account balance is "([^"]*)" for the market "([^"]*)" before MTM$`, func(amountStr, market string) error {
		return steps.SettlementAccountBalanceIsForMarket(execsetup.broker, amountStr, market)
	})
	s.Step(`^the following transfers happened:$`, func(table *gherkin.DataTable) error {
		return steps.TheFollowingTransfersHappened(execsetup.broker, table)
	})
	s.Step(`^the insurance pool initial balance for the markets is "([^"]*)":$`, theInsurancePoolInitialBalanceForTheMarketsIs)
	s.Step(`^the insurance pool balance is "([^"]*)" for the market "([^"]*)"$`, func(amountStr, market string) error {
		return steps.TheInsurancePoolBalanceIsForTheMarket(execsetup.broker, amountStr, market)
	})
	s.Step(`^the markets start on "([^"]*)" and expire on "([^"]*)"$`, func(startDate, expiryDate string) error {
		start, expiry, err := steps.MarketsStartOnAndExpireOn(startDate, expiryDate)
		if err == nil {
			marketExpiry = expiry
			marketStart = start
		}
		return err
	})
	s.Step(`^time is updated to "([^"]*)"$`, func(rawTime string) error {
		return steps.TimeIsUpdatedTo(execsetup.timesvc, rawTime)
	})
	s.Step(`^traders cannot place the following orders anymore:$`, tradersCannotPlaceTheFollowingOrdersAnymore)
	s.Step(`^the margins levels for the traders are:$`, func(table *gherkin.DataTable) error {
		return steps.TheMarginsLevelsForTheTradersAre(execsetup.broker, table)
	})
	s.Step(`^traders place the following invalid orders:$`, func(table *gherkin.DataTable) error {
		return steps.TradersPlaceFollowingInvalidOrders(execsetup.engine, table)
	})
	s.Step(`^the following orders are rejected:$`, theFollowingOrdersAreRejected)
	s.Step(`^traders place following orders with references:$`, func(table *gherkin.DataTable) error {
		return steps.TradersPlaceFollowingOrdersWithReferences(execsetup.engine, table)
	})
	s.Step(`^missing traders place following orders with references:$`, missingTradersPlaceFollowingOrdersWithReferences)
	s.Step(`^traders cancel the following orders:$`, func(table *gherkin.DataTable) error {
		return steps.TradersCancelTheFollowingOrders(execsetup.broker, execsetup.engine, table)
	})
	s.Step(`^traders attempt to cancel the following filled orders:$`, func(table *gherkin.DataTable) error {
		return steps.TradersAttemptToCancelTheFollowingFilledOrders(execsetup.broker, execsetup.engine, table)
	})
	s.Step(`^missing traders cancels the following orders reference:$`, missingTradersCancelsTheFollowingOrdersReference)
	s.Step(`^traders have the following profit and loss:$`, func(table *gherkin.DataTable) error {
		return steps.TradersHaveTheFollowingProfitAndLoss(execsetup.positionPlugin, table)
	})
	s.Step(`^the mark price for the market "([^"]*)" is "([^"]*)"$`, func(market, markPriceStr string) error {
		return steps.TheMarkPriceForTheMarketIs(execsetup.engine, market, markPriceStr)
	})
	s.Step(`^the trading mode for the market "([^"]*)" is "([^"]*)"$`, func(marketID, tradingMode string) error {
		return steps.TradingModeForMarketIs(execsetup.engine, marketID, tradingMode)
	})
	s.Step(`^the following network trades happened:$`, theFollowingNetworkTradesHappened)
	s.Step(`^traders amends the following orders reference:$`, tradersAmendsTheFollowingOrdersReference)
	s.Step(`^the following trades happened:$`, func(table *gherkin.DataTable) error {
		return steps.TheFollowingTradesHappened(execsetup.broker, table)
	})
	s.Step(`^verify the status of the order reference:$`, verifyTheStatusOfTheOrderReference)
	s.Step(`^executed trades:$`, executedTrades)
	s.Step(`^clear order events$`, clearOrderEvents)
	s.Step(`^traders place pegged orders:$`, tradersPlacePeggedOrders)
	s.Step(`^I see the following order events:$`, seeTheFollowingOrderEvents)
	s.Step(`^clear order events by reference:$`, clearOrdersByRef)
	s.Step(`^clear transfer events$`, clearTransferEvents)
	s.Step(`^traders cancel pegged orders and clear:$`, tradersCancelPeggedOrdersAndClear)
	s.Step(`^the trader submits LP:$`, submitLP)
	s.Step(`^I see the LP events:$`, seeLPEvents)
	s.Step(`^the opening auction period for market "([^"]+)" ends$`, theOpeningAuctionPeriodEnds)
	s.Step(`^oracles broadcast data signed with "([^"]*)":$`, func(pubKeys string, properties *gherkin.DataTable) error {
		return steps.OraclesBroadcastDataSignedWithKeys(execsetup.oracleEngine, pubKeys, properties)
	})

	// Debug steps
	s.Step(`^debug transfers$`, func() error {
		return steps.DebugTransfers(execsetup.broker, execsetup.log)
	})
	s.Step(`^debug trades$`, func() error {
		return steps.DebugTrades(execsetup.broker, execsetup.log)
	})
	s.Step(`^debug orders$`, func() error {
		return steps.DebugOrders(execsetup.broker, execsetup.log)
	})
}
