package entities_test

import (
	"testing"
)

func TestLiquidityProvision(t *testing.T) {
	t.Run("should parse all valid prices", testParseAllValidPrices)
	t.Run("should return zero for prices if string is empty", testParseEmptyPrices)
	t.Run("should return error if an invalid price string is provided", testParseInvalidPriceString)
	t.Run("should parse valid market data records successfully", testParseMarketDataSuccessfully)
}
