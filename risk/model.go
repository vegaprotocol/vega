package risk

import (
	"errors"
	"time"

	"code.vegaprotocol.io/vega/risk/models"
	"code.vegaprotocol.io/vega/types/num"

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
	// this should probably go? @witgaw.
	CalculationInterval() time.Duration
	CalculateRiskFactors() *types.RiskFactor
	PriceRange(price, yearFraction, probability num.Decimal) (minPrice, maxPrice num.Decimal)
	ProbabilityOfTrading(currentP, orderP *num.Uint, minP, maxP, yFrac num.Decimal, isBid, applyMinMax bool) num.Decimal
	GetProjectionHorizon() num.Decimal
}

// NewModel instantiate a new risk model from a market framework configuration.
func NewModel(prm interface{}, asset string) (Model, error) {
	if prm == nil {
		return nil, ErrNilRiskModel
	}

	switch rm := prm.(type) {
	case *types.TradableInstrumentLogNormalRiskModel:
		return models.NewBuiltinFutures(rm.LogNormalRiskModel, asset)
	case *types.TradableInstrumentSimpleRiskModel:
		return models.NewSimple(rm.SimpleRiskModel, asset)
	default:
		return nil, ErrUnimplementedRiskModel
	}
}
