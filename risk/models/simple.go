// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package models

import (
	"time"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
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

// CalculationInterval return the calculation interval for the simple/dummy risk model.
func (f *Simple) CalculationInterval() time.Duration {
	return time.Duration(0)
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
	return currentP.Sub(f.minMoveDown), currentP.Add(f.maxMoveUp)
}

// ProbabilityOfTrading of trading returns the probability of trading given current mark price, projection horizon expressed as year fraction, order price and side (isBid).
// Additional arguments control optional truncation of probability density outside the [minPrice,maxPrice] range.
func (f *Simple) ProbabilityOfTrading(currentP, orderP *num.Uint, minP, maxP, yFrac num.Decimal, isBid, applyMinMax bool) num.Decimal {
	if applyMinMax {
		p := orderP.ToDecimal()
		if p.LessThan(minP) || p.GreaterThan(maxP) {
			return num.DecimalZero()
		}
	}
	return f.prob
}

// GetProjectionHorizon returns 0 and the simple model doesn't rely on any proabilistic calculations.
func (f *Simple) GetProjectionHorizon() num.Decimal {
	return num.DecimalZero()
}
