// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package amm

import (
	"testing"

	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/assert"
)

func TestEstimateSeparateFunctions(t *testing.T) {
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

	// test liquidity unit
	unitLower := LiquidityUnit(basePrice, lowerPrice)
	unitUpper := LiquidityUnit(upperPrice, basePrice)

	assert.Equal(t, num.DecimalFromFloat(584.6049894).String(), unitLower.Round(7).String())
	assert.Equal(t, num.DecimalFromFloat(257.2170745).String(), unitUpper.Round(7).String())

	// test average entry price
	avgEntryLower := AverageEntryPrice(unitLower, basePrice)
	avgEntryUpper := AverageEntryPrice(unitUpper, upperPrice)
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

func TestEstimate(t *testing.T) {
	initialMargin := num.DecimalFromFloat(1)
	riskFactorShort := num.DecimalFromFloat(0.01)
	riskFactorLong := num.DecimalFromFloat(0.01)
	linearSlippageFactor := num.DecimalFromFloat(0)

	t.Run("test 0014-NP-VAMM-001", func(t *testing.T) {
		lowerPrice := num.NewUint(900)
		basePrice := num.NewUint(1000)
		upperPrice := num.NewUint(1100)
		leverageUpper := num.DecimalFromFloat(2.00)
		leverageLower := num.DecimalFromFloat(2.00)
		balance := num.NewUint(100)

		expectedMetrics := EstimatedBounds{
			PositionSizeAtUpper:     num.DecimalFromFloat(-0.166),
			PositionSizeAtLower:     num.DecimalFromFloat(0.201),
			LossOnCommitmentAtUpper: num.DecimalFromFloat(8.515),
			LossOnCommitmentAtLower: num.DecimalFromFloat(9.762),
			LiquidationPriceAtUpper: num.DecimalFromFloat(1633.663),
			LiquidationPriceAtLower: num.DecimalFromFloat(454.545),
		}

		metrics := EstimateBounds(
			lowerPrice,
			basePrice,
			upperPrice,
			leverageLower,
			leverageUpper,
			balance,
			linearSlippageFactor,
			initialMargin,
			riskFactorShort,
			riskFactorLong,
		)

		assert.Equal(t, expectedMetrics.PositionSizeAtUpper.String(), metrics.PositionSizeAtUpper.Round(3).String())
		assert.Equal(t, expectedMetrics.PositionSizeAtLower.String(), metrics.PositionSizeAtLower.Round(3).String())
		assert.Equal(t, expectedMetrics.LossOnCommitmentAtUpper.String(), metrics.LossOnCommitmentAtUpper.Round(3).String())
		assert.Equal(t, expectedMetrics.LossOnCommitmentAtLower.String(), metrics.LossOnCommitmentAtLower.Round(3).String())
		assert.Equal(t, expectedMetrics.LiquidationPriceAtUpper.String(), metrics.LiquidationPriceAtUpper.Round(3).String())
		assert.Equal(t, expectedMetrics.LiquidationPriceAtLower.String(), metrics.LiquidationPriceAtLower.Round(3).String())
	})

	t.Run("test 0014-NP-VAMM-004", func(t *testing.T) {
		lowerPrice := num.NewUint(900)
		basePrice := num.NewUint(1000)
		upperPrice := num.NewUint(1300)
		leverageUpper := num.DecimalFromFloat(1)
		leverageLower := num.DecimalFromFloat(5)
		balance := num.NewUint(100)

		expectedMetrics := EstimatedBounds{
			PositionSizeAtUpper:     num.DecimalFromFloat(-0.069),
			PositionSizeAtLower:     num.DecimalFromFloat(0.437),
			LossOnCommitmentAtUpper: num.DecimalFromFloat(10.948),
			LossOnCommitmentAtLower: num.DecimalFromFloat(21.289),
			LiquidationPriceAtUpper: num.DecimalFromFloat(2574.257),
			LiquidationPriceAtLower: num.DecimalFromFloat(727.273),
		}

		metrics := EstimateBounds(
			lowerPrice,
			basePrice,
			upperPrice,
			leverageLower,
			leverageUpper,
			balance,
			linearSlippageFactor,
			initialMargin,
			riskFactorShort,
			riskFactorLong,
		)

		assert.Equal(t, expectedMetrics.PositionSizeAtUpper.String(), metrics.PositionSizeAtUpper.Round(3).String())
		assert.Equal(t, expectedMetrics.PositionSizeAtLower.String(), metrics.PositionSizeAtLower.Round(3).String())
		assert.Equal(t, expectedMetrics.LossOnCommitmentAtUpper.String(), metrics.LossOnCommitmentAtUpper.Round(3).String())
		assert.Equal(t, expectedMetrics.LossOnCommitmentAtLower.String(), metrics.LossOnCommitmentAtLower.Round(3).String())
		assert.Equal(t, expectedMetrics.LiquidationPriceAtUpper.String(), metrics.LiquidationPriceAtUpper.Round(3).String())
		assert.Equal(t, expectedMetrics.LiquidationPriceAtLower.String(), metrics.LiquidationPriceAtLower.Round(3).String())
	})
}
