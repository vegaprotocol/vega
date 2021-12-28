package models

import (
	"errors"
	"math"
	"time"

	"code.vegaprotocol.io/quant/interfaces"
	pd "code.vegaprotocol.io/quant/pricedistribution"
	"code.vegaprotocol.io/quant/riskmodelbs"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

var ErrMissingLogNormalParameter = errors.New("missing log normal parameters")

// LogNormal represent a future risk model.
type LogNormal struct {
	riskAversionParameter, tau num.Decimal
	params                     riskmodelbs.ModelParamsBS
	asset                      string

	distCache    interfaces.AnalyticalDistribution
	cachePrice   *num.Uint
	cacheHorizon num.Decimal
}

// NewBuiltinFutures instantiate a new builtin future.
func NewBuiltinFutures(pf *types.LogNormalRiskModel, asset string) (*LogNormal, error) {
	if pf.Params == nil {
		return nil, ErrMissingLogNormalParameter
	}
	// the quant stuff really needs to be updated to use the same num types...
	mu, _ := pf.Params.Mu.Float64()
	r, _ := pf.Params.R.Float64()
	sigma, _ := pf.Params.Sigma.Float64()
	return &LogNormal{
		riskAversionParameter: pf.RiskAversionParameter,
		tau:                   pf.Tau,
		cachePrice:            num.Zero(),
		params: riskmodelbs.ModelParamsBS{
			Mu:    mu,
			R:     r,
			Sigma: sigma,
		},
		asset: asset,
	}, nil
}

// CalculationInterval return the calculation interval for
// the Forward risk model.
func (f *LogNormal) CalculationInterval() time.Duration {
	return time.Duration(0)
}

// CalculateRiskFactors calls the risk model in order to get
// the new risk models.
func (f *LogNormal) CalculateRiskFactors() *types.RiskFactor {
	rav, _ := f.riskAversionParameter.Float64()
	tau, _ := f.tau.Float64()
	rawrf := riskmodelbs.RiskFactorsForward(rav, tau, f.params)
	return &types.RiskFactor{
		Long:  num.DecimalFromFloat(rawrf.Long),
		Short: num.DecimalFromFloat(rawrf.Short),
	}
}

// PriceRange returns the minimum and maximum price as implied by the model's probability distribution with horizon given by yearFraction (e.g. 0.5 for half a year) and probability level (e.g. 0.95 for 95%).
func (f *LogNormal) PriceRange(currentP, yFrac, probabilityLevel num.Decimal) (num.Decimal, num.Decimal) {
	uPrice, _ := num.UintFromDecimal(currentP)
	dist := f.getDistribution(uPrice, yFrac)
	// damn you quant!
	pl, _ := probabilityLevel.Float64()
	min, max := pd.PriceRange(dist, pl)
	return num.DecimalFromFloat(min), num.DecimalFromFloat(max)
}

// ProbabilityOfTrading of trading returns the probability of trading given current mark price, projection horizon expressed as year fraction, order price and side (isBid).
// Additional arguments control optional truncation of probability density outside the [minPrice,maxPrice] range.
func (f *LogNormal) ProbabilityOfTrading(currentP, orderP *num.Uint, minP, maxP num.Decimal, yFrac num.Decimal, isBid, applyMinMax bool) num.Decimal {
	dist := f.getDistribution(currentP, yFrac)
	min := math.Max(minP.InexactFloat64(), 0)
	// still, quant uses floats
	prob := pd.ProbabilityOfTrading(dist, orderP.Float64(), isBid, applyMinMax, min, maxP.InexactFloat64())
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

// GetProjectionHorizon returns the projection horizon used by the model for margin calculation pruposes.
func (f *LogNormal) GetProjectionHorizon() num.Decimal {
	return f.tau
}
