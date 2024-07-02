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
	"context"
	"fmt"
	"strconv"
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
	// ensure we simulate the end of the governance period
	if data.MarketState != types.MarketStatePending {
		if err := execEngine.StartOpeningAuction(context.Background(), mkt.ID); err != nil {
			return err
		}
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

func MarketAuctionStartTime(execEngine Execution, marketID, startTime string) error {
	data, err := execEngine.GetMarketData(marketID)
	if err != nil {
		return errMarketDataNotFound(marketID, err)
	}

	st, err := strconv.ParseInt(startTime, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to convert time to int64: %s", startTime)
	}

	if data.AuctionStart == st {
		return nil
	}
	return fmt.Errorf("start auction time did not match %d != %d", data.AuctionStart, st)
}

func MarketAuctionEndTime(execEngine Execution, marketID, endTime string) error {
	data, err := execEngine.GetMarketData(marketID)
	if err != nil {
		return errMarketDataNotFound(marketID, err)
	}

	et, err := strconv.ParseInt(endTime, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to convert time to int64: %s", endTime)
	}

	if data.AuctionEnd == et {
		return nil
	}
	return fmt.Errorf("end auction time did not match %d != %d", data.AuctionEnd, et)
}
