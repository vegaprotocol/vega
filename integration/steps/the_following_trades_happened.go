package steps

import (
	"fmt"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/cucumber/godog/gherkin"
)

func TheFollowingTradesHappened(
	broker interface{ GetTrades() []types.Trade },
	table *gherkin.DataTable,
) error {
	var err error
	for _, row := range TableWrapper(*table).Parse() {
		var (
			buyer        = row.Str("buyer")
			seller       = row.Str("seller")
			price, perr  = row.U64("price")
			volume, verr = row.U64("volume")
		)
		panicW(perr)
		panicW(verr)

		data := broker.GetTrades()
		var found bool
		for _, v := range data {
			if v.Buyer == buyer && v.Seller == seller && v.Price == price && v.Size == volume {
				found = true
			}
		}

		if !found {
			return fmt.Errorf(
				"expecting trade was missing: buyer(%v), seller(%v), price(%v), volume(%v)",
				buyer, seller, price, volume,
			)
		}
	}

	return err
}
