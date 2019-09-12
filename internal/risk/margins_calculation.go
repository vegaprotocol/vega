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

func newMarginLevels(maintenance int64, scalingFactors *types.ScalingFactors) MarginLevels {
	return MarginLevels{
		MarginMaintenance: maintenance,
		SearchLevel:       int64(float64(maintenance) * scalingFactors.SearchLevel),
		InitialMargin:     int64(float64(maintenance) * scalingFactors.InitialMargin),
		ReleaseLevel:      int64(float64(maintenance) * scalingFactors.CollateralRelease),
	}
}

func (r *Engine) calculateMargins(e events.Margin, markPrice int64, rf types.RiskFactor) MarginLevels {
	lngCloseoutPNL, shtCloseoutPNL := r.calculateCloseoutPNL(e, markPrice)
	lngMaintenance := lngCloseoutPNL + e.Size()*int64(rf.Long*float64(markPrice))
	shtMaintenance := shtCloseoutPNL + e.Size()*int64(rf.Long*float64(markPrice))

	if lngMaintenance > shtMaintenance {
		return newMarginLevels(lngMaintenance, r.marginCalculator.ScalingFactors)
	}

	return newMarginLevels(shtMaintenance, r.marginCalculator.ScalingFactors)
}

// calculateCloseoutPNL
// closeoutPNL = position_size * (Product.value(closeout_price) - Product.value(current_price))
// in here all errors are logged only, as the GetCloseountPrice return an error if there is not
// enough Order in the book
//
// altho the specs says:
// if there is insufficient order book volume for this closeout_price to be calculated for an
// individual trader, the closeout_price is the price that would be achieved for as much of
// the volume that could theoretically be closed
func (r *Engine) calculateCloseoutPNL(
	e events.Margin, markPrice int64) (lngCloseoutPNL, shrtCloseoutPNL int64) {
	size := e.Size()
	potentialLong := size + e.Buy()
	potentialShort := size - e.Sell()

	if potentialLong > 0 {
		closeoutPrice, err := r.ob.GetCloseoutPrice(uint64(potentialLong), types.Side_Buy)
		if err != nil {
			r.log.Warn("got non critical error from GetCloseoutPrice for Buy side",
				logging.Error(err))
		}
		lngCloseoutPNL = potentialLong * (int64(closeoutPrice) - markPrice)
	}

	if potentialShort < 0 {
		closeoutPrice, err := r.ob.GetCloseoutPrice(uint64(potentialShort), types.Side_Sell)
		if err != nil {
			r.log.Warn("got non critical error from GetCloseoutPrice for Sell side",
				logging.Error(err))

		}
		shrtCloseoutPNL = potentialShort * (int64(closeoutPrice) - markPrice)
	}

	return
}
