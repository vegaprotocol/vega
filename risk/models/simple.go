package models

import (
	"time"

	types "code.vegaprotocol.io/vega/proto"
)

// Simple represents a dummy risk model with fixed risk params.
type Simple struct {
	factorLong, factorShort float64
	asset                   string
}

// NewSimple instantiates a new simple/dummy risk model with fixed risk params.
func NewSimple(ps *types.SimpleRiskModel, asset string) (*Simple, error) {
	return &Simple{
		factorLong:  ps.Params.FactorLong,
		factorShort: ps.Params.FactorShort,
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
			f.asset: &types.RiskFactor{
				Long:  f.factorLong,
				Short: f.factorShort,
			},
		},
		PredictedNextRiskFactors: map[string]*types.RiskFactor{
			f.asset: &types.RiskFactor{
				Long:  f.factorLong,
				Short: f.factorShort,
			},
		},
	}
	return true, rf
}
