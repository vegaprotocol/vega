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
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

var (
	exp    = num.UintZero().Exp(num.NewUint(10), num.NewUint(5))
	expDec = num.DecimalFromUint(exp)
)

type scalingFactorsUint struct {
	search  *num.Uint
	initial *num.Uint
	release *num.Uint
}

func scalingFactorsUintFromDecimals(sf *types.ScalingFactors) *scalingFactorsUint {
	search, _ := num.UintFromDecimal(sf.SearchLevel.Mul(expDec))
	initial, _ := num.UintFromDecimal(sf.InitialMargin.Mul(expDec))
	release, _ := num.UintFromDecimal(sf.CollateralRelease.Mul(expDec))

	return &scalingFactorsUint{
		search:  search,
		initial: initial,
		release: release,
	}
}

func newMarginLevels(maintenance num.Decimal, scalingFactors *scalingFactorsUint) *types.MarginLevels {
	umaintenance, _ := num.UintFromDecimal(maintenance.Ceil())
	return &types.MarginLevels{
		MaintenanceMargin:      umaintenance,
		SearchLevel:            num.UintZero().Div(num.UintZero().Mul(scalingFactors.search, umaintenance), exp),
		InitialMargin:          num.UintZero().Div(num.UintZero().Mul(scalingFactors.initial, umaintenance), exp),
		CollateralReleaseLevel: num.UintZero().Div(num.UintZero().Mul(scalingFactors.release, umaintenance), exp),
		OrderMargin:            num.UintZero(),
		MarginMode:             types.MarginModeCrossMargin,
		MarginFactor:           num.DecimalZero(),
	}
}

