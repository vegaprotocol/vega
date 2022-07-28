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

	"code.vegaprotocol.io/vega/types"
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
