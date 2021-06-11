package risk

import (
	"errors"
	"time"

	"code.vegaprotocol.io/vega/risk/models"
	"code.vegaprotocol.io/vega/types/num"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
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
	PriceRange(price *num.Uint, yearFraction, probability num.Decimal) (minPrice, maxPrice *num.Uint)
	ProbabilityOfTrading(currentP, orderP, minP, maxP *num.Uint, yFrac num.Decimal, isBid, applyMinMax bool) num.Decimal
	GetProjectionHorizon() num.Decimal
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