// Implementation of the margin calculator per specs:
// https://github.com/vegaprotocol/product/blob/master/specs/0019-margin-calculator.md
func (e *Engine) calculateMargins(m events.Margin, markPrice *num.Uint, rf types.RiskFactor, withPotentialBuyAndSell, auction bool, inc num.Decimal, auctionPrice *num.Uint) *types.MarginLevels {
	var (
		marginMaintenanceLng num.Decimal
		marginMaintenanceSht num.Decimal
	)
	// convert volumn to a decimal number from a * 10^pdp
	openVolume := num.DecimalFromInt64(m.Size()).Div(e.positionFactor)
	var (
		riskiestLng = openVolume
		riskiestSht = openVolume
	)
	if withPotentialBuyAndSell {
		// calculate both long and short riskiest positions
		riskiestLng = riskiestLng.Add(num.DecimalFromInt64(m.Buy()).Div(e.positionFactor))
		riskiestSht = riskiestSht.Sub(num.DecimalFromInt64(m.Sell()).Div(e.positionFactor))
	}
	// the party has no open positions that we need to calculate margin for
	if riskiestLng.IsZero() && riskiestSht.IsZero() {
		return &types.MarginLevels{
			MaintenanceMargin:      num.UintZero(),
			SearchLevel:            num.UintZero(),
			InitialMargin:          num.UintZero(),
			CollateralReleaseLevel: num.UintZero(),
			OrderMargin:            num.UintZero(),
			MarginMode:             types.MarginModeCrossMargin,
			MarginFactor:           num.DecimalZero(),
		}
	}

	mPriceDec := markPrice.ToDecimal()
	// calculate margin maintenance long only if riskiest is > 0
	// marginMaintenanceLng will be 0 by default
	if riskiestLng.IsPositive() {
		slippageVolume := num.MaxD(openVolume, num.DecimalZero())
		minV := mPriceDec.Mul(e.linearSlippageFactor.Mul(slippageVolume).Add(e.quadraticSlippageFactor.Mul(slippageVolume.Mul(slippageVolume))))
		if auction {
			marginMaintenanceLng = minV.Add(slippageVolume.Mul(mPriceDec.Mul(rf.Long)))
			if withPotentialBuyAndSell {
				p := m.BuySumProduct()
				if auctionPrice != nil {
					p = num.Max(p, num.UintZero().Mul(num.UintFromUint64(uint64(m.Buy())), auctionPrice))
				}
				maintenanceMarginLongOpenOrders := p.ToDecimal().Div(e.positionFactor).Mul(rf.Long)
				marginMaintenanceLng = marginMaintenanceLng.Add(maintenanceMarginLongOpenOrders)
			}
		} else {
			// 	maintenance_margin_long_open_position =
			//  	max(
			//              0,
			// 				mark_price * (slippage_volume * market.maxSlippageFraction[1] + slippage_volume^2 * market.maxSlippageFraction[2])
			// 		) + slippage_volume * [ quantitative_model.risk_factors_long ] . [ Product.value(market_observable) ]
			//
			// maintenance_margin_long_open_orders = buy_orders * [ quantitative_model.risk_factors_long ] . [ Product.value(market_observable) ]
			marginMaintenanceLng = num.MaxD(
				num.DecimalZero(),
				minV,
			).Add(slippageVolume.Mul(rf.Long).Mul(mPriceDec))
			if withPotentialBuyAndSell {
				bDec := num.DecimalFromInt64(m.Buy()).Div(e.positionFactor)
				maintenanceMarginLongOpenOrders := bDec.Mul(rf.Long).Mul(mPriceDec)
				marginMaintenanceLng = marginMaintenanceLng.Add(maintenanceMarginLongOpenOrders)
			}
		}
	}
	// calculate margin maintenance short only if riskiest is < 0
	// marginMaintenanceSht will be 0 by default
	if riskiestSht.IsNegative() {
		absSlippageVolume := num.MinD(openVolume, num.DecimalZero()).Abs()
		linearSlippage := absSlippageVolume.Mul(e.linearSlippageFactor)
		quadraticSlipage := absSlippageVolume.Mul(absSlippageVolume).Mul(e.quadraticSlippageFactor)
		minV := mPriceDec.Mul(linearSlippage.Add(quadraticSlipage))
		if auction {
			marginMaintenanceSht = minV.Add(absSlippageVolume.Mul(mPriceDec.Mul(rf.Short)))
			if withPotentialBuyAndSell {
				p := m.SellSumProduct()
				if auctionPrice != nil {
					p = num.Max(p, num.UintZero().Mul(num.UintFromUint64(uint64(m.Sell())), auctionPrice))
				}
				maintenanceMarginShortOpenOrders := p.ToDecimal().Div(e.positionFactor).Mul(rf.Short)
				marginMaintenanceSht = marginMaintenanceSht.Add(maintenanceMarginShortOpenOrders)
			}
		} else {
			// maintenance_margin_short_open_position =
			// 		max(
			//					0,
			//					mark_price * market.maxSlippageFraction[1] + abs(slippage_volume)^2 * market.maxSlippageFraction[2])
			//		) + abs(slippage_volume) * [ quantitative_model.risk_factors_short ] . [ Product.value(market_observable) ]
			//
			// maintenance_margin_short_open_orders = abs(sell_orders) * [ quantitative_model.risk_factors_short ] . [ Product.value(market_observable) ]
			marginMaintenanceSht = num.MaxD(
				num.DecimalZero(),
				minV,
			).Add(absSlippageVolume.Mul(mPriceDec).Mul(rf.Short))
			if withPotentialBuyAndSell {
				sDec := num.DecimalFromInt64(m.Sell()).Div(e.positionFactor)
				maintenanceMarginShortOpenOrders := sDec.Abs().Mul(mPriceDec).Mul(rf.Short)
				marginMaintenanceSht = marginMaintenanceSht.Add(maintenanceMarginShortOpenOrders)
			}
		}
	}

	if !inc.IsZero() && !openVolume.IsZero() {
		// openVolume and inc are signed, but this is fine, we only apply the positive values
		incD := num.MaxD(num.DecimalZero(), inc.Mul(openVolume))
		marginMaintenanceLng = marginMaintenanceLng.Add(incD)
		marginMaintenanceSht = marginMaintenanceSht.Add(incD)
	}

	// the greatest liability is the most positive number
	if marginMaintenanceLng.GreaterThan(marginMaintenanceSht) && marginMaintenanceLng.IsPositive() {
		return newMarginLevels(marginMaintenanceLng, e.scalingFactorsUint)
	}
	if marginMaintenanceSht.IsPositive() {
		return newMarginLevels(marginMaintenanceSht, e.scalingFactorsUint)
	}

	return &types.MarginLevels{
		MaintenanceMargin:      num.UintZero(),
		SearchLevel:            num.UintZero(),
		InitialMargin:          num.UintZero(),
		CollateralReleaseLevel: num.UintZero(),
		OrderMargin:            num.UintZero(),
		MarginMode:             types.MarginModeCrossMargin,
		MarginFactor:           num.DecimalZero(),
	}
}

