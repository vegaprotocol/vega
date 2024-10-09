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

package matching

import (
	"slices"
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"golang.org/x/exp/maps"
)

type IndicativePriceAndVolume struct {
	log    *logging.Logger
	levels []ipvPriceLevel

	// this is just used to avoid allocations
	buf []CumulativeVolumeLevel

	// keep track of previouses {min/max}Price
	// and if orders has been add in the book
	// with a price in the range
	lastMinPrice, lastMaxPrice *num.Uint
	lastMaxTradable            uint64
	lastCumulativeVolumes      []CumulativeVolumeLevel
	needsUpdate                bool

	// keep track of expanded off book orders
	offbook   OffbookSource
	generated map[string]*ipvGeneratedOffbook
}

type ipvPriceLevel struct {
	price  *num.Uint
	buypl  ipvVolume
	sellpl ipvVolume
}

type ipvVolume struct {
	volume        uint64
	offbookVolume uint64 // how much of the above total volume is coming from AMMs
}

type ipvGeneratedOffbook struct {
	buy    []*types.Order
	sell   []*types.Order
	approx bool
}

func (g *ipvGeneratedOffbook) add(order *types.Order) {
	if order.Side == types.SideSell {
		g.sell = append(g.sell, order)
		return
	}
	g.buy = append(g.buy, order)
}

func NewIndicativePriceAndVolume(log *logging.Logger, buy, sell *OrderBookSide) *IndicativePriceAndVolume {
	bestBid, _, err := buy.BestPriceAndVolume()
	if err != nil {
		bestBid = num.UintZero()
	}
	bestAsk, _, err := sell.BestPriceAndVolume()
	if err != nil {
		bestAsk = num.UintZero()
	}

	if buy.offbook != nil {
		bid, _, ask, _ := buy.offbook.BestPricesAndVolumes()
		if bid != nil {
			if bestBid.IsZero() {
				bestBid = bid
			} else {
				bestBid = num.Max(bestBid, bid)
			}
		}
		if ask != nil {
			if bestAsk.IsZero() {
				bestAsk = ask
			} else {
				bestAsk = num.Min(bestAsk, ask)
			}
		}
	}

	ipv := IndicativePriceAndVolume{
		levels:       []ipvPriceLevel{},
		log:          log,
		lastMinPrice: num.UintZero(),
		lastMaxPrice: num.UintZero(),
		needsUpdate:  true,
		offbook:      buy.offbook,
		generated:    map[string]*ipvGeneratedOffbook{},
	}

	// if they are crossed set the last min/max values otherwise leave as zero
	if bestAsk.LTE(bestBid) {
		ipv.lastMinPrice = bestAsk
		ipv.lastMaxPrice = bestBid
	}

	ipv.buildInitialCumulativeLevels(buy, sell)
	// initialize at the size of all levels at start, we most likely
	// not gonna need any other allocation if we start an auction
	// on an existing market
	ipv.buf = make([]CumulativeVolumeLevel, len(ipv.levels))
	return &ipv
}

func (ipv *IndicativePriceAndVolume) buildInitialOffbookShape(offbook OffbookSource, mplm map[num.Uint]ipvPriceLevel) {
	min, max := ipv.lastMinPrice, ipv.lastMaxPrice
	if min.IsZero() || max.IsZero() || min.GT(max) {
		// region is not crossed so we won't expand just yet
		return
	}

	// expand all AMM's into orders within the crossed region and add them to the price-level cache
	r := offbook.OrderbookShape(min, max, nil)

	for _, shape := range r {
		buys := shape.Buys
		sells := shape.Sells

		for i := len(buys) - 1; i >= 0; i-- {
			o := buys[i]
			mpl, ok := mplm[*o.Price]
			if !ok {
				mpl = ipvPriceLevel{price: o.Price, buypl: ipvVolume{0, 0}, sellpl: ipvVolume{0, 0}}
			}
			// increment the volume at this level
			mpl.buypl.volume += o.Size
			mpl.buypl.offbookVolume += o.Size
			mplm[*o.Price] = mpl

			if ipv.generated[o.Party] == nil {
				ipv.generated[o.Party] = &ipvGeneratedOffbook{approx: shape.Approx}
			}
			ipv.generated[o.Party].add(o)
		}

		for _, o := range sells {
			mpl, ok := mplm[*o.Price]
			if !ok {
				mpl = ipvPriceLevel{price: o.Price, buypl: ipvVolume{0, 0}, sellpl: ipvVolume{0, 0}}
			}

			mpl.sellpl.volume += o.Size
			mpl.sellpl.offbookVolume += o.Size
			mplm[*o.Price] = mpl

			if ipv.generated[o.Party] == nil {
				ipv.generated[o.Party] = &ipvGeneratedOffbook{approx: shape.Approx}
			}
			ipv.generated[o.Party].add(o)
		}
	}
}

