package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/execution"
	types "code.vegaprotocol.io/vega/proto"
)

func TheSuppliedStakeShouldBeForTheMarket(engine *execution.Engine, marketID string, wantSuppliedStake string) error {
	marketData, err := engine.GetMarketData(marketID)
	if err != nil {
		return errMarketDataNotFound(marketID, err)
	}

	if marketData.GetSuppliedStake() != wantSuppliedStake {
		return errUnexpectedSuppliedStake(marketData, wantSuppliedStake)
	}

	return nil
}

func errUnexpectedSuppliedStake(md types.MarketData, wantSuppliedStake string) error {
	return fmt.Errorf("unexpected supplied stake for market %s got %s, want %s", md.Market, md.GetSuppliedStake(), wantSuppliedStake)
}
