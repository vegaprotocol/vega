package gql

import (
	"context"

	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

type myTradableInstrumentResolver VegaResolverRoot

func (r *myTradableInstrumentResolver) RiskModel(ctx context.Context, obj *types.TradableInstrument) (RiskModel, error) {
	return RiskModelFromProto(obj.RiskModel)
}
func (r *myTradableInstrumentResolver) MarginCalculator(ctx context.Context, obj *types.TradableInstrument) (*MarginCalculator, error) {
	return MarginCalculatorFromProto(obj.MarginCalculator)
}
