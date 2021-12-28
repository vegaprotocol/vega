package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/integration/stubs"
)

func TheLiquidityFeeFactorShouldForTheMarket(
	broker *stubs.BrokerStub,
	feeStr, market string,
) error {
	mkt := broker.GetMarket(market)
	if mkt == nil {
		return fmt.Errorf("invalid market id %v", market)
	}

	got := mkt.Fees.Factors.LiquidityFee
	if got != feeStr {
		return errInvalidLiquidityFeeFactor(market, feeStr, got)
	}

	return nil
}

func errInvalidLiquidityFeeFactor(market string, expected, got string) error {
	return fmt.Errorf("invalid liquidity fee factor for market %s want %s got %s", market, expected, got)
}
