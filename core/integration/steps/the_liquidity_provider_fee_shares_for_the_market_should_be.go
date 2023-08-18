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
