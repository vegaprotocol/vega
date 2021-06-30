package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/types"
	"github.com/cucumber/godog/gherkin"
)

func ThePriceMonitoringBoundsForTheMarketShouldBe(engine *execution.Engine, marketID string, table *gherkin.DataTable) error {
	marketData, err := engine.GetMarketData(marketID)
	if err != nil {
		return errMarketDataNotFound(marketID, err)
	}

	for _, row := range parsePriceMonitoringBoundsTable(table) {
		expected := types.PriceMonitoringBounds{
			MinValidPrice: row.MustUint("min bound"),
			MaxValidPrice: row.MustUint("max bound"),
		}

		var found bool
		for _, v := range marketData.PriceMonitoringBounds {
			if v.MinValidPrice.EQ(expected.MinValidPrice) &&
				v.MaxValidPrice.EQ(expected.MaxValidPrice) {
				found = true
			}
		}

		if !found {
			return errMissingPriceMonitoringBounds(marketID, expected)
		}
	}

	return nil
}

func errMissingPriceMonitoringBounds(market string, expected types.PriceMonitoringBounds) error {
	return fmt.Errorf("missing price monitoring bounds for market %s  want %v", market, expected)
}

func parsePriceMonitoringBoundsTable(table *gherkin.DataTable) []RowWrapper {
	return StrictParseTable(table, []string{
		"min bound",
		"max bound",
	}, []string{})
}
