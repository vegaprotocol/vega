package risk

import (
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/types/statevar"
)

type RiskFactorConverter struct{}

var riskFactorTolerance = num.MustDecimalFromString("1e-6")

func (RiskFactorConverter) BundleToInterface(kvb *statevar.KeyValueBundle) statevar.StateVariableResult {
	return &types.RiskFactor{
		Short: kvb.KVT[0].Val.(*statevar.DecimalScalar).Val,
		Long:  kvb.KVT[1].Val.(*statevar.DecimalScalar).Val,
	}
}

func (RiskFactorConverter) InterfaceToBundle(res statevar.StateVariableResult) *statevar.KeyValueBundle {
	value := res.(*types.RiskFactor)
	return &statevar.KeyValueBundle{
		KVT: []statevar.KeyValueTol{
			{Key: "short", Val: &statevar.DecimalScalar{Val: value.Short}, Tolerance: riskFactorTolerance},
			{Key: "long", Val: &statevar.DecimalScalar{Val: value.Long}, Tolerance: riskFactorTolerance},
		},
	}
}