func (ipv *IndicativePriceAndVolume) removeOffbookShape(party string) {
	orders, ok := ipv.generated[party]
	if !ok {
		return
	}

	// remove all the old volume for the AMM's
	for _, o := range orders.buy {
		ipv.RemoveVolumeAtPrice(o.Price, o.Size, o.Side, true)
	}
	for _, o := range orders.sell {
		ipv.RemoveVolumeAtPrice(o.Price, o.Size, o.Side, true)
	}

	// clear it out the saved generated orders for the offbook shape
	delete(ipv.generated, party)
}

func (ipv *IndicativePriceAndVolume) addOffbookShape(party *string, minPrice, maxPrice *num.Uint, excludeMin, excludeMax bool) {
	// recalculate new orders for the shape and add the volume in
	r := ipv.offbook.OrderbookShape(minPrice, maxPrice, party)

	for _, shape := range r {
		buys := shape.Buys
		sells := shape.Sells

		if len(buys) == 0 && len(sells) == 0 {
			continue
		}

		if _, ok := ipv.generated[shape.AmmParty]; !ok {
			ipv.generated[shape.AmmParty] = &ipvGeneratedOffbook{approx: shape.Approx}
		}

		// add buys backwards so that the best-bid is first
		for i := len(buys) - 1; i >= 0; i-- {
			o := buys[i]

			if excludeMin && o.Price.EQ(minPrice) {
				continue
			}
			if excludeMax && o.Price.EQ(maxPrice) {
				continue
			}

			ipv.AddVolumeAtPrice(o.Price, o.Size, o.Side, true)
			ipv.generated[shape.AmmParty].add(o)
		}

		// add buys fowards so that the best-ask is first
		for _, o := range sells {
			if excludeMin && o.Price.EQ(minPrice) {
				continue
			}
			if excludeMax && o.Price.EQ(maxPrice) {
				continue
			}

			ipv.AddVolumeAtPrice(o.Price, o.Size, o.Side, true)
			ipv.generated[shape.AmmParty].add(o)
		}
	}
}

func (ipv *IndicativePriceAndVolume) updateOffbookState(minPrice, maxPrice *num.Uint) {
	parties := maps.Keys(ipv.generated)
	for _, p := range parties {
		ipv.removeOffbookShape(p)
	}

	if minPrice.GT(maxPrice) {
		// region is not crossed so we won't expand just yet
		return
	}

	ipv.addOffbookShape(nil, minPrice, maxPrice, false, false)
}

