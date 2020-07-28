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

func (r *Engine) calculateAuctionMargin(e events.Margin, rf types.RiskFactor, o *types.Order) *types.MarginLevels {
	factor := rf.Long
	if o.Side == types.Side_SIDE_SELL {
		factor = rf.Short
	}
	maintenance := float64(o.Size) * (factor * float64(o.Price))
	return newMarginLevels(maintenance, r.marginCalculator.ScalingFactors)
}

// Implementation of the margin calculator per specs:
// https://github.com/vegaprotocol/product/blob/master/specs/0019-margin-calculator.md
func (r *Engine) calculateMargins(e events.Margin, markPrice int64, rf types.RiskFactor, withPotentialBuyAndSell bool) *types.MarginLevels {
	var (
		marginMaintenanceLng float64
		marginMaintenanceSht float64
	)
	openVolume := e.Size()
	var (
		riskiestLng = openVolume
		riskiestSht = openVolume
	)
	if withPotentialBuyAndSell {
		// calculate both long and short riskiest positions
		riskiestLng += e.Buy()
		riskiestSht -= e.Sell()
	}

	// calculate margin maintenance long only if riskiest is > 0
	// marginMaintenanceLng will be 0 by default
	if riskiestLng > 0 {
		var (
			slippageVolume  = max(openVolume, 0)
			slippagePerUnit int64
		)
		if slippageVolume > 0 {
			exitPrice, err := r.ob.GetCloseoutPrice(uint64(slippageVolume), types.Side_SIDE_BUY)
			if err != nil && r.log.GetLevel() == logging.DebugLevel {
				r.log.Debug("got non critical error from GetCloseoutPrice for Buy side",
					logging.Error(err))
			}
			slippagePerUnit = markPrice - int64(exitPrice)
		}

		marginMaintenanceLng = float64(max(slippageVolume*slippagePerUnit, 0)) + float64(slippageVolume)*(rf.Long*float64(markPrice)) + (float64(e.Buy()) * rf.Long *
			float64(markPrice))
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
			exitPrice, err := r.ob.GetCloseoutPrice(uint64(-slippageVolume), types.Side_SIDE_SELL)
			if err != nil && r.log.GetLevel() == logging.DebugLevel {
				r.log.Debug("got non critical error from GetCloseoutPrice for Sell side",
					logging.Error(err))
			}
			slippagePerUnit = -1 * (markPrice - int64(exitPrice))
		}

		marginMaintenanceSht = float64(max(abs(slippageVolume)*slippagePerUnit, 0)) + float64(abs(slippageVolume))*(rf.Short*float64(markPrice)) + (float64(abs(e.Sell())) * rf.Short * float64(markPrice))
	}

	// the greatest liability is the most positive number
	if marginMaintenanceLng > marginMaintenanceSht && marginMaintenanceLng > 0 {
		return newMarginLevels(marginMaintenanceLng, r.marginCalculator.ScalingFactors)
	}
	if marginMaintenanceSht > 0 {
		return newMarginLevels(marginMaintenanceSht, r.marginCalculator.ScalingFactors)
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
