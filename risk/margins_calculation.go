package risk

import (
	"math"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

func newMarginLevels(maintenance float64, scalingFactors *types.ScalingFactors) *types.MarginLevels {
	maintenance = math.Ceil(maintenance)
	return &types.MarginLevels{
		MaintenanceMargin:      uint64(maintenance),
		SearchLevel:            uint64(maintenance * scalingFactors.SearchLevel),
		InitialMargin:          uint64(maintenance * scalingFactors.InitialMargin),
		CollateralReleaseLevel: uint64(maintenance * scalingFactors.CollateralRelease),
	}
}

func addMarginLevels(ml *types.MarginLevels, maintenance float64, scalingFactors *types.ScalingFactors) {
	ml.MaintenanceMargin += uint64(maintenance)
	ml.SearchLevel += uint64(maintenance * scalingFactors.SearchLevel)
	ml.InitialMargin += uint64(maintenance * scalingFactors.InitialMargin)
	ml.CollateralReleaseLevel += uint64(maintenance * scalingFactors.CollateralRelease)
}

func (e *Engine) calculateAuctionMargins(m events.Margin, markPrice int64, rf types.RiskFactor) *types.MarginLevels {
	// calculate margins without order positions
	ml := e.calculateMargins(m, markPrice, rf, true, true)
	// now add the margin levels for orders
	long, short := m.Buy(), m.Sell()
	var (
		lMargin, sMargin float64
	)
	if long > 0 {
		lMargin = float64(long) * (rf.Long * float64(m.VWBuy()))
	}
	if short > 0 {
		sMargin = float64(short) * (rf.Short * float64(m.VWSell()))
	}
	// add buy/sell order margins to the margin requirements
	if lMargin > sMargin {
		addMarginLevels(ml, lMargin, e.marginCalculator.ScalingFactors)
	} else {
		addMarginLevels(ml, sMargin, e.marginCalculator.ScalingFactors)
	}
	// this is a bit of a hack, perhaps, but it keeps the remaining flow in the core simple:
	// artificially increase the release level so we never release the margin balance during auction
	ml.CollateralReleaseLevel += m.MarginBalance()
	return ml
}

// Implementation of the margin calculator per specs:
// https://github.com/vegaprotocol/product/blob/master/specs/0019-margin-calculator.md
func (e *Engine) calculateMargins(m events.Margin, markPrice int64, rf types.RiskFactor, withPotentialBuyAndSell, auction bool) *types.MarginLevels {
	var (
		marginMaintenanceLng float64
		marginMaintenanceSht float64
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

	// calculate margin maintenance long only if riskiest is > 0
	// marginMaintenanceLng will be 0 by default
	if riskiestLng > 0 {
		var (
			slippageVolume  = max(openVolume, 0)
			slippagePerUnit int64
		)
		if slippageVolume > 0 {
			var (
				exitPrice uint64
				err       error
			)
			if auction {
				exitPrice = e.ob.GetIndicativePrice()
			} else {
				exitPrice, err = e.ob.GetCloseoutPrice(uint64(slippageVolume), types.Side_SIDE_BUY)
				if err != nil && e.log.GetLevel() == logging.DebugLevel {
					e.log.Debug("got non critical error from GetCloseoutPrice for Buy side",
						logging.Error(err))
				}
			}
			slippagePerUnit = markPrice - int64(exitPrice)
		}

		if auction {
			marginMaintenanceLng = float64(slippageVolume)*(rf.Long*float64(markPrice)) + (float64(m.Buy()) * rf.Long * float64(markPrice))
		} else {
			marginMaintenanceLng = float64(max(slippageVolume*slippagePerUnit, 0)) + float64(slippageVolume)*(rf.Long*float64(markPrice)) + (float64(m.Buy()) * rf.Long *
				float64(markPrice))
		}
	}
	// calculate margin maintenance short only if riskiest is < 0
	// marginMaintenanceSht will be 0 by default
	if riskiestSht < 0 {
		var (
			slippageVolume  = min(openVolume, 0)
			slippagePerUnit int64
		)
		// slippageVolume would be negative we abs it in the next phase
		if slippageVolume < 0 {
			var (
				exitPrice uint64
				err       error
			)
			if auction {
				exitPrice = e.ob.GetIndicativePrice()
			} else {
				exitPrice, err = e.ob.GetCloseoutPrice(uint64(-slippageVolume), types.Side_SIDE_SELL)
				if err != nil && e.log.GetLevel() == logging.DebugLevel {
					e.log.Debug("got non critical error from GetCloseoutPrice for Sell side",
						logging.Error(err))
				}
			}
			slippagePerUnit = -1 * (markPrice - int64(exitPrice))
		}

		if auction {
			marginMaintenanceSht = float64(abs(slippageVolume))*(rf.Short*float64(markPrice)) + (float64(abs(m.Sell())) * rf.Short * float64(markPrice))
		} else {
			marginMaintenanceSht = float64(max(abs(slippageVolume)*slippagePerUnit, 0)) + float64(abs(slippageVolume))*(rf.Short*float64(markPrice)) + (float64(abs(m.Sell())) * rf.Short * float64(markPrice))
		}
	}

	// the greatest liability is the most positive number
	if marginMaintenanceLng > marginMaintenanceSht && marginMaintenanceLng > 0 {
		return newMarginLevels(marginMaintenanceLng, e.marginCalculator.ScalingFactors)
	}
	if marginMaintenanceSht > 0 {
		return newMarginLevels(marginMaintenanceSht, e.marginCalculator.ScalingFactors)
	}

	return &types.MarginLevels{}
}

func abs(a int64) int64 {
	if a < 0 {
		return -a
	}
	return a
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
