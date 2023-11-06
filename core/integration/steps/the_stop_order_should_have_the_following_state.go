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
