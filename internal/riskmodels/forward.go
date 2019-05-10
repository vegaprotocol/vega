package riskmodels

import (
	"time"

	"code.vegaprotocol.io/quant/riskmodelbs"
	types "code.vegaprotocol.io/vega/proto"
)

type Forward struct {
	lambd, tau float64
	params     riskmodelbs.ModelParamsBS
}

func newBuiltinFutures(pf *types.Forward) (*Forward, error) {
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

func (f *Forward) CalculationInterval() time.Duration {
	return time.Duration(0)
}

func (f *Forward) CalculateRiskFactors(
	current *types.RiskResult) (bool, *types.RiskResult) {
	rawrf := riskmodelbs.RiskFactorsForward(f.lambd, f.tau, f.params)
	rf := &types.RiskResult{
		RiskFactors: map[string]*types.RiskFactor{
			"Ethereum/Ether": &types.RiskFactor{
				Long:  rawrf.Long,
				Short: rawrf.Short,
			},
		},
		PredictedNextRiskFactors: map[string]*types.RiskFactor{
			"Ethereum/Ether": &types.RiskFactor{
				Long:  rawrf.Long,
				Short: rawrf.Short,
			},
		},
	}
	return true, rf
}