func CalculateMaintenanceMarginWithSlippageFactors(sizePosition int64, buyOrders, sellOrders []*OrderInfo, marketObservable, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort, fundingPaymntPerUnitPosition num.Decimal, auction bool, auctionPrice num.Decimal) num.Decimal {
	buySumProduct, sellSumProduct := num.DecimalZero(), num.DecimalZero()
	sizeSells, sizeBuys := int64(0), int64(0)
	for _, o := range buyOrders {
		size := int64(o.TrueRemaining)
		if o.IsMarketOrder {
			// assume market order fills
			sizePosition += size
		} else {
			buySumProduct = buySumProduct.Add(num.DecimalFromInt64(size).Mul(o.Price))
			sizeBuys += size
		}
	}
	for _, o := range sellOrders {
		size := int64(o.TrueRemaining)
		if o.IsMarketOrder {
			// assume market order fills
			sizePosition -= size
		} else {
			sellSumProduct = sellSumProduct.Add(num.DecimalFromInt64(size).Mul(o.Price))
			sizeSells += size
		}
	}
	return computeMaintenanceMargin(sizePosition, sizeBuys, sizeSells, buySumProduct, sellSumProduct, marketObservable, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort, fundingPaymntPerUnitPosition, auction, auctionPrice)
}

func calculateSlippageFactor(slippageVolume, linearSlippageFactor, quadraticSlippageFactor num.Decimal) num.Decimal {
	return linearSlippageFactor.Mul(slippageVolume.Abs()).Add(quadraticSlippageFactor.Mul(slippageVolume.Mul(slippageVolume)))
}

