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

package sqlsubscribers

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/assert"
)

func TestSubscriberSequenceNumber(t *testing.T) {
	ts := testStore{}
	sub := NewTradesSubscriber(&ts)

	now := time.Now()
	nowPlusOne := time.Now().Add(time.Second)

	sub.SetVegaTime(now)

	tradeEvent := events.NewTradeEvent(context.Background(), newTrade())
	tradeEvent.SetSequenceID(1)
	sub.Push(context.Background(), tradeEvent)

	tradeEvent = events.NewTradeEvent(context.Background(), newTrade())
	tradeEvent.SetSequenceID(2)
	sub.Push(context.Background(), tradeEvent)

	sub.SetVegaTime(nowPlusOne)

	tradeEvent = events.NewTradeEvent(context.Background(), newTrade())
	tradeEvent.SetSequenceID(1)
	sub.Push(context.Background(), tradeEvent)

	tradeEvent = events.NewTradeEvent(context.Background(), newTrade())
	tradeEvent.SetSequenceID(2)
	sub.Push(context.Background(), tradeEvent)

	assert.Equal(t, now, ts.trades[0].VegaTime)
	assert.Equal(t, uint64(1), ts.trades[0].SeqNum)
	assert.Equal(t, now, ts.trades[1].VegaTime)
	assert.Equal(t, uint64(2), ts.trades[1].SeqNum)

	assert.Equal(t, nowPlusOne, ts.trades[2].VegaTime)
	assert.Equal(t, uint64(1), ts.trades[2].SeqNum)
	assert.Equal(t, nowPlusOne, ts.trades[3].VegaTime)
	assert.Equal(t, uint64(2), ts.trades[3].SeqNum)
}

type testStore struct {
	trades []*entities.Trade
}

func (ts *testStore) Add(t *entities.Trade) error {
	ts.trades = append(ts.trades, t)
	return nil
}

func (ts *testStore) Flush(ctx context.Context) error {
	return nil
}

func newTrade() types.Trade {
	trade := types.Trade{
		ID:                 "bc2001bddac588f8aaae0d9bec3d6881a447b888447e5d0a9de92d149ba4e877",
		MarketID:           "8cc0e020c0bc2f9eba77749d81ecec8283283b85941722c2cb88318aaf8b8cd8",
		Price:              num.NewUint(12),
		Size:               16,
		Buyer:              "2e4f34a38204a2a155be678e670903ed8df96e813700729deacd3daf7e55039e",
		Seller:             "8b6be1a03cc4d529f682887a78b66e6879d17f81e2b37356ca0acbc5d5886eb8",
		Aggressor:          types.SideBuy,
		BuyOrder:           "cf951606211775c43449807fe15f908704a85c514d65d549d67bbd6b5eef66bb",
		SellOrder:          "6a94947f724cdb7851bee793aca6888f68abbf8d49dfd0f778424a7ce42e7b7d",
		Type:               types.TradeTypeNetworkCloseOutGood,
		BuyerAuctionBatch:  3,
		SellerAuctionBatch: 4,
		MarketPrice:        num.NewUint(22),
	}

	return trade
}
