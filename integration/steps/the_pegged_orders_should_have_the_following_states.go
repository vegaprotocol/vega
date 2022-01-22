package steps

import (
	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/integration/stubs"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/cucumber/godog"
)

func ThePeggedOrdersShouldHaveTheFollowingStates(broker *stubs.BrokerStub, table *godog.Table) error {
	data := broker.GetOrderEvents()

	for _, r := range parsePeggedOrdersStatesTable(table) {
		row := peggedOrdersStatusAssertionRow{row: r}
		party := row.party()
		marketID := row.marketID()
		side := row.side()
		volume := row.volume()
		reference := row.reference()
		offset := row.offset()
		price := row.price()
		status := row.status()

		match := false
		for _, e := range data {
			o := e.Order()
			if o.PartyId != party || o.Status != status || o.MarketId != marketID || o.Side != side || o.Size != volume || stringToU64(o.Price) != price {
				continue
			}
			if o.PeggedOrder == nil {
				continue
			}
			if o.PeggedOrder.Offset != offset.String() || o.PeggedOrder.Reference != reference {
				continue
			}
			match = true
			break
		}
		if !match {
			return errOrderEventsNotFound(party, marketID, side, volume, price)
		}
	}
	return nil
}

func parsePeggedOrdersStatesTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"market id",
		"side",
		"volume",
		"reference",
		"offset",
		"price",
		"status",
	}, []string{})
}

type peggedOrdersStatusAssertionRow struct {
	row RowWrapper
}

func (r peggedOrdersStatusAssertionRow) party() string {
	return r.row.MustStr("party")
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

func (r peggedOrdersStatusAssertionRow) offset() *num.Uint {
	return r.row.MustUint("offset")
}

func (r peggedOrdersStatusAssertionRow) price() uint64 {
	return r.row.MustU64("price")
}

func (r peggedOrdersStatusAssertionRow) status() proto.Order_Status {
	return r.row.MustOrderStatus("status")
}
