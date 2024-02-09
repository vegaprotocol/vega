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
	"fmt"

	"code.vegaprotocol.io/vega/core/integration/stubs"

	"github.com/cucumber/godog"
)

func TheIcebergOrdersShouldHaveTheFollowingStates(broker *stubs.BrokerStub, table *godog.Table) error {
	data := broker.GetOrderEvents()

	for _, row := range parseIcebergOrdersStatesTable(table) {
		party := row.MustStr("party")
		marketID := row.MustStr("market id")
		side := row.MustSide("side")
		visible := row.MustU64("visible volume")
		price := row.MustU64("price")
		status := row.MustOrderStatus("status")
		ref, hasRef := row.StrB("reference")
		reservedRemaining := row.MustU64("reserved volume")

		match := false
		for _, e := range data {
			o := e.Order()
			if hasRef {
				if ref != o.Reference {
					continue
				}
				if o.PartyId == party && o.Status == status && o.MarketId == marketID && o.Side == side {
					if o.Remaining != visible || stringToU64(o.Price) != price {
						return fmt.Errorf("side: %s, expected price: %v actual: %v, expected volume: %v, actual %v", side.String(), price, o.Price, visible, o.Size)
					}
				}
			}
			if o.PartyId != party || o.Status != status || o.MarketId != marketID || o.Side != side || o.Remaining != visible || stringToU64(o.Price) != price {
				continue
			}

			if o.IcebergOrder == nil || o.IcebergOrder.ReservedRemaining != reservedRemaining {
				continue
			}

			match = true
			break
		}
		if !match {
			return errOrderEventsNotFound(party, marketID, side, visible, price)
		}
	}
	return nil
}

func parseIcebergOrdersStatesTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"market id",
		"side",
		"visible volume",
		"price",
		"status",
		"reserved volume",
	}, []string{"reference"})
}
