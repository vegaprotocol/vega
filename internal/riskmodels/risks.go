package riskmodels

import (
	"errors"
	"time"

	types "code.vegaprotocol.io/vega/proto"
)

var (
	ErrNilRiskModel           = errors.New("nil risk model")
	ErrUnimplementedRiskModel = errors.New("unimplemented risk model")
)

type Model interface {
	CalculationInterval() time.Duration
	CalculateRiskFactors(current *types.RiskResult) (bool, *types.RiskResult)
}

func New(prm interface{}) (Model, error) {
	if prm == nil {
		return nil, ErrNilRiskModel
	}

	switch rm := prm.(type) {
	case *types.TradableInstrument_BuiltinFutures:
		return newBuiltinFutures(rm.BuiltinFutures)
	case *types.TradableInstrument_ExternalRiskModel:
		return newExternal(rm.ExternalRiskModel)
	default:
		return nil, ErrUnimplementedRiskModel
	}
}
