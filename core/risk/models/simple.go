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

package models

import (
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

// Simple represents a dummy risk model with fixed risk params.
type Simple struct {
	factorLong, factorShort num.Decimal
	maxMoveUp, minMoveDown  num.Decimal
	asset                   string
	prob                    num.Decimal
}

// NewSimple instantiates a new simple/dummy risk model with fixed risk params.
func NewSimple(ps *types.SimpleRiskModel, asset string) (*Simple, error) {
	return &Simple{
		factorLong:  ps.Params.FactorLong,
		factorShort: ps.Params.FactorShort,
		maxMoveUp:   ps.Params.MaxMoveUp,
		minMoveDown: ps.Params.MinMoveDown.Abs(), // use Abs in case the value is negative
		asset:       asset,
		prob:        ps.Params.ProbabilityOfTrading,
	}, nil
}

// CalculateRiskFactors returns the fixed risk factors for the simple risk model.
func (f *Simple) CalculateRiskFactors() *types.RiskFactor {
	return &types.RiskFactor{
		Long:  f.factorLong,
		Short: f.factorShort,
	}
}

// PriceRange returns the minimum and maximum price as implied by the model's maxMoveUp/minMoveDown parameters and the current price.
func (f *Simple) PriceRange(currentP, _, _ num.Decimal) (num.Decimal, num.Decimal) {
	return num.MaxD(currentP.Sub(f.minMoveDown), num.DecimalZero()), currentP.Add(f.maxMoveUp)
}

// ProbabilityOfTrading of trading returns the probability of trading given current mark price, projection horizon expressed as year fraction, order price and side (isBid).
// Additional arguments control optional truncation of probability density outside the [minPrice,maxPrice] range.
func (f *Simple) ProbabilityOfTrading(currentP, orderP, minP, maxP, yFrac num.Decimal, isBid, applyMinMax bool) num.Decimal {
	if applyMinMax {
		if orderP.LessThan(minP) || orderP.GreaterThan(maxP) {
			return num.DecimalZero()
		}
	}
	return f.prob
}

// GetProjectionHorizon returns 0 and the simple model doesn't rely on any proabilistic calculations.
func (f *Simple) GetProjectionHorizon() num.Decimal {
	return num.DecimalZero()
}

func (f *Simple) DefaultRiskFactors() *types.RiskFactor {
	return &types.RiskFactor{
		Long:  f.factorLong,
		Short: f.factorShort,
	}
}
