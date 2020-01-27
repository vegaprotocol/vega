package models

import (
	"time"

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
