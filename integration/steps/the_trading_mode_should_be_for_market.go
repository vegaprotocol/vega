package steps

import (
	"fmt"

	types "code.vegaprotocol.io/protos/vega"
)

func TheTradingModeShouldBeForMarket(
	engine Execution,
	market, tradingModeStr string,
) error {
	tradingMode, err := TradingMode(tradingModeStr)
	panicW("trading mode", err)

	marketData, err := engine.GetMarketData(market)
	if err != nil {
		return errMarketDataNotFound(market, err)
	}

	if marketData.MarketTradingMode != tradingMode {
		return errMismatchedTradingMode(market, tradingMode, marketData.MarketTradingMode)
	}
	return nil
}

func errMismatchedTradingMode(market string, expectedTradingMode, gotTradingMode types.Market_TradingMode) error {
	return formatDiff(
		fmt.Sprintf("unexpected market trading mode for market \"%s\"", market),
		map[string]string{
			"trading mode": expectedTradingMode.String(),
		},
		map[string]string{
			"trading mode": gotTradingMode.String(),
		},
	)
}
