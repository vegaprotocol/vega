// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package supplied

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/core/types/statevar"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

type probabilityOfTradingConverter struct{}

type probabilityOfTrading struct {
	bidOffset      []num.Decimal // represents offsets from best bid
	bidProbability []num.Decimal // probability[0] = probability of the best bid, probabtility[len-1] = probability of the worst bid
	askOffset      []num.Decimal // represents offsets from best ask
	askProbability []num.Decimal // probability[0] = probability of the best ask, probabtility[len-1] = probability of the worst ask
}

var (
	defaultInRangeProbabilityOfTrading = num.DecimalFromFloat(0.5)
	defaultMinimumProbabilityOfTrading = num.DecimalFromFloat(1e-8)
	tolerance                          = num.DecimalFromFloat(1e-6)
	incrementInPct                     = num.DecimalFromFloat(0.1)                                    // we calculate the probability of trading in increments of 0.1% of the best bid/ask
	IncrementFactor                    = incrementInPct.Div(num.DecimalFromFloat(100))                // we calculate the probability of trading in increments of 0.001 from of the best bid/ask
	maxDistanceWhenNoConsensusPct      = num.DecimalFromFloat(20)                                     // if there's no consensus yet and the price is within 20% of the best bid/ask it gets the default probability
	maxDistanceWhenNoConsensusFactor   = maxDistanceWhenNoConsensusPct.Div(num.DecimalFromFloat(100)) // if there's no consensus yet and the price is within 0.2 of the best bid/ask it gets the default probability

	maxAskMultiplier = num.DecimalFromFloat(6) // arbitrarily large factor on the best ask - 600%
)

func (probabilityOfTradingConverter) BundleToInterface(kvb *statevar.KeyValueBundle) statevar.StateVariableResult {
	return &probabilityOfTrading{
		bidOffset:      kvb.KVT[0].Val.(*statevar.DecimalVector).Val,
		bidProbability: kvb.KVT[1].Val.(*statevar.DecimalVector).Val,
		askOffset:      kvb.KVT[2].Val.(*statevar.DecimalVector).Val,
		askProbability: kvb.KVT[3].Val.(*statevar.DecimalVector).Val,
	}
}

func (probabilityOfTradingConverter) InterfaceToBundle(res statevar.StateVariableResult) *statevar.KeyValueBundle {
	value := res.(*probabilityOfTrading)
	return &statevar.KeyValueBundle{
		KVT: []statevar.KeyValueTol{
			{Key: "bidOffset", Val: &statevar.DecimalVector{Val: value.bidOffset}, Tolerance: tolerance},
			{Key: "bidProbability", Val: &statevar.DecimalVector{Val: value.bidProbability}, Tolerance: tolerance},
			{Key: "askOffset", Val: &statevar.DecimalVector{Val: value.askOffset}, Tolerance: tolerance},
			{Key: "askProbability", Val: &statevar.DecimalVector{Val: value.askProbability}, Tolerance: tolerance},
		},
	}
}

func (e *Engine) IsPoTInitialised() bool {
	return e.potInitialised
}

// startCalcPriceRanges kicks off the probability of trading calculation.
func (e *Engine) startCalcProbOfTrading(eventID string, endOfCalcCallback statevar.FinaliseCalculation) {
	tauScaled := e.horizon.Mul(e.probabilityOfTradingTauScaling)

	// get the best bid and ask
	bestBid, bestAsk, err := e.getBestStaticPrices()
	if err != nil {
		e.log.Error("failed to get static price for probability of trading state var", logging.String("error", err.Error()))
		endOfCalcCallback.CalculationFinished(eventID, nil, err)
		return
	}

	// calculate offsets and probabilities for the range
	bidOffsets, bidProbabilities := calculateBidRange(bestBid, IncrementFactor, tauScaled, e.rm.ProbabilityOfTrading)
	askOffsets, askProbabilities := calculateAskRange(bestAsk, IncrementFactor, tauScaled, e.rm.ProbabilityOfTrading)

	res := &probabilityOfTrading{
		bidOffset:      bidOffsets,
		bidProbability: bidProbabilities,
		askOffset:      askOffsets,
		askProbability: askProbabilities,
	}
	endOfCalcCallback.CalculationFinished(eventID, res, nil)
}

