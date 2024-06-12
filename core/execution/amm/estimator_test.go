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

	shortRiskFactor := num.NewDecimalFromFloat(0.01)
	longRiskFactor := num.NewDecimalFromFloat(0.01)
	linearSlippage := num.NewDecimalFromFloat(0.05)
	initialMargin := num.DecimalOne()

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
	riskFactorLower := RiskFactor(leverageAtLower, longRiskFactor, linearSlippage, initialMargin)
	riskFactorUpper := RiskFactor(leverageAtUpper, shortRiskFactor, linearSlippage, initialMargin)
	assert.Equal(t, leverageAtLower.String(), riskFactorLower.String())
	assert.Equal(t, leverageAtUpper.String(), riskFactorUpper.String())

	lowerPriceD := lowerPrice.ToDecimal()
	upperPriceD := upperPrice.ToDecimal()

	// test position at bounds
	lowerBoundPos := PositionAtLowerBound(riskFactorLower, balance.ToDecimal(), lowerPriceD, avgEntryLower)
	upperBoundPos := PositionAtUpperBound(riskFactorUpper, balance.ToDecimal(), upperPriceD, avgEntryUpper)
	assert.Equal(t, num.DecimalFromFloat(0.437).String(), lowerBoundPos.Round(3).String())
	assert.Equal(t, num.DecimalFromFloat(-0.069).String(), upperBoundPos.Round(3).String())

	// test loss on commitment
	lossAtLower := LossOnCommitment(avgEntryLower, lowerPriceD, lowerBoundPos)
	lossAtUpper := LossOnCommitment(avgEntryUpper, upperPriceD, upperBoundPos)
	assert.Equal(t, num.DecimalFromFloat(21.28852368).String(), lossAtLower.Round(8).String())
	assert.Equal(t, num.DecimalFromFloat(10.94820416).String(), lossAtUpper.Round(8).String())

	linearSlippageFactor := num.DecimalZero()

	// test liquidation price
	liquidationPriceAtLower := LiquidationPrice(balance.ToDecimal(), lossAtLower, lowerBoundPos, lowerPriceD, linearSlippageFactor, longRiskFactor)
	liquidationPriceAtUpper := LiquidationPrice(balance.ToDecimal(), lossAtUpper, upperBoundPos, upperPriceD, linearSlippageFactor, shortRiskFactor)
	assert.Equal(t, num.DecimalFromFloat(727.2727273).String(), liquidationPriceAtLower.Round(7).String())
	assert.Equal(t, num.DecimalFromFloat(2574.257426).String(), liquidationPriceAtUpper.Round(6).String())
}
