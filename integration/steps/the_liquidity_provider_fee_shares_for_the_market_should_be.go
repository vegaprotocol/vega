package steps

import (
	"fmt"

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
		for _, v := range marketData.LiquidityProviderFeeShare {
			if v.Party == expected.Party &&
				v.EquityLikeShare == expected.EquityLikeShare &&
				v.AverageEntryValuation == expected.AverageEntryValuation {
				found = true
			}
		}

		if !found {
			return errMissingLPFeeShare(marketID, expected)
		}
	}

	return nil
}

func errMissingLPFeeShare(market string, expected types.LiquidityProviderFeeShare) error {
	return fmt.Errorf("missing fee share for market %s got %v, want %v", market, expected)
}
