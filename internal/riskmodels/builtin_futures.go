package riskmodels

import (
	"time"

	types "code.vegaprotocol.io/vega/proto"
)

type BuiltinFutures struct {
	HistoricVolatility float64
}

func newBuiltinFutures(pbf *types.BuiltinFutures) (*BuiltinFutures, error) {
	return &BuiltinFutures{
		HistoricVolatility: pbf.HistoricVolatility,
	}, nil
}

func (bf *BuiltinFutures) CalculationInterval() time.Duration {
	return time.Duration(0)
}

func (bf *BuiltinFutures) CalculateRiskFactors(
	current *types.RiskResult) (bool, *types.RiskResult) {
	if current == nil {
		return true, &types.RiskResult{
			RiskFactors: map[string]*types.RiskFactor{
				"Ethereum/Ether": &types.RiskFactor{
					Long:  0.15,
					Short: 0.25,
				},
			},
			PredictedNextRiskFactors: map[string]*types.RiskFactor{
				"Ethereum/Ether": &types.RiskFactor{
					Long:  0.15,
					Short: 0.25,
				},
			},
		}
	}
	return true, current
}
