package steps

import (
	"errors"

	"code.vegaprotocol.io/vega/integration/stubs"
	"github.com/cucumber/godog/gherkin"
)

func OrderEventsSent(broker *stubs.BrokerStub, table *gherkin.DataTable) error {
	data := broker.GetOrderEvents()
	for _, row := range TableWrapper(*table).Parse() {
		trader := row.Str("trader")
		marketID := row.Str("market id")
		side := row.Str("side")
		size := row.U64("volume")
		reference := row.Str("reference")
		offset := row.I64("offset")
		price := row.U64("price")
		status := row.OrderStatus("status")

		if trader == "trader" {
			continue
		}

		match := false
		for _, e := range data {
			o := e.Order()
			if o.PartyId != trader || o.Status != status || o.MarketId != marketID || o.Side.String() != side || o.Size != size || o.Price != price {
				// if o.MarketId != id || o.Side != side || o.Size != vol || o.Price != price {
				continue
			}
			// check if pegged:
			if offset != 0 {
				// nope
				if o.PeggedOrder == nil {
					continue
				}
				if o.PeggedOrder.Offset != offset || o.PeggedOrder.Reference.String() != reference {
					continue
				}
				// this matches
			}
			// we've checked all fields and found this order to be a match
			match = true
			break
		}
		if !match {
			return errOrderEventsNotFound()
		}
	}
	return nil
}

func errOrderEventsNotFound() error {
	return errors.New("no matching order event found")
}
