package models

import (
	"time"

	types "code.vegaprotocol.io/vega/proto"
)

// Simple represents a dummy risk model with fixed risk params.
type Simple struct {
	factorLong, factorShort float64
	maxMoveUp, minMoveDown  float64
	asset                   string
	prob                    float64
}

// NewSimple instantiates a new simple/dummy risk model with fixed risk params.
func NewSimple(ps *types.SimpleRiskModel, asset string) (*Simple, error) {
	return &Simple{
		factorLong:  ps.Params.FactorLong,
		factorShort: ps.Params.FactorShort,
		maxMoveUp:   ps.Params.MaxMoveUp,
		minMoveDown: ps.Params.MinMoveDown,
		asset:       asset,
		prob:        ps.Params.ProbabilityOfTrading,
	}, nil
}

// CalculationInterval return the calculation interval for the simple/dummy risk model.
func (f *Simple) CalculationInterval() time.Duration {
	return time.Duration(0)
}

// CalculateRiskFactors returns the fixed risk factors for the simple risk model.
func (f *Simple) CalculateRiskFactors(current *types.RiskResult) (bool, *types.RiskResult) {
	rf := &types.RiskResult{
		RiskFactors: map[string]*types.RiskFactor{
			f.asset: {
				Long:  f.factorLong,
				Short: f.factorShort,
			},
		},
		PredictedNextRiskFactors: map[string]*types.RiskFactor{
			f.asset: {
				Long:  f.factorLong,
				Short: f.factorShort,
			},
		},
	}
	return true, rf
}

// PriceRange returns the minimum and maximum price as implied by the model's maxMoveUp/minMoveDown parameters and the current price
func (f *Simple) PriceRange(currentPrice, yearFraction, probabilityLevel float64) (float64, float64) {
	return currentPrice + f.minMoveDown, currentPrice + f.maxMoveUp
}

// ProbabilityOfTrading of trading returns the probability of trading given current mark price, projection horizon expressed as year fraction, order price and side (isBid).
// Additional arguments control optional truncation of probability density outside the [minPrice,maxPrice] range.
func (f *Simple) ProbabilityOfTrading(currentPrice, yearFraction, orderPrice float64, isBid bool, applyMinMax bool, minPrice float64, maxPrice float64) float64 {
	if applyMinMax && (orderPrice < minPrice || orderPrice > maxPrice) {
		return 0
	}
	return f.prob
}

// GetProjectionHorizon returns 0 and the simple model doesn't rely on any proabilistic calculations
func (f *Simple) GetProjectionHorizon() float64 {
	return 0
}
