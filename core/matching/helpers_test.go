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
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

type tstOB struct {
	*OrderBook
	log *logging.Logger
}

func (t *tstOB) Finish() {
	t.log.Sync()
}

func peggedOrderCounterForTest(int64) {}

func getTestOrderBook(_ *testing.T, market string) *tstOB {
	tob := tstOB{
		log: logging.NewTestLogger(),
	}
	tob.OrderBook = NewOrderBook(tob.log, NewDefaultConfig(), market, false, peggedOrderCounterForTest)

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
	var volume uint64

	buys := b.buy.getLevels()
	for _, pl := range buys {
		volume += pl.volume
	}
	return volume
}

//nolint:unparam
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
	var volume uint64
	sells := b.sell.getLevels()
	for _, pl := range sells {
		volume += pl.volume
	}
	return volume
}
