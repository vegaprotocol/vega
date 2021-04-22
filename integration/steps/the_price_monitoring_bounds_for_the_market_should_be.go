package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/execution"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/cucumber/godog/gherkin"
)

func ThePriceMonitoringBoundsForTheMarketShouldBe(engine *execution.Engine, marketID string, table *gherkin.DataTable) error {
	marketData, err := engine.GetMarketData(marketID)
	if err != nil {
		return errMarketDataNotFound(marketID, err)
	}

	for _, row := range TableWrapper(*table).Parse() {
		expected := types.PriceMonitoringBounds{
			MinValidPrice: row.MustU64("min bound"),
			MaxValidPrice: row.MustU64("max bound"),
		}

		var found bool
		for _, v := range marketData.PriceMonitoringBounds {
			fmt.Printf("BOUND: %#v\n", v)

			if v.MinValidPrice == expected.MinValidPrice &&
				v.MaxValidPrice == expected.MaxValidPrice {
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
