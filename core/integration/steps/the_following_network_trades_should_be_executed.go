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

func TheFollowingNetworkTradesShouldBeExecuted(broker *stubs.BrokerStub, table *godog.Table) error {
	for _, row := range parseNetworkTradesTable(table) {
		var (
			party         = row.MustStr("party")
			aggressorSide = row.MustSide("aggressor side")
			volume        = row.MustU64("volume")
		)

		ok := false
		data := broker.GetTrades()
		for _, v := range data {
			if (v.Buyer == party || v.Seller == party) && v.Aggressor == aggressorSide && v.Size == volume {
				ok = true
				break
			}
		}

		if !ok {
			return errTradeMissing(party, aggressorSide, volume)
		}
	}

	return nil
}

func errTradeMissing(party string, aggressorSide types.Side, volume uint64) error {
	return fmt.Errorf("expecting trade was missing: %v, %v, %v", party, aggressorSide, volume)
}

func parseNetworkTradesTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"aggressor side",
		"volume",
	}, []string{})
}
