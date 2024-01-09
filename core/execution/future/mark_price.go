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

package future

import (
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

type CompositePriceCalculator struct {
	config           *types.CompositePriceConfiguration
	trades           []*types.Trade
	sourceLastUpdate []int64
	bookPriceAtTime  map[int64]*num.Uint
	price            *num.Uint
	// [0] trade mark price
	// [1] book mark price
	// [2] first oracel mark price
	// [2+n] median mark price
	priceSources []*num.Uint
}

const (
	TradePriceIndex       = 0
	BookPriceIndex        = 1
	FirstOraclePriceIndex = 2
)

func NewCompositePriceCalculatorFromSnapshot(mp *num.Uint, mpc *snapshot.CompositePriceCalculator) *CompositePriceCalculator {
	if mpc == nil {
		// migration - for existing markets loaded from snapshot, set the configuration to default to use last trade price
		// for mark price
		return &CompositePriceCalculator{
			config: &types.CompositePriceConfiguration{
				DecayWeight:        num.DecimalZero(),
				DecayPower:         num.DecimalZero(),
				CashAmount:         num.UintZero(),
				CompositePriceType: types.CompositePriceTypeByLastTrade,
			},
			trades:           []*types.Trade{},
			price:            mp,
			priceSources:     make([]*num.Uint, 1),
			sourceLastUpdate: make([]int64, 1),
		}
	}

	config := types.CompositePriceConfigurationFromProto(mpc.PriceConfiguration)
	trades := make([]*types.Trade, 0, len(mpc.Trades))
	for _, t := range mpc.Trades {
		trades = append(trades, types.TradeFromProto(t))
	}
	priceSources := make([]*num.Uint, 0, len(mpc.PriceSources))
	for _, v := range mpc.PriceSources {
		if len(v) == 0 {
			priceSources = append(priceSources, nil)
		} else {
			priceSources = append(priceSources, num.MustUintFromString(v, 10))
		}
	}
	var compositePrice *num.Uint
	if len(mpc.CompositePrice) > 0 {
		compositePrice = num.MustUintFromString(mpc.CompositePrice, 10)
	}

	bookPriceAtTime := make(map[int64]*num.Uint, len(mpc.BookPriceAtTime))
	for _, tp := range mpc.BookPriceAtTime {
		bookPriceAtTime[tp.Time] = num.MustUintFromString(tp.Price, 10)
	}

	return &CompositePriceCalculator{
		config:           config,
		trades:           trades,
		sourceLastUpdate: mpc.PriceSourceLastUpdate,
		priceSources:     priceSources,
		bookPriceAtTime:  bookPriceAtTime,
		price:            compositePrice,
	}
}

func NewCompositePriceCalculator(config *types.CompositePriceConfiguration) *CompositePriceCalculator {
	priceSourcesLen := len(config.SourceStalenessTolerance)
	if priceSourcesLen == 0 {
		priceSourcesLen = 1
	}

	return &CompositePriceCalculator{
		config:           config,
		priceSources:     make([]*num.Uint, priceSourcesLen),
		sourceLastUpdate: make([]int64, priceSourcesLen),
		bookPriceAtTime:  map[int64]*num.Uint{},
	}
}

func (mpc *CompositePriceCalculator) UpdateConfig(config *types.CompositePriceConfiguration) {
	priceSourcesLen := len(config.SourceStalenessTolerance)
	if priceSourcesLen == 0 {
		priceSourcesLen = 1
	}
	mpc.config = config
	mpc.priceSources = make([]*num.Uint, priceSourcesLen)
	mpc.sourceLastUpdate = make([]int64, priceSourcesLen)
}

// NewTrade is called to inform the mark price calculator on a new trade.
// All the trades for a given mark price calculation interval are saved until the end of the interval.
func (mpc *CompositePriceCalculator) NewTrade(trade *types.Trade) {
	if trade.Seller == "network" || trade.Buyer == "network" {
		return
	}
	mpc.trades = append(mpc.trades, trade)
	mpc.sourceLastUpdate[TradePriceIndex] = trade.Timestamp
}

// UpdateOraclePrice is called when a new oracle price is available.
// TODO call from market when a new price is received from one of the oracles.
func (mpc *CompositePriceCalculator) UpdateOraclePrice(oracleIndex int, price *num.Uint, t int64) {
	if mpc.config.CompositePriceType == types.CompositePriceTypeByLastTrade {
		return
	}
	mpc.priceSources[FirstOraclePriceIndex+oracleIndex] = price.Clone()
	mpc.sourceLastUpdate[FirstOraclePriceIndex+oracleIndex] = t
}

// CalculateBookMarkPriceAtTimeT is called every interval (currently at the end of each block) to calculate
// the mark price implied by the book.
// If there is insufficient quantity in the book, ignore this price
// IF the market is in auction set the mark price to the indicative price if not zero.
func (mpc *CompositePriceCalculator) CalculateBookMarkPriceAtTimeT(initialScalingFactor, slippageFactor, shortRiskFactor, longRiskFactor num.Decimal, t int64, ob *matching.CachedOrderBook) {
	if mpc.config.CompositePriceType == types.CompositePriceTypeByLastTrade {
		return
	}
	if ob.InAuction() {
		indicative := ob.GetIndicativePrice()
		if !indicative.IsZero() {
			mpc.bookPriceAtTime[t] = indicative
			mpc.sourceLastUpdate[BookPriceIndex] = t
		}
		return
	}
	mp := PriceFromBookAtTime(mpc.config.CashAmount, initialScalingFactor, slippageFactor, shortRiskFactor, longRiskFactor, ob)
	if mp != nil {
		mpc.bookPriceAtTime[t] = mp
		mpc.sourceLastUpdate[BookPriceIndex] = t
	}
}

// CalculateMarkPrice is called at the end of each mark price calculation interval and calculates the mark price
// using the mark price type methodology.
func (mpc *CompositePriceCalculator) CalculateMarkPrice(t int64, ob *matching.CachedOrderBook, markPriceFrequency time.Duration, initialScalingFactor, slippageFactor, shortRiskFactor, longRiskFactor num.Decimal) *num.Uint {
	if mpc.config.CompositePriceType == types.CompositePriceTypeByLastTrade {
		// if there are no trades, the mark price remains what it was before.
		if len(mpc.trades) > 0 {
			mpc.price = mpc.trades[len(mpc.trades)-1].Price
		}
		mpc.trades = []*types.Trade{}
		return mpc.price
	}
	if len(mpc.trades) > 0 {
		if pft := PriceFromTrades(mpc.trades, mpc.config.DecayWeight, num.DecimalFromInt64(markPriceFrequency.Nanoseconds()), mpc.config.DecayPower, t); pft != nil && !pft.IsZero() {
			mpc.priceSources[TradePriceIndex] = pft
		}
	}
	if p := CalculateTimeWeightedAverageBookPrice(mpc.bookPriceAtTime, t); p != nil {
		mpc.priceSources[BookPriceIndex] = p
	}
	if median := MedianPrice(mpc.priceSources[:len(mpc.priceSources)-1]); median != nil {
		mpc.priceSources[len(mpc.priceSources)-1] = median
	}
	if mpc.config.CompositePriceType == types.CompositePriceTypeByMedian {
		if p := CompositePriceByMedian(mpc.priceSources, mpc.sourceLastUpdate, mpc.config.SourceStalenessTolerance, t); p != nil && !p.IsZero() {
			mpc.price = p
		}
	} else {
		if p := CompositePriceByWeight(mpc.priceSources, mpc.config.SourceWeights, mpc.sourceLastUpdate, mpc.config.SourceStalenessTolerance, t); p != nil && !p.IsZero() {
			mpc.price = p
		}
	}
	mpc.trades = []*types.Trade{}
	mpc.bookPriceAtTime = map[int64]*num.Uint{}
	if p := PriceFromBookAtTime(mpc.config.CashAmount, initialScalingFactor, slippageFactor, shortRiskFactor, longRiskFactor, ob); p != nil {
		mpc.bookPriceAtTime[t] = p
	}
	return mpc.price
}

func (mpc *CompositePriceCalculator) IntoProto() *snapshot.CompositePriceCalculator {
	var compositePrice string
	if mpc.price != nil {
		compositePrice = mpc.price.String()
	}

	priceSources := make([]string, 0, len(mpc.priceSources))
	for _, u := range mpc.priceSources {
		if u == nil {
			priceSources = append(priceSources, "")
		} else {
			priceSources = append(priceSources, u.String())
		}
	}
	trades := make([]*vega.Trade, 0, len(mpc.trades))
	for _, t := range mpc.trades {
		trades = append(trades, t.IntoProto())
	}
	bookPriceAtTime := make([]*snapshot.TimePrice, 0, len(mpc.bookPriceAtTime))
	for k, u := range mpc.bookPriceAtTime {
		var p string
		if u != nil {
			p = u.String()
		}
		bookPriceAtTime = append(bookPriceAtTime, &snapshot.TimePrice{Time: k, Price: p})
	}
	sort.Slice(bookPriceAtTime, func(i, j int) bool {
		return bookPriceAtTime[i].Time < bookPriceAtTime[j].Time
	})

	return &snapshot.CompositePriceCalculator{
		CompositePrice:        compositePrice,
		PriceConfiguration:    mpc.config.IntoProto(),
		PriceSources:          priceSources,
		Trades:                trades,
		PriceSourceLastUpdate: mpc.sourceLastUpdate,
		BookPriceAtTime:       bookPriceAtTime,
	}
}
