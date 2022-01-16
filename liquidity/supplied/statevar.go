package supplied

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/types/statevar"
)

type probabilityOfTradingConverter struct{}

type probabilityOfTrading struct {
	bidPrice       []num.Decimal
	bidProbability []num.Decimal
	askPrice       []num.Decimal
	askProbability []num.Decimal
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
		bidPrice:       kvb.KVT[0].Val.(*statevar.DecimalVector).Val,
		bidProbability: kvb.KVT[1].Val.(*statevar.DecimalVector).Val,
		askPrice:       kvb.KVT[2].Val.(*statevar.DecimalVector).Val,
		askProbability: kvb.KVT[3].Val.(*statevar.DecimalVector).Val,
	}
}

func (probabilityOfTradingConverter) InterfaceToBundle(res statevar.StateVariableResult) *statevar.KeyValueBundle {
	value := res.(*probabilityOfTrading)
	return &statevar.KeyValueBundle{
		KVT: []statevar.KeyValueTol{
			{Key: "bidPrice", Val: &statevar.DecimalVector{Val: value.bidPrice}, Tolerance: tolerance},
			{Key: "bidProbability", Val: &statevar.DecimalVector{Val: value.bidProbability}, Tolerance: tolerance},
			{Key: "askPrice", Val: &statevar.DecimalVector{Val: value.askPrice}, Tolerance: tolerance},
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
	bidFrom := minPrice.Representation()
	askTo := maxPrice.Representation()

	// NB: to clip the range we're calculating for in case max is max uint or min is zero so that we're calculating for a small enough range
	// we're confining the range to 0.25/4 times the current best bid/ask. This is primarily for testing, in reality the bounds could probably be much tighter.
	if maxPrice.Representation().EQ(num.MaxUint()) {
		mn, _ := num.UintFromDecimal(bestBid.ToDecimal().Mul(num.DecimalFromFloat(0.25)))
		mx, _ := num.UintFromDecimal(bestAsk.ToDecimal().Mul(num.DecimalFromFloat(4)))
		bidFrom = mn
		askTo = mx
	}

	bidPrices, bidProbabilities := calculateRange(bestBid, bidFrom, bestBid, minPrice.Original(), bestBid.ToDecimal(), tauScaled, true, e.rm.ProbabilityOfTrading)
	askPrices, askProbabilities := calculateRange(bestAsk, bestAsk, askTo, bestAsk.ToDecimal(), maxPrice.Original(), tauScaled, false, e.rm.ProbabilityOfTrading)

	res := &probabilityOfTrading{
		bidPrice:       bidPrices,
		bidProbability: bidProbabilities,
		askPrice:       askPrices,
		askProbability: askProbabilities,
	}
	endOfCalcCallback.CalculationFinished(eventID, res, nil)
}

// calculateRange generates a range of prices and corresponding probabilities between the given min price and given max price
// for bid this is expected to go from the min price in the price range to the best bid
// for ask this is expected to go from best ask to the max price in the price range
// the price increment in the range is attempting to have 100 price points but may have more (up to 199 - e.g. range between 100-199).
func calculateRange(best, from, to *num.Uint, min, max, tauScaled num.Decimal, isBid bool, probabilityFunc func(*num.Uint, *num.Uint, num.Decimal, num.Decimal, num.Decimal, bool, bool) num.Decimal) ([]num.Decimal, []num.Decimal) {
	tickSize := num.Zero().Div(num.Zero().Sub(to, from), numberOfPricePoints)
	cappedTickSize := num.Max(tickSize, num.NewUint(1))

	prices := make([]num.Decimal, 0, int(numberOfPricePoints.Uint64()))
	probabilities := make([]num.Decimal, 0, int(numberOfPricePoints.Uint64()))

	p := from.Clone()
	for p.LTE(to) {
		prices = append(prices, p.ToDecimal())
		prob := probabilityFunc(best, p, min, max, tauScaled, isBid, true)
		probabilities = append(probabilities, prob)
		p.AddSum(cappedTickSize)
	}
	if p.LT(to) {
		p = to
		prices = append(prices, p.ToDecimal())
		prob := probabilityFunc(best, p, min, max, tauScaled, isBid, true)
		probabilities = append(probabilities, prob)
	}
	return prices, probabilities
}

// updatePriceBounds is called back from the state variable consensus engine when consensus is reached for the down/up factors and updates the price bounds.
func (e *Engine) updateProbabilities(ctx context.Context, res statevar.StateVariableResult) error {
	e.pot = res.(*probabilityOfTrading)
	e.potInitialised = true
	e.changed = true
	return nil
}

// getProbabilityOfTrading returns the probability of trading for the given price
// if we can't get best bid/ask it returns <minProbabilityOfTrading>
// if the given price is between best bid and best ask it returns <defaultInRangeProbabilityOfTrading>
// if there is not yet first consensus probabilities and the price is within <defaultTickDistance> ticks from the best bid and is a bid price it returns <defaultProbability>
// if there is not yet first consensus probabilities and the price is within <defaultTickDistance> ticks from the best ask and is a ask price it returns <defaultProbability>
// else if there is not yet first consensus probabilities it returns <defaultInRangeProbabilityOfTrading>
// if the price is worse than the min bid and is a bid price or is worse than the max ask and is an ask price it returns <minProbabilityOfTrading>
// if it matches a price point - the corresponding probability is returned (scaled by <defaultInRangeProbabilityOfTrading>)
// otherwise it finds the first price point that is greater than the given price and returns the interpolation of the probabilities of this price point and the preceding one rescaled by <defaultInRangeProbabilityOfTrading>.
func getProbabilityOfTrading(bestBid, bestAsk num.Decimal, pot *probabilityOfTrading, price num.Decimal, isBid bool, minProbabilityOfTrading num.Decimal) num.Decimal {
	// if the price is between the *current* bid and ask, return the default in range probability
	if price.GreaterThanOrEqual(bestBid) && price.LessThanOrEqual(bestAsk) {
		return defaultInRangeProbabilityOfTrading
	}

	// no consensus yet
	if (len(pot.bidPrice) == 0 && isBid) || (len(pot.askPrice) == 0 && !isBid) {
		if isBid && bestBid.Sub(price).Abs().LessThanOrEqual(defaultTickDistance) {
			return defaultProbability
		}
		if !isBid && bestAsk.Sub(price).Abs().LessThanOrEqual(defaultTickDistance) {
			return defaultProbability
		}
		return minProbabilityOfTrading
	}

	// find the first price greater or equal the given price
	prices := pot.bidPrice
	probabilities := pot.bidProbability
	if !isBid {
		prices = pot.askPrice
		probabilities = pot.askProbability
	}

	// NB: if the price is a bid order and it's better than the best bid at the time the probabilities were calculated
	// or the price is an ask order and it's better than the best ask at the time the probabilities were calculated
	// we return the <defaultInRangeProbabilityOfTrading>
	if (isBid && price.GreaterThanOrEqual(prices[len(prices)-1])) || (!isBid && price.LessThanOrEqual(prices[0])) {
		return defaultInRangeProbabilityOfTrading
	}

	// check out of bounds
	if isBid && price.LessThan(prices[0]) || !isBid && price.GreaterThan(prices[len(prices)-1]) {
		return minProbabilityOfTrading
	}

	// find the first price >= price
	i := sort.Search(len(prices), func(i int) bool { return prices[i].GreaterThanOrEqual(price) })
	if prices[i].Equals(price) {
		return num.MaxD(minProbabilityOfTrading, rescaleProbability(probabilities[i]))
	}

	// linear interpolation
	prev := prices[i-1]
	size := prices[i].Sub(prev)
	ratio := price.Sub(prev).Div(size)
	cRatio := num.DecimalFromInt64(1).Sub(ratio)
	prob := ratio.Mul(probabilities[i]).Add(cRatio.Mul(probabilities[i-1]))
	return num.MaxD(minProbabilityOfTrading, rescaleProbability(prob))
}

// rescaleProbability rescales probability so that it's at most the value returned between bid and ask.
func rescaleProbability(prob num.Decimal) num.Decimal {
	return prob.Mul(defaultInRangeProbabilityOfTrading)
}
