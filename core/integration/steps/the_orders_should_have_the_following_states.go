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

func TheOrdersShouldHaveTheFollowingStates(broker *stubs.BrokerStub, table *godog.Table) error {
	orderEvents := broker.GetOrderEvents()

	for _, row := range parseOrdersStatesTable(table) {
		party := row.MustStr("party")
		marketID := row.MustStr("market id")
		side := row.MustSide("side")
		size := row.MustU64("volume")
		price := row.MustU64("price")
		remaining := row.MustU64("remaining")
		status := row.MustOrderStatus("status")
		ref, hasRef := row.StrB("reference")

		match := false
		for i := len(orderEvents) - 1; i >= 0; i-- {
			e := orderEvents[i]
			o := e.Order()
			if hasRef {
				if ref != o.Reference {
					continue
				}
				if o.PartyId == party && o.Status == status && o.MarketId == marketID && o.Side == side {
					if o.Size != size || stringToU64(o.Price) != price || o.Remaining != remaining {
						return formatDiff(fmt.Sprintf("the order didn't match the expectation for reference %q", ref),
							map[string]string{
								"size":      u64ToS(size),
								"price":     u64ToS(price),
								"remaining": u64ToS(remaining),
							},
							map[string]string{
								"size":      u64ToS(o.Size),
								"price":     o.Price,
								"remaining": u64ToS(o.Remaining),
							},
						)
					}
				}
			}
			if o.PartyId != party || o.Status != status || o.MarketId != marketID || o.Side != side || o.Size != size || stringToU64(o.Price) != price || o.Remaining != remaining {
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

func parseOrdersStatesTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"market id",
		"side",
		"volume",
		"price",
		"remaining",
		"status",
	}, []string{
		"reference",
	})
}
