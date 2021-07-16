package steps

import (
	"code.vegaprotocol.io/vega/integration/stubs"
	"github.com/cucumber/godog/gherkin"
)

func TheOrdersShouldHaveTheFollowingStates(broker *stubs.BrokerStub, table *gherkin.DataTable) error {
	data := broker.GetOrderEvents()

	for _, row := range parseOrdersStatesTable(table) {
		party := row.MustStr("party")
		marketID := row.MustStr("market id")
		side := row.MustSide("side")
		size := row.MustU64("volume")
		price := row.MustU64("price")
		status := row.MustOrderStatus("status")

		match := false
		for _, e := range data {
			o := e.Order()
			if o.PartyId != party || o.Status != status || o.MarketId != marketID || o.Side != side || o.Size != size || o.Price != price {
				continue
			}
			match = true
			break
		}
		if !match {
			return errOrderEventsNotFound(party, marketID, side, size, price)
		}
	}
	return nil
}

func parseOrdersStatesTable(table *gherkin.DataTable) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"market id",
		"side",
		"volume",
		"price",
		"status",
	}, []string{})
}
