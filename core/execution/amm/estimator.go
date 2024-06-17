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
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/shopspring/decimal"
)

type EstimatedBounds struct {
	PositionSizeAtUpper     num.Decimal
	PositionSizeAtLower     num.Decimal
	LossOnCommitmentAtUpper num.Decimal
	LossOnCommitmentAtLower num.Decimal
	LiquidationPriceAtUpper num.Decimal
	LiquidationPriceAtLower num.Decimal
}

var sqrter *Sqrter

func init() {
	sqrter = NewSqrter()
}

func EstimateBounds(
	lowerPrice, basePrice, upperPrice *num.Uint,
	leverageLower, leverageUpper num.Decimal,
	balance *num.Uint,
	linearSlippageFactor, initialMargin decimal.Decimal,
	riskFactorShort, riskFactorLong decimal.Decimal,
) EstimatedBounds {
	// test liquidity unit
	unitLower := LiquidityUnit(basePrice, lowerPrice)
	unitUpper := LiquidityUnit(upperPrice, basePrice)

	// test average entry price
	avgEntryLower := AverageEntryPrice(unitLower, basePrice)
	avgEntryUpper := AverageEntryPrice(unitUpper, upperPrice)

	// test risk factor
	riskFactorLower := RiskFactor(leverageLower, riskFactorLong, linearSlippageFactor, initialMargin)
	riskFactorUpper := RiskFactor(leverageUpper, riskFactorShort, linearSlippageFactor, initialMargin)

	lowerPriceD := lowerPrice.ToDecimal()
	upperPriceD := upperPrice.ToDecimal()
	balanceD := balance.ToDecimal()

	// test position at bounds
	boundPosLower := PositionAtLowerBound(riskFactorLower, balanceD, lowerPriceD, avgEntryLower)
	boundPosUpper := PositionAtUpperBound(riskFactorUpper, balanceD, upperPriceD, avgEntryUpper)

	// test loss on commitment
	lossLower := LossOnCommitment(avgEntryLower, lowerPriceD, boundPosLower)
	lossUpper := LossOnCommitment(avgEntryUpper, upperPriceD, boundPosUpper)

	// test liquidation price
	liquidationPriceAtLower := LiquidationPrice(balanceD, lossLower, boundPosLower, lowerPriceD, linearSlippageFactor, riskFactorLong)
	liquidationPriceAtUpper := LiquidationPrice(balanceD, lossUpper, boundPosUpper, upperPriceD, linearSlippageFactor, riskFactorShort)

	return EstimatedBounds{
		PositionSizeAtUpper:     boundPosUpper,
		PositionSizeAtLower:     boundPosLower,
		LossOnCommitmentAtUpper: lossUpper,
		LossOnCommitmentAtLower: lossLower,
		LiquidationPriceAtUpper: liquidationPriceAtUpper,
		LiquidationPriceAtLower: liquidationPriceAtLower,
	}
}

// Lu = (sqrt(pu) * sqrt(pl)) / (sqrt(pu) - sqrt(pl)).
func LiquidityUnit(pu, pl *num.Uint) num.Decimal {
	sqrtPu := sqrter.sqrt(pu)
	sqrtPl := sqrter.sqrt(pl)

	return sqrtPu.Mul(sqrtPl).Div(sqrtPu.Sub(sqrtPl))
}

// Rf = min(Lb, 1 / (Fs + Fl) * Fi).
func RiskFactor(lb, fs, fl, fi num.Decimal) num.Decimal {
	b := num.DecimalOne().Div(fs.Add(fl).Mul(fi))
	return num.MinD(lb, b)
}

// Pa = Lu * sqrt(pu) * (1 - (Lu / (Lu + sqrt(pu)))).
func AverageEntryPrice(lu num.Decimal, pu *num.Uint) num.Decimal {
	sqrtPu := sqrter.sqrt(pu)
	// (1 - Lu / (Lu + sqrt(pu)))
	oneSubLuDivLuWithUpSquared := num.DecimalOne().Sub(lu.Div(lu.Add(sqrtPu)))
	return lu.Mul(sqrtPu).Mul(oneSubLuDivLuWithUpSquared)
}

// Pvl = rf * b / (pl * (1 - rf) + rf * pa).
func PositionAtLowerBound(rf, b, pl, pa num.Decimal) num.Decimal {
	oneSubRf := num.DecimalOne().Sub(rf)
	rfMulPa := rf.Mul(pa)

	return rf.Mul(b).Div(
		pl.Mul(oneSubRf).Add(rfMulPa),
	)
}

// Pvl = -rf * b / (pl * (1 + rf) - rf * pa).
func PositionAtUpperBound(rf, b, pl, pa num.Decimal) num.Decimal {
	onePlusRf := num.DecimalOne().Add(rf)
	rfMulPa := rf.Mul(pa)

	return rf.Neg().Mul(b).Div(
		pl.Mul(onePlusRf).Sub(rfMulPa),
	)
}

// lc = |pa - pb * pB|.
func LossOnCommitment(pa, pb, pB num.Decimal) num.Decimal {
	return pa.Sub(pb).Mul(pB).Abs()
}

// Pliq = (b - lc - Pb * pb) / (|Pb| * (fl + mr) - Pb).
func LiquidationPrice(b, lc, pB, pb, fl, mr num.Decimal) num.Decimal {
	return b.Sub(lc).Sub(pB.Mul(pb)).Div(
		pB.Abs().Mul(fl.Add(mr)).Sub(pB),
	)
}
