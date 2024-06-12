package amm

import (
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/shopspring/decimal"
)

type EstimatorMetrics struct {
	PositionSizeAtUpperBound     num.Decimal
	PositionSizeAtLowerBound     num.Decimal
	LossOnCommitmentAtUpperBound num.Decimal
	LossOnCommitmentAtLowerBound num.Decimal
	LiquidationPriceAtUpperBound num.Decimal
	LiquidationPriceAtLowerBound num.Decimal
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
) EstimatorMetrics {
	// test liquidity unit
	unitLower := LiquidityUnit(sqrter, basePrice, lowerPrice)
	unitUpper := LiquidityUnit(sqrter, upperPrice, basePrice)

	// test average entry price
	avgEntryLower := AverageEntryPrice(sqrter, unitLower, basePrice)
	avgEntryUpper := AverageEntryPrice(sqrter, unitUpper, upperPrice)

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

	return EstimatorMetrics{
		PositionSizeAtUpperBound:     liquidationPriceAtUpper,
		PositionSizeAtLowerBound:     liquidationPriceAtLower,
		LossOnCommitmentAtUpperBound: lossUpper,
		LossOnCommitmentAtLowerBound: lossLower,
		LiquidationPriceAtUpperBound: liquidationPriceAtUpper,
		LiquidationPriceAtLowerBound: liquidationPriceAtLower,
	}
}

// Lu = (sqrt(pu) * sqrt(pl)) / (sqrt(pu) - sqrt(pl))
func LiquidityUnit(sqrter *Sqrter, pu, pl *num.Uint) num.Decimal {
	sqrtPu := sqrter.sqrt(pu)
	sqrtPl := sqrter.sqrt(pl)

	return sqrtPu.Mul(sqrtPl).Div(sqrtPu.Sub(sqrtPl))
}

// Rf = min(Lb, 1 / (Fs + Fl) * Fi)
func RiskFactor(lb, fs, fl, fi num.Decimal) num.Decimal {
	b := num.DecimalOne().Div(fs.Add(fl).Mul(fi))
	return num.MinD(lb, b)
}

// Pa = Lu * sqrt(pu) * (1 - (Lu / (Lu + sqrt(pu))))
func AverageEntryPrice(sqrter *Sqrter, lu num.Decimal, pu *num.Uint) num.Decimal {
	sqrtPu := sqrter.sqrt(pu)
	// (1 - Lu / (Lu + sqrt(pu)))
	oneSubLuDivLuWithUpSquared := num.DecimalOne().Sub(lu.Div(lu.Add(sqrtPu)))
	return lu.Mul(sqrtPu).Mul(oneSubLuDivLuWithUpSquared)
}

// Pvl = rf * b / (pl * (1 - rf) + rf * pa)
func PositionAtLowerBound(rf, b, pl, pa num.Decimal) num.Decimal {
	oneSubRf := num.DecimalOne().Sub(rf)
	rfMulPa := rf.Mul(pa)

	return rf.Mul(b).Div(
		pl.Mul(oneSubRf).Add(rfMulPa),
	)
}

// Pvl = -rf * b / (pl * (1 + rf) - rf * pa)
func PositionAtUpperBound(rf, b, pl, pa num.Decimal) num.Decimal {
	onePlusRf := num.DecimalOne().Add(rf)
	rfMulPa := rf.Mul(pa)

	return rf.Neg().Mul(b).Div(
		pl.Mul(onePlusRf).Sub(rfMulPa),
	)
}

// lc = |pa - pb * pB|
func LossOnCommitment(pa, pb, pB num.Decimal) num.Decimal {
	return pa.Sub(pb).Mul(pB).Abs()
}

// Pliq = (b - lc - Pb * pb) / (|Pb| * (fl + mr) - Pb)
func LiquidationPrice(b, lc, pB, pb, fl, mr num.Decimal) num.Decimal {
	return b.Sub(lc).Sub(pB.Mul(pb)).Div(
		pB.Abs().Mul(fl.Add(mr)).Sub(pB),
	)
}
