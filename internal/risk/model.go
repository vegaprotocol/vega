package risk

import (
	"errors"
	"time"

	"code.vegaprotocol.io/vega/internal/risk/models"

	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"
)

var (
	// ErrNilRiskModel ...
	ErrNilRiskModel = errors.New("nil risk model")
	// ErrUnimplementedRiskModel ...
	ErrUnimplementedRiskModel = errors.New("unimplemented risk model")
)

// Model represents a risk model interface
//go:generate go run github.com/golang/mock/mockgen -destination mocks/risk_model_mock.go -package mocks code.vegaprotocol.io/vega/internal/risk Model
type Model interface {
	CalculationInterval() time.Duration
	CalculateRiskFactors(current *types.RiskResult) (bool, *types.RiskResult)
}

// NewModel instantiate a new risk model from a market framework configuration
func NewModel(log *logging.Logger, prm interface{}, asset string) (Model, error) {
	if prm == nil {
		return nil, ErrNilRiskModel
	}

	switch rm := prm.(type) {
	case *types.TradableInstrument_Forward:
		return models.NewBuiltinFutures(rm.Forward, asset)
	case *types.TradableInstrument_ExternalRiskModel:
		return models.NewExternal(log, rm.ExternalRiskModel)
	case *types.TradableInstrument_SimpleRiskModel:
		return models.NewSimple(rm.SimpleRiskModel, asset)
	default:
		return nil, ErrUnimplementedRiskModel
	}
}
