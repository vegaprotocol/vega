package matching

import (
	"sort"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
)

type IndicativePriceAndVolume struct {
	log    *logging.Logger
	levels []ipvPriceLevel

	// this is just used to avoid allocations
	buf []CumulativeVolumeLevel
}

type ipvPriceLevel struct {
	price  uint64
	buypl  *ipvVolume
	sellpl *ipvVolume
}

type ipvVolume struct {
	volume uint64
}

func NewIndicativePriceAndVolume(log *logging.Logger, buy, sell *OrderBookSide) *IndicativePriceAndVolume {
	ipv := IndicativePriceAndVolume{
		levels: []ipvPriceLevel{},
		log:    log,
	}
	ipv.buildInitialCumulativeLevels(buy, sell)
	// initialize at the size of all levels at start, we most likely
	// not gonna need any other allocation if we start an auction
	// on an existing market
	ipv.buf = make([]CumulativeVolumeLevel, len(ipv.levels))
	return &ipv
}

// this will be user to build the initial set of price levels, when the auction is being started.
func (ipv *IndicativePriceAndVolume) buildInitialCumulativeLevels(buy, sell *OrderBookSide) {
	// we'll keep track of all the pl we encounter
	mplm := map[uint64]ipvPriceLevel{}

	for i := len(buy.levels) - 1; i >= 0; i-- {
		mplm[buy.levels[i].price] = ipvPriceLevel{price: buy.levels[i].price, buypl: &ipvVolume{buy.levels[i].volume}}
	}

	// now we add all the sells
	// to our list of pricelevel
	// making sure we have no duplicates
	for i := len(sell.levels) - 1; i >= 0; i-- {
		var price = sell.levels[i].price
		if mpl, ok := mplm[price]; ok {
			mpl.sellpl = &ipvVolume{sell.levels[i].volume}
			mplm[price] = mpl
		} else {
			mplm[price] = ipvPriceLevel{price: price, sellpl: &ipvVolume{sell.levels[i].volume}}
		}
	}

	// now we insert them all in the slice.
	// so we can sort them
	ipv.levels = make([]ipvPriceLevel, 0, len(mplm))
	for _, v := range mplm {
		ipv.levels = append(ipv.levels, v)
	}

	// sort the slice so we can go through each levels nicely
	sort.Slice(ipv.levels, func(i, j int) bool { return ipv.levels[i].price > ipv.levels[j].price })
}

func (ipv *IndicativePriceAndVolume) incrementLevelVolume(idx int, volume uint64, side types.Side) {
	switch side {
	case types.Side_SIDE_BUY:
		if ipv.levels[idx].buypl == nil {
			ipv.levels[idx].buypl = &ipvVolume{}
		}
		ipv.levels[idx].buypl.volume += volume
	case types.Side_SIDE_SELL:
		if ipv.levels[idx].sellpl == nil {
			ipv.levels[idx].sellpl = &ipvVolume{}
		}
		ipv.levels[idx].sellpl.volume += volume
	}
}

func (ipv *IndicativePriceAndVolume) AddVolumeAtPrice(price, volume uint64, side types.Side) {
	i := sort.Search(len(ipv.levels), func(i int) bool {
		return ipv.levels[i].price <= price
	})
	if i < len(ipv.levels) && ipv.levels[i].price == price {
		// we found the price level, let's add the volume there, and we are done
		ipv.incrementLevelVolume(i, volume, side)
	} else {
		ipv.levels = append(ipv.levels, ipvPriceLevel{})
		copy(ipv.levels[i+1:], ipv.levels[i:])
		ipv.levels[i] = ipvPriceLevel{price: price}
		ipv.incrementLevelVolume(i, volume, side)
	}
}

func (ipv *IndicativePriceAndVolume) decrementLevelVolume(idx int, volume uint64, side types.Side) {
	switch side {
	case types.Side_SIDE_BUY:
		if ipv.levels[idx].buypl == nil {
			ipv.log.Panic("cannot decrement volume from a non-existing level",
				logging.String("side", side.String()),
				logging.Uint64("price", ipv.levels[idx].price),
				logging.Uint64("volume", volume))
		}
		ipv.levels[idx].buypl.volume -= volume
	case types.Side_SIDE_SELL:
		if ipv.levels[idx].sellpl == nil {
			ipv.log.Panic("cannot decrement volume from a non-existing level",
				logging.String("side", side.String()),
				logging.Uint64("price", ipv.levels[idx].price),
				logging.Uint64("volume", volume))
		}
		ipv.levels[idx].sellpl.volume -= volume
	}
}

func (ipv *IndicativePriceAndVolume) RemoveVolumeAtPrice(price, volume uint64, side types.Side) {
	i := sort.Search(len(ipv.levels), func(i int) bool {
		return ipv.levels[i].price <= price
	})
	if i < len(ipv.levels) && ipv.levels[i].price == price {
		// we found the price level, let's add the volume there, and we are done
		ipv.decrementLevelVolume(i, volume, side)
	} else {
		ipv.log.Panic("cannot remove volume from a non-existing level",
			logging.String("side", side.String()),
			logging.Uint64("price", price),
			logging.Uint64("volume", volume))
	}
}

func (ipv *IndicativePriceAndVolume) getLevelsWithinRange(maxPrice, minPrice uint64) []ipvPriceLevel {
	// these are ordered descending, se we gonna find first the maxPrice then
	// the minPrice, and using that we can then subslice like a boss
	maxPricePos := sort.Search(len(ipv.levels), func(i int) bool {
		return ipv.levels[i].price <= maxPrice
	})
	if maxPricePos >= len(ipv.levels) || ipv.levels[maxPricePos].price != maxPrice {
		// price level not present, that should not be possible?
		ipv.log.Panic("missing max price in levels",
			logging.Uint64("max-price", maxPrice))
	}
	minPricePos := sort.Search(len(ipv.levels), func(i int) bool {
		return ipv.levels[i].price <= minPrice
	})
	if minPricePos >= len(ipv.levels) || ipv.levels[minPricePos].price != minPrice {
		// price level not present, that should not be possible?
		ipv.log.Panic("missing min price in levels",
			logging.Uint64("min-price", minPrice))
	}

	return ipv.levels[maxPricePos : minPricePos+1]
}

func (ipv *IndicativePriceAndVolume) GetCumulativePriceLevels(maxPrice, minPrice uint64) ([]CumulativeVolumeLevel, uint64) {
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

		// if we had a price level in the bug side, use it
		if rangedLevels[j].buypl != nil {
			cumulativeVolumeBuy += rangedLevels[j].buypl.volume
			cumulativeVolumes[j].bidVolume = rangedLevels[j].buypl.volume
		}

		// same same
		if rangedLevels[i].sellpl != nil {
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

	return cumulativeVolumes, maxTradable
}
