package steps

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/integration/stubs"
	"code.vegaprotocol.io/vega/types"
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
