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

	"code.vegaprotocol.io/vega/core/types"
)

func TheTargetStakeShouldBeForMarket(engine Execution, marketID string, wantTargetStake string) error {
	marketData, err := engine.GetMarketData(marketID)
	if err != nil {
		return errMarketDataNotFound(marketID, err)
	}

	if marketData.TargetStake != wantTargetStake {
		return errUnexpectedTargetStake(marketData, wantTargetStake)
	}

	return nil
}

func errUnexpectedTargetStake(md types.MarketData, wantTargetStake string) error {
	return fmt.Errorf("unexpected target stake for market %s got %s, want %s", md.Market, md.TargetStake, wantTargetStake)
}
