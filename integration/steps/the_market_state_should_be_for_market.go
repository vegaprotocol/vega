package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/execution"
	types "code.vegaprotocol.io/vega/proto"
)

func TheMarketStateShouldBeForMarket(
	engine *execution.Engine,
	market, expectedMarketStateStr string,
) error {
	expectedMarketState, err := MarketState(expectedMarketStateStr)
	panicW("market state", err)

	marketState, err := engine.GetMarketState(market)
	if err != nil {
		return errMarketDataNotFound(market, err)
	}

	if marketState != expectedMarketState {
		return errMismatchedMarketState(market, expectedMarketState, marketState)
	}
	return nil
}

func errMismatchedMarketState(market string, expectedMarketState, marketState types.Market_State) error {
	return formatDiff(
		fmt.Sprintf("unexpected market state for market \"%s\"", market),
		map[string]string{
			"market state": expectedMarketState.String(),
		},
		map[string]string{
			"market state": marketState.String(),
		},
	)
}
