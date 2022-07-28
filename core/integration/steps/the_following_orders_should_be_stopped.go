// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"github.com/cucumber/godog"
)

func TheFollowingOrdersShouldBeStopped(broker *stubs.BrokerStub, table *godog.Table) error {
	var orderNotStopped []string
	count := len(table.Rows) - 1
	for _, row := range parseStoppedOrdersTable(table) {
		party := row.MustStr("party")
		marketID := row.MustStr("market id")
		reason := row.MustStr("reason")

		data := broker.GetOrderEvents()
		for _, o := range data {
			v := o.Order()
			if v.PartyId == party && v.MarketId == marketID {
				if v.Status == types.Order_STATUS_STOPPED && v.Reason.String() == reason {
					count -= 1
					continue
				}
				orderNotStopped = append(orderNotStopped, v.Reference)
			}
		}
	}

	if count > 0 {
		return errOrderNotStopped(orderNotStopped)
	}

	return nil
}

func errOrderNotStopped(orderNotRejected []string) error {
	return fmt.Errorf("orders with reference %v were not stopped", orderNotRejected)
}

func parseStoppedOrdersTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"market id",
		"reason",
	}, []string{})
}
