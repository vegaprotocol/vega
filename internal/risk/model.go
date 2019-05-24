package risk

import (
	"errors"
	"time"

	"code.vegaprotocol.io/vega/internal/risk/models"

	"code.vegaprotocol.io/vega/internal/logging"
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

func NewModel(log *logging.Logger, prm interface{}) (Model, error) {
	if prm == nil {
		return nil, ErrNilRiskModel
	}

	switch rm := prm.(type) {
	case *types.TradableInstrument_Forward:
		return models.NewBuiltinFutures(rm.Forward)
	case *types.TradableInstrument_ExternalRiskModel:
		return models.NewExternal(log, rm.ExternalRiskModel)
	default:
		return nil, ErrUnimplementedRiskModel
	}
}
