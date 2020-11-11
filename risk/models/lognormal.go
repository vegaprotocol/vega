package models

import (
	"time"

	pd "code.vegaprotocol.io/quant/pricedistribution"
	"code.vegaprotocol.io/quant/riskmodelbs"
	types "code.vegaprotocol.io/vega/proto"
)

// LogNormal represent a future risk model
type LogNormal struct {
	riskAversionParameter, tau float64
	params                     riskmodelbs.ModelParamsBS
	asset                      string
}

// NewBuiltinFutures instantiate a new builtin future
func NewBuiltinFutures(pf *types.LogNormalRiskModel, asset string) (*LogNormal, error) {
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
	dist := f.params.GetProbabilityDistribution(currentPrice, yearFraction)
	return pd.PriceRange(dist, probabilityLevel)
}

// ProbabilityOfTrading of trading returns the probability of trading given current mark price, projection horizon expressed as year fraction, order price and side (isBid).
// Additional arguments control optional truncation of probability density outside the [minPrice,maxPrice] range.
func (f *LogNormal) ProbabilityOfTrading(currentPrice, yearFraction, orderPrice float64, isBid bool, applyMinMax bool, minPrice float64, maxPrice float64) float64 {
	dist := f.params.GetProbabilityDistribution(currentPrice, yearFraction)
	return pd.ProbabilityOfTrading(dist, orderPrice, isBid, applyMinMax, minPrice, maxPrice)
}
