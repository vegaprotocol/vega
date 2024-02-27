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

package common

import (
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

// timeWeight calculates the time weight for the given trade time given the current time.
func timeWeight(alpha, lambda, decayPower num.Decimal, t, tradeTime int64) num.Decimal {
	if alpha.IsZero() {
		return num.DecimalOne()
	}
	timeFraction := num.DecimalFromInt64(t - tradeTime).Div(lambda)
	res := num.DecimalOne().Sub(alpha.Mul(timeFraction.Pow(decayPower)))
	if res.IsNegative() {
		return num.DecimalZero()
	}
	return res
}

// PriceFromTrades calculates the mark price from trades in the current price frequency.
func PriceFromTrades(trades []*types.Trade, decayWeight, lambda, decayPower num.Decimal, t int64) *num.Uint {
	if lambda.IsZero() {
		return nil
	}
	wSum := num.DecimalZero()
	ptSum := num.DecimalZero()

	totalTradedSize := int64(0)
	for _, trade := range trades {
		totalTradedSize += int64(trade.Size)
	}
	totalTradedSizeD := num.DecimalFromInt64(totalTradedSize)

	for _, trade := range trades {
		weightedSize := timeWeight(decayWeight, lambda, decayPower, t, trade.Timestamp).Mul(num.DecimalFromInt64(int64(trade.Size)).Div(totalTradedSizeD))
		wSum = wSum.Add(weightedSize)
		ptSum = ptSum.Add(weightedSize.Mul(trade.Price.ToDecimal()))
	}
	// if all trades have time weight 0, there's no price from trades.
	if wSum.IsZero() {
		return nil
	}
	ptUint, _ := num.UintFromDecimal(ptSum.Div(wSum))
	return ptUint
}

// PriceFromBookAtTime calculate the mark price as the average price of buying/selling the quantity implied by scaling C
// by the factors. If there is no bid or ask price for the required quantity, returns nil.
func PriceFromBookAtTime(C *num.Uint, initialScalingFactor, slippageFactor, shortRiskFactor, longRiskFactor num.Decimal, orderBook *matching.CachedOrderBook) *num.Uint {
	bestAsk, err := orderBook.GetBestAskPrice()
	// no best ask
	if err != nil {
		return nil
	}
	bestBid, err := orderBook.GetBestBidPrice()
	// no best bid
	if err != nil {
		return nil
	}

	vBuy := uint64(C.ToDecimal().Div(initialScalingFactor.Mul(slippageFactor.Add(shortRiskFactor))).Div(bestBid.ToDecimal()).IntPart())
	vwapBuy, err := orderBook.VWAP(vBuy, types.SideBuy)
	// insufficient quantity in the book for vbuy quantity
	if err != nil {
		return nil
	}

	vSell := uint64(C.ToDecimal().Div(initialScalingFactor.Mul(slippageFactor.Add(longRiskFactor))).Div(bestAsk.ToDecimal()).IntPart())
	vwapSell, err := orderBook.VWAP(vSell, types.SideSell)
	// insufficient quantity in the book for vsell quantity
	if err != nil {
		return nil
	}

	return num.UintZero().Div(vwapSell.AddSum(vwapBuy), num.NewUint(2))
}

// MedianPrice returns the median of the given prices (pBook, pTrades, pOracle1..n).
func MedianPrice(prices []*num.Uint) *num.Uint {
	if prices == nil {
		return nil
	}

	return num.Median(prices)
}

// CompositePriceByMedian returns the median mark price out of the non stale ones or nil if there is none.
func CompositePriceByMedian(prices []*num.Uint, lastUpdate []int64, delta []time.Duration, t int64) *num.Uint {
	pricesToConsider := []*num.Uint{}
	for i, u := range prices {
		if t-lastUpdate[i] <= delta[i].Nanoseconds() && u != nil && !u.IsZero() {
			pricesToConsider = append(pricesToConsider, u)
		}
	}
	if len(pricesToConsider) == 0 {
		return nil
	}
	return num.Median(pricesToConsider)
}

// CompositePriceByWeight calculates the mid price out of the non-stale price by the weights assigned to each mid price.
func CompositePriceByWeight(prices []*num.Uint, weights []num.Decimal, lastUpdateTime []int64, delta []time.Duration, t int64) *num.Uint {
	pricesToConsider := []*num.Uint{}
	priceWeights := []num.Decimal{}
	weightSum := num.DecimalZero()
	for i, u := range prices {
		if t-lastUpdateTime[i] <= delta[i].Nanoseconds() && u != nil && !u.IsZero() {
			pricesToConsider = append(pricesToConsider, u)
			priceWeights = append(priceWeights, weights[i])
			weightSum = weightSum.Add(weights[i])
		}
	}
	if len(pricesToConsider) == 0 || weightSum.IsZero() {
		return nil
	}
	price := num.UintZero()
	for i := 0; i < len(pricesToConsider); i++ {
		mp, _ := num.UintFromDecimal(pricesToConsider[i].ToDecimal().Mul(priceWeights[i]).Div(weightSum))
		price.AddSum(mp)
	}
	return price
}

// CalculateTimeWeightedAverageBookPrice calculates the time weighted average of the timepoints where book price
// was calculated.
func CalculateTimeWeightedAverageBookPrice(timeToPrice map[int64]*num.Uint, t int64, markPricePeriod int64) *num.Uint {
	if len(timeToPrice) == 0 {
		return nil
	}

	keys := make([]int64, 0, len(timeToPrice))
	for k := range timeToPrice {
		if k >= t-markPricePeriod {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	if len(keys) == 0 {
		return nil
	}
	totalDuration := num.DecimalFromInt64(t - keys[0])
	mp := num.DecimalZero()
	for i, timepoint := range keys {
		var duration int64
		if i == len(keys)-1 {
			duration = t - timepoint
		} else {
			duration = keys[i+1] - timepoint
		}
		var timeWeight num.Decimal
		if totalDuration.IsZero() {
			if len(keys) == 1 {
				// if there's just one observation it should get all the weight
				timeWeight = num.DecimalOne()
			} else {
				timeWeight = num.DecimalZero()
			}
		} else {
			timeWeight = num.DecimalFromInt64(duration).Div(totalDuration)
		}

		mp = mp.Add(timeWeight.Mul(timeToPrice[timepoint].ToDecimal()))
	}
	mpAsU, _ := num.UintFromDecimal(mp)
	return mpAsU
}