// this will be used to build the initial set of price levels, when the auction is being started.
func (ipv *IndicativePriceAndVolume) buildInitialCumulativeLevels(buy, sell *OrderBookSide) {
	// we'll keep track of all the pl we encounter
	mplm := map[num.Uint]ipvPriceLevel{}

	for i := len(buy.levels) - 1; i >= 0; i-- {
		mplm[*buy.levels[i].price] = ipvPriceLevel{price: buy.levels[i].price.Clone(), buypl: ipvVolume{buy.levels[i].volume, 0}, sellpl: ipvVolume{0, 0}}
	}

	// now we add all the sells
	// to our list of pricelevel
	// making sure we have no duplicates
	for i := len(sell.levels) - 1; i >= 0; i-- {
		price := sell.levels[i].price.Clone()
		if mpl, ok := mplm[*price]; ok {
			mpl.sellpl = ipvVolume{sell.levels[i].volume, 0}
			mplm[*price] = mpl
		} else {
			mplm[*price] = ipvPriceLevel{price: price, sellpl: ipvVolume{sell.levels[i].volume, 0}, buypl: ipvVolume{0, 0}}
		}
	}

	if buy.offbook != nil {
		ipv.buildInitialOffbookShape(buy.offbook, mplm)
	}

	// now we insert them all in the slice.
	// so we can sort them
	ipv.levels = make([]ipvPriceLevel, 0, len(mplm))
	for _, v := range mplm {
		ipv.levels = append(ipv.levels, v)
	}

	// sort the slice so we can go through each levels nicely
	sort.Slice(ipv.levels, func(i, j int) bool { return ipv.levels[i].price.GT(ipv.levels[j].price) })
}

func (ipv *IndicativePriceAndVolume) incrementLevelVolume(idx int, volume uint64, side types.Side, isOffbook bool) {
	switch side {
	case types.SideBuy:
		ipv.levels[idx].buypl.volume += volume
		if isOffbook {
			ipv.levels[idx].buypl.offbookVolume += volume
		}
	case types.SideSell:
		ipv.levels[idx].sellpl.volume += volume
		if isOffbook {
			ipv.levels[idx].sellpl.offbookVolume += volume
		}
	}
}

func (ipv *IndicativePriceAndVolume) AddVolumeAtPrice(price *num.Uint, volume uint64, side types.Side, isOffbook bool) {
	if price.GTE(ipv.lastMinPrice) || price.LTE(ipv.lastMaxPrice) {
		// the new price added is in the range, that will require
		// to recompute the whole range when we call GetCumulativePriceLevels
		// again
		ipv.needsUpdate = true
	}
	i := sort.Search(len(ipv.levels), func(i int) bool {
		return ipv.levels[i].price.LTE(price)
	})
	if i < len(ipv.levels) && ipv.levels[i].price.EQ(price) {
		// we found the price level, let's add the volume there, and we are done
		ipv.incrementLevelVolume(i, volume, side, isOffbook)
	} else {
		ipv.levels = append(ipv.levels, ipvPriceLevel{})
		copy(ipv.levels[i+1:], ipv.levels[i:])
		ipv.levels[i] = ipvPriceLevel{price: price.Clone()}
		ipv.incrementLevelVolume(i, volume, side, isOffbook)
	}
}

func (ipv *IndicativePriceAndVolume) decrementLevelVolume(idx int, volume uint64, side types.Side, isOffbook bool) {
	switch side {
	case types.SideBuy:
		ipv.levels[idx].buypl.volume -= volume
		if isOffbook {
			ipv.levels[idx].buypl.offbookVolume -= volume
		}
	case types.SideSell:
		ipv.levels[idx].sellpl.volume -= volume
		if isOffbook {
			ipv.levels[idx].sellpl.offbookVolume -= volume
		}
	}
}

func (ipv *IndicativePriceAndVolume) RemoveVolumeAtPrice(price *num.Uint, volume uint64, side types.Side, isOffbook bool) {
	if price.GTE(ipv.lastMinPrice) || price.LTE(ipv.lastMaxPrice) {
		// the new price added is in the range, that will require
		// to recompute the whole range when we call GetCumulativePriceLevels
		// again
		ipv.needsUpdate = true
	}
	i := sort.Search(len(ipv.levels), func(i int) bool {
		return ipv.levels[i].price.LTE(price)
	})
	if i < len(ipv.levels) && ipv.levels[i].price.EQ(price) {
		// we found the price level, let's add the volume there, and we are done
		ipv.decrementLevelVolume(i, volume, side, isOffbook)
	} else {
		ipv.log.Panic("cannot remove volume from a non-existing level",
			logging.String("side", side.String()),
			logging.BigUint("price", price),
			logging.Uint64("volume", volume))
	}
}

