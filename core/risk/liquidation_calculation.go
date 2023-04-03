package risk

import (
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/libs/num"
)

type OrderInfo struct {
	Size  int64
	Price *num.Uint
}

func CalculateLiquidationPriceWithSlippageFactors(sizePosition int64, buyOrders, sellOrders []OrderInfo, currentPrice, collateralAvailable num.Decimal, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort num.Decimal) (liquidationPriceForOpenVolume, liquidationPriceWithSellOrders, liquidationPriceWithBuyOrders num.Decimal, err error) {
	openVolume := num.DecimalFromInt64(sizePosition).Div(positionFactor)

	if sizePosition != 0 {
		liquidationPriceForOpenVolume, err = calculateLiquidationPrice(openVolume, currentPrice, collateralAvailable, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort)
	}

	liquidationPriceWithSellOrders, liquidationPriceWithBuyOrders = liquidationPriceForOpenVolume, liquidationPriceForOpenVolume
	if err != nil || len(buyOrders)+len(sellOrders) == 0 {
		return
	}

	sort.Slice(buyOrders, func(i, j int) bool {
		return buyOrders[i].Price.GT(buyOrders[j].Price)
	})
	sort.Slice(sellOrders, func(i, j int) bool {
		return sellOrders[i].Price.LT(sellOrders[j].Price)
	})

	liquidationPriceWithBuyOrders, err = calculateLiquidationPriceWithOrders(liquidationPriceForOpenVolume, buyOrders, true, openVolume, currentPrice, collateralAvailable, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort)
	if err != nil {
		liquidationPriceWithBuyOrders = num.DecimalZero()
		return
	}
	liquidationPriceWithSellOrders, err = calculateLiquidationPriceWithOrders(liquidationPriceForOpenVolume, sellOrders, false, openVolume, currentPrice, collateralAvailable, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort)
	if err != nil {
		liquidationPriceWithSellOrders = num.DecimalZero()
		return
	}

	return liquidationPriceForOpenVolume, liquidationPriceWithBuyOrders, liquidationPriceWithSellOrders, nil
}

func calculateLiquidationPrice(openVolume num.Decimal, currentPrice, collateralAvailable num.Decimal, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort num.Decimal) (num.Decimal, error) {
	rf := riskFactorLong
	if openVolume.IsNegative() {
		rf = riskFactorShort
	}

	denominator := calculateSlippageFactor(openVolume, linearSlippageFactor, quadraticSlippageFactor).Add(openVolume.Abs().Mul(rf)).Sub(openVolume)
	if denominator.IsZero() {
		return num.DecimalZero(), fmt.Errorf("liquidation price not defined")
	}

	ret := collateralAvailable.Sub(openVolume.Mul(currentPrice)).Div(denominator)
	if ret.IsNegative() {
		return num.DecimalZero(), nil
	}
	return ret, nil
}

func calculateLiquidationPriceWithOrders(liquidationPriceOpenVolumeOnly num.Decimal, orders []OrderInfo, buySide bool, openVolume num.Decimal, currentPrice, collateralAvailable num.Decimal, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort num.Decimal) (num.Decimal, error) {
	var err error
	liquidationPrice := liquidationPriceOpenVolumeOnly
	exposureWithOrders := openVolume
	collateralWithOrders := collateralAvailable
	for _, o := range orders {
		price := num.DecimalFromUint(o.Price)
		if !exposureWithOrders.IsZero() && ((buySide && exposureWithOrders.IsPositive() && price.LessThan(liquidationPrice)) || (!buySide && exposureWithOrders.IsNegative() && price.GreaterThan(liquidationPrice))) {
			// party gets marked for closeout before this order gets a chance to fill
			break
		}
		mtm := exposureWithOrders.Mul(price.Sub(currentPrice))
		currentPrice = price

		collateralWithOrders = collateralWithOrders.Add(mtm)
		if buySide {
			exposureWithOrders = exposureWithOrders.Add(num.DecimalFromInt64(o.Size).Div(positionFactor))
		} else {
			exposureWithOrders = exposureWithOrders.Sub(num.DecimalFromInt64(o.Size).Div(positionFactor))
		}

		liquidationPrice, err = calculateLiquidationPrice(exposureWithOrders, price, collateralWithOrders, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort)
		if err != nil {
			return num.DecimalZero(), err
		}
	}
	return liquidationPrice, nil
}
