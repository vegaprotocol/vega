// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package risk

import (
	"errors"

	"code.vegaprotocol.io/vega/core/risk/models"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

var (
	// ErrNilRiskModel ...
	ErrNilRiskModel = errors.New("nil risk model")
	// ErrUnimplementedRiskModel ...
	ErrUnimplementedRiskModel = errors.New("unimplemented risk model")
)

// Model represents a risk model interface.
type Model interface {
	CalculateRiskFactors() *types.RiskFactor
	DefaultRiskFactors() *types.RiskFactor
	PriceRange(price, yearFraction, probability num.Decimal) (minPrice, maxPrice num.Decimal)
	ProbabilityOfTrading(currentP, orderP, minP, maxP, yFrac num.Decimal, isBid, applyMinMax bool) num.Decimal
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
