package steps

import (
	"fmt"
	"strconv"

	"code.vegaprotocol.io/vega/execution"
)

func TheMarkPriceForTheMarketIs(
	exec *execution.Engine,
	market, markPriceStr string,
) error {
	markPrice, err := strconv.ParseUint(markPriceStr, 10, 0)
	if err != nil {
		return fmt.Errorf("markPrice is not a integer: markPrice(%v), err(%v)", markPriceStr, err)
	}

	mktdata, err := exec.GetMarketData(market)
	if err != nil {
		return fmt.Errorf("unable to get mark price for market(%v), err(%v)", markPriceStr, err)
	}

	if mktdata.MarkPrice != markPrice {
		return fmt.Errorf("mark price if wrong for market(%v), expected(%v) got(%v)", market, markPrice, mktdata.MarkPrice)
	}

	return nil
}
