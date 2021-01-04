package risk

import (
	"errors"
	"time"

	"code.vegaprotocol.io/vega/risk/models"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

var (
	// ErrNilRiskModel ...
	ErrNilRiskModel = errors.New("nil risk model")
	// ErrUnimplementedRiskModel ...
	ErrUnimplementedRiskModel = errors.New("unimplemented risk model")
)

// Model represents a risk model interface
//go:generate go run github.com/golang/mock/mockgen -destination mocks/risk_model_mock.go -package mocks code.vegaprotocol.io/vega/risk Model
type Model interface {
	CalculationInterval() time.Duration
	CalculateRiskFactors(current *types.RiskResult) (bool, *types.RiskResult)
	PriceRange(price float64, yearFraction float64, probability float64) (minPrice float64, maxPrice float64)
	ProbabilityOfTrading(currentPrice, yearFraction, orderPrice float64, isBid bool, applyMinMax bool, minPrice float64, maxPrice float64) float64
	GetProjectionHorizon() float64
}

// NewModel instantiate a new risk model from a market framework configuration
func NewModel(log *logging.Logger, prm interface{}, asset string) (Model, error) {
	if prm == nil {
		return nil, ErrNilRiskModel
	}

	switch rm := prm.(type) {
	case *types.TradableInstrument_LogNormalRiskModel:
		return models.NewBuiltinFutures(rm.LogNormalRiskModel, asset)
	case *types.TradableInstrument_SimpleRiskModel:
		return models.NewSimple(rm.SimpleRiskModel, asset)
	default:
		return nil, ErrUnimplementedRiskModel
	}
}
