package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

func TheMarkPriceForTheMarketIs(
	exec Execution,
	market, markPriceStr string,
) error {
	markPrice := parseMarkPrice(markPriceStr)

	marketData, err := exec.GetMarketData(market)
	if err != nil {
		return errMarkPriceNotFound(market, err)
	}

	if marketData.MarkPrice.NEQ(markPrice) {
		return errWrongMarkPrice(market, markPrice, marketData)
	}

	return nil
}

func parseMarkPrice(markPriceStr string) *num.Uint {
	markPrice, err := U64(markPriceStr)
	panicW("mark price", err)
	return num.NewUint(markPrice)
}

func errWrongMarkPrice(market string, markPrice *num.Uint, marketData types.MarketData) error {
	return fmt.Errorf("wrong mark price for market(%v), expected(%v) got(%v)",
		market, markPrice, marketData.MarkPrice,
	)
}

func errMarkPriceNotFound(market string, err error) error {
	return fmt.Errorf("unable to get mark price for market(%v), err(%v)", market, err)
}
