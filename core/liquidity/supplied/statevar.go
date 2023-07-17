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
	bidOffset      []uint32      // represents offsets from best bid - not sent over the wire
	bidProbability []num.Decimal // probability[0] = probability of the best bid, probabtility[len-1] = probability of the worst bid
	askOffset      []uint32      // represents offsets from best ask - not sent over the wire
	askProbability []num.Decimal // probability[0] = probability of the best ask, probabtility[len-1] = probability of the worst ask
}

var (
	defaultInRangeProbabilityOfTrading = num.DecimalFromFloat(0.5)
	defaultMinimumProbabilityOfTrading = num.DecimalFromFloat(1e-8)
	tolerance                          = num.DecimalFromFloat(1e-6)
	OffsetIncrement                    = uint32(1000) // the increment of offsets in the offset slice - 1/PriceIncrementFactor as integer
	OffsetIncrementAsDecimal           = num.DecimalFromInt64(int64(OffsetIncrement))
	OffsetOneDecimal                   = OffsetIncrementAsDecimal.Mul(OffsetIncrementAsDecimal)             // an offset of 100%
	PriceIncrementFactor               = num.DecimalOne().Div(num.DecimalFromInt64(int64(OffsetIncrement))) // we calculate the probability of trading in increments of 0.001 from of the best bid/ask
	maxDistanceWhenNoConsensusPct      = num.DecimalFromFloat(20)                                           // if there's no consensus yet and the price is within 20% of the best bid/ask it gets the default probability
	maxDistanceWhenNoConsensusFactor   = maxDistanceWhenNoConsensusPct.Div(num.DecimalFromFloat(100))       // if there's no consensus yet and the price is within 0.2 of the best bid/ask it gets the default probability

	maxAskMultiplier = num.DecimalFromFloat(6) // arbitrarily large factor on the best ask - 600%
)

func (probabilityOfTradingConverter) BundleToInterface(kvb *statevar.KeyValueBundle) statevar.StateVariableResult {
	return &probabilityOfTrading{
		bidProbability: kvb.KVT[0].Val.(*statevar.DecimalVector).Val,
		askProbability: kvb.KVT[1].Val.(*statevar.DecimalVector).Val,
	}
}

func (probabilityOfTradingConverter) InterfaceToBundle(res statevar.StateVariableResult) *statevar.KeyValueBundle {
	value := res.(*probabilityOfTrading)
	return &statevar.KeyValueBundle{
		KVT: []statevar.KeyValueTol{
			{Key: "bidProbability", Val: &statevar.DecimalVector{Val: value.bidProbability}, Tolerance: tolerance},
			{Key: "askProbability", Val: &statevar.DecimalVector{Val: value.askProbability}, Tolerance: tolerance},
		},
	}
}

