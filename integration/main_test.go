package core_test

import (
	"flag"
	"os"
	"testing"

	"github.com/DATA-DOG/godog"
	"github.com/DATA-DOG/godog/colors"
	"github.com/DATA-DOG/godog/gherkin"
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

	s.Step(`^"([^"]*)" have only on margin account per market$`, haveOnlyOnMarginAccountPerMarket)
	s.Step(`^The "([^"]*)" withdraw "([^"]*)" from the "([^"]*)" account$`, theWithdrawFromTheAccount)
	s.Step(`^The "([^"]*)" makes a deposit of "([^"]*)" into the "([^"]*)" account$`, theMakesADepositOfIntoTheAccount)
	s.Step(`^"([^"]*)" general account for asset "([^"]*)" balance is "([^"]*)"$`, generalAccountForAssetBalanceIs)
	s.Step(`^"([^"]*)" have only one account per asset$`, haveOnlyOneAccountPerAsset)
	s.Step(`^the following traders:$`, theFollowingTraders)
	s.Step(`^I Expect the traders to have new general account:$`, iExpectTheTradersToHaveNewGeneralAccount)
	s.Step(`^"([^"]*)" general accounts balance is "([^"]*)"$`, generalAccountsBalanceIs)

	s.Step(`^the market:$`, theMarket)
	s.Step(`^the system accounts:$`, theSystemAccounts)
	s.Step(`^traders have the following state:$`, tradersHaveTheFollowingState)
	s.Step(`^the following orders:$`, theFollowingOrders)
	s.Step(`^I place the following orders:$`, theFollowingOrders)
	s.Step(`^I expect the trader to have a margin liability:$`, tradersLiability)
	s.Step(`^"([^"]*)" has not been added to the market$`, hasNotBeenAddedToTheMarket)
	s.Step(`^the mark price is "([^"]+)"$`, theMarkPriceIs)
	s.Step(`^the executon engine have these markets:$`, theExecutonEngineHaveTheseMarkets)
	s.Step(`^traders place following orders:$`, tradersPlaceFollowingOrders)
	s.Step(`^I expect the trader to have a margin:$`, iExpectTheTraderToHaveAMargin)
	s.Step(`^a transfer of "([^"]*)" is made to the settlement account$`, aTransferOfIsMadeToTheSettlementAccount)
}
