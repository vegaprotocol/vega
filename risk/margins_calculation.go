package risk

import (
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

func newMarginLevels(maintenance num.Decimal, scalingFactors *types.ScalingFactors) *types.MarginLevels {
	maintenance = maintenance.Ceil()
	mUint, _ := num.UintFromDecimal(maintenance)
	sl, _ := num.UintFromDecimal(maintenance.Mul(scalingFactors.SearchLevel))
	im, _ := num.UintFromDecimal(maintenance.Mul(scalingFactors.InitialMargin))
	cr, _ := num.UintFromDecimal(maintenance.Mul(scalingFactors.CollateralRelease))
	return &types.MarginLevels{
		MaintenanceMargin:      mUint,
		SearchLevel:            sl,
		InitialMargin:          im,
		CollateralReleaseLevel: cr,
	}
}

func addMarginLevels(ml *types.MarginLevels, maintenance num.Decimal, scalingFactors *types.ScalingFactors) {
	mtl, _ := num.UintFromDecimal(maintenance)
	sl, _ := num.UintFromDecimal(maintenance.Mul(scalingFactors.SearchLevel))
	im, _ := num.UintFromDecimal(maintenance.Mul(scalingFactors.InitialMargin))
	cr, _ := num.UintFromDecimal(maintenance.Mul(scalingFactors.CollateralRelease))
	ml.MaintenanceMargin.AddSum(mtl)
	ml.SearchLevel.AddSum(sl)
	ml.InitialMargin.AddSum(im)
	ml.CollateralReleaseLevel.AddSum(cr)
}

func (e *Engine) calculateAuctionMargins(m events.Margin, markPrice *num.Uint, rf types.RiskFactor) *types.MarginLevels {
	// calculate margins without order positions
	ml := e.calculateMargins(m, markPrice, rf, true, true)
	// now add the margin levels for orders
	long, short := num.DecimalFromFloat(float64(m.Buy())), num.DecimalFromFloat(float64(m.Sell()))
	zeroD := num.DecimalFromFloat(0)
	var (
		lMargin, sMargin num.Decimal
	)
	if long.GreaterThan(zeroD) {
		lMargin = long.Mul(rf.Long.Mul(m.VWBuy().ToDecimal()))
	}
	if short.GreaterThan(zeroD) {
		sMargin = short.Mul(rf.Short.Mul(m.VWSell().ToDecimal()))
	}
	// add buy/sell order margins to the margin requirements
	if lMargin.GreaterThan(sMargin) {
		addMarginLevels(ml, lMargin, e.marginCalculator.ScalingFactors)
	} else {
		addMarginLevels(ml, sMargin, e.marginCalculator.ScalingFactors)
	}
	// this is a bit of a hack, perhaps, but it keeps the remaining flow in the core simple:
	// artificially increase the release level so we never release the margin balance during auction
	ml.CollateralReleaseLevel.AddSum(m.MarginBalance())
	return ml
}

// Implementation of the margin calculator per specs:
// https://github.com/vegaprotocol/product/blob/master/specs/0019-margin-calculator.md
func (e *Engine) calculateMargins(m events.Margin, markPrice *num.Uint, rf types.RiskFactor, withPotentialBuyAndSell, auction bool) *types.MarginLevels {
	var (
		marginMaintenanceLng num.Decimal
		marginMaintenanceSht num.Decimal
	)
	openVolume := m.Size()
	var (
		riskiestLng = openVolume
		riskiestSht = openVolume
	)
	if withPotentialBuyAndSell {
		// calculate both long and short riskiest positions
		riskiestLng += m.Buy()
		riskiestSht -= m.Sell()
	}

	mPriceDec := markPrice.ToDecimal()
	zeroD := num.DecimalFromFloat(0)
	// calculate margin maintenance long only if riskiest is > 0
	// marginMaintenanceLng will be 0 by default
	if riskiestLng > 0 {
		var (
			slippageVolume  = num.DecimalFromFloat(float64(max(openVolume, 0)))
			slippagePerUnit = num.NewUint(0)
			negSlippage     bool
		)
		if slippageVolume.GreaterThan(zeroD) {
			var (
				exitPrice *num.Uint
				err       error
			)
			if auction {
				exitPrice = e.ob.GetIndicativePrice()
			} else {
				svol, _ := slippageVolume.Float64()
				exitPrice, err = e.ob.GetCloseoutPrice(uint64(svol), types.Side_SIDE_BUY)
				if err != nil && e.log.GetLevel() == logging.DebugLevel {
					e.log.Debug("got non critical error from GetCloseoutPrice for Buy side",
						logging.Error(err))
				}
			}
			slippagePerUnit, negSlippage = num.NewUint(0).Delta(markPrice, exitPrice)
		}

		bDec := num.DecimalFromFloat(float64(m.Buy()))
		if auction {
			marginMaintenanceLng = slippageVolume.Mul(rf.Long.Mul(mPriceDec)).Add(bDec.Mul(rf.Long).Mul(mPriceDec))
			// marginMaintenanceLng = float64(slippageVolume)*(rf.Long*float64(markPrice)) + (float64(m.Buy()) * rf.Long * float64(markPrice))
		} else {
			slip := slippagePerUnit.ToDecimal().Mul(slippageVolume)
			if negSlippage {
				slip = slip.Mul(num.DecimalFromFloat(-1))
			}
			marginMaintenanceLng = slippageVolume.Mul(rf.Long.Mul(mPriceDec)).Add(bDec.Mul(rf.Long).Mul(mPriceDec))
			if slip.GreaterThan(zeroD) {
				marginMaintenanceLng = marginMaintenanceLng.Add(slip)
			}
		}
	}
	// calculate margin maintenance short only if riskiest is < 0
	// marginMaintenanceSht will be 0 by default
	if riskiestSht < 0 {
		var (
			slippageVolume  = num.DecimalFromFloat(float64(min(openVolume, 0)))
			slippagePerUnit = num.NewUint(0)
		)
		// slippageVolume would be negative we abs it in the next phase
		if slippageVolume.LessThan(zeroD) {
			var (
				exitPrice *num.Uint
				err       error
			)
			if auction {
				exitPrice = e.ob.GetIndicativePrice()
			} else {
				svol, _ := slippageVolume.Abs().Float64()
				exitPrice, err = e.ob.GetCloseoutPrice(uint64(svol), types.Side_SIDE_SELL)
				if err != nil && e.log.GetLevel() == logging.DebugLevel {
					e.log.Debug("got non critical error from GetCloseoutPrice for Sell side",
						logging.Error(err))
				}
			}
			// exitPrice - markPrice == -1*(markPrice - exitPrice)
			slippagePerUnit, _ = num.NewUint(0).Delta(exitPrice, markPrice) // we don't care about neg/pos, we're using Abs() anyway
			// slippagePerUnit = -1 * (markPrice - int64(exitPrice))
		}

		sDec := num.DecimalFromFloat(float64(m.Sell()))
		if auction {
			marginMaintenanceSht = slippageVolume.Abs().Mul(rf.Short.Mul(mPriceDec)).Add(sDec.Mul(rf.Short).Mul(mPriceDec))
		} else {
			marginMaintenanceSht = slippageVolume.Abs().Mul(slippagePerUnit.ToDecimal()).Add(slippageVolume.Abs().Mul(rf.Short.Mul(mPriceDec)).Add(sDec.Abs().Mul(rf.Short).Mul(mPriceDec)))
		}
	}

	// the greatest liability is the most positive number
	if marginMaintenanceLng.GreaterThan(marginMaintenanceSht) && marginMaintenanceLng.GreaterThan(zeroD) {
		return newMarginLevels(marginMaintenanceLng, e.marginCalculator.ScalingFactors)
	}
	if marginMaintenanceSht.GreaterThan(zeroD) {
		return newMarginLevels(marginMaintenanceSht, e.marginCalculator.ScalingFactors)
	}

	return &types.MarginLevels{}
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
