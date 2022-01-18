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
	numberOfPricePoints                = num.NewUint(100)
	defaultProbability                 = num.DecimalFromFloat(0.05) //@witold should it be 0.05 or 0.005?
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

	bestBid, bestAsk, err := e.getBestStaticPrices()
	if err != nil {
		return
	}

	minPrice, maxPrice := e.pm.GetValidPriceRange()
	bidTo := minPrice.Representation()
	askTo := maxPrice.Representation()

	// NB: to clip the range we're calculating for in case max is max uint or min is zero so that we're calculating for a small enough range
	// we're confining the range to 0.25/4 times the current best bid/ask. This is primarily for testing, in reality the bounds could probably be much tighter.
	if maxPrice.Representation().EQ(num.MaxUint()) {
		mn, _ := num.UintFromDecimal(bestBid.ToDecimal().Mul(num.DecimalFromFloat(0.25)))
		mx, _ := num.UintFromDecimal(bestAsk.ToDecimal().Mul(num.DecimalFromFloat(4)))
		bidTo = mn
		askTo = mx
	}

	// calculate offset and probabilities between the best bid/ask and the [bid|ask]To
	bidOffsets, bidProbabilities := calculateBidRange(bestBid, bidTo, minPrice.Original(), bestBid.ToDecimal(), tauScaled, e.rm.ProbabilityOfTrading)
	askOffsets, askProbabilities := calculateAskRange(bestAsk, askTo, bestAsk.ToDecimal(), maxPrice.Original(), tauScaled, e.rm.ProbabilityOfTrading)

	res := &probabilityOfTrading{
		bidOffset:      bidOffsets,
		bidProbability: bidProbabilities,
		askOffset:      askOffsets,
		askProbability: askProbabilities,
	}
	endOfCalcCallback.CalculationFinished(eventID, res, nil)
}

// calculateBidRange calculates the probabilities of price between bestBid and the worst bid (minBid)
// in increments of <cappedTickSize> and records the offsets and probabilities
// such that offset 0 is stored in offsets[0] which corresponds to the best bid
// and similarly probabilities[0] corresponds to best bid trading probability
// whereas the last entry in offset equals to the maximum distance from bid for which
// probability is calculated, and the last entry in probabilities corresponds to
// the probability of trading at the price implied by this offset from best bid.
func calculateBidRange(bestBid, minBid *num.Uint, min, max, tauScaled num.Decimal, probabilityFunc func(*num.Uint, *num.Uint, num.Decimal, num.Decimal, num.Decimal, bool, bool) num.Decimal) ([]num.Decimal, []num.Decimal) {
	tickSize := num.Zero().Div(num.Zero().Sub(bestBid, minBid), numberOfPricePoints)
	cappedTickSize := num.Max(tickSize, num.NewUint(1))

	offsets := make([]num.Decimal, 0, int(numberOfPricePoints.Uint64()))
	probabilities := make([]num.Decimal, 0, int(numberOfPricePoints.Uint64()))

	p := bestBid.Clone()
	offset := num.Zero()
	for p.GT(minBid) {
		offsets = append(offsets, offset.ToDecimal())
		prob := probabilityFunc(bestBid, p, min, max, tauScaled, true, true)
		probabilities = append(probabilities, prob)
		if p.EQ(minBid) {
			break
		}
		offset.AddSum(cappedTickSize)
		p = p.Sub(p, cappedTickSize)
	}
	if p.LTE(minBid) {
		p = minBid
		offsets = append(offsets, offset.ToDecimal())
		prob := probabilityFunc(bestBid, p, min, max, tauScaled, true, true)
		probabilities = append(probabilities, prob)
	}
	return offsets, probabilities
}

