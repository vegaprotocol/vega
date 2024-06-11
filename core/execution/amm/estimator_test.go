package amm

import (
	"testing"

	"code.vegaprotocol.io/vega/libs/num"
	"github.com/stretchr/testify/assert"
)

func TestEstimateBounds(t *testing.T) {
	balance := num.NewUint(100)
	lowerPrice := num.NewUint(900)
	basePrice := num.NewUint(1000)
	upperPrice := num.NewUint(1300)
	leverageAtUpper := num.NewDecimalFromFloat(1.00)
	leverageAtLower := num.NewDecimalFromFloat(5.00)

	sqrter := NewSqrter()

	// test liquidity unit
	unitLower := LiquidityUnit(sqrter, basePrice, lowerPrice)
	unitUpper := LiquidityUnit(sqrter, upperPrice, basePrice)

	assert.Equal(t, num.DecimalFromFloat(584.6049894).String(), unitLower.Round(7).String())
	assert.Equal(t, num.DecimalFromFloat(257.2170745).String(), unitUpper.Round(7).String())

	// test average entry price
	avgEntryLower := AverageEntryPrice(sqrter, unitLower, basePrice)
	avgEntryUpper := AverageEntryPrice(sqrter, unitUpper, upperPrice)
	assert.Equal(t, num.DecimalFromFloat(948.683).String(), avgEntryLower.Round(3).String())
	assert.Equal(t, num.DecimalFromFloat(1140.175).String(), avgEntryUpper.Round(3).String())

	// test risk factor
	zeroPointZeroFive := num.NewDecimalFromFloat(0.05)
	one := num.DecimalOne()

	riskFactorLower := RiskFactor(leverageAtLower, zeroPointZeroFive, zeroPointZeroFive, one)
	riskFactorUpper := RiskFactor(leverageAtUpper, zeroPointZeroFive, zeroPointZeroFive, one)
	assert.Equal(t, leverageAtLower.String(), riskFactorLower.String())
	assert.Equal(t, leverageAtUpper.String(), riskFactorUpper.String())

	lowerBoundPos := PositionAtBound(riskFactorLower, balance.ToDecimal(), lowerPrice.ToDecimal(), avgEntryLower)
	upperBoundPos := PositionAtBound(riskFactorUpper, balance.ToDecimal(), upperPrice.ToDecimal(), avgEntryUpper)
	assert.Equal(t, num.DecimalFromFloat(0.437).String(), lowerBoundPos.Round(3).String())
	// TODO: Tom needs to clarify the formula for upper bound position
	assert.Equal(t, num.DecimalFromFloat(-0.069).String(), upperBoundPos.Round(3).String())

	// out := EstimateBounds(basePrice, upperPrice, lowerPrice, leverageAtUpper, leverageAtLower, balance)
	// assert.Equal(t, num.DecimalFromFloat(-0.166).String(), out.PositionSizeAtUpperBound.String())
	// assert.Equal(t, num.DecimalFromFloat(0.201).String(), out.PositionSizeAtLowerBound.String())
}
