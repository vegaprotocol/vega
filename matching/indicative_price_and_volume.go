package matching

import (
	"fmt"
	"sort"

	types "code.vegaprotocol.io/vega/proto"
)

type IndicativePriceAndVolume struct {
	levels []ipvPriceLevel
}

type ipvPriceLevel struct {
	price  uint64
	buypl  *ipvVolume
	sellpl *ipvVolume
}

type ipvVolume struct {
	volume uint64
}

func NewIndicativePriceAndVolume(buy, sell *OrderBookSide) *IndicativePriceAndVolume {
	ipv := IndicativePriceAndVolume{
		levels: []ipvPriceLevel{},
	}
	ipv.buildInitialCumulativeLevels(buy, sell)
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
			ipv.levels[idx].buypl = &ipvVolume{}
		}
		ipv.levels[idx].buypl.volume -= volume
	case types.Side_SIDE_SELL:
		if ipv.levels[idx].sellpl == nil {
			ipv.levels[idx].sellpl = &ipvVolume{}
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
		ipv.levels = append(ipv.levels, ipvPriceLevel{})
		copy(ipv.levels[i+1:], ipv.levels[i:])
		ipv.levels[i] = ipvPriceLevel{price: price}
		ipv.decrementLevelVolume(i, volume, side)
	}
}

func (ipv *IndicativePriceAndVolume) getLevelsWithinRange(maxPrice, minPrice uint64) []ipvPriceLevel {
	fmt.Printf("maxPrice(%d), minPrice(%d)\n", maxPrice, minPrice)
	for _, v := range ipv.levels {
		fmt.Printf("price: %v\n", v.price)
	}

	// these are ordered descending, se we gonna find first the maxPrice then
	// the minPrice, and using that we can then subslice like a boss
	maxPricePos := sort.Search(len(ipv.levels), func(i int) bool {
		return ipv.levels[i].price <= maxPrice
	})
	if maxPricePos >= len(ipv.levels) || ipv.levels[maxPricePos].price != maxPrice {
		// price level not present, that should not be possible?
		panic("missing max price level")
	}
	minPricePos := sort.Search(len(ipv.levels), func(i int) bool {
		return ipv.levels[i].price <= minPrice
	})
	if minPricePos >= len(ipv.levels) || ipv.levels[minPricePos].price != minPrice {
		// price level not present, that should not be possible?
		panic("missing min price level")
	}

	fmt.Printf("maxPricePos(%d) minPricePos(%d)\n", maxPricePos, minPricePos)

	return ipv.levels[maxPricePos : minPricePos+1]
}

func (ipv *IndicativePriceAndVolume) GetCumulativePriceLevels(maxPrice, minPrice uint64) ([]CumulativeVolumeLevel, uint64) {
	rangedLevels := ipv.getLevelsWithinRange(maxPrice, minPrice)
	// now we iterate other all the OK price levels
	var (
		cumulativeVolumeSell, cumulativeVolumeBuy, maxTradable uint64
		cumulativeVolumes                                      = make([]CumulativeVolumeLevel, len(rangedLevels))
		ln                                                     = len(rangedLevels) - 1
	)

	for i := ln; i >= 0; i-- {
		j := ln - i
		cumulativeVolumes[i].price = rangedLevels[i].price
		if rangedLevels[j].buypl != nil {
			cumulativeVolumeBuy += rangedLevels[j].buypl.volume
			cumulativeVolumes[j].bidVolume = rangedLevels[j].buypl.volume
		}

		if rangedLevels[i].sellpl != nil {
			cumulativeVolumeSell += rangedLevels[i].sellpl.volume
			cumulativeVolumes[i].askVolume = rangedLevels[i].sellpl.volume

		}
		cumulativeVolumes[j].cumulativeBidVolume = cumulativeVolumeBuy
		cumulativeVolumes[i].cumulativeAskVolume = cumulativeVolumeSell

		cumulativeVolumes[i].maxTradableAmount = min(cumulativeVolumes[i].cumulativeAskVolume, cumulativeVolumes[i].cumulativeBidVolume)
		cumulativeVolumes[j].maxTradableAmount = min(cumulativeVolumes[j].cumulativeAskVolume, cumulativeVolumes[j].cumulativeBidVolume)
		maxTradable = max(maxTradable, max(cumulativeVolumes[i].maxTradableAmount, cumulativeVolumes[j].maxTradableAmount))
	}

	return cumulativeVolumes, maxTradable
}