// calculateAskRange calculates the probabilities of price between bestAsk and the worst ask (maxAsk)
// in increments of <cappedTickSize> and records the offsets and probabilities
// such that offset 0 is stored in offsets[0] which corresponds to the best ask
// and similarly probabilities[0] corresponds to best ask trading probability
// whereas the last entry in offset equals to the maximum distance from ask for which
// probability is calculated, and the last entry in probabilities corresponds to probability of trading
// at the price implied by this offset from best ask.
func calculateAskRange(bestAsk, maxAsk *num.Uint, min, max, tauScaled num.Decimal, probabilityFunc func(*num.Uint, *num.Uint, num.Decimal, num.Decimal, num.Decimal, bool, bool) num.Decimal) ([]num.Decimal, []num.Decimal) {
	tickSize := num.Zero().Div(num.Zero().Sub(maxAsk, bestAsk), numberOfPricePoints)
	cappedTickSize := num.Max(tickSize, num.NewUint(1))

	offsets := make([]num.Decimal, 0, int(numberOfPricePoints.Uint64()))
	probabilities := make([]num.Decimal, 0, int(numberOfPricePoints.Uint64()))

	p := bestAsk.Clone()
	offset := num.Zero()
	for p.LT(maxAsk) {
		offsets = append(offsets, offset.ToDecimal())
		prob := probabilityFunc(bestAsk, p, min, max, tauScaled, false, true)
		probabilities = append(probabilities, prob)
		if p.EQ(maxAsk) {
			break
		}
		offset.AddSum(cappedTickSize)
		p.AddSum(cappedTickSize)
	}
	if p.GTE(maxAsk) {
		p = maxAsk
		offsets = append(offsets, offset.ToDecimal())
		prob := probabilityFunc(bestAsk, p, min, max, tauScaled, false, true)
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
// if the price is beween the best bid and ask it returns <defaultInRangeProbabilityOfTrading>
// otherwise if we've not seen a consensus value and the price is within 100 ticks from the best (by side)
// it returns <defaultProbability> else if there is not yet consensus value it returns <minProbabilityOfTrading>.
// If there is consensus value and the price is worse than the best price for the relevant side worsened by the maximal
// offset then <minProbabilityOfTrading> is returned. If the price is within the range, the corresponding
// probability implied by the offset is returned, scaled, with lower bound of <minProbabilityOfTrading>.
func getProbabilityOfTrading(bestBid, bestAsk num.Decimal, pot *probabilityOfTrading, price num.Decimal, isBid bool, minProbabilityOfTrading num.Decimal, log *logging.Logger) num.Decimal {
	// if the price is between the *current* bid and ask, return the default in range probability
	if price.GreaterThanOrEqual(bestBid) && price.LessThanOrEqual(bestAsk) {
		log.Info("getProbabilityOfTrading price is greater than best bid and smaller than best ask", logging.Decimal("price", price), logging.Decimal("best-bid", bestBid), logging.Decimal("best-ask", bestAsk), logging.Decimal("prob", defaultInRangeProbabilityOfTrading))
		return defaultInRangeProbabilityOfTrading
	}

	if isBid {
		return getBidProbabilityOfTrading(bestBid, bestAsk, pot.bidOffset, pot.bidProbability, price, minProbabilityOfTrading, log)
	}
	return getAskProbabilityOfTrading(bestBid, bestAsk, pot.askOffset, pot.askProbability, price, minProbabilityOfTrading, log)
}

func getAskProbabilityOfTrading(bestBid, bestAsk num.Decimal, offsets, probabilities []num.Decimal, price num.Decimal, minProbabilityOfTrading num.Decimal, log *logging.Logger) num.Decimal {
	// no consensus yet
	if len(offsets) == 0 {
		if bestAsk.Sub(price).Abs().LessThanOrEqual(defaultTickDistance) {
			log.Info("getProbabilityOfTrading no first consensus yet, price is within 100 ticks from best ask", logging.Decimal("price", price), logging.Decimal("best-ask", bestAsk), logging.Decimal("prob", defaultProbability))
			return defaultProbability
		}
		log.Info("getProbabilityOfTrading no first consensus yet, price is more than 100 ticks away from best bid/ask", logging.Decimal("price", price), logging.Decimal("best-Bid", bestBid), logging.Decimal("best-ask", bestAsk), logging.Decimal("prob", minProbabilityOfTrading))
		return minProbabilityOfTrading
	}
	// check out of bounds
	maxOffset := offsets[len(offsets)-1]
	if price.GreaterThan(bestAsk.Add(maxOffset)) {
		log.Info("getProbabilityOfTrading ask price is worse than the worst consensus ask", logging.Decimal("price", price), logging.Decimal("cons-worst-ask", bestAsk.Add(maxOffset)), logging.Decimal("prob", minProbabilityOfTrading))
		return minProbabilityOfTrading
	}

	// linear interpolation
	interpolatedProbability := linterp(offsets, probabilities, price.Sub(bestAsk), func(i int) bool {
		r := bestAsk.Add(offsets[i]).GreaterThan(price)
		return r
	}, minProbabilityOfTrading)
	return interpolatedProbability
}

func getBidProbabilityOfTrading(bestBid, bestAsk num.Decimal, offsets, probabilities []num.Decimal, price num.Decimal, minProbabilityOfTrading num.Decimal, log *logging.Logger) num.Decimal {
	// no consensus yet
	if len(offsets) == 0 {
		if bestBid.Sub(price).Abs().LessThanOrEqual(defaultTickDistance) {
			log.Info("getProbabilityOfTrading no first consensus yet, price is within 100 ticks from best bid", logging.Decimal("price", price), logging.Decimal("best-bid", bestBid), logging.Decimal("prob", defaultProbability))
			return defaultProbability
		}
		log.Info("getProbabilityOfTrading no first consensus yet, price is more than 100 ticks away from best bid/ask", logging.Decimal("price", price), logging.Decimal("best-Bid", bestBid), logging.Decimal("best-ask", bestAsk), logging.Decimal("prob", minProbabilityOfTrading))
		return minProbabilityOfTrading
	}

	// check out of bounds
	maxOffset := offsets[len(offsets)-1]
	if price.LessThan(bestBid.Sub(maxOffset)) {
		log.Info("getProbabilityOfTrading bid price is worse than the worst consensus bid", logging.Decimal("price", price), logging.Decimal("cons-worst-bid", bestBid.Sub(maxOffset)), logging.Decimal("prob", minProbabilityOfTrading))
		return minProbabilityOfTrading
	}

	// linear interpolation
	interpolatedProbability := linterp(offsets, probabilities, bestBid.Sub(price), func(i int) bool {
		r := bestBid.Sub(offsets[i]).LessThan(price)
		return r
	}, minProbabilityOfTrading)
	return interpolatedProbability
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
