package steps

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/data-node/integration/stubs"
	"code.vegaprotocol.io/data-node/types"
)

func MarketOpeningAuctionPeriodEnds(timeStub *stubs.TimeStub, markets []types.Market, marketID string) error {
	var mkt *types.Market
	for _, m := range markets {
		if m.Id == marketID {
			mkt = &m
			break
		}
	}
	if mkt == nil {
		return errMarketNotFound(marketID)
	}
	// double the time, so it's definitely past opening auction time
	now, err := timeStub.GetTimeNow()
	if err != nil {
		return err

	}
	timeStub.SetTime(now.Add(time.Duration(mkt.OpeningAuction.Duration*2) * time.Second))
	return nil
}

func errMarketNotFound(marketID string) error {
	return fmt.Errorf("market %s not found", marketID)
}
