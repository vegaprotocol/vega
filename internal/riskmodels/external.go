package riskmodels

import (
	"time"

	types "code.vegaprotocol.io/vega/proto"
)

type External struct {
	Name   string
	Config map[string]string
}

func newExternal(pe *types.ExternalRiskModel) (*External, error) {
	return &External{
		Name:   pe.Name,
		Config: pe.Config,
	}, nil
}

func (e *External) CalculationInterval() time.Duration {
	return time.Duration(0)
}

func (e *External) CalculateRiskFactors(
	current *types.RiskResult) (bool, *types.RiskResult) {
	return false, nil
}
