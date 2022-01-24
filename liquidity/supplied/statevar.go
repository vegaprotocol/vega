package supplied

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/types/statevar"
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
	numberOfPricePoints                = num.NewUint(500)
	defaultProbability                 = num.DecimalFromFloat(0.05)
	defaultTickDistance                = num.DecimalFromFloat(100)
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
		return
	}

	// calculate how far from the best we want to calculate
	distanceFromBest := e.tickSize.ToDecimal().Mul(numberOfPricePoints.ToDecimal())

	// we're calculating between the best ask up to ask+distanceFromBest
	askTo, _ := num.UintFromDecimal(bestAsk.ToDecimal().Add(distanceFromBest))
	// we're calculating between the best bid down to the max(0, bid-distanceFromBest)
	bidTo, _ := num.UintFromDecimal(num.MaxD(num.DecimalZero(), bestBid.ToDecimal().Sub(distanceFromBest)))

	// calculate offsets and probabilities for the range
	bidOffsets, bidProbabilities := calculateBidRange(bestBid, bidTo, e.tickSize, tauScaled, e.rm.ProbabilityOfTrading)
	askOffsets, askProbabilities := calculateAskRange(bestAsk, askTo, e.tickSize, tauScaled, e.rm.ProbabilityOfTrading)

	res := &probabilityOfTrading{
		bidOffset:      bidOffsets,
		bidProbability: bidProbabilities,
		askOffset:      askOffsets,
		askProbability: askProbabilities,
	}
	endOfCalcCallback.CalculationFinished(eventID, res, nil)
}

// calculateBidRange calculates the probabilities of price between bestBid and the worst bid (minBid)
// in increments of <tickSize> and records the offsets and probabilities
// such that offset 0 is stored in offsets[0] which corresponds to the best bid
// and similarly probabilities[0] corresponds to best bid trading probability
// whereas the last entry in offset equals to the maximum distance from bid for which
// probability is calculated, and the last entry in probabilities corresponds to
// the probability of trading at the price implied by this offset from best bid.
func calculateBidRange(bestBid, minBid, tickSize *num.Uint, tauScaled num.Decimal, probabilityFunc func(*num.Uint, *num.Uint, num.Decimal, num.Decimal, num.Decimal, bool, bool) num.Decimal) ([]num.Decimal, []num.Decimal) {
	bbDecimal := bestBid.ToDecimal()
	mbDecimal := minBid.ToDecimal()

	offsets := make([]num.Decimal, 0, int(numberOfPricePoints.Uint64()))
	probabilities := make([]num.Decimal, 0, int(numberOfPricePoints.Uint64()))

	p := bestBid.Clone()
	offset := num.Zero()
	for p.GT(minBid) && !p.IsNegative() {
		offsets = append(offsets, offset.ToDecimal())
		prob := probabilityFunc(bestBid, p, mbDecimal, bbDecimal, tauScaled, true, true)
		probabilities = append(probabilities, prob)
		if p.EQ(minBid) {
			break
		}
		offset.AddSum(tickSize)
		p = p.Sub(p, tickSize)
	}
	if p.LTE(minBid) || p.IsNegative() {
		p = minBid
		offsets = append(offsets, offset.ToDecimal())
		prob := probabilityFunc(bestBid, p, mbDecimal, bbDecimal, tauScaled, true, true)
		probabilities = append(probabilities, prob)
	}
	return offsets, probabilities
}

// calculateAskRange calculates the probabilities of price between bestAsk and the worst ask (maxAsk)
// in increments of <tickSize> and records the offsets and probabilities
// such that offset 0 is stored in offsets[0] which corresponds to the best ask
// and similarly probabilities[0] corresponds to best ask trading probability
// whereas the last entry in offset equals to the maximum distance from ask for which
// probability is calculated, and the last entry in probabilities corresponds to probability of trading
// at the price implied by this offset from best ask.
func calculateAskRange(bestAsk, maxAsk, tickSize *num.Uint, tauScaled num.Decimal, probabilityFunc func(*num.Uint, *num.Uint, num.Decimal, num.Decimal, num.Decimal, bool, bool) num.Decimal) ([]num.Decimal, []num.Decimal) {
	baDecimal := bestAsk.ToDecimal()
	maDecimal := maxAsk.ToDecimal()

	offsets := make([]num.Decimal, 0, int(numberOfPricePoints.Uint64()))
	probabilities := make([]num.Decimal, 0, int(numberOfPricePoints.Uint64()))

	p := bestAsk.Clone()
	offset := num.Zero()
	for p.LT(maxAsk) {
		offsets = append(offsets, offset.ToDecimal())
		prob := probabilityFunc(bestAsk, p, baDecimal, maDecimal, tauScaled, false, true)
		probabilities = append(probabilities, prob)
		if p.EQ(maxAsk) {
			break
		}
		offset.AddSum(tickSize)
		p.AddSum(tickSize)
	}
	if p.GTE(maxAsk) {
		p = maxAsk
		offsets = append(offsets, offset.ToDecimal())
		prob := probabilityFunc(bestAsk, p, baDecimal, maDecimal, tauScaled, false, true)
		probabilities = append(probabilities, prob)
	}
	return offsets, probabilities
}

