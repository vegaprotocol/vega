package risk

import (
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
)

type orderInfo struct {
	size  uint64
	price *num.Uint
}

func EstimateLiquidationLevel(sizePosition int64, activeOrders []*vega.Order, currentPrice num.Decimal, collateralAvailable num.Uint, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort num.Decimal) (num.Decimal, error) {
	openVolume := num.DecimalFromInt64(sizePosition).Div(positionFactor)

	buyOrders := make([]orderInfo, 0, len(activeOrders))
	sellOrders := make([]orderInfo, 0, len(activeOrders))
	liquidationPrice := num.DecimalZero()
	for _, o := range activeOrders {
		r := o.GetRemaining()
		p, e := num.UintFromString(o.GetPrice(), 10)
		if e {
			return liquidationPrice, fmt.Errorf("could not parse %s to Uint", o.GetPrice())
		}
		s := o.GetSide()
		ord := orderInfo{size: r, price: p}
		if s == vega.Side_SIDE_BUY {
			buyOrders = append(buyOrders, ord)
			continue
		}
		sellOrders = append(sellOrders, ord)
	}

	sort.Slice(buyOrders, func(i, j int) bool {
		return buyOrders[i].price.GT(buyOrders[j].price)
	})
	sort.Slice(sellOrders, func(i, j int) bool {
		return sellOrders[i].price.LT(sellOrders[j].price)
	})

	// calculate liquidation price for position itself

	slippage_factor := CalculateSlippageFactor(openVolume, linearSlippageFactor, quadraticSlippageFactor)

	rf := riskFactorLong
	if sizePosition < 0 {
		rf = riskFactorShort
	}

	liquidationPrice = collateralAvailable.ToDecimal().Sub(openVolume.Mul(currentPrice)).Div(slippage_factor.Add(openVolume.Abs().Mul(rf)).Sub(openVolume))

	return liquidationPrice, nil
}