func computeMaintenanceMargin(sizePosition, buySize, sellSize int64, buySumProduct, sellSumProduct, marketObservable, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactorLong, riskFactorShort, fundingPaymntPerUnitPosition num.Decimal, auction bool, auctionPrice num.Decimal) num.Decimal {
	var (
		marginMaintenanceLng num.Decimal
		marginMaintenanceSht num.Decimal
	)
	// convert volumn to a decimal number from a * 10^pdp
	openVolume := num.DecimalFromInt64(sizePosition).Div(positionFactor)
	// calculate both long and short riskiest positions
	var (
		riskiestLng = openVolume.Add(num.DecimalFromInt64(buySize).Div(positionFactor))
		riskiestSht = openVolume.Sub(num.DecimalFromInt64(sellSize).Div(positionFactor))
	)

	// calculate margin maintenance long only if riskiest is > 0
	// marginMaintenanceLng will be 0 by default
	if riskiestLng.IsPositive() {
		slippageVolume := num.MaxD(openVolume, num.DecimalZero())
		slippageCap := marketObservable.Mul(calculateSlippageFactor(slippageVolume, linearSlippageFactor, quadraticSlippageFactor))
		if auction {
			marginMaintenanceLng = slippageCap.Add(slippageVolume.Mul(marketObservable.Mul(riskFactorLong)))
			p := buySumProduct
			if !auctionPrice.IsZero() {
				p = num.MaxD(p, auctionPrice.Mul(num.DecimalFromInt64(buySize)))
			}
			maintenanceMarginLongOpenOrders := p.Div(positionFactor).Mul(riskFactorLong)
			marginMaintenanceLng = marginMaintenanceLng.Add(maintenanceMarginLongOpenOrders)
		} else {
			marginMaintenanceLng = num.MaxD(
				num.DecimalZero(),
				slippageCap,
			).Add(slippageVolume.Mul(riskFactorLong).Mul(marketObservable))
			if buySize > 0 {
				maintenanceMarginLongOpenOrders := num.DecimalFromInt64(buySize).Div(positionFactor).Mul(riskFactorLong).Mul(marketObservable)
				marginMaintenanceLng = marginMaintenanceLng.Add(maintenanceMarginLongOpenOrders)
			}
		}
	}
	// calculate margin maintenance short only if riskiest is < 0
	// marginMaintenanceSht will be 0 by default
	if riskiestSht.IsNegative() {
		slippageVolume := num.MinD(openVolume, num.DecimalZero())
		absSlippageVolume := slippageVolume.Abs()
		slippageCap := marketObservable.Mul(calculateSlippageFactor(slippageVolume, linearSlippageFactor, quadraticSlippageFactor))
		if auction {
			marginMaintenanceSht = slippageCap.Add(absSlippageVolume.Mul(marketObservable.Mul(riskFactorShort)))
			p := sellSumProduct
			if !auctionPrice.IsZero() {
				p = num.MaxD(p, auctionPrice.Mul(num.DecimalFromInt64(sellSize)))
			}
			maintenanceMarginShortOpenOrders := p.Div(positionFactor).Mul(riskFactorShort)
			marginMaintenanceSht = marginMaintenanceSht.Add(maintenanceMarginShortOpenOrders)
		} else {
			marginMaintenanceSht = num.MaxD(
				num.DecimalZero(),
				slippageCap,
			).Add(absSlippageVolume.Mul(marketObservable).Mul(riskFactorShort))
			if sellSize > 0 {
				maintenanceMarginShortOpenOrders := num.DecimalFromInt64(sellSize).Div(positionFactor).Abs().Mul(marketObservable).Mul(riskFactorShort)
				marginMaintenanceSht = marginMaintenanceSht.Add(maintenanceMarginShortOpenOrders)
			}
		}
	}

	if !fundingPaymntPerUnitPosition.IsZero() && !openVolume.IsZero() {
		// calculate margin increase based on position
		// incD = max(0, inc * open volume)
		incD := num.MaxD(num.DecimalZero(), fundingPaymntPerUnitPosition.Mul(openVolume))
		marginMaintenanceLng = marginMaintenanceLng.Add(incD)
		marginMaintenanceSht = marginMaintenanceSht.Add(incD)
	}

	// the greatest liability is the most positive number
	if marginMaintenanceLng.GreaterThan(marginMaintenanceSht) && marginMaintenanceLng.IsPositive() {
		return marginMaintenanceLng
	}
	if marginMaintenanceSht.IsPositive() {
		return marginMaintenanceSht
	}
	return num.DecimalZero()
}

// CalcOrderMarginIsolatedMode calculates the the order margin required for the party in isolated margin mode given their current orders and margin factor.
func CalcOrderMarginIsolatedMode(positionSize int64, buyOrders, sellOrders []*OrderInfo, positionFactor, marginFactor, auctionPrice num.Decimal) num.Decimal {
	// sort orders from best to worst
	sort.Slice(buyOrders, func(i, j int) bool { return buyOrders[i].Price.GreaterThan(buyOrders[j].Price) })
	sort.Slice(sellOrders, func(i, j int) bool { return sellOrders[i].Price.LessThan(sellOrders[j].Price) })

	// calc the side margin
	marginByBuy := calcOrderSideMarginIsolatedMode(positionSize, buyOrders, positionFactor, marginFactor, auctionPrice, true)
	marginBySell := calcOrderSideMarginIsolatedMode(positionSize, sellOrders, positionFactor, marginFactor, auctionPrice, false)
	if marginBySell.GreaterThan(marginByBuy) {
		return marginBySell
	}
	return marginByBuy
}

