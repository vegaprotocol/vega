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
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"github.com/cucumber/godog"
)

func TheStopOrdersShouldHaveTheFollowingStates(broker *stubs.BrokerStub, table *godog.Table) error {
	data := broker.GetStopOrderEvents()

	for _, row := range parseStopOrdersStatesTable(table) {
		party := row.MustStr("party")
		marketID := row.MustStr("market id")
		status := row.MustStopOrderStatus("status")
		ref, hasRef := row.StrB("reference")

		match := false
		for _, e := range data {
			o := e.StopOrder()
			if hasRef {
				if ref != o.Submission.Reference {
					continue
				}
			}
			if o.StopOrder.PartyId != party || o.StopOrder.Status != status || o.StopOrder.MarketId != marketID {
				continue
			}
			match = true
			break
		}
		if !match {
			return errStopOrderEventsNotFound(party, marketID, status)
		}
	}
	return nil
}

func parseStopOrdersStatesTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"market id",
		"status",
	}, []string{"reference"})
}
