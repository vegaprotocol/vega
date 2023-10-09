// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package steps

import (
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/libs/num"
	proto "code.vegaprotocol.io/vega/protos/vega"

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
