package gql

import (
	"context"

	"code.vegaprotocol.io/vega/protos/vega"
)

type liquiditySLAParametersResolver VegaResolverRoot

func (r liquiditySLAParametersResolver) PerformanceHysteresisEpochs(ctx context.Context, obj *vega.LiquiditySLAParameters) (int, error) {
	return int(obj.PerformanceHysteresisEpochs), nil
}

func (r liquiditySLAParametersResolver) SLACompetitionFactor(ctx context.Context, obj *vega.LiquiditySLAParameters) (string, error) {
	return obj.SlaCompetitionFactor, nil
}
