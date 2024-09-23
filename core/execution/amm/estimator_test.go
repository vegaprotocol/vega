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
	sqrter := NewSqrter()

	shortRiskFactor := num.NewDecimalFromFloat(0.01)
	longRiskFactor := num.NewDecimalFromFloat(0.01)
	linearSlippage := num.NewDecimalFromFloat(0.05)
	initialMargin := num.DecimalOne()

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
	lowerBoundPos := PositionAtLowerBound(riskFactorLower, balance.ToDecimal(), lowerPriceD, avgEntryLower, num.DecimalOne())
	upperBoundPos := PositionAtUpperBound(riskFactorUpper, balance.ToDecimal(), upperPriceD, avgEntryUpper, num.DecimalOne())
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
	sqrter := NewSqrter()

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
			sqrter,
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
			num.DecimalOne(),
			num.DecimalOne(),
			0,
		)

		assert.Equal(t, expectedMetrics.PositionSizeAtUpper.String(), metrics.PositionSizeAtUpper.Round(3).String())
		assert.Equal(t, expectedMetrics.PositionSizeAtLower.String(), metrics.PositionSizeAtLower.Round(3).String())
		assert.Equal(t, expectedMetrics.LossOnCommitmentAtUpper.String(), metrics.LossOnCommitmentAtUpper.Round(3).String())
		assert.Equal(t, expectedMetrics.LossOnCommitmentAtLower.String(), metrics.LossOnCommitmentAtLower.Round(3).String())
		assert.Equal(t, expectedMetrics.LiquidationPriceAtUpper.String(), metrics.LiquidationPriceAtUpper.Round(3).String())
		assert.Equal(t, expectedMetrics.LiquidationPriceAtLower.String(), metrics.LiquidationPriceAtLower.Round(3).String())
		assert.True(t, metrics.TooWideLower)
		assert.True(t, metrics.TooWideUpper)
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
			sqrter,
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
			num.DecimalOne(),
			num.DecimalOne(),
			0,
		)

		assert.Equal(t, expectedMetrics.PositionSizeAtUpper.String(), metrics.PositionSizeAtUpper.Round(3).String())
		assert.Equal(t, expectedMetrics.PositionSizeAtLower.String(), metrics.PositionSizeAtLower.Round(3).String())
		assert.Equal(t, expectedMetrics.LossOnCommitmentAtUpper.String(), metrics.LossOnCommitmentAtUpper.Round(3).String())
		assert.Equal(t, expectedMetrics.LossOnCommitmentAtLower.String(), metrics.LossOnCommitmentAtLower.Round(3).String())
		assert.Equal(t, expectedMetrics.LiquidationPriceAtUpper.String(), metrics.LiquidationPriceAtUpper.Round(3).String())
		assert.Equal(t, expectedMetrics.LiquidationPriceAtLower.String(), metrics.LiquidationPriceAtLower.Round(3).String())
	})
}

func TestEstimatePositionFactor(t *testing.T) {
	initialMargin := num.DecimalFromFloat(1.2)
	riskFactorShort := num.DecimalFromFloat(0.05529953589167391)
	riskFactorLong := num.DecimalFromFloat(0.05529953589167391)
	linearSlippageFactor := num.DecimalFromFloat(0.01)
	sqrter := NewSqrter()

	lowerPrice := num.MustUintFromString("80000000000000000000", 10)
	basePrice := num.MustUintFromString("100000000000000000000", 10)
	upperPrice := num.MustUintFromString("120000000000000000000", 10)
	leverageUpper := num.DecimalFromFloat(0.5)
	leverageLower := num.DecimalFromFloat(0.5)
	balance := num.MustUintFromString("390500000000000000000000000", 10)

	expectedMetrics := EstimatedBounds{
		PositionSizeAtUpper: num.DecimalFromFloat(-1559159.284),
		PositionSizeAtLower: num.DecimalFromFloat(2304613.63),
	}

	metrics := EstimateBounds(
		sqrter,
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
		num.DecimalFromInt64(1000000000000000000),
		num.DecimalOne(),
		0,
	)

	assert.Equal(t, expectedMetrics.PositionSizeAtUpper.String(), metrics.PositionSizeAtUpper.Round(3).String())
	assert.Equal(t, expectedMetrics.PositionSizeAtLower.String(), metrics.PositionSizeAtLower.Round(3).String())
	assert.False(t, metrics.TooWideLower)
	assert.False(t, metrics.TooWideUpper)

	// if commitment is super low then we could panic, so test that we don't
	metrics = EstimateBounds(
		sqrter,
		lowerPrice,
		basePrice,
		upperPrice,
		leverageLower,
		leverageUpper,
		num.MustUintFromString("390500000000000000000", 10),
		linearSlippageFactor,
		initialMargin,
		riskFactorShort,
		riskFactorLong,
		num.DecimalFromInt64(1000000000000000000),
		num.DecimalOne(),
		10,
	)

	assert.Equal(t, "-1.559", metrics.PositionSizeAtUpper.Round(3).String())
	assert.Equal(t, "2.305", metrics.PositionSizeAtLower.Round(3).String())
	assert.False(t, metrics.TooWideLower) // is valid as there are less than 10 empty price levels
	assert.True(t, metrics.TooWideUpper)  // isn't valid as there are more than 10 empty price levels
}
