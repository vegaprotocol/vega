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
	types "code.vegaprotocol.io/vega/protos/vega"

	"github.com/cucumber/godog"
)

func TheFollowingOrdersShouldBeRejected(broker *stubs.BrokerStub, table *godog.Table) error {
	var orderNotRejected []string
	count := len(table.Rows) - 1
	for _, row := range parseRejectedOrdersTable(table) {
		party := row.MustStr("party")
		marketID := row.MustStr("market id")
		reason := row.MustStr("reason")

		data := broker.GetOrderEvents()
		for _, o := range data {
			v := o.Order()
			if v.PartyId == party && v.MarketId == marketID {
				if v.Status == types.Order_STATUS_REJECTED && v.Reason.String() == reason {
					count--
					continue
				}
				orderNotRejected = append(orderNotRejected, v.Reference)
			}
		}
	}

	if count > 0 {
		return errOrderNotRejected(orderNotRejected)
	}

	return nil
}

func errOrderNotRejected(orderNotRejected []string) error {
	return fmt.Errorf("orders with reference %v were not rejected", orderNotRejected)
}

func parseRejectedOrdersTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"market id",
		"reason",
	}, []string{})
}
