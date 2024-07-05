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
)

type EstimatedBounds struct {
	PositionSizeAtUpper     num.Decimal
	PositionSizeAtLower     num.Decimal
	LossOnCommitmentAtUpper num.Decimal
	LossOnCommitmentAtLower num.Decimal
	LiquidationPriceAtUpper num.Decimal
	LiquidationPriceAtLower num.Decimal
}

func EstimateBounds(
	sqrter *Sqrter,
	lowerPrice, basePrice, upperPrice *num.Uint,
	leverageLower, leverageUpper num.Decimal,
	balance *num.Uint,
	linearSlippageFactor, initialMargin,
	riskFactorShort, riskFactorLong num.Decimal,
) EstimatedBounds {
	r := EstimatedBounds{}

	balanceD := balance.ToDecimal()
	if lowerPrice != nil {
		unitLower := LiquidityUnit(sqrter, basePrice, lowerPrice)
		avgEntryLower := AverageEntryPrice(sqrter, unitLower, basePrice)
		riskFactorLower := RiskFactor(leverageLower, riskFactorLong, linearSlippageFactor, initialMargin)
		lowerPriceD := lowerPrice.ToDecimal()
		boundPosLower := PositionAtLowerBound(riskFactorLower, balanceD, lowerPriceD, avgEntryLower)
		lossLower := LossOnCommitment(avgEntryLower, lowerPriceD, boundPosLower)
		liquidationPriceAtLower := LiquidationPrice(balanceD, lossLower, boundPosLower, lowerPriceD, linearSlippageFactor, riskFactorLong)

		r.PositionSizeAtLower = boundPosLower.Truncate(5)
		r.LiquidationPriceAtLower = liquidationPriceAtLower.Truncate(5)
		r.LossOnCommitmentAtLower = lossLower.Truncate(5)
	}

	if upperPrice != nil {
		unitUpper := LiquidityUnit(sqrter, upperPrice, basePrice)
		avgEntryUpper := AverageEntryPrice(sqrter, unitUpper, upperPrice)
		riskFactorUpper := RiskFactor(leverageUpper, riskFactorShort, linearSlippageFactor, initialMargin)
		upperPriceD := upperPrice.ToDecimal()
		boundPosUpper := PositionAtUpperBound(riskFactorUpper, balanceD, upperPriceD, avgEntryUpper)
		lossUpper := LossOnCommitment(avgEntryUpper, upperPriceD, boundPosUpper)
		liquidationPriceAtUpper := LiquidationPrice(balanceD, lossUpper, boundPosUpper, upperPriceD, linearSlippageFactor, riskFactorShort)

		r.PositionSizeAtUpper = boundPosUpper.Truncate(5)
		r.LiquidationPriceAtUpper = liquidationPriceAtUpper.Truncate(5)
		r.LossOnCommitmentAtUpper = lossUpper.Truncate(5)
	}

	return r
}

// Lu = (sqrt(pu) * sqrt(pl)) / (sqrt(pu) - sqrt(pl)).
func LiquidityUnit(sqrter *Sqrter, pu, pl *num.Uint) num.Decimal {
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
func AverageEntryPrice(sqrter *Sqrter, lu num.Decimal, pu *num.Uint) num.Decimal {
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
