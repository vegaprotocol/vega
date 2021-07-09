package steps

import (
	"fmt"

	"code.vegaprotocol.io/data-node/execution"
	"code.vegaprotocol.io/data-node/types"
)

func TheTargetStakeShouldBeForMarket(engine *execution.Engine, marketID string, wantTargetStake string) error {
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
