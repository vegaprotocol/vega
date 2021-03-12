package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/execution"
	types "code.vegaprotocol.io/vega/proto"
)

func TheMarkPriceForTheMarketIs(
	exec *execution.Engine,
	market, markPriceStr string,
) error {
	markPrice := parseMarkPrice(markPriceStr)

	marketData, err := exec.GetMarketData(market)
	if err != nil {
		return errMarkPriceNotFound(markPriceStr, err)
	}

	if marketData.MarkPrice != markPrice {
		return errWrongMarkPrice(market, markPrice, marketData)
	}

	return nil
}

func parseMarkPrice(markPriceStr string) uint64 {
	markPrice, err := U64(markPriceStr)
	panicW("mark price", err)
	return markPrice
}

func errWrongMarkPrice(market string, markPrice uint64, marketData types.MarketData) error {
	return fmt.Errorf("mark price if wrong for market(%v), expected(%v) got(%v)",
		market, markPrice, marketData.MarkPrice,
	)
}

func errMarkPriceNotFound(markPriceStr string, err error) error {
	return fmt.Errorf("unable to get mark price for market(%v), err(%v)", markPriceStr, err)
}