func (ipv *IndicativePriceAndVolume) getLevelsWithinRange(maxPrice, minPrice *num.Uint) []ipvPriceLevel {
	// these are ordered descending, se we gonna find first the maxPrice then
	// the minPrice, and using that we can then subslice like a boss
	maxPricePos := sort.Search(len(ipv.levels), func(i int) bool {
		return ipv.levels[i].price.LTE(maxPrice)
	})
	if maxPricePos >= len(ipv.levels) || ipv.levels[maxPricePos].price.NEQ(maxPrice) {
		// price level not present, that should not be possible?
		ipv.log.Panic("missing max price in levels",
			logging.BigUint("max-price", maxPrice))
	}
	minPricePos := sort.Search(len(ipv.levels), func(i int) bool {
		return ipv.levels[i].price.LTE(minPrice)
	})
	if minPricePos >= len(ipv.levels) || ipv.levels[minPricePos].price.NEQ(minPrice) {
		// price level not present, that should not be possible?
		ipv.log.Panic("missing min price in levels",
			logging.BigUint("min-price", minPrice))
	}

	return ipv.levels[maxPricePos : minPricePos+1]
}

func (ipv *IndicativePriceAndVolume) GetCrossedRegion() (*num.Uint, *num.Uint) {
	min := ipv.lastMinPrice
	if min != nil {
		min = min.Clone()
	}

	max := ipv.lastMaxPrice
	if max != nil {
		max = max.Clone()
	}
	return min, max
}

func (ipv *IndicativePriceAndVolume) GetCumulativePriceLevels(maxPrice, minPrice *num.Uint) ([]CumulativeVolumeLevel, uint64) {
	var crossedRegionChanged bool
	if maxPrice.NEQ(ipv.lastMaxPrice) {
		maxPrice = maxPrice.Clone()
		crossedRegionChanged = true
	}
	if minPrice.NEQ(ipv.lastMinPrice) {
		minPrice = minPrice.Clone()
		crossedRegionChanged = true
	}

	// if the crossed region hasn't changed and no new orders were added/removed from the crossed region then we do not need
	// to recalculate
	if !ipv.needsUpdate && !crossedRegionChanged {
		return ipv.lastCumulativeVolumes, ipv.lastMaxTradable
	}

	if crossedRegionChanged && ipv.offbook != nil {
		ipv.updateOffbookState(minPrice, maxPrice)
	}

	rangedLevels := ipv.getLevelsWithinRange(maxPrice, minPrice)
	// now re-allocate the slice only if needed
	if ipv.buf == nil || cap(ipv.buf) < len(rangedLevels) {
		ipv.buf = make([]CumulativeVolumeLevel, len(rangedLevels))
	}

	var (
		cumulativeVolumeSell, cumulativeVolumeBuy, maxTradable uint64
		cumulativeOffbookSell, cumulativeOffbookBuy            uint64
		// here the caching buf is already allocated, we can just resize it
		// based on the required length
		cumulativeVolumes = ipv.buf[:len(rangedLevels)]
		ln                = len(rangedLevels) - 1
	)

	half := ln / 2
	// now we iterate other all the OK price levels
	for i := ln; i >= 0; i-- {
		j := ln - i
		// reset just to make sure
		cumulativeVolumes[j].bidVolume = 0
		cumulativeVolumes[i].askVolume = 0

		if j < i {
			cumulativeVolumes[j].cumulativeAskVolume = 0
			cumulativeVolumes[i].cumulativeBidVolume = 0
		}

		// always set the price
		cumulativeVolumes[i].price = rangedLevels[i].price

		// if we had a price level in the buy side, use it
		if rangedLevels[j].buypl.volume > 0 {
			cumulativeVolumeBuy += rangedLevels[j].buypl.volume
			cumulativeOffbookBuy += rangedLevels[j].buypl.offbookVolume
			cumulativeVolumes[j].bidVolume = rangedLevels[j].buypl.volume
		}

		// same same
		if rangedLevels[i].sellpl.volume > 0 {
			cumulativeVolumeSell += rangedLevels[i].sellpl.volume
			cumulativeOffbookSell += rangedLevels[i].sellpl.offbookVolume
			cumulativeVolumes[i].askVolume = rangedLevels[i].sellpl.volume
		}

		// this will always erase the previous values
		cumulativeVolumes[j].cumulativeBidVolume = cumulativeVolumeBuy
		cumulativeVolumes[j].cumulativeBidOffbook = cumulativeOffbookBuy

		cumulativeVolumes[i].cumulativeAskVolume = cumulativeVolumeSell
		cumulativeVolumes[i].cumulativeAskOffbook = cumulativeOffbookSell

		// we just do that
		// price | sell | buy | vol | ibuy | isell
		// 100   | 0    | 1   | 0   | 0    | 0
		// 110   | 14   | 2   | 2   | 0    | 2
		// 120   | 13   | 5   | 5   | 5    | 0
		// 130   | 10   | 0   | 0   | 0    | 0
		// or we just do that
		// price | sell | buy | vol | ibuy | isell
		// 100   | 0    | 1   | 0   | 0    | 0
		// 110   | 14   | 2   | 2   | 0    | 2
		// 120   | 13   | 5   | 5   | 5    | 5
		// 130   | 11   | 6   | 6   | 6    | 0
		// 150   | 10   | 0   | 0   | 0    | 0
		if j >= half {
			cumulativeVolumes[i].maxTradableAmount = min(cumulativeVolumes[i].cumulativeAskVolume, cumulativeVolumes[i].cumulativeBidVolume)
			cumulativeVolumes[j].maxTradableAmount = min(cumulativeVolumes[j].cumulativeAskVolume, cumulativeVolumes[j].cumulativeBidVolume)
			maxTradable = max(maxTradable, max(cumulativeVolumes[i].maxTradableAmount, cumulativeVolumes[j].maxTradableAmount))
		}
	}

	// reset those fields
	ipv.needsUpdate = false
	ipv.lastMinPrice = minPrice.Clone()
	ipv.lastMaxPrice = maxPrice.Clone()
	ipv.lastMaxTradable = maxTradable
	ipv.lastCumulativeVolumes = cumulativeVolumes
	return cumulativeVolumes, maxTradable
}

