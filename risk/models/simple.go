package models

import (
	"time"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

// Simple represents a dummy risk model with fixed risk params.
type Simple struct {
	factorLong, factorShort num.Decimal
	maxMoveUp, minMoveDown  num.Decimal
	asset                   string
	prob                    num.Decimal
}

// NewSimple instantiates a new simple/dummy risk model with fixed risk params.
func NewSimple(ps *types.SimpleRiskModel, asset string) (*Simple, error) {
	return &Simple{
		factorLong:  num.DecimalFromFloat(ps.Params.FactorLong),
		factorShort: num.DecimalFromFloat(ps.Params.FactorShort),
		maxMoveUp:   num.DecimalFromFloat(ps.Params.MaxMoveUp),
		minMoveDown: num.DecimalFromFloat(ps.Params.MinMoveDown),
		asset:       asset,
		prob:        num.DecimalFromFloat(ps.Params.ProbabilityOfTrading),
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
func (f *Simple) PriceRange(currentP, _, _ num.Decimal) (num.Decimal, num.Decimal) {
	return currentP.Sub(f.minMoveDown), currentP.Add(f.maxMoveUp)
}

// ProbabilityOfTrading of trading returns the probability of trading given current mark price, projection horizon expressed as year fraction, order price and side (isBid).
// Additional arguments control optional truncation of probability density outside the [minPrice,maxPrice] range.
func (f *Simple) ProbabilityOfTrading(currentP, orderP, minP, maxP *num.Uint, yFrac num.Decimal, isBid, applyMinMax bool) num.Decimal {
	if !applyMinMax {
		return f.prob
	}
	if orderP.LT(minP) || orderP.GT(maxP) {
		return num.DecimalFromFloat(0)
	}
	return f.prob
}

// GetProjectionHorizon returns 0 and the simple model doesn't rely on any proabilistic calculations
func (f *Simple) GetProjectionHorizon() num.Decimal {
	return num.DecimalFromFloat(0)
}
