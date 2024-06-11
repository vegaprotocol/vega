package amm

import (
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

type EstimatorMetrics struct {
	LossOnCommitmentAtUpperBound num.Decimal
	LossOnCommitmentAtLowerBound num.Decimal
	PositionSizeAtUpperBound     num.Decimal
	PositionSizeAtLowerBound     num.Decimal
	LiquidationPriceAtUpperBound num.Decimal
	LiquidationPriceAtLowerBound num.Decimal
}

func EstimateBounds(
	basePrice, upperPrice, lowerPrice *num.Uint, leverageUpper, leverageLower num.Decimal, commitment *num.Uint,
) EstimatorMetrics {
	// FS = market sided risk factor
	rfs := &types.RiskFactor{
		Short: num.DecimalOne(),
		Long:  num.DecimalOne(),
	}
	// FL = market's linear slippage
	linearSlippage := num.DecimalOne()
	// Fi = market's initial margin factor
	sfs := &types.ScalingFactors{
		InitialMargin: num.DecimalOne(),
	}

	sqrter := NewSqrter()
	luUpperRange := LiquidityUnit(sqrter, upperPrice, basePrice)
	luLowerRange := LiquidityUnit(sqrter, basePrice, lowerPrice)

	aepUpperRange := AverageEntryPrice(sqrter, luUpperRange, upperPrice)
	aepLowerRange := AverageEntryPrice(sqrter, luLowerRange, basePrice)

	rfShortUpper := RiskFactor(leverageUpper, rfs.Short, linearSlippage, sfs.InitialMargin)
	rfLongLower := RiskFactor(leverageLower, rfs.Long, linearSlippage, sfs.InitialMargin)

	commitmentDecimal := commitment.ToDecimal()

	positionSizeAtUpperBound := PositionAtBound(rfShortUpper, commitmentDecimal, basePrice.ToDecimal(), aepUpperRange)
	positionSizeAtLowerBound := PositionAtBound(rfLongLower, commitmentDecimal, lowerPrice.ToDecimal(), aepLowerRange)

	return EstimatorMetrics{
		PositionSizeAtUpperBound: positionSizeAtUpperBound,
		PositionSizeAtLowerBound: positionSizeAtLowerBound,
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
func PositionAtBound(rf, b, pl, pa num.Decimal) num.Decimal {
	oneSubRf := num.DecimalOne().Sub(rf)
	rfMulPa := rf.Mul(pa)

	return rf.Mul(b).Div(
		pl.Mul(oneSubRf).Add(rfMulPa),
	)
}