func (e *Engine) IsProbabilityOfTradingInitialised() bool {
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
	bidProbabilities := calculateBidRange(bestBid, PriceIncrementFactor, tauScaled, e.rm.ProbabilityOfTrading)
	askProbabilities := calculateAskRange(bestAsk, PriceIncrementFactor, tauScaled, e.rm.ProbabilityOfTrading)

	res := &probabilityOfTrading{
		bidProbability: bidProbabilities,
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
func calculateBidRange(bestBid num.Decimal, priceIncrementFactor, tauScaled num.Decimal, probabilityFunc func(num.Decimal, num.Decimal, num.Decimal, num.Decimal, num.Decimal, bool, bool) num.Decimal) []num.Decimal {
	probabilities := []num.Decimal{}

	p := bestBid
	increment := priceIncrementFactor.Mul(bestBid)
	for {
		prob := probabilityFunc(bestBid, p, num.DecimalZero(), bestBid, tauScaled, true, true)
		if prob.LessThanOrEqual(defaultMinimumProbabilityOfTrading) {
			break
		}
		probabilities = append(probabilities, prob)
		p = p.Sub(increment)
	}
	return probabilities
}

// calculateAskRange calculates the probabilities of price between bestAsk and the worst ask (maxAsk)
// in increments of incrementFactor of the best ask and records the offsets and probabilities
// such that offset 0 is stored in offsets[0] which corresponds to the best ask
// and similarly probabilities[0] corresponds to best ask trading probability
// whereas the last entry in offset equals to the maximum distance from ask for which
// probability is calculated, and the last entry in probabilities corresponds to probability of trading
// at the price implied by this offset from best ask.
func calculateAskRange(bestAsk num.Decimal, priceIncrementFactor, tauScaled num.Decimal, probabilityFunc func(num.Decimal, num.Decimal, num.Decimal, num.Decimal, num.Decimal, bool, bool) num.Decimal) []num.Decimal {
	probabilities := []num.Decimal{}

	maxAsk := bestAsk.Mul(maxAskMultiplier)
	increment := priceIncrementFactor.Mul(bestAsk)

	p := bestAsk
	for {
		prob := probabilityFunc(bestAsk, p, bestAsk, maxAsk, tauScaled, false, true)
		if prob.LessThanOrEqual(defaultMinimumProbabilityOfTrading) {
			break
		}
		probabilities = append(probabilities, prob)
		p = p.Add(increment)
	}
	return probabilities
}

// updatePriceBounds is called back from the state variable consensus engine when consensus is reached for the down/up factors and updates the price bounds.
func (e *Engine) updateProbabilities(ctx context.Context, res statevar.StateVariableResult) error {
	e.pot = res.(*probabilityOfTrading)

	e.pot.bidOffset = make([]uint32, 0, len(e.pot.bidProbability))
	for i := range e.pot.bidProbability {
		e.pot.bidOffset = append(e.pot.bidOffset, uint32(i)*OffsetIncrement)
	}

	e.pot.askOffset = make([]uint32, 0, len(e.pot.bidProbability))
	for i := range e.pot.askProbability {
		e.pot.askOffset = append(e.pot.askOffset, uint32(i)*OffsetIncrement)
	}

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
func getProbabilityOfTrading(bestBid, bestAsk, minPrice, maxPrice num.Decimal, pot *probabilityOfTrading, price num.Decimal, isBid bool, minProbabilityOfTrading num.Decimal, offsetOne num.Decimal) num.Decimal {
	if (isBid && price.GreaterThanOrEqual(bestBid)) || (!isBid && price.LessThanOrEqual(bestAsk)) {
		return defaultInRangeProbabilityOfTrading
	}

	if isBid {
		if price.LessThan(minPrice) {
			return minProbabilityOfTrading
		}
		return getBidProbabilityOfTrading(bestBid, pot.bidOffset, pot.bidProbability, price, minProbabilityOfTrading, offsetOne)
	}
	if price.GreaterThan(maxPrice) {
		return minProbabilityOfTrading
	}
	return getAskProbabilityOfTrading(bestAsk, pot.askOffset, pot.askProbability, price, minProbabilityOfTrading, offsetOne)
}

func getAskProbabilityOfTrading(bestAsk num.Decimal, offsets []uint32, probabilities []num.Decimal, price, minProbabilityOfTrading num.Decimal, offsetOne num.Decimal) num.Decimal {
	// no consensus yet
	if len(offsets) == 0 {
		maxDistance := maxDistanceWhenNoConsensusFactor.Mul(bestAsk)
		if bestAsk.Sub(price).Abs().LessThanOrEqual(maxDistance) {
			return defaultInRangeProbabilityOfTrading
		}
		return minProbabilityOfTrading
	}
	offset := uint32(offsetOne.Mul(price.Sub(bestAsk).Div(bestAsk)).Floor().IntPart())

	// if outside the range - extrapolate
	if offset > offsets[len(offsets)-1] {
		return lexterp(offsets, probabilities, offset, minProbabilityOfTrading)
	}

	idx := sort.Search(len(offsets), func(i int) bool {
		return offset < offsets[i]
	})
	// linear interpolation
	interpolatedProbability := linterp(offsets, probabilities, offset, idx, minProbabilityOfTrading)
	return interpolatedProbability
}

func getBidProbabilityOfTrading(bestBid num.Decimal, offsets []uint32, probabilities []num.Decimal, price, minProbabilityOfTrading num.Decimal, offsetOne num.Decimal) num.Decimal {
	// no consensus yet
	if len(offsets) == 0 {
		maxDistance := maxDistanceWhenNoConsensusFactor.Mul(bestBid)
		if bestBid.Sub(price).Abs().LessThanOrEqual(maxDistance) {
			return defaultInRangeProbabilityOfTrading
		}
		return minProbabilityOfTrading
	}

	offset := uint32(offsetOne.Mul(bestBid.Sub(price).Div(bestBid)).Floor().IntPart())

	// if outside the range - extrapolate
	if offset > offsets[len(offsets)-1] {
		return lexterp(offsets, probabilities, offset, minProbabilityOfTrading)
	}

	idx := sort.Search(len(offsets), func(i int) bool {
		return offset < offsets[i]
	})
	// linear interpolation
	interpolatedProbability := linterp(offsets, probabilities, offset, idx, minProbabilityOfTrading)
	return interpolatedProbability
}

func lexterp(offsets []uint32, probabilities []num.Decimal, offset uint32, minProbabilityOfTrading num.Decimal) num.Decimal {
	if len(offsets) == 1 {
		return minProbabilityOfTrading
	}
	last := offsets[len(offsets)-1]
	last2 := offsets[len(offsets)-2]
	probLast := probabilities[len(offsets)-1]
	probLast2 := probabilities[len(offsets)-2]
	slopeNum := num.DecimalFromInt64(int64(offset - last2))
	slopeDenom := num.DecimalFromInt64(int64(last - last2))
	slope := slopeNum.Div(slopeDenom)
	prob := probLast2.Add(probLast.Sub(probLast2).Mul(slope))
	scaled := rescaleProbability(prob)
	prob = num.MinD(num.DecimalFromInt64(1), num.MaxD(minProbabilityOfTrading, scaled))
	return prob
}

func linterp(offsets []uint32, probabilities []num.Decimal, priceOffset uint32, i int, minProbabilityOfTrading num.Decimal) num.Decimal {
	if i >= len(probabilities) {
		return num.MaxD(minProbabilityOfTrading, rescaleProbability(probabilities[len(probabilities)-1]))
	}
	prev := offsets[i-1]
	size := offsets[i] - (prev)
	ratio := num.DecimalFromInt64(int64(priceOffset - prev)).Div(num.DecimalFromInt64(int64(size)))
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
