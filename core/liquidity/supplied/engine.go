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

package supplied

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/statevar"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

// ErrNoValidOrders informs that there weren't any valid orders to cover the liquidity obligation with.
// This could happen when for a given side (buy or sell) limit orders don't supply enough liquidity and there aren't any
// valid pegged orders (all the prives are invalid) to cover it with.
var (
	ErrNoValidOrders = errors.New("no valid orders to cover the liquidity obligation with")
)

// RiskModel allows calculation of min/max price range and a probability of trading.
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/risk_model_mock.go -package mocks code.vegaprotocol.io/vega/core/liquidity/supplied RiskModel
type RiskModel interface {
	ProbabilityOfTrading(currentPrice, orderPrice, minPrice, maxPrice num.Decimal, yearFraction num.Decimal, isBid, applyMinMax bool) num.Decimal
	GetProjectionHorizon() num.Decimal
}

// PriceMonitor provides the range of valid prices, that is prices that wouldn't trade the current trading mode
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/price_monitor_mock.go -package mocks code.vegaprotocol.io/vega/core/liquidity/supplied PriceMonitor
type PriceMonitor interface {
	GetValidPriceRange() (num.WrappedDecimal, num.WrappedDecimal)
}

type StateVarEngine interface {
	RegisterStateVariable(asset, market, name string, converter statevar.Converter, startCalculation func(string, statevar.FinaliseCalculation), trigger []statevar.EventType, result func(context.Context, statevar.StateVariableResult) error) error
}

// Engine provides functionality related to supplied liquidity.
type Engine struct {
	rm                             RiskModel
	pm                             PriceMonitor
	marketID                       string
	horizon                        num.Decimal // projection horizon used in probability calculations
	probabilityOfTradingTauScaling num.Decimal
	minProbabilityOfTrading        num.Decimal
	pot                            *probabilityOfTrading
	potInitialised                 bool

	getBestStaticPrices func() (num.Decimal, num.Decimal, error)
	log                 *logging.Logger
	positionFactor      num.Decimal
}

// NewEngine returns a reference to a new supplied liquidity calculation engine.
func NewEngine(riskModel RiskModel, priceMonitor PriceMonitor, asset, marketID string, stateVarEngine StateVarEngine, log *logging.Logger, positionFactor num.Decimal) *Engine {
	e := &Engine{
		rm:                             riskModel,
		pm:                             priceMonitor,
		marketID:                       marketID,
		horizon:                        riskModel.GetProjectionHorizon(),
		probabilityOfTradingTauScaling: num.DecimalFromInt64(1), // this is the same as the default in the netparams
		minProbabilityOfTrading:        defaultMinimumProbabilityOfTrading,
		pot:                            &probabilityOfTrading{},
		potInitialised:                 false,
		log:                            log,
		positionFactor:                 positionFactor,
	}

	stateVarEngine.RegisterStateVariable(asset, marketID, "probability_of_trading", probabilityOfTradingConverter{}, e.startCalcProbOfTrading, []statevar.EventType{statevar.EventTypeTimeTrigger, statevar.EventTypeAuctionEnded, statevar.EventTypeOpeningAuctionFirstUncrossingPrice}, e.updateProbabilities)
	return e
}

func (e *Engine) UpdateMarketConfig(riskModel RiskModel, monitor PriceMonitor) {
	e.rm = riskModel
	e.pm = monitor
	e.horizon = riskModel.GetProjectionHorizon()
	e.potInitialised = false
}

func (e *Engine) SetGetStaticPricesFunc(f func() (num.Decimal, num.Decimal, error)) {
	e.getBestStaticPrices = f
}

func (e *Engine) OnMinProbabilityOfTradingLPOrdersUpdate(v num.Decimal) {
	e.minProbabilityOfTrading = v
}

func (e *Engine) OnProbabilityOfTradingTauScalingUpdate(v num.Decimal) {
	e.probabilityOfTradingTauScaling = v
}

// CalculateLiquidityScore returns the current liquidity scores (volume-weighted probability of trading).
func (e *Engine) CalculateLiquidityScore(
	orders []*types.Order,
	bestBid, bestAsk num.Decimal,
	minLpPrice, maxLpPrice *num.Uint,
) num.Decimal {
	minPMPrice, maxPMPrice := e.pm.GetValidPriceRange()

	bLiq := num.DecimalZero()
	sLiq := num.DecimalZero()
	bSize := num.DecimalZero()
	sSize := num.DecimalZero()
	for _, o := range orders {
		if o.Price.LT(minLpPrice) || o.Price.GT(maxLpPrice) {
			continue
		}
		prob := num.DecimalZero()
		// if order is outside of price monitoring bounds then probability is set to 0.
		if o.Price.GTE(minPMPrice.Representation()) && o.Price.LTE(maxPMPrice.Representation()) {
			prob = getProbabilityOfTrading(bestBid, bestAsk, minPMPrice.Original(), maxPMPrice.Original(), e.pot, o.Price.ToDecimal(), o.Side == types.SideBuy, e.minProbabilityOfTrading, OffsetOneDecimal)
			if prob.LessThanOrEqual(e.minProbabilityOfTrading) {
				prob = num.DecimalZero()
			}
		}
		s := num.DecimalFromUint(num.NewUint(o.Remaining))
		l := prob.Mul(s)
		if o.Side == types.SideBuy {
			bLiq = bLiq.Add(l)
			bSize = bSize.Add(s)
		}
		if o.Side == types.SideSell {
			sLiq = sLiq.Add(l)
			sSize = sSize.Add(s)
		}
	}
	// descale by total volume per side
	if !bSize.IsZero() {
		bLiq = bLiq.Div(bSize)
	}
	if !sSize.IsZero() {
		sLiq = sLiq.Div(sSize)
	}

	// return the minimum of the two
	if bLiq.LessThanOrEqual(sLiq) {
		return bLiq
	}
	return sLiq
}