// calculateBidRange calculates the probabilities of price between bestBid and the worst bid
// in increments of incrementFactor of the best bid and records the offsets and probabilities
// such that offset 0 is stored in offsets[0] which corresponds to the best bid
// and similarly probabilities[0] corresponds to best bid trading probability
// whereas the last entry in offset equals to the maximum distance from bid for which
// probability is calculated and is greater than the default minimum acceptable probability of trading.
func calculateBidRange(bestBid, incrementFactor, tauScaled num.Decimal, probabilityFunc func(num.Decimal, num.Decimal, num.Decimal, num.Decimal, num.Decimal, bool, bool) num.Decimal) ([]num.Decimal, []num.Decimal) {
	offsets := []num.Decimal{}
	probabilities := []num.Decimal{}

	p := bestBid
	offset := num.DecimalZero()
	increment := incrementFactor.Mul(bestBid)
	for {
		prob := probabilityFunc(bestBid, p, num.DecimalZero(), bestBid, tauScaled, true, true)
		if prob.LessThanOrEqual(defaultMinimumProbabilityOfTrading) {
			break
		}
		offsets = append(offsets, offset)
		probabilities = append(probabilities, prob)
		offset = offset.Add(incrementFactor)
		p = p.Sub(increment)
	}
	return offsets, probabilities
}

// calculateAskRange calculates the probabilities of price between bestAsk and the worst ask (maxAsk)
// in increments of incrementFactor of the best ask and records the offsets and probabilities
// such that offset 0 is stored in offsets[0] which corresponds to the best ask
// and similarly probabilities[0] corresponds to best ask trading probability
// whereas the last entry in offset equals to the maximum distance from ask for which
// probability is calculated, and the last entry in probabilities corresponds to probability of trading
// at the price implied by this offset from best ask.
func calculateAskRange(bestAsk, incrementFactor, tauScaled num.Decimal, probabilityFunc func(num.Decimal, num.Decimal, num.Decimal, num.Decimal, num.Decimal, bool, bool) num.Decimal) ([]num.Decimal, []num.Decimal) {
	offsets := []num.Decimal{}
	probabilities := []num.Decimal{}

	maxAsk := bestAsk.Mul(maxAskMultiplier)
	increment := incrementFactor.Mul(bestAsk)
	p := bestAsk
	offset := num.DecimalZero()
	for {
		prob := probabilityFunc(bestAsk, p, bestAsk, maxAsk, tauScaled, false, true)
		if prob.LessThanOrEqual(defaultMinimumProbabilityOfTrading) {
			break
		}
		offsets = append(offsets, offset)
		probabilities = append(probabilities, prob)
		offset = offset.Add(incrementFactor)
		p = p.Add(increment)
	}
	return offsets, probabilities
}

// updatePriceBounds is called back from the state variable consensus engine when consensus is reached for the down/up factors and updates the price bounds.
func (e *Engine) updateProbabilities(ctx context.Context, res statevar.StateVariableResult) error {
	e.pot = res.(*probabilityOfTrading)
	if e.log.GetLevel() <= logging.DebugLevel {
		e.log.Debug("consensus reached for probability of trading", logging.String("market", e.marketID))
	}

	e.potInitialised = true
	return nil
}

// getProbabilityOfTrading returns the probability of trading for the given price
// if the price is a bid order that is better than the best bid or an ask order better than the best and ask it returns <defaultInRangeProbabilityOfTrading>
// if the price is a bid worse than min or an ask worse than max it returns minProbabilityOfTrading
// otherwise if we've not seen a consensus value and the price is within 10% the best (by side)
// it returns <defaultProbability> else if there is not yet consensus value it returns <minProbabilityOfTrading>.
// If there is consensus value and the price is worse than the worse price, we extrapolate using the last 2 price points
// If the price is within the range, the corresponding probability implied by the offset is returned, scaled, with lower bound of <minProbabilityOfTrading>.
func getProbabilityOfTrading(bestBid, bestAsk, minPrice, maxPrice num.Decimal, pot *probabilityOfTrading, price num.Decimal, isBid bool, minProbabilityOfTrading num.Decimal) num.Decimal {
	if (isBid && price.GreaterThanOrEqual(bestBid)) || (!isBid && price.LessThanOrEqual(bestAsk)) {
		return defaultInRangeProbabilityOfTrading
	}

	if isBid {
		if price.LessThan(minPrice) {
			return minProbabilityOfTrading
		}
		return getBidProbabilityOfTrading(bestBid, pot.bidOffset, pot.bidProbability, price, minProbabilityOfTrading)
	}
	if price.GreaterThan(maxPrice) {
		return minProbabilityOfTrading
	}
	return getAskProbabilityOfTrading(bestAsk, pot.askOffset, pot.askProbability, price, minProbabilityOfTrading)
}