func calcOrderSideMarginIsolatedMode(currentPosition int64, orders []*OrderInfo, positionFactor, marginFactor num.Decimal, auctionPrice num.Decimal, buy bool) num.Decimal {
	for _, o := range orders {
		if o.IsMarketOrder {
			// assume market order fills
			if buy {
				currentPosition += int64(o.TrueRemaining)
			} else {
				currentPosition -= int64(o.TrueRemaining)
			}
		}
	}

	margin := num.DecimalZero()
	remainingCovered := int64Abs(currentPosition)
	for _, o := range orders {
		size := o.TrueRemaining
		// for long position we don't need to count margin for the top <currentPosition> size for sell orders
		// for short position we don't need to count margin for the top <currentPosition> size for buy orders
		if remainingCovered != 0 && (buy && currentPosition < 0) || (!buy && currentPosition > 0) {
			if size >= remainingCovered { // part of the order doesn't require margin
				size = size - remainingCovered
				remainingCovered = 0
			} else { // the entire order doesn't require margin
				remainingCovered -= size
				size = 0
			}
		}
		if size > 0 {
			// if we're in auction we need to use the larger between auction price (which is the max(indicativePrice, markPrice)) and the order price
			p := o.Price
			if auctionPrice.GreaterThan(p) {
				p = auctionPrice
			}
			// add the margin for the given order
			margin = margin.Add(num.DecimalFromInt64(int64(size)).Mul(p))
		}
	}
	// factor the margin by margin factor and divide by position factor to get to the right decimals
	return margin.Mul(marginFactor).Div(positionFactor)
}

func CalculateRequiredMarginInIsolatedMode(sizePosition int64, averageEntryPrice, marketObservable num.Decimal, buyOrders, sellOrders []*OrderInfo, positionFactor, marginFactor num.Decimal, auctionPrice *num.Uint) (num.Decimal, num.Decimal) {
	marketOrderAdjustedPositionNotional := averageEntryPrice.Copy().Mul(num.DecimalFromInt64(sizePosition))
	var orders []*types.Order = make([]*types.Order, 0, len(buyOrders)+len(sellOrders))

	// assume market orders fill immediately at marketObservable price
	for _, o := range buyOrders {
		if o.IsMarketOrder {
			sizePosition += int64(o.TrueRemaining)
			marketOrderAdjustedPositionNotional = marketOrderAdjustedPositionNotional.Add(marketObservable.Mul(num.DecimalFromInt64(int64(o.TrueRemaining))))
		} else {
			price, _ := num.UintFromDecimal(o.Price)
			ord := &types.Order{
				Status:    types.OrderStatusActive,
				Remaining: o.TrueRemaining,
				Price:     price,
				Side:      types.SideBuy,
			}
			orders = append(orders, ord)
		}
	}
	for _, o := range sellOrders {
		if o.IsMarketOrder {
			sizePosition -= int64(o.TrueRemaining)
			marketOrderAdjustedPositionNotional = marketOrderAdjustedPositionNotional.Sub(marketObservable.Mul(num.DecimalFromInt64(int64(o.TrueRemaining))))
		} else {
			price, _ := num.UintFromDecimal(o.Price)
			ord := &types.Order{
				Status:    types.OrderStatusActive,
				Remaining: o.TrueRemaining,
				Price:     price,
				Side:      types.SideSell,
			}
			orders = append(orders, ord)
		}
	}

	requiredPositionMargin := marketOrderAdjustedPositionNotional.Abs().Mul(marginFactor).Div(positionFactor)
	requiredOrderMargin := CalcOrderMargins(sizePosition, orders, positionFactor, marginFactor, auctionPrice)

	return requiredPositionMargin, requiredOrderMargin.ToDecimal()
}
