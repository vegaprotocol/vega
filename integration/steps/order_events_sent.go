package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/cucumber/godog/gherkin"
)

func OrderEventsSent(broker *stubs.BrokerStub, table *gherkin.DataTable) error {
	data := broker.GetOrderEvents()
	for _, row := range TableWrapper(*table).Parse() {
		trader := row.MustStr("trader")
		marketID := row.MustStr("market id")
		side := row.MustSide("side")
		size := row.MustU64("volume")
		reference := row.MustPeggedReference("reference")
		offset := row.MustI64("offset")
		price := row.MustU64("price")
		status := row.MustOrderStatus("status")

		match := false
		for _, e := range data {
			o := e.Order()
			if o.PartyId != trader || o.Status != status || o.MarketId != marketID || o.Side != side || o.Size != size || o.Price != price {
				continue
			}
			// check if pegged:
			if offset != 0 {
				// nope
				if o.PeggedOrder == nil {
					continue
				}
				if o.PeggedOrder.Offset != offset || o.PeggedOrder.Reference != reference {
					continue
				}
				// this matches
			}
			// we've checked all fields and found this order to be a match
			match = true
			break
		}
		if !match {
			return errOrderEventsNotFound(trader, marketID, side, size, price)
		}
	}
	return nil
}

func errOrderEventsNotFound(trader, marketID string, side types.Side, size, price uint64) error {
	return fmt.Errorf("no matching order event found %v, %v, %v, %v, %v", trader, marketID, side.String(), size, price)
}
