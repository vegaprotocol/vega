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

	"code.vegaprotocol.io/vega/core/types"

	"github.com/cucumber/godog"
)

func ThePriceMonitoringBoundsForTheMarketShouldBe(engine Execution, marketID string, table *godog.Table) error {
	marketData, err := engine.GetMarketData(marketID)
	if err != nil {
		return errMarketDataNotFound(marketID, err)
	}

	for _, row := range parsePriceMonitoringBoundsTable(table) {
		expected := types.PriceMonitoringBounds{
			MinValidPrice: row.MustUint("min bound"),
			MaxValidPrice: row.MustUint("max bound"),
		}

		var found bool
		for _, v := range marketData.PriceMonitoringBounds {
			if v.MinValidPrice.EQ(expected.MinValidPrice) &&
				v.MaxValidPrice.EQ(expected.MaxValidPrice) {
				found = true
			}
		}

		if !found {
			return errMissingPriceMonitoringBounds(marketID, expected)
		}
	}

	return nil
}

func errMissingPriceMonitoringBounds(market string, expected types.PriceMonitoringBounds) error {
	return fmt.Errorf("missing price monitoring bounds for market %s  want %v", market, expected)
}

func parsePriceMonitoringBoundsTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"min bound",
		"max bound",
	}, []string{})
}
