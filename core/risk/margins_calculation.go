// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package risk

import (
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
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
	}
}

func addMarginLevels(ml *types.MarginLevels, maintenance num.Decimal, scalingFactors *scalingFactorsUint) {
	mtl, _ := num.UintFromDecimal(maintenance.Ceil())
	ml.MaintenanceMargin.AddSum(mtl)
	ml.SearchLevel.AddSum(num.UintZero().Div(num.UintZero().Mul(scalingFactors.search, mtl), exp))
	ml.InitialMargin.AddSum(num.UintZero().Div(num.UintZero().Mul(scalingFactors.initial, mtl), exp))
	ml.CollateralReleaseLevel.AddSum(num.UintZero().Div(num.UintZero().Mul(scalingFactors.release, mtl), exp))
}

func (e *Engine) calculateAuctionMargins(m events.Margin, markPrice *num.Uint, rf types.RiskFactor) *types.MarginLevels {
	// calculate margins without order positions
	ml := e.calculateMargins(m, markPrice, rf, false, true)
	// now add the margin levels for orders
	long, short := m.BuySumProduct().ToDecimal().Div(e.positionFactor), m.SellSumProduct().ToDecimal().Div(e.positionFactor)
	var lMargin, sMargin num.Decimal
	if long.IsPositive() {
		lMargin = long.Mul(rf.Long)
	}
	if short.IsPositive() {
		sMargin = short.Mul(rf.Short)
	}
	// add buy/sell order margins to the margin requirements
	if lMargin.GreaterThan(sMargin) {
		addMarginLevels(ml, lMargin, e.scalingFactorsUint)
	} else {
		addMarginLevels(ml, sMargin, e.scalingFactorsUint)
	}
	return ml
}

