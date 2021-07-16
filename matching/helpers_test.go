package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

type tstOB struct {
	*OrderBook
	log *logging.Logger
}

func (t *tstOB) Finish() {
	t.log.Sync()
}

func getTestOrderBook(_ *testing.T, market string) *tstOB {
	tob := tstOB{
		log: logging.NewTestLogger(),
	}
	tob.OrderBook = NewOrderBook(tob.log, NewDefaultConfig(), market, false)

	// Turn on all the debug levels so we can cover more lines of code
	tob.OrderBook.LogPriceLevelsDebug = true
	tob.OrderBook.LogRemovedOrdersDebug = true
	return &tob
}

func (b *OrderBook) getNumberOfBuyLevels() int {
	buys := b.buy.getLevels()
	return len(buys)
}

func (b *OrderBook) getNumberOfSellLevels() int {
	sells := b.sell.getLevels()
	return len(sells)
}

func (b *OrderBook) getTotalBuyVolume() uint64 {
	var volume uint64 = 0

	buys := b.buy.getLevels()
	for _, pl := range buys {
		volume += pl.volume
	}
	return volume
}

func (b *OrderBook) getVolumeAtLevel(price uint64, side types.Side) uint64 {
	if side == types.SideBuy {
		priceLevel := b.buy.getPriceLevel(num.NewUint(price))
		if priceLevel != nil {
			return priceLevel.volume
		}
	} else {
		priceLevel := b.sell.getPriceLevel(num.NewUint(price))
		if priceLevel != nil {
			return priceLevel.volume
		}
	}
	return 0
}

func (b *OrderBook) getTotalSellVolume() uint64 {
	var volume uint64 = 0
	sells := b.sell.getLevels()
	for _, pl := range sells {
		volume += pl.volume
	}
	return volume
}
