package amm

import (
	"fmt"

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
	basePrice, upperPrice, lowerPrice *num.Uint, leverageUpper, leverageLower *num.Decimal, commitment *num.Uint,
) EstimatorMetrics {
	sqrter := NewSqrter()
	lu := liquidityUnit(sqrter, upperPrice, lowerPrice)
	aep := averageEntryPrice(sqrter, lu, upperPrice)

	fmt.Println("lu", lu, aep)

	return EstimatorMetrics{}
}

// Rf = min(Lb, 1 / (Fs + Fl) * Fi)
func riskFactor(sqrter *Sqrter, lu num.Decimal, upperPrice, lowerPrice *num.Uint) num.Decimal {
	return lu
}

// Pa = Lu * sqrt(pu) * (1 - Lu / (Lu + sqrt(pu)))
func averageEntryPrice(sqrter *Sqrter, lu num.Decimal, upperPrice *num.Uint) num.Decimal {
	upperSquared := sqrter.sqrt(upperPrice)
	// (1 - Lu / (Lu + sqrt(pu)))
	oneSubLuDivLuWithUpSquared := num.DecimalOne().Sub(lu).Div(
		lu.Add(upperSquared),
	)

	// Lu * sqrt(pu) * (1 - Lu/(Lu+sqrt(pu)))
	return lu.Mul(upperSquared).Mul(oneSubLuDivLuWithUpSquared)
}

// Lu = (sqrt(pu) * sqrt(pl)) / (sqrt(pu) - sqrt(pl))
func liquidityUnit(sqrter *Sqrter, upperPrice, lowerPrice *num.Uint) num.Decimal {
	upperSquared := sqrter.sqrt(upperPrice)
	lowerSquared := sqrter.sqrt(lowerPrice)

	return upperSquared.Mul(lowerSquared).Div(
		upperSquared.Sub(lowerSquared),
	)
}
