package steps

import (
	"fmt"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/integration/stubs"
)

func TheFollowingTradesHappened(
	broker *stubs.BrokerStub,
	table *gherkin.DataTable,
) error {
	var err error
	for _, row := range TableWrapper(*table).Parse() {
		buyer := row.Str("buyer")
		seller := row.Str("seller")
		price := row.U64("price")
		volume := row.U64("volume")

		data := broker.GetTrades()
		var found bool
		for _, v := range data {
			if v.Buyer == buyer && v.Seller == seller && v.Price == price && v.Size == volume {
				found = true
			}
		}

		if !found {
			return errMissingTrade(buyer, seller, price, volume)
		}
	}

	return err
}

func errMissingTrade(buyer string, seller string, price uint64, volume uint64) error {
	return fmt.Errorf(
		"expecting trade was missing: buyer(%v), seller(%v), price(%v), volume(%v)",
		buyer, seller, price, volume,
	)
}
