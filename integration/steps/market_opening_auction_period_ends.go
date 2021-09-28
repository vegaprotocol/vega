package steps

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/integration/stubs"
	"code.vegaprotocol.io/vega/types"
)

func MarketOpeningAuctionPeriodEnds(timeStub *stubs.TimeStub, markets []types.Market, marketID string) error {
	var mkt *types.Market
	for _, m := range markets {
		if m.ID == marketID {
			mkt = &m
			break
		}
	}
	if mkt == nil {
		return errMarketNotFound(marketID)
	}
	// double the time, so it's definitely past opening auction time
	now := timeStub.GetTimeNow()
	timeStub.SetTime(now.Add(time.Duration(mkt.OpeningAuction.Duration*2) * time.Second))
	return nil
}

func errMarketNotFound(marketID string) error {
	return fmt.Errorf("market %s not found", marketID)
}
