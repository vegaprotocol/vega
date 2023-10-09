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

package risk

import (
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/libs/num"
)

type OrderInfo struct {
	Size          uint64
	Price         num.Decimal
	IsMarketOrder bool
}

func CalculateLiquidationPriceWithSlippageFactors(sizePosition int64, buyOrders, sellOrders []*OrderInfo, currentPrice, collateralAvailable num.Decimal, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort, fundingPaymentPerUnitPosition num.Decimal) (liquidationPriceForOpenVolume, liquidationPriceWithSellOrders, liquidationPriceWithBuyOrders num.Decimal, err error) {
	openVolume := num.DecimalFromInt64(sizePosition).Div(positionFactor)

	if sizePosition != 0 {
		liquidationPriceForOpenVolume, err = calculateLiquidationPrice(openVolume, currentPrice, collateralAvailable, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort, fundingPaymentPerUnitPosition)
	}

	liquidationPriceWithSellOrders, liquidationPriceWithBuyOrders = liquidationPriceForOpenVolume, liquidationPriceForOpenVolume
	if err != nil || len(buyOrders)+len(sellOrders) == 0 {
		return
	}

	// assume market orders will trade immediately
	for _, o := range buyOrders {
		if o.IsMarketOrder {
			o.Price = currentPrice
		}
	}

	for _, o := range sellOrders {
		if o.IsMarketOrder {
			o.Price = currentPrice
		}
	}

	sort.Slice(buyOrders, func(i, j int) bool {
		return buyOrders[i].Price.GreaterThan(buyOrders[j].Price)
	})
	sort.Slice(sellOrders, func(i, j int) bool {
		return sellOrders[i].Price.LessThan(sellOrders[j].Price)
	})

	liquidationPriceWithBuyOrders, err = calculateLiquidationPriceWithOrders(liquidationPriceForOpenVolume, buyOrders, true, openVolume, currentPrice, collateralAvailable, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort, fundingPaymentPerUnitPosition)
	if err != nil {
		liquidationPriceWithBuyOrders = num.DecimalZero()
		return
	}
	liquidationPriceWithSellOrders, err = calculateLiquidationPriceWithOrders(liquidationPriceForOpenVolume, sellOrders, false, openVolume, currentPrice, collateralAvailable, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort, fundingPaymentPerUnitPosition)
	if err != nil {
		liquidationPriceWithSellOrders = num.DecimalZero()
		return
	}

	return liquidationPriceForOpenVolume, liquidationPriceWithBuyOrders, liquidationPriceWithSellOrders, nil
}

func calculateLiquidationPrice(openVolume num.Decimal, currentPrice, collateralAvailable num.Decimal, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort, fundingPaymentPerUnitPosition num.Decimal) (num.Decimal, error) {
	if openVolume.IsZero() {
		return num.DecimalZero(), nil
	}

	rf := riskFactorLong
	if openVolume.IsNegative() {
		rf = riskFactorShort
	}

	denominator := calculateSlippageFactor(openVolume, linearSlippageFactor, quadraticSlippageFactor).Add(openVolume.Abs().Mul(rf)).Sub(openVolume)
	if denominator.IsZero() {
		return num.DecimalZero(), fmt.Errorf("liquidation price not defined")
	}

	numerator := collateralAvailable

	if !openVolume.IsZero() {
		numerator = numerator.Sub(openVolume.Mul(currentPrice))
		if !fundingPaymentPerUnitPosition.IsZero() {
			numerator = numerator.Sub(num.MaxD(num.DecimalZero(), openVolume.Mul(fundingPaymentPerUnitPosition)))
		}
	}

	ret := numerator.Div(denominator)
	if ret.IsNegative() {
		return num.DecimalZero(), nil
	}
	return ret, nil
}

func calculateLiquidationPriceWithOrders(liquidationPriceOpenVolumeOnly num.Decimal, orders []*OrderInfo, buySide bool, openVolume num.Decimal, currentPrice, collateralAvailable num.Decimal, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort, constant num.Decimal) (num.Decimal, error) {
	var err error
	liquidationPrice := liquidationPriceOpenVolumeOnly
	exposureWithOrders := openVolume
	collateralWithOrders := collateralAvailable
	for _, o := range orders {
		if !exposureWithOrders.IsZero() && ((buySide && exposureWithOrders.IsPositive() && o.Price.LessThan(liquidationPrice)) || (!buySide && exposureWithOrders.IsNegative() && o.Price.GreaterThan(liquidationPrice))) {
			// party gets marked for closeout before this order gets a chance to fill
			break
		}
		mtm := exposureWithOrders.Mul(o.Price.Sub(currentPrice))
		currentPrice = o.Price

		collateralWithOrders = collateralWithOrders.Add(mtm)
		if buySide {
			exposureWithOrders = exposureWithOrders.Add(num.DecimalFromInt64(int64(o.Size)).Div(positionFactor))
		} else {
			exposureWithOrders = exposureWithOrders.Sub(num.DecimalFromInt64(int64(o.Size)).Div(positionFactor))
		}

		liquidationPrice, err = calculateLiquidationPrice(exposureWithOrders, o.Price, collateralWithOrders, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort, constant)
		if err != nil {
			return num.DecimalZero(), err
		}
	}
	return liquidationPrice, nil
}
