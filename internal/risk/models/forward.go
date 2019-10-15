package models

import (
	"time"

	"code.vegaprotocol.io/quant/riskmodelbs"
	types "code.vegaprotocol.io/vega/proto"
)

// Forward represent a future risk model
type Forward struct {
	lambd, tau float64
	params     riskmodelbs.ModelParamsBS
}

// NewBuiltinFutures instantiate a new builtin future
func NewBuiltinFutures(pf *types.Forward) (*Forward, error) {
	return &Forward{
		lambd: pf.Lambd,
		tau:   pf.Tau,
		params: riskmodelbs.ModelParamsBS{
			Mu:    pf.Params.Mu,
			R:     pf.Params.R,
			Sigma: pf.Params.Sigma,
		},
	}, nil
}

// CalculationInterval return the calculation interval for
// the Forward risk model
func (f *Forward) CalculationInterval() time.Duration {
	return time.Duration(0)
}

// CalculateRiskFactors calls the risk model in order to get
// the new risk models
func (f *Forward) CalculateRiskFactors(
	current *types.RiskResult) (bool, *types.RiskResult) {
	rawrf := riskmodelbs.RiskFactorsForward(f.lambd, f.tau, f.params)
	rf := &types.RiskResult{
		RiskFactors: map[string]*types.RiskFactor{
			"ETH": &types.RiskFactor{
				Long:  rawrf.Long,
				Short: rawrf.Short,
			},
		},
		PredictedNextRiskFactors: map[string]*types.RiskFactor{
			"ETH": &types.RiskFactor{
				Long:  rawrf.Long,
				Short: rawrf.Short,
			},
		},
	}
	return true, rf
}