// Implementation of the margin calculator per specs:
// https://github.com/vegaprotocol/product/blob/master/specs/0019-margin-calculator.md
func (e *Engine) calculateMargins(m events.Margin, markPrice *num.Uint, rf types.RiskFactor, withPotentialBuyAndSell, auction bool) *types.MarginLevels {
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

	mPriceDec := markPrice.ToDecimal()
	// calculate margin maintenance long only if riskiest is > 0
	// marginMaintenanceLng will be 0 by default
	if riskiestLng.IsPositive() {
		var (
			slippageVolume  = num.MaxD(openVolume, num.DecimalZero())
			slippagePerUnit = num.UintZero()
			negSlippage     bool
			noExit          bool
		)
		if slippageVolume.IsPositive() {
			var (
				exitPrice *num.Uint
				err       error
			)
			if auction {
				noExit = true
			} else {
				svol, _ := num.UintFromDecimal(slippageVolume.Abs().Mul(e.positionFactor))
				exitPrice, err = e.ob.GetCloseoutPrice(svol.Uint64(), types.SideBuy)
				if err != nil {
					noExit = true
					if e.log.IsDebug() {
						e.log.Debug("got non critical error from GetCloseoutPrice for Buy side",
							logging.Error(err))
					}
				}
				slippagePerUnit, negSlippage = num.UintZero().Delta(markPrice, exitPrice)
			}
		}

		bDec := num.DecimalFromInt64(m.Buy()).Div(e.positionFactor)
		minV := mPriceDec.Mul(e.linearSlippageFactor.Mul(slippageVolume).Add(e.quadraticSlippageFactor.Mul(slippageVolume.Mul(slippageVolume))))
		if auction {
			marginMaintenanceLng = minV.Add(slippageVolume.Mul(mPriceDec.Mul(rf.Long)))
			if withPotentialBuyAndSell {
				marginMaintenanceLng = marginMaintenanceLng.Add(bDec.Mul(rf.Long).Mul(mPriceDec))
			}
		} else {
			// 	maintenance_margin_long_open_position =
			//  	max(
			// 			min(
			// 				slippage_volume * slippage_per_unit,
			// 				mark_price * (slippage_volume * market.maxSlippageFraction[1] + slippage_volume^2 * market.maxSlippageFraction[2])
			// 				),
			//		  	0
			// 		) + slippage_volume * [ quantitative_model.risk_factors_long ] . [ Product.value(market_observable) ]
			//
			// maintenance_margin_long_open_orders = buy_orders * [ quantitative_model.risk_factors_long ] . [ Product.value(market_observable) ]
			//
			// if we don't have an exit price, the min() part of the formula above =
			//
			// mark_price * (slippage_volume * market.maxSlippageFraction[1] + slippage_volume^2 * market.maxSlippageFraction[2])
			// changing the formula to:
			// maintenance_margin_long_open_position =
			//	max(
			//		mark_price * (slippage_volume * market.linearSlippageFactor + slippage_volume^2 * market.quadraticSlippageFactor),
			//		0
			//	) + slippage_volume * [quantitative_model.risk_factors_long] . [ Product.value(market_observable) ]

			if !noExit {
				slip := slippagePerUnit.ToDecimal().Mul(slippageVolume)
				if negSlippage {
					slip = slip.Mul(num.DecimalFromInt64(-1))
				}
				minV = num.MinD(
					slip,
					minV,
				)
			}
			maintenanceMarginLongOpenPosition := num.MaxD(
				num.DecimalZero(),
				minV,
			).Add(slippageVolume.Mul(rf.Long).Mul(mPriceDec))
			maintenanceMarginLongOpenOrders := bDec.Mul(rf.Long).Mul(mPriceDec)
			marginMaintenanceLng = maintenanceMarginLongOpenPosition.Add(maintenanceMarginLongOpenOrders)
		}
	}
	// calculate margin maintenance short only if riskiest is < 0
	// marginMaintenanceSht will be 0 by default
	if riskiestSht.IsNegative() {
		var (
			slippageVolume  = num.MinD(openVolume, num.DecimalZero())
			slippagePerUnit = num.UintZero()
			noExit          = false
		)
		// slippageVolume would be negative we abs it in the next phase
		if slippageVolume.IsNegative() {
			var (
				exitPrice *num.Uint
				err       error
			)
			if auction {
				noExit = true
			} else {
				// convert back into vol * 10^pdp
				svol, _ := num.UintFromDecimal(slippageVolume.Abs().Mul(e.positionFactor))
				exitPrice, err = e.ob.GetCloseoutPrice(svol.Uint64(), types.SideSell)
				if err != nil {
					noExit = true
					if e.log.IsDebug() {
						e.log.Debug("got non critical error from GetCloseoutPrice for Sell side",
							logging.Error(err))
					}
				}
				// exitPrice - markPrice == -1*(markPrice - exitPrice)
				slippagePerUnit, _ = num.UintZero().Delta(exitPrice, markPrice) // we don't care about neg/pos, we're using Abs() anyway
				// slippagePerUnit = -1 * (markPrice - int64(exitPrice))
			}
		}
		sDec := num.DecimalFromInt64(m.Sell()).Div(e.positionFactor)
		absSlippageVolume := slippageVolume.Abs()
		linearSlippage := absSlippageVolume.Mul(e.linearSlippageFactor)
		quadraticSlipage := absSlippageVolume.Mul(absSlippageVolume).Mul(e.quadraticSlippageFactor)
		minV := mPriceDec.Mul(linearSlippage.Add(quadraticSlipage))
		if auction {
			marginMaintenanceSht = minV.Add(absSlippageVolume.Mul(mPriceDec.Mul(rf.Short)))
			if withPotentialBuyAndSell {
				marginMaintenanceSht = marginMaintenanceSht.Add(sDec.Mul(rf.Short).Mul(mPriceDec))
			}
		} else {
			// maintenance_margin_short_open_position =
			// 		max(
			//			min(
			//					abs(slippage_volume) * slippage_per_unit,
			//					mark_price * market.maxSlippageFraction[1] + abs(slippage_volume)^2 * market.maxSlippageFraction[2])
			//			   ),
			//			0
			//		) + abs(slippage_volume) * [ quantitative_model.risk_factors_short ] . [ Product.value(market_observable) ]
			//
			// maintenance_margin_short_open_orders = abs(sell_orders) * [ quantitative_model.risk_factors_short ] . [ Product.value(market_observable) ]
			//
			// again the same applies here if the exit_price is invalid (+Infinity, or some other error):
			// we replace the min part in the formula above, ignoring abs(slippage_volume) * slippage_per_unit:
			// 		max(
			//			mark_price * market.maxSlippageFraction[1] + abs(slippage_volume)^2 * market.maxSlippageFraction[2])
			//			0
			//		) + abs(slippage_volume) * [ quantitative_model.risk_factors_short ] . [ Product.value(market_observable) ]
			//
			// maintenance_margin_short_open_orders = abs(sell_orders) * [ quantitative_model.risk_factors_short ] . [ Product.value(market_observable) ]
			if !noExit {
				minV = num.MinD(
					absSlippageVolume.Mul(slippagePerUnit.ToDecimal()),
					minV,
				)
			}
			maintenanceMarginShortOpenPosition := num.MaxD(
				num.DecimalZero(),
				minV,
			).Add(absSlippageVolume.Mul(mPriceDec).Mul(rf.Short))
			maintenanceMarginShortOpenOrders := sDec.Abs().Mul(mPriceDec).Mul(rf.Short)
			marginMaintenanceSht = maintenanceMarginShortOpenPosition.Add(maintenanceMarginShortOpenOrders)
		}
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
	}
}
