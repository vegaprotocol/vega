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

func TheOrderBookOfMarketShouldHaveTheFollowingVolumes(broker *stubs.BrokerStub, marketID string, table *godog.Table) error {
	for _, row := range parseOrderBookTable(table) {
		volume := row.MustU64("volume")
		price := row.MustU64("price")
		side := row.MustSide("side")

		sell, buy := broker.GetBookDepth(marketID)
		if side == types.Side_SIDE_SELL {
			vol := sell[u64ToS(price)]
			if vol != volume {
				return fmt.Errorf("invalid volume(%d) at price(%d) and side(%s) for market(%v), expected(%v)", vol, price, side.String(), marketID, volume)
			}
			continue
		}
		vol := buy[u64ToS(price)]
		if vol != volume {
			return fmt.Errorf("invalid volume(%d) at price(%d) and side(%s) for market(%v), expected(%v)", vol, price, side.String(), marketID, volume)
		}
	}
	return nil
}

func parseOrderBookTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"volume",
		"price",
		"side",
	}, []string{})
}