// updatePriceBounds is called back from the state variable consensus engine when consensus is reached for the down/up factors and updates the price bounds.
func (e *Engine) updateProbabilities(ctx context.Context, res statevar.StateVariableResult) error {
	e.pot = res.(*probabilityOfTrading)
	e.log.Info("consensus reached for probability of trading", logging.String("market", e.marketID))

	e.potInitialised = true
	e.changed = true
	return nil
}

// getProbabilityOfTrading returns the probability of trading for the given price
// if the price is a bid order that is better than the best bid or an ask order better than the best and ask it returns <defaultInRangeProbabilityOfTrading>
// if the price is a bid worse than min or an ask worse than max it returns minProbabilityOfTrading
// otherwise if we've not seen a consensus value and the price is within 100 ticks from the best (by side)
// it returns <defaultProbability> else if there is not yet consensus value it returns <minProbabilityOfTrading>.
// If there is consensus value and the price is worse than the worse price, we extrapolate using the last 2 price points
// If the price is within the range, the corresponding probability implied by the offset is returned, scaled, with lower bound of <minProbabilityOfTrading>.
func getProbabilityOfTrading(bestBid, bestAsk, minPrice, maxPrice num.Decimal, pot *probabilityOfTrading, price num.Decimal, isBid bool, minProbabilityOfTrading num.Decimal, tickSize num.Decimal) num.Decimal {
	if (isBid && price.GreaterThanOrEqual(bestBid)) || (!isBid && price.LessThanOrEqual(bestAsk)) {
		return defaultInRangeProbabilityOfTrading
	}

	// when we don't have consensus yet we'll allow prices that are within
	// tickSize * defaultTickDistance from the best
	maxDistanceWhenNoConsensus := defaultTickDistance.Mul(tickSize)

	if isBid {
		if price.LessThan(minPrice) {
			return minProbabilityOfTrading
		}
		return getBidProbabilityOfTrading(bestBid, bestAsk, pot.bidOffset, pot.bidProbability, price, minProbabilityOfTrading, maxDistanceWhenNoConsensus)
	}
	if price.GreaterThan(maxPrice) {
		return minProbabilityOfTrading
	}
	return getAskProbabilityOfTrading(bestBid, bestAsk, pot.askOffset, pot.askProbability, price, minProbabilityOfTrading, maxDistanceWhenNoConsensus)
}

func getAskProbabilityOfTrading(bestBid, bestAsk num.Decimal, offsets, probabilities []num.Decimal, price num.Decimal, minProbabilityOfTrading num.Decimal, maxDistance num.Decimal) num.Decimal {
	// no consensus yet
	if len(offsets) == 0 {
		if bestAsk.Sub(price).Abs().LessThanOrEqual(maxDistance) {
			return defaultProbability
		}
		return minProbabilityOfTrading
	}
	offset := price.Sub(bestAsk)
	// if outside the range - extrapolate
	if offset.GreaterThan(offsets[len(offsets)-1]) {
		return lexterp(offsets, probabilities, offset, minProbabilityOfTrading)
	}

	// linear interpolation
	interpolatedProbability := linterp(offsets, probabilities, offset, func(i int) bool {
		r := bestAsk.Add(offsets[i]).GreaterThan(price)
		return r
	}, minProbabilityOfTrading)
	return interpolatedProbability
}

func getBidProbabilityOfTrading(bestBid, bestAsk num.Decimal, offsets, probabilities []num.Decimal, price num.Decimal, minProbabilityOfTrading, maxDistance num.Decimal) num.Decimal {
	// no consensus yet
	if len(offsets) == 0 {
		if bestBid.Sub(price).Abs().LessThanOrEqual(maxDistance) {
			return defaultProbability
		}
		return minProbabilityOfTrading
	}

	offset := bestBid.Sub(price)
	// if outside the range - extrapolate
	if offset.GreaterThan(offsets[len(offsets)-1]) {
		return lexterp(offsets, probabilities, offset, minProbabilityOfTrading)
	}

	// linear interpolation
	interpolatedProbability := linterp(offsets, probabilities, offset, func(i int) bool {
		r := bestBid.Sub(offsets[i]).LessThan(price)
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
