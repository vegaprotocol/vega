package steps

import (
	"fmt"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/integration/stubs"
)

func TheFollowingTradesWereExecuted(
	broker *stubs.BrokerStub,
	table *gherkin.DataTable,
) error {
	var err error
	for _, row := range TableWrapper(*table).Parse() {
		buyer := row.MustStr("buyer")
		seller := row.MustStr("seller")
		price := row.MustU64("price")
		size := row.MustU64("size")

		data := broker.GetTrades()
		var found bool
		for _, v := range data {
			if v.Buyer == buyer && v.Seller == seller && v.Price == price && v.Size == size {
				found = true
			}
		}

		if !found {
			return errMissingTrade(buyer, seller, price, size)
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
