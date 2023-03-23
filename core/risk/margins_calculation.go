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
			noExit          = true
		)
		if slippageVolume.IsPositive() {
			if !auction {
				svol, _ := num.UintFromDecimal(slippageVolume.Abs().Mul(e.positionFactor))
				exitPrice, err := e.ob.GetCloseoutPrice(svol.Uint64(), types.SideBuy)
				if err != nil {
					if e.log.IsDebug() {
						e.log.Debug("got non critical error from GetCloseoutPrice for Buy side",
							logging.Error(err))
					}
				} else {
					noExit = false
					var negative bool
					slippagePerUnit, negative = num.UintZero().Delta(markPrice, exitPrice)
					if negative {
						slippagePerUnit = num.UintZero()
					}
				}
			}
		}

		minV := mPriceDec.Mul(e.linearSlippageFactor.Mul(slippageVolume).Add(e.quadraticSlippageFactor.Mul(slippageVolume.Mul(slippageVolume))))
		if auction {
			marginMaintenanceLng = minV.Add(slippageVolume.Mul(mPriceDec.Mul(rf.Long)))
			if withPotentialBuyAndSell {
				maintenanceMarginLongOpenOrders := m.BuySumProduct().ToDecimal().Div(e.positionFactor).Mul(rf.Long)
				marginMaintenanceLng = marginMaintenanceLng.Add(maintenanceMarginLongOpenOrders)
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
				minV = num.MinD(
					slip,
					minV,
				)
			}
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
		var (
			slippageVolume  = num.MinD(openVolume, num.DecimalZero())
			slippagePerUnit = num.UintZero()
			noExit          = true
		)
		// slippageVolume would be negative we abs it in the next phase
		if slippageVolume.IsNegative() {
			if !auction {
				// convert back into vol * 10^pdp
				svol, _ := num.UintFromDecimal(slippageVolume.Abs().Mul(e.positionFactor))
				exitPrice, err := e.ob.GetCloseoutPrice(svol.Uint64(), types.SideSell)
				if err != nil {
					if e.log.IsDebug() {
						e.log.Debug("got non critical error from GetCloseoutPrice for Sell side",
							logging.Error(err))
					}
				} else {
					noExit = false
					var negative bool
					slippagePerUnit, negative = num.UintZero().Delta(exitPrice, markPrice)
					if negative {
						slippagePerUnit = num.UintZero()
					}
				}
			}
		}
		absSlippageVolume := slippageVolume.Abs()
		linearSlippage := absSlippageVolume.Mul(e.linearSlippageFactor)
		quadraticSlipage := absSlippageVolume.Mul(absSlippageVolume).Mul(e.quadraticSlippageFactor)
		minV := mPriceDec.Mul(linearSlippage.Add(quadraticSlipage))
		if auction {
			marginMaintenanceSht = minV.Add(absSlippageVolume.Mul(mPriceDec.Mul(rf.Short)))
			if withPotentialBuyAndSell {
				maintenanceMarginShortOpenOrders := m.SellSumProduct().ToDecimal().Div(e.positionFactor).Mul(rf.Short)
				marginMaintenanceSht = marginMaintenanceSht.Add(maintenanceMarginShortOpenOrders)
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
