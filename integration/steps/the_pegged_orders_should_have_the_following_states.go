package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
	"code.vegaprotocol.io/vega/proto"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/cucumber/godog/gherkin"
)

func ThePeggedOrdersShouldHaveTheFollowingStates(broker *stubs.BrokerStub, table *gherkin.DataTable) error {
	data := broker.GetOrderEvents()

	for _, r := range parsePeggedOrdersStatesTable(table) {
		row := peggedOrdersStatusAssertionRow{row: r}
		trader := row.trader()
		marketID := row.marketID()
		side := row.side()
		volume := row.volume()
		reference := row.reference()
		offset := row.offset()
		price := row.price()
		status := row.status()

		var ord types.Order
		match := false
		for _, e := range data {
			o := e.Order()
			if o.PartyId != trader || o.Status != status || o.MarketId != marketID || o.Side != side || o.Size != volume {
				if o.Price != price {
					continue
				}
				ord = *o
			}
			if o.PeggedOrder == nil {
				continue
			}
			if o.PeggedOrder.Offset != offset || o.PeggedOrder.Reference != reference {
				continue
			}
			match = true
			break
		}
		if !match {
			err := errOrderEventsNotFound(trader, marketID, side, volume, price)
			return fmt.Errorf("%v - order: %s", err, ord.String())
		}
	}
	return nil
}

func parsePeggedOrdersStatesTable(table *gherkin.DataTable) []RowWrapper {
	return TableWrapper(*table).StrictParse(
		"trader",
		"market id",
		"side",
		"volume",
		"reference",
		"offset",
		"price",
		"status",
	)
}

type peggedOrdersStatusAssertionRow struct {
	row RowWrapper
}

func (r peggedOrdersStatusAssertionRow) trader() string {
	return r.row.MustStr("trader")
}

func (r peggedOrdersStatusAssertionRow) marketID() string {
	return r.row.MustStr("market id")
}

func (r peggedOrdersStatusAssertionRow) side() proto.Side {
	return r.row.MustSide("side")
}

func (r peggedOrdersStatusAssertionRow) volume() uint64 {
	return r.row.MustU64("volume")
}

func (r peggedOrdersStatusAssertionRow) reference() proto.PeggedReference {
	return r.row.MustPeggedReference("reference")
}

func (r peggedOrdersStatusAssertionRow) offset() int64 {
	return r.row.MustI64("offset")
}

func (r peggedOrdersStatusAssertionRow) price() uint64 {
	return r.row.MustU64("price")
}

func (r peggedOrdersStatusAssertionRow) status() proto.Order_Status {
	return r.row.MustOrderStatus("status")
}
