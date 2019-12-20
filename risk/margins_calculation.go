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
		MaintenanceMargin:      int64(maintenance),
		SearchLevel:            int64(maintenance * scalingFactors.SearchLevel),
		InitialMargin:          int64(maintenance * scalingFactors.InitialMargin),
		CollateralReleaseLevel: int64(maintenance * scalingFactors.CollateralRelease),
	}
}

// Implementation of the margin calculator per specs:
// https://gitlab.com/vega-protocol/product/blob/master/specs/0019-margin-calculator.md
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
			exitPrice, err := r.ob.GetCloseoutPrice(uint64(slippageVolume), types.Side_Buy)
			if err != nil {
				r.log.Warn("got non critical error from GetCloseoutPrice for Buy side",
					logging.Error(err))
			}
			slippagePerUnit = int64(exitPrice) - markPrice
		}
		marginMaintenanceLng = float64(slippageVolume)*(float64(slippagePerUnit)+(rf.Long*float64(markPrice))) + (float64(e.Buy()) * rf.Long * float64(markPrice))
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
			exitPrice, err := r.ob.GetCloseoutPrice(uint64(-slippageVolume), types.Side_Sell)
			if err != nil {
				r.log.Warn("got non critical error from GetCloseoutPrice for Sell side",
					logging.Error(err))
			}
			slippagePerUnit = int64(exitPrice) - markPrice
		}
		marginMaintenanceSht = float64(-slippageVolume)*(float64(slippagePerUnit)+(rf.Short*float64(markPrice))) + (float64(e.Sell()) * rf.Short * float64(markPrice))
	}

	// the greatest liability is the most positive number
	if marginMaintenanceLng > marginMaintenanceSht && marginMaintenanceLng > 0 {
		return newMarginLevels(marginMaintenanceLng, r.marginCalculator.ScalingFactors)
	}
	if marginMaintenanceSht > 0 {
		return newMarginLevels(marginMaintenanceSht, r.marginCalculator.ScalingFactors)
	}

	return nil
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
