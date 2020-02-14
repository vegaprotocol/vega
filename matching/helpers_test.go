package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"
)

type tstOB struct {
	*OrderBook
	log *logging.Logger
}

func (t *tstOB) Finish() {
	t.log.Sync()
}

func getCurrentUtcTimestampNano() int64 {
	return vegatime.Now().UnixNano()
}

func getTestOrderBook(t *testing.T, market string) *tstOB {
	tob := tstOB{
		log: logging.NewTestLogger(),
	}
	tob.OrderBook = NewOrderBook(tob.log, NewDefaultConfig(), market, 100.0)
	return &tob
}

func (ob *OrderBook) getNumberOfBuyLevels() int {
	buys := ob.buy.getLevels()

	return len(buys)
}

func (ob *OrderBook) getNumberOfSellLevels() int {
	sells := ob.sell.getLevels()

	return len(sells)
}

func (ob *OrderBook) getTotalBuyVolume() uint64 {
	var volume uint64 = 0

	buys := ob.buy.getLevels()
	for _, pl := range buys {
		volume += pl.volume
	}
	return volume
}

func (ob *OrderBook) getVolumeAtLevel(price uint64, side types.Side) uint64 {
	if side == types.Side_Buy {
		priceLevel := ob.buy.getPriceLevel(price, side)
		if priceLevel != nil {
			return priceLevel.volume
		}
	} else {
		priceLevel := ob.sell.getPriceLevel(price, side)
		if priceLevel != nil {
			return priceLevel.volume
		}
	}
	return 0
}

func (ob *OrderBook) getTotalSellVolume() uint64 {
	var volume uint64 = 0
	sells := ob.sell.getLevels()
	for _, pl := range sells {
		volume += pl.volume
	}
	return volume
}
