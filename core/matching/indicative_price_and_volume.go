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

package matching

import (
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

type IndicativePriceAndVolume struct {
	log *logging.Logger

	// keep track of previouses {min/max}Price
	// and if orders has been add in the book
	// with a price in the range
	lastMinPrice, lastMaxPrice *num.Uint
	levels                     []ipvPriceLevel

	// this is just used to avoid allocations
	buf []CumulativeVolumeLevel

	lastCumulativeVolumes []CumulativeVolumeLevel
	lastMaxTradable       uint64
	needsUpdate           bool
}

type ipvPriceLevel struct {
	price  *num.Uint
	buypl  ipvVolume
	sellpl ipvVolume
}

type ipvVolume struct {
	volume uint64
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

	ipv := IndicativePriceAndVolume{
		levels:       []ipvPriceLevel{},
		log:          log,
		lastMinPrice: bestBid,
		lastMaxPrice: bestAsk,
		needsUpdate:  true,
	}

	ipv.buildInitialCumulativeLevels(buy, sell)
	// initialize at the size of all levels at start, we most likely
	// not gonna need any other allocation if we start an auction
	// on an existing market
	ipv.buf = make([]CumulativeVolumeLevel, len(ipv.levels))
	return &ipv
}

// this will be used to build the initial set of price levels, when the auction is being started.
func (ipv *IndicativePriceAndVolume) buildInitialCumulativeLevels(buy, sell *OrderBookSide) {
	// we'll keep track of all the pl we encounter
	mplm := map[num.Uint]ipvPriceLevel{}

	for i := len(buy.levels) - 1; i >= 0; i-- {
		mplm[*buy.levels[i].price] = ipvPriceLevel{price: buy.levels[i].price.Clone(), buypl: ipvVolume{buy.levels[i].volume}, sellpl: ipvVolume{0}}
	}

	// now we add all the sells
	// to our list of pricelevel
	// making sure we have no duplicates
	for i := len(sell.levels) - 1; i >= 0; i-- {
		price := sell.levels[i].price.Clone()
		if mpl, ok := mplm[*price]; ok {
			mpl.sellpl = ipvVolume{sell.levels[i].volume}
			mplm[*price] = mpl
		} else {
			mplm[*price] = ipvPriceLevel{price: price, sellpl: ipvVolume{sell.levels[i].volume}, buypl: ipvVolume{0}}
		}
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

func (ipv *IndicativePriceAndVolume) incrementLevelVolume(idx int, volume uint64, side types.Side) {
	switch side {
	case types.SideBuy:
		ipv.levels[idx].buypl.volume += volume
	case types.SideSell:
		ipv.levels[idx].sellpl.volume += volume
	}
}

func (ipv *IndicativePriceAndVolume) AddVolumeAtPrice(price *num.Uint, volume uint64, side types.Side) {
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
		ipv.incrementLevelVolume(i, volume, side)
	} else {
		ipv.levels = append(ipv.levels, ipvPriceLevel{})
		copy(ipv.levels[i+1:], ipv.levels[i:])
		ipv.levels[i] = ipvPriceLevel{price: price.Clone()}
		ipv.incrementLevelVolume(i, volume, side)
	}
}

func (ipv *IndicativePriceAndVolume) decrementLevelVolume(idx int, volume uint64, side types.Side) {
	switch side {
	case types.SideBuy:
		ipv.levels[idx].buypl.volume -= volume
	case types.SideSell:
		ipv.levels[idx].sellpl.volume -= volume
	}
}

func (ipv *IndicativePriceAndVolume) RemoveVolumeAtPrice(price *num.Uint, volume uint64, side types.Side) {
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
		ipv.decrementLevelVolume(i, volume, side)
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

func (ipv *IndicativePriceAndVolume) GetCumulativePriceLevels(maxPrice, minPrice *num.Uint) ([]CumulativeVolumeLevel, uint64) {
	needsUpdate := ipv.needsUpdate
	if maxPrice.NEQ(ipv.lastMaxPrice) {
		maxPrice = maxPrice.Clone()
		needsUpdate = true
	}
	if minPrice.NEQ(ipv.lastMinPrice) {
		minPrice = minPrice.Clone()
		needsUpdate = true
	}

	if !needsUpdate {
		return ipv.lastCumulativeVolumes, ipv.lastMaxTradable
	}

	rangedLevels := ipv.getLevelsWithinRange(maxPrice, minPrice)
	// now re-allocate the slice only if needed
	if ipv.buf == nil || cap(ipv.buf) < len(rangedLevels) {
		ipv.buf = make([]CumulativeVolumeLevel, len(rangedLevels))
	}

	var (
		cumulativeVolumeSell, cumulativeVolumeBuy, maxTradable uint64
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
			cumulativeVolumes[j].bidVolume = rangedLevels[j].buypl.volume
		}

		// same same
		if rangedLevels[i].sellpl.volume > 0 {
			cumulativeVolumeSell += rangedLevels[i].sellpl.volume
			cumulativeVolumes[i].askVolume = rangedLevels[i].sellpl.volume
		}

		// this will always erase the previous values
		cumulativeVolumes[j].cumulativeBidVolume = cumulativeVolumeBuy
		cumulativeVolumes[i].cumulativeAskVolume = cumulativeVolumeSell

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
