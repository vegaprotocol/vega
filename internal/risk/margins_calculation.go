package risk

import (
	"code.vegaprotocol.io/vega/internal/events"
	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"
)

type MarginLevels struct {
	MarginMaintenance int64
	SearchLevel       int64
	InitialMargin     int64
	ReleaseLevel      int64
}

func newMarginLevels(maintenance int64, scalingFactors *types.ScalingFactors) *MarginLevels {
	return &MarginLevels{
		MarginMaintenance: maintenance,
		SearchLevel:       int64(float64(maintenance) * scalingFactors.SearchLevel),
		InitialMargin:     int64(float64(maintenance) * scalingFactors.InitialMargin),
		ReleaseLevel:      int64(float64(maintenance) * scalingFactors.CollateralRelease),
	}
}

func abs(i int64) int64 {
	if i <= 0 {
		return -i
	}
	return i
}

// Implementation of the margin calculator per specs:
// https://gitlab.com/vega-protocol/product/blob/master/specs/0019-margin-calculator.md
func (r *Engine) calculateMargins(e events.Margin, markPrice int64, rf types.RiskFactor) *MarginLevels {
	var (
		marginMaintenanceLng int64
		marginMaintenanceSht int64
	)
	openPos := e.Size()
	// calculate both long and short riskiest positions
	riskiestLng := openPos + e.Buy()
	riskiestSht := openPos - e.Sell()

	// calculate margin maintenance long only if riskiest is > 0
	// marginMaintenanceLng will be 0 by default
	if riskiestLng > 0 {
		exitPrice, err := r.ob.GetCloseoutPrice(uint64(riskiestLng), types.Side_Sell)
		if err != nil {
			r.log.Warn("got non critical error from GetCloseoutPrice for Buy side",
				logging.Error(err))
		}
		slippagePerUnit := int64(exitPrice) - markPrice
		marginMaintenanceLng = openPos*(slippagePerUnit+int64(rf.Long*float64(markPrice))) + e.Buy()*int64(rf.Long*float64(markPrice))

	}
	// calculate margin maintenace short only if riskiest is < 0
	// marginMaintenanceSht will be 0 by default
	if riskiestSht < 0 {
		exitPrice, err := r.ob.GetCloseoutPrice(uint64(-riskiestSht), types.Side_Buy)
		if err != nil {
			r.log.Warn("got non critical error from GetCloseoutPrice for Buy side",
				logging.Error(err))
		}
		slippagePerUnit := int64(exitPrice) - markPrice
		marginMaintenanceSht = openPos*(slippagePerUnit+int64(rf.Short*float64(markPrice))) + e.Sell()*int64(rf.Short*float64(markPrice))
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
