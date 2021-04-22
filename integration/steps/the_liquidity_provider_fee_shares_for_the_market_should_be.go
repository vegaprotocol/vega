package steps

import (
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/execution"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/cucumber/godog/gherkin"
)

func TheLiquidityProviderFeeSharesForTheMarketShouldBe(engine *execution.Engine, marketID string, table *gherkin.DataTable) error {
	marketData, err := engine.GetMarketData(marketID)
	if err != nil {
		return errMarketDataNotFound(marketID, err)
	}

	for _, row := range TableWrapper(*table).Parse() {
		expected := types.LiquidityProviderFeeShare{
			Party:                 row.MustStr("party"),
			EquityLikeShare:       row.MustStr("equity like share"),
			AverageEntryValuation: row.MustStr("average entry valuation"),
		}

		var found bool
		var got []types.LiquidityProviderFeeShare
		for _, v := range marketData.LiquidityProviderFeeShare {
			got = append(got, *v)
			if v.Party == expected.Party &&
				// ok it's trick not pretty here, but the actual numbers are
				// something like 0.6666666666666, and I don't want to create
				// a float, so just checking if they start the same should be fine...
				strings.HasPrefix(v.EquityLikeShare, expected.EquityLikeShare) &&
				strings.HasPrefix(v.AverageEntryValuation, expected.AverageEntryValuation) {
				found = true
			}
		}

		if !found {
			return errMissingLPFeeShare(marketID, expected, got)
		}
	}

	return nil
}

func errMissingLPFeeShare(market string, expected types.LiquidityProviderFeeShare, got []types.LiquidityProviderFeeShare) error {
	return fmt.Errorf("missing fee share for market %s got %#v, want %#v", market, expected, got)
}
