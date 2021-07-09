package gql

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/proto"
)

type myTradableInstrumentResolver VegaResolverRoot

func (r *myTradableInstrumentResolver) RiskModel(ctx context.Context, obj *proto.TradableInstrument) (RiskModel, error) {
	switch rm := obj.RiskModel.(type) {
	case *proto.TradableInstrument_LogNormalRiskModel:
		return rm.LogNormalRiskModel, nil
	case *proto.TradableInstrument_SimpleRiskModel:
		return rm.SimpleRiskModel, nil
	default:
		return nil, errors.New("invalid risk model")
	}
}
