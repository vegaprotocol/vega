package models

import (
	"errors"
	"math"
	"time"

	"code.vegaprotocol.io/quant/interfaces"
	pd "code.vegaprotocol.io/quant/pricedistribution"
	"code.vegaprotocol.io/quant/riskmodelbs"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/types/num"
)

var (
	ErrMissingLogNormalParameter = errors.New("missing log normal parameters")
)

// LogNormal represent a future risk model
type LogNormal struct {
	riskAversionParameter, tau num.Decimal
	params                     riskmodelbs.ModelParamsBS
	asset                      string

	distCache    interfaces.AnalyticalDistribution
	cachePrice   *num.Uint
	cacheHorizon num.Decimal
}

// NewBuiltinFutures instantiate a new builtin future
func NewBuiltinFutures(pf *types.LogNormalRiskModel, asset string) (*LogNormal, error) {
	if pf.Params == nil {
		return nil, ErrMissingLogNormalParameter
	}
	return &LogNormal{
		riskAversionParameter: num.DecimalFromFloat(pf.RiskAversionParameter),
		tau:                   num.DecimalFromFloat(pf.Tau),
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
	rav, _ := f.riskAversionParameter.Float64()
	tau, _ := f.tau.Float64()
	rawrf := riskmodelbs.RiskFactorsForward(rav, tau, f.params)
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
func (f *LogNormal) PriceRange(currentP *num.Uint, yFrac, probabilityLevel num.Decimal) (*num.Uint, *num.Uint) {
	dist := f.getDistribution(currentP, yFrac)
	// damn you quant!
	pl, _ := probabilityLevel.Float64()
	minF, maxF := pd.PriceRange(dist, pl)
	min, _ := num.UintFromDecimal(num.DecimalFromFloat(minF))
	max, _ := num.UintFromDecimal(num.DecimalFromFloat(maxF))
	return min, max
}

// ProbabilityOfTrading of trading returns the probability of trading given current mark price, projection horizon expressed as year fraction, order price and side (isBid).
// Additional arguments control optional truncation of probability density outside the [minPrice,maxPrice] range.
func (f *LogNormal) ProbabilityOfTrading(currentP, orderP, minP, maxP *num.Uint, yFrac num.Decimal, isBid, applyMinMax bool) num.Decimal {
	dist := f.getDistribution(currentP, yFrac)
	min := math.Max(minP.Float64(), 0)
	// still, quant uses floats
	prob := pd.ProbabilityOfTrading(dist, orderP.Float64(), isBid, applyMinMax, min, maxP.Float64())
	return num.DecimalFromFloat(prob)
}

func (f *LogNormal) getDistribution(currentP *num.Uint, yFrac num.Decimal) interfaces.AnalyticalDistribution {
	if f.distCache == nil || f.cachePrice.NEQ(currentP) || !f.cacheHorizon.Equal(yFrac) {
		// quant still uses floats... sad
		yf, _ := yFrac.Float64()
		f.distCache = f.params.GetProbabilityDistribution(currentP.Float64(), yf)
	}
	return f.distCache
}

// GetProjectionHorizon returns the projection horizon used by the model for margin calculation pruposes
func (f *LogNormal) GetProjectionHorizon() num.Decimal {
	return f.tau
}
