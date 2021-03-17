package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/execution"
	types "code.vegaprotocol.io/vega/proto"
)

func TradingModeForMarketIs(
	engine *execution.Engine,
	market, tradingModeStr string,
) error {
	tradingMode, err := TradingMode(tradingModeStr)
	panicW("trading mode", err)

	marketData, err := engine.GetMarketData(market)
	if err != nil {
		return errCannotGetMarketData(market, err)
	}

	if marketData.MarketTradingMode != tradingMode {
		return errMismatchedTradingMode(market, tradingMode, marketData.MarketTradingMode)
	}
	return nil
}

func errMismatchedTradingMode(market string, expectedTradingMode, gotTradingMode types.Market_TradingMode) error {
	return fmt.Errorf("market trading tradingMode is wrong for market(%v), expected(%v) got(%v)", market, expectedTradingMode, gotTradingMode)
}

func errCannotGetMarketData(marketID string, err error) error {
	return fmt.Errorf("couldn't get order for marked data for market(%v): %s", marketID, err.Error())
}
