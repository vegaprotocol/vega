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
)

func TheOpenInterestShouldBeForTheMarket(engine Execution, marketID string, wantOpenInterest string) error {
	marketData, err := engine.GetMarketData(marketID)
	if err != nil {
		return errMarketDataNotFound(marketID, err)
	}

	if fmt.Sprintf("%d", marketData.OpenInterest) != wantOpenInterest {
		return errUnexpectedOpenInterest(marketData, wantOpenInterest)
	}

	return nil
}

func errUnexpectedOpenInterest(md types.MarketData, wantOpenInterest string) error {
	return fmt.Errorf("unexpected open interest for market %s got %d, want %s", md.Market, md.OpenInterest, wantOpenInterest)
}
