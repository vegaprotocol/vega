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

	types "code.vegaprotocol.io/vega/protos/vega"

	"github.com/cucumber/godog"
)

func TheLiquidityProviderFeeSharesForTheMarketShouldBe(engine Execution, marketID string, table *godog.Table) error {
	marketData, err := engine.GetMarketData(marketID)
	if err != nil {
		return errMarketDataNotFound(marketID, err)
	}

	for _, row := range parseLiquidityFeeSharesTable(table) {
		expected := types.LiquidityProviderFeeShare{
			Party:                 row.MustStr("party"),
			EquityLikeShare:       row.MustStr("equity like share"),
			AverageEntryValuation: row.MustStr("average entry valuation"),
		}

		var found bool
		var got []types.LiquidityProviderFeeShare
		for _, v := range marketData.LiquidityProviderFeeShare {
			got = append(got, *v)
			fmt.Printf("v: %v\n", v)
			if v.Party == expected.Party &&
				v.EquityLikeShare == expected.EquityLikeShare &&
				v.AverageEntryValuation == expected.AverageEntryValuation {
				found = true
			}
			if row.HasColumn("average score") &&
				v.AverageScore != row.MustStr("average score") {
				found = false
			}
			if row.HasColumn("virtual stake") &&
				v.VirtualStake != row.MustStr("virtual stake") {
				found = false
			}
			if found {
				break // No need to continue checking once a match is found
			}
		}

		if !found {
			return errMissingLPFeeShare(marketID, expected, got)
		}
	}

	return nil
}

func errMissingLPFeeShare(market string, expected types.LiquidityProviderFeeShare, got []types.LiquidityProviderFeeShare) error {
	return fmt.Errorf("missing fee share for market %s got %#v, want %#v", market, expected, got)
}

func parseLiquidityFeeSharesTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"equity like share",
		"average entry valuation",
	}, []string{
		"average score",
		"virtual stake",
	})
}
