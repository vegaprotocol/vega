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
}

// NewSimple instantiates a new simple/dummy risk model with fixed risk params.
func NewSimple(ps *types.SimpleRiskModel, asset string) (*Simple, error) {
	return &Simple{
		factorLong:  ps.Params.FactorLong,
		factorShort: ps.Params.FactorShort,
		maxMoveUp:   ps.Params.MaxMoveUp,
		minMoveDown: ps.Params.MinMoveDown,
		asset:       asset,
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
func (f *Simple) PriceRange(currentPrice float64, yearFraction float64, probabilityLevel float64) (minPrice float64, maxPrice float64) {
	minPrice, maxPrice = currentPrice+f.minMoveDown, currentPrice+f.maxMoveUp
	return
}