func getAskProbabilityOfTrading(bestAsk num.Decimal, offsets, probabilities []num.Decimal, price, minProbabilityOfTrading num.Decimal) num.Decimal {
	// no consensus yet
	if len(offsets) == 0 {
		maxDistance := maxDistanceWhenNoConsensusFactor.Mul(bestAsk)
		if bestAsk.Sub(price).Abs().LessThanOrEqual(maxDistance) {
			return defaultInRangeProbabilityOfTrading
		}
		return minProbabilityOfTrading
	}
	offset := price.Sub(bestAsk).Div(bestAsk)

	// if outside the range - extrapolate
	if offset.GreaterThan(offsets[len(offsets)-1]) {
		return lexterp(offsets, probabilities, offset, minProbabilityOfTrading)
	}

	// linear interpolation
	interpolatedProbability := linterp(offsets, probabilities, offset, func(i int) bool {
		r := offset.LessThan(offsets[i])
		return r
	}, minProbabilityOfTrading)
	return interpolatedProbability
}

func getBidProbabilityOfTrading(bestBid num.Decimal, offsets, probabilities []num.Decimal, price, minProbabilityOfTrading num.Decimal) num.Decimal {
	// no consensus yet
	if len(offsets) == 0 {
		maxDistance := maxDistanceWhenNoConsensusFactor.Mul(bestBid)
		if bestBid.Sub(price).Abs().LessThanOrEqual(maxDistance) {
			return defaultInRangeProbabilityOfTrading
		}
		return minProbabilityOfTrading
	}

	offset := bestBid.Sub(price).Div(bestBid)

	// if outside the range - extrapolate
	if offset.GreaterThan(offsets[len(offsets)-1]) {
		return lexterp(offsets, probabilities, offset, minProbabilityOfTrading)
	}

	// linear interpolation
	interpolatedProbability := linterp(offsets, probabilities, offset, func(i int) bool {
		r := offset.LessThan(offsets[i])
		return r
	}, minProbabilityOfTrading)
	return interpolatedProbability
}

func lexterp(offsets, probabilities []num.Decimal, offset, minProbabilityOfTrading num.Decimal) num.Decimal {
	if len(offsets) == 1 {
		return minProbabilityOfTrading
	}
	last := offsets[len(offsets)-1]
	last2 := offsets[len(offsets)-2]
	probLast := probabilities[len(offsets)-1]
	probLast2 := probabilities[len(offsets)-2]
	slopeNum := offset.Sub(last2)
	slopeDenom := last.Sub(last2)
	slope := slopeNum.Div(slopeDenom)
	prob := probLast2.Add(probLast.Sub(probLast2).Mul(slope))
	scaled := rescaleProbability(prob)
	prob = num.MinD(num.DecimalFromInt64(1), num.MaxD(minProbabilityOfTrading, scaled))
	return prob
}

func linterp(offsets, probabilities []num.Decimal, priceOffset num.Decimal, searchFunc func(i int) bool, minProbabilityOfTrading num.Decimal) num.Decimal {
	i := sort.Search(len(offsets), searchFunc)

	if i >= len(probabilities) {
		return num.MaxD(minProbabilityOfTrading, rescaleProbability(probabilities[len(probabilities)-1]))
	}
	prev := offsets[i-1]
	size := offsets[i].Sub(prev)
	ratio := priceOffset.Sub(prev).Div(size)
	cRatio := num.DecimalFromInt64(1).Sub(ratio)
	prob := ratio.Mul(probabilities[i]).Add(cRatio.Mul(probabilities[i-1]))
	scaled := rescaleProbability(prob)
	capped := num.MaxD(minProbabilityOfTrading, scaled)
	return capped
}

// rescaleProbability rescales probability so that it's at most the value returned between bid and ask.
func rescaleProbability(prob num.Decimal) num.Decimal {
	return prob.Mul(defaultInRangeProbabilityOfTrading)
}
