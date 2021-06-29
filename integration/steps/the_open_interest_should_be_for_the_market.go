package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/types"
)

func TheOpenInterestShouldBeForTheMarket(engine *execution.Engine, marketID string, wantOpenInterest string) error {
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
