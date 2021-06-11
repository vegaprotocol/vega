package models

import (
	"errors"
	"math"
	"time"

	"code.vegaprotocol.io/quant/interfaces"
	pd "code.vegaprotocol.io/quant/pricedistribution"
	"code.vegaprotocol.io/quant/riskmodelbs"
	types "code.vegaprotocol.io/vega/proto"
)

var (
	ErrMissingLogNormalParameter = errors.New("missing log normal parameters")
)

// LogNormal represent a future risk model
type LogNormal struct {
	riskAversionParameter, tau float64
	params                     riskmodelbs.ModelParamsBS
	asset                      string

	distCache    interfaces.AnalyticalDistribution
	cachePrice   float64
	cacheHorizon float64
}

// NewBuiltinFutures instantiate a new builtin future
func NewBuiltinFutures(pf *types.LogNormalRiskModel, asset string) (*LogNormal, error) {
	if pf.Params == nil {
		return nil, ErrMissingLogNormalParameter
	}
	return &LogNormal{
		riskAversionParameter: pf.RiskAversionParameter,
		tau:                   pf.Tau,
		params: riskmodelbs.ModelParamsBS{
			Mu:    pf.Params.Mu,
			R:     pf.Params.R,
			Sigma: pf.Params.Sigma,
		},
		asset: asset,
	}, nil
}

// CalculationInterval return the calculation interval for
// the Forward risk model
func (f *LogNormal) CalculationInterval() time.Duration {
	return time.Duration(0)
}

// CalculateRiskFactors calls the risk model in order to get
// the new risk models
func (f *LogNormal) CalculateRiskFactors(
	current *types.RiskResult) (bool, *types.RiskResult) {
	rawrf := riskmodelbs.RiskFactorsForward(f.riskAversionParameter, f.tau, f.params)
	rf := &types.RiskResult{
		RiskFactors: map[string]*types.RiskFactor{
			f.asset: {
				Long:  rawrf.Long,
				Short: rawrf.Short,
			},
		},
		PredictedNextRiskFactors: map[string]*types.RiskFactor{
			f.asset: {
				Long:  rawrf.Long,
				Short: rawrf.Short,
			},
		},
	}
	return true, rf
}

// PriceRange returns the minimum and maximum price as implied by the model's probability distribution with horizon given by yearFraction (e.g. 0.5 for half a year) and probability level (e.g. 0.95 for 95%).
func (f *LogNormal) PriceRange(currentPrice, yearFraction, probabilityLevel float64) (float64, float64) {
	dist := f.getDistribution(currentPrice, yearFraction)
	return pd.PriceRange(dist, probabilityLevel)
}

// ProbabilityOfTrading of trading returns the probability of trading given current mark price, projection horizon expressed as year fraction, order price and side (isBid).
// Additional arguments control optional truncation of probability density outside the [minPrice,maxPrice] range.
func (f *LogNormal) ProbabilityOfTrading(currentPrice, yearFraction, orderPrice float64, isBid bool, applyMinMax bool, minPrice float64, maxPrice float64) float64 {
	dist := f.getDistribution(currentPrice, yearFraction)
	//Floor min price at zero since lognormal distribution has support [0, inf)
	minPrice = math.Max(minPrice, 0)
	return pd.ProbabilityOfTrading(dist, currentPrice, orderPrice, isBid, applyMinMax, minPrice, maxPrice)
}

func (f *LogNormal) getDistribution(currentPrice, yearFraction float64) interfaces.AnalyticalDistribution {
	if f.cachePrice != currentPrice || f.cacheHorizon != yearFraction || f.distCache == nil {
		f.distCache = f.params.GetProbabilityDistribution(currentPrice, yearFraction)
	}
	return f.distCache
}

// GetProjectionHorizon returns the projection horizon used by the model for margin calculation pruposes
func (f *LogNormal) GetProjectionHorizon() float64 {
	return f.tau
}
