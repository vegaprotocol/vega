// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"github.com/cucumber/godog"
)

func TheOrdersShouldHaveTheFollowingStates(broker *stubs.BrokerStub, table *godog.Table) error {
	data := broker.GetOrderEvents()

	for _, row := range parseOrdersStatesTable(table) {
		party := row.MustStr("party")
		marketID := row.MustStr("market id")
		side := row.MustSide("side")
		size := row.MustU64("volume")
		price := row.MustU64("price")
		status := row.MustOrderStatus("status")
		ref, hasRef := row.StrB("reference")

		match := false
		for _, e := range data {
			o := e.Order()
			if hasRef {
				if ref != o.Reference {
					continue
				}
				if o.PartyId == party && o.Status == status && o.MarketId == marketID && o.Side == side {
					if o.Size != size || stringToU64(o.Price) != price {
						return fmt.Errorf("side: %s, expected price: %v actual: %v, expected volume: %v, actual %v", side.String(), price, o.Price, size, o.Size)
					}
				}
			}
			if o.PartyId != party || o.Status != status || o.MarketId != marketID || o.Side != side || o.Size != size || stringToU64(o.Price) != price {
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
		"status",
	}, []string{"reference"})
}
