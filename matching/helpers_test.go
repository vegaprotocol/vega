package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/logging"
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

func getTestOrderBook(t *testing.T, market string, proRata bool) *tstOB {
	tob := tstOB{
		log: logging.NewTestLogger(),
	}
	tob.OrderBook = NewOrderBook(tob.log, NewDefaultConfig(), market, 100.0, proRata)
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

func (ob *OrderBook) getTotalSellVolume() uint64 {
	var volume uint64 = 0
	sells := ob.sell.getLevels()
	for _, pl := range sells {
		volume += pl.volume
	}
	return volume
}
