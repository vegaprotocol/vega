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
	"time"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/types"
)

func MarketOpeningAuctionPeriodEnds(execEngine Execution, timeStub *stubs.TimeStub, markets []types.Market, marketID string) error {
	var mkt *types.Market
	for _, m := range markets {
		if m.ID == marketID {
			m := m
			mkt = &m
			break
		}
	}
	if mkt == nil {
		return errMarketNotFound(marketID)
	}
	// double the time, so it's definitely past opening auction time
	data, err := execEngine.GetMarketData(mkt.ID)
	if err != nil {
		return errMarketDataNotFound(marketID, err)
	}

	end := time.Unix(0, data.AuctionEnd)
	now := timeStub.GetTimeNow()
	if end.Before(now) {
		// already out of auction step a second to make things happen
		timeStub.SetTime(now.Add(time.Second))
		return nil
	}

	timeStub.SetTime(now.Add(2 * end.Sub(now)))
	return nil
}

func errMarketNotFound(marketID string) error {
	return fmt.Errorf("market %s not found", marketID)
}
