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
	PositionSizeAtLower     num.Decimal
	LossOnCommitmentAtLower num.Decimal
	LiquidationPriceAtLower num.Decimal
	TooWideLower            bool

	PositionSizeAtUpper     num.Decimal
	LossOnCommitmentAtUpper num.Decimal
	LiquidationPriceAtUpper num.Decimal
	TooWideUpper            bool
}

func EstimateBounds(
	sqrter *Sqrter,
	lowerPrice, basePrice, upperPrice *num.Uint,
	leverageLower, leverageUpper num.Decimal,
	balance *num.Uint,
	linearSlippageFactor, initialMargin,
	riskFactorShort, riskFactorLong,
	priceFactor, positionFactor num.Decimal,
	allowedMaxEmptyLevels uint64,
) EstimatedBounds {
	r := EstimatedBounds{}

	balanceD := balance.ToDecimal()

	oneTick, _ := num.UintFromDecimal(priceFactor)
	oneTick = num.Max(num.UintOne(), oneTick)

	if lowerPrice != nil {
		unitLower := LiquidityUnit(sqrter, basePrice, lowerPrice)

		avgEntryLower := AverageEntryPrice(sqrter, unitLower, basePrice)
		riskFactorLower := RiskFactor(leverageLower, riskFactorLong, linearSlippageFactor, initialMargin)
		lowerPriceD := lowerPrice.ToDecimal()
		boundPosLower := PositionAtLowerBound(riskFactorLower, balanceD, lowerPriceD, avgEntryLower, positionFactor)

		// if the commitment is *so low* that the position at the bound is 0 then we will panic trying to calculate the rest
		// and the "too wide" check below will flag it up as an invalid AMM defn
		if !boundPosLower.IsZero() {
			lossLower := LossOnCommitment(avgEntryLower, lowerPriceD, boundPosLower)

			liquidationPriceAtLower := LiquidationPrice(balanceD, lossLower, boundPosLower, lowerPriceD, linearSlippageFactor, riskFactorLong)

			r.PositionSizeAtLower = boundPosLower.Mul(positionFactor)
			r.LiquidationPriceAtLower = liquidationPriceAtLower
			r.LossOnCommitmentAtLower = lossLower
		}

		// now lets check that the lower bound is not too wide that the volume is spread too thin
		l := unitLower.Mul(boundPosLower).Abs()

		cu := &curve{
			l:        l,
			high:     basePrice,
			low:      lowerPrice,
			sqrtHigh: sqrter.sqrt(basePrice),
			isLower:  true,
			pv:       r.PositionSizeAtLower,
		}

		if err := cu.check(sqrter.sqrt, oneTick, allowedMaxEmptyLevels); err != nil {
			r.TooWideLower = true
		}
	}

	if upperPrice != nil {
		unitUpper := LiquidityUnit(sqrter, upperPrice, basePrice)

		avgEntryUpper := AverageEntryPrice(sqrter, unitUpper, upperPrice)
		riskFactorUpper := RiskFactor(leverageUpper, riskFactorShort, linearSlippageFactor, initialMargin)
		upperPriceD := upperPrice.ToDecimal()

		boundPosUpper := PositionAtUpperBound(riskFactorUpper, balanceD, upperPriceD, avgEntryUpper, positionFactor)

		// if the commitment is *so low* that the position at the bound is 0 then we will panic trying to calculate the rest
		// and the "too wide" check below will flag it up as an invalid AMM defn
		if !boundPosUpper.IsZero() {
			lossUpper := LossOnCommitment(avgEntryUpper, upperPriceD, boundPosUpper)

			liquidationPriceAtUpper := LiquidationPrice(balanceD, lossUpper, boundPosUpper, upperPriceD, linearSlippageFactor, riskFactorShort)
			r.PositionSizeAtUpper = boundPosUpper.Mul(positionFactor)
			r.LiquidationPriceAtUpper = liquidationPriceAtUpper
			r.LossOnCommitmentAtUpper = lossUpper
		}

		// now lets check that the lower bound is not too wide that the volume is spread too thin
		l := unitUpper.Mul(boundPosUpper).Abs()

		cu := &curve{
			l:        l,
			high:     upperPrice,
			low:      basePrice,
			sqrtHigh: sqrter.sqrt(upperPrice),
			pv:       r.PositionSizeAtUpper.Neg(),
		}
		if err := cu.check(sqrter.sqrt, oneTick, allowedMaxEmptyLevels); err != nil {
			r.TooWideUpper = true
		}
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
func PositionAtLowerBound(rf, b, pl, pa, positionFactor num.Decimal) num.Decimal {
	oneSubRf := num.DecimalOne().Sub(rf)
	rfMulPa := rf.Mul(pa)

	pv := rf.Mul(b).Div(
		pl.Mul(oneSubRf).Add(rfMulPa),
	)
	return pv
}

// Pvl = -rf * b / (pl * (1 + rf) - rf * pa).
func PositionAtUpperBound(rf, b, pl, pa, positionFactor num.Decimal) num.Decimal {
	onePlusRf := num.DecimalOne().Add(rf)
	rfMulPa := rf.Mul(pa)

	pv := rf.Neg().Mul(b).Div(
		pl.Mul(onePlusRf).Sub(rfMulPa),
	)
	return pv
}

// lc = |(pa - pb) * pB|.
func LossOnCommitment(pa, pb, pB num.Decimal) num.Decimal {
	res := pa.Sub(pb).Mul(pB).Abs()
	return res
}

// Pliq = (b - lc - Pb * pb) / (|Pb| * (fl + mr) - Pb).
func LiquidationPrice(b, lc, pB, pb, fl, mr num.Decimal) num.Decimal {
	// (b - lc - Pb * pb)
	numer := b.Sub(lc).Sub(pB.Mul(pb))

	// (|Pb| * (fl + mr) - Pb)
	denom := pB.Abs().Mul(fl.Add(mr)).Sub(pB)

	return num.MaxD(num.DecimalZero(), numer.Div(denom))
}