// ExtractOffbookOrders returns the cached expanded orders of AM M's in the crossed region of the given side. These
// are the order that we will send in aggressively to uncrossed the book.
func (ipv *IndicativePriceAndVolume) ExtractOffbookOrders(price *num.Uint, side types.Side, target uint64) []*types.Order {
	if target == 0 {
		return []*types.Order{}
	}

	var volume uint64
	orders := []*types.Order{}
	// the ipv keeps track of all the expand AMM orders in the crossed region
	parties := maps.Keys(ipv.generated)
	slices.Sort(parties)

	for _, p := range parties {
		cpm := func(p *num.Uint) bool { return p.LT(price) }
		oo := ipv.generated[p].buy
		if side == types.SideSell {
			oo = ipv.generated[p].sell
			cpm = func(p *num.Uint) bool { return p.GT(price) }
		}

		var combined *types.Order
		for _, o := range oo {
			if cpm(o.Price) {
				continue
			}

			// we want to combine all the uncrossing orders into one big one of the combined volume so that
			// we only uncross with 1 order and not 1000s of expanded ones for a single AMM. We can take the price
			// to the best of the lot so that it trades -- it'll get overridden by the uncrossing price after uncrossing
			// anyway.
			if combined == nil {
				combined = o.Clone()
				orders = append(orders, combined)
			} else {
				combined.Size += o.Size
				combined.Remaining += o.Remaining

				if side == types.SideBuy {
					combined.Price = num.Max(combined.Price, o.Price)
				} else {
					combined.Price = num.Min(combined.Price, o.Price)
				}
			}
			volume += o.Size

			// if we're extracted enough we can stop now
			if volume == target {
				return orders
			}
		}
	}

	if volume != target {
		ipv.log.Panic("Failed to extract AMM orders for uncrossing",
			logging.BigUint("price", price),
			logging.Uint64("volume", volume),
			logging.Uint64("extracted-volume", volume),
			logging.Uint64("target-volume", target),
		)
	}

	return orders
}
