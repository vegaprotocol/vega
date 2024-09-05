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

package entities_test

// No race condition checks on these tests, the channels are buffered to avoid actual issues
// we are aware that the tests themselves can be written in an unsafe way, but that's the tests
// not the code itsel. The behaviour of the tests is 100% reliable.
import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultipleTradesOfSameSize(t *testing.T) {
	ctx := context.Background()
	market := "market-id"
	party := "party1"
	position := entities.NewEmptyPosition(entities.MarketID(market), entities.PartyID(party))
	ps := events.NewSettlePositionEvent(ctx, party, market, num.NewUint(1000), []events.TradeSettlement{
		tradeStub{
			size:  -1,
			price: num.NewUint(1000),
		},
		tradeStub{
			size:  -1,
			price: num.NewUint(1000),
		},
	}, 1, num.DecimalFromFloat(1))
	position.UpdateWithPositionSettlement(ps)
	pp := position.ToProto()
	// average entry price should be 1k
	assert.Equal(t, ps.Price().String(), pp.AverageEntryPrice)
}

func TestMultipleTradesAndLossSocializationPartyNoOpenVolume(t *testing.T) {
	ctx := context.Background()
	market := "market-id"
	party := "party1"
	position := entities.NewEmptyPosition(entities.MarketID(market), entities.PartyID(party))

	ps := events.NewSettlePositionEvent(ctx, party, market, num.NewUint(1000), []events.TradeSettlement{
		tradeStub{
			size:  2,
			price: num.NewUint(1000),
		},
		tradeStub{
			size:  -2,
			price: num.NewUint(1500),
		},
	}, 1, num.DecimalFromFloat(1))
	position.UpdateWithPositionSettlement(ps)
	pp := position.ToProto()
	assert.Equal(t, "1000", pp.RealisedPnl)

	// then we process the event for LossSocialization
	lsevt := events.NewLossSocializationEvent(ctx, party, market, num.NewUint(300), true, 1, types.LossTypeUnspecified)
	position.UpdateWithLossSocialization(lsevt)
	pp = position.ToProto()
	assert.Equal(t, "700", pp.RealisedPnl)
	assert.Equal(t, "0", pp.UnrealisedPnl)
}

func TestDistressedPartyUpdate(t *testing.T) {
	ctx := context.Background()
	market := "market-id"
	party := "party1"
	position := entities.NewEmptyPosition(entities.MarketID(market), entities.PartyID(party))
	pf := num.DecimalFromFloat(1)

	ps := events.NewSettlePositionEvent(ctx, party, market, num.NewUint(1000), []events.TradeSettlement{
		tradeStub{
			size:  2,
			price: num.NewUint(1000),
		},
		tradeStub{
			size:  3,
			price: num.NewUint(1200),
		},
	}, 1, pf)
	position.UpdateWithPositionSettlement(ps)
	pp := position.ToProto()
	assert.Equal(t, "0", pp.RealisedPnl)
	assert.Equal(t, "-600", pp.UnrealisedPnl)

	// then we process the event for LossSocialization
	lsevt := events.NewLossSocializationEvent(ctx, party, market, num.NewUint(300), true, 1, types.LossTypeUnspecified)
	position.UpdateWithLossSocialization(lsevt)
	pp = position.ToProto()
	assert.Equal(t, "-300", pp.RealisedPnl)
	assert.Equal(t, "-600", pp.UnrealisedPnl)

	// now assume this party is distressed, and we've taken all their funds
	sde := events.NewSettleDistressed(ctx, party, market, num.UintZero(), num.NewUint(100), 1)
	position.UpdateWithSettleDistressed(sde)
	// ensure the position is flagged as distressed.
	assert.Equal(t, entities.PositionStatusClosedOut, position.DistressedStatus)
	pp = position.ToProto()
	assert.Equal(t, "0", pp.UnrealisedPnl)
	assert.Equal(t, "-1000", pp.RealisedPnl)

	// now submit process a closeout trade event
	position.UpdateWithTrade(vega.Trade{
		Size:       5,
		Price:      "1200",
		AssetPrice: "1200",
		Type:       vega.Trade_TYPE_NETWORK_CLOSE_OUT_BAD,
	}, true, pf)
	// now ensure the position status still is what we expect it to be
	assert.Equal(t, entities.PositionStatusClosedOut, position.DistressedStatus)

	// next, assume the party has topped up, and traded again.
	position.UpdateWithTrade(vega.Trade{
		Size:       1,
		Price:      "1200",
		AssetPrice: "1200",
		Type:       vega.Trade_TYPE_DEFAULT,
	}, true, pf)
	// now the distressed status ought to be cleared.
	assert.Equal(t, entities.PositionStatusUnspecified, position.DistressedStatus)
}

func TestMultipleTradesAndLossSocializationPartyWithOpenVolume(t *testing.T) {
	ctx := context.Background()
	market := "market-id"
	party := "party1"
	position := entities.NewEmptyPosition(entities.MarketID(market), entities.PartyID(party))

	ps := events.NewSettlePositionEvent(ctx, party, market, num.NewUint(1000), []events.TradeSettlement{
		tradeStub{
			size:  2,
			price: num.NewUint(1000),
		},
		tradeStub{
			size:  3,
			price: num.NewUint(1200),
		},
	}, 1, num.DecimalFromFloat(1))
	position.UpdateWithPositionSettlement(ps)
	pp := position.ToProto()
	assert.Equal(t, "0", pp.RealisedPnl)
	assert.Equal(t, "-600", pp.UnrealisedPnl)

	// then we process the event for LossSocialization
	lsevt := events.NewLossSocializationEvent(ctx, party, market, num.NewUint(300), true, 1, types.LossTypeUnspecified)
	position.UpdateWithLossSocialization(lsevt)
	pp = position.ToProto()
	assert.Equal(t, "-300", pp.RealisedPnl)
	assert.Equal(t, "-600", pp.UnrealisedPnl)
}

func TestPnLWithPositionDecimals(t *testing.T) {
	ctx := context.Background()
	market := "market-id"
	party := "party1"
	position := entities.NewEmptyPosition(entities.MarketID(market), entities.PartyID(party))
	dp := num.DecimalFromFloat(10).Pow(num.DecimalFromInt64(3))

	// first update with trades
	trade := vega.Trade{
		Id:         "t1",
		MarketId:   market,
		Price:      "1000",
		Size:       2,
		Buyer:      party,
		Seller:     "seller",
		AssetPrice: "1000",
	}
	position.UpdateWithTrade(trade, false, dp)
	trade.Id = "t2"
	trade.Size = 3
	trade.Price = "1200"
	trade.AssetPrice = "1200"
	position.UpdateWithTrade(trade, false, dp)
	pp := position.ToProto()
	assert.Equal(t, "0", pp.RealisedPnl)
	assert.Equal(t, "0", pp.UnrealisedPnl)
	// now MTM settlement event, contains the same trades, mark price is 1k
	ps := events.NewSettlePositionEvent(ctx, party, market, num.NewUint(1000), []events.TradeSettlement{
		tradeStub{
			size:  2,
			price: num.NewUint(1000),
		},
		tradeStub{
			size:  3,
			price: num.NewUint(1200),
		},
	}, 1, dp)
	position.UpdateWithPositionSettlement(ps)
	pp = position.ToProto()
	assert.Equal(t, "0", pp.RealisedPnl)
	assert.Equal(t, "-1", pp.UnrealisedPnl)
	assert.EqualValues(t, 5, pp.OpenVolume)

	// let's make it look like this party is trading, buyer in this case
	trade = vega.Trade{
		Id:         "t3",
		MarketId:   market,
		Price:      "1150",
		Size:       1,
		Buyer:      party,
		Seller:     "seller",
		AssetPrice: "1150",
	}
	// position.UpdateWithTrade(trade, false, num.DecimalFromFloat(1))
	position.UpdateWithTrade(trade, false, dp)
	pp = position.ToProto()
	assert.Equal(t, "0", pp.RealisedPnl)
	assert.Equal(t, "0", pp.UnrealisedPnl)
	assert.EqualValues(t, 6, pp.OpenVolume)
	// now assume this last trade was the only trade that occurred before MTM
	ps = events.NewSettlePositionEvent(ctx, party, market, num.NewUint(1150), []events.TradeSettlement{
		tradeStub{
			size:  1,
			price: num.NewUint(1150),
		},
	}, 1, dp)
	position.UpdateWithPositionSettlement(ps)
	pp = position.ToProto()
	assert.Equal(t, "0", pp.RealisedPnl)
	assert.Equal(t, "0", pp.UnrealisedPnl)
	assert.EqualValues(t, 6, pp.OpenVolume)
	// now close a position to see some realised PnL
	trade = vega.Trade{
		Id:         "t4",
		MarketId:   market,
		Price:      "1250",
		Size:       1,
		Buyer:      "buyer",
		Seller:     party,
		AssetPrice: "1250",
	}
	position.UpdateWithTrade(trade, true, dp)
	pp = position.ToProto()
	assert.Equal(t, "0", pp.RealisedPnl)
	assert.Equal(t, "1", pp.UnrealisedPnl)
	assert.EqualValues(t, 5, pp.OpenVolume)
	ps = events.NewSettlePositionEvent(ctx, party, market, num.NewUint(1250), []events.TradeSettlement{
		tradeStub{
			size:  -1,
			price: num.NewUint(1250),
		},
	}, 1, dp)
	position.UpdateWithPositionSettlement(ps)
	pp = position.ToProto()
	assert.Equal(t, "0", pp.RealisedPnl)
	assert.Equal(t, "1", pp.UnrealisedPnl)
	assert.EqualValues(t, 5, pp.OpenVolume)
	// now close the position
	trade = vega.Trade{
		Id:         "t5",
		MarketId:   market,
		Price:      "1300",
		Size:       5,
		Buyer:      "buyer",
		Seller:     party,
		AssetPrice: "1300",
	}
	position.UpdateWithTrade(trade, true, dp)
	pp = position.ToProto()
	assert.Equal(t, "1", pp.RealisedPnl)
	assert.Equal(t, "0", pp.UnrealisedPnl)
	assert.EqualValues(t, 0, pp.OpenVolume)
	ps = events.NewSettlePositionEvent(ctx, party, market, num.NewUint(1250), []events.TradeSettlement{
		tradeStub{
			size:  -5,
			price: num.NewUint(1300),
		},
	}, 1, dp)
	position.UpdateWithPositionSettlement(ps)
	pp = position.ToProto()
	assert.Equal(t, "1", pp.RealisedPnl)
	assert.Equal(t, "0", pp.UnrealisedPnl)
	assert.EqualValues(t, 0, pp.OpenVolume)
}

func TestPnLWithTradeDecimals(t *testing.T) {
	ctx := context.Background()
	market := "market-id"
	party := "party1"
	position := entities.NewEmptyPosition(entities.MarketID(market), entities.PartyID(party))
	dp := num.DecimalFromFloat(3)

	ps := events.NewSettlePositionEvent(ctx, party, market, num.NewUint(1000), []events.TradeSettlement{
		tradeStub{
			size:  2,
			price: num.NewUint(1000),
		},
		tradeStub{
			size:  3,
			price: num.NewUint(1200),
		},
	}, 1, dp)
	position.UpdateWithPositionSettlement(ps)
	pp := position.ToProto()
	assert.Equal(t, "0", pp.RealisedPnl)
	assert.Equal(t, "-200", pp.UnrealisedPnl)

	// then we process the event for LossSocialization
	lsevt := events.NewLossSocializationEvent(ctx, party, market, num.NewUint(300), true, 1, types.LossTypeUnspecified)
	position.UpdateWithLossSocialization(lsevt)
	pp = position.ToProto()
	assert.Equal(t, "-300", pp.RealisedPnl)
	assert.Equal(t, "-200", pp.UnrealisedPnl)
	// let's make it look like this party is trading, buyer in this case
	trade := vega.Trade{
		Id:         "t1",
		MarketId:   market,
		Price:      "1150",
		Size:       1,
		Buyer:      party,
		Seller:     "seller",
		AssetPrice: "1150",
	}
	// position.UpdateWithTrade(trade, false, num.DecimalFromFloat(1))
	position.UpdateWithTrade(trade, false, dp)
	pp = position.ToProto()
	assert.Equal(t, "-300", pp.RealisedPnl)
	assert.Equal(t, "50", pp.UnrealisedPnl)
}

func TestUpdateWithTradesAndFundingPayment(t *testing.T) {
	ctx := context.Background()
	market := "market-id"
	party := "party1"
	position := entities.NewEmptyPosition(entities.MarketID(market), entities.PartyID(party))
	dp := num.DecimalFromFloat(3)
	trades := []tradeStub{
		{
			size:  2,
			price: num.NewUint(1200),
		},
		{
			size:  3,
			price: num.NewUint(1000),
		},
	}
	// this is the order in which the events will be sent/received
	position.UpdateWithTrade(trades[0].ToVega(dp), false, dp)
	pp := position.ToProto()
	assert.Equal(t, "0", pp.RealisedPnl)
	assert.Equal(t, "0", pp.UnrealisedPnl)
	position.ApplyFundingPayment(num.NewInt(100))
	pp = position.ToProto()
	assert.Equal(t, "100", pp.RealisedPnl)
	assert.Equal(t, "0", pp.UnrealisedPnl, pp.AverageEntryPrice)
	position.UpdateWithTrade(trades[1].ToVega(dp), false, dp)
	pp = position.ToProto()
	assert.Equal(t, "100", pp.RealisedPnl)
	assert.Equal(t, "-133", pp.UnrealisedPnl, pp.AverageEntryPrice)
	ps := events.NewSettlePositionEvent(ctx, party, market, num.NewUint(1000), []events.TradeSettlement{trades[0], trades[1]}, 1, dp)
	position.UpdateWithPositionSettlement(ps)
	psp := position.ToProto()
	assert.Equal(t, "100", psp.RealisedPnl)
	assert.Equal(t, "-133", psp.UnrealisedPnl)
	position.ApplyFundingPayment(num.NewInt(-50))
	pp = position.ToProto()
	assert.Equal(t, "50", pp.RealisedPnl)
	assert.Equal(t, "-133", pp.UnrealisedPnl, pp.AverageEntryPrice)
}

type tradeStub struct {
	size        int64
	price       *num.Uint
	marketPrice *num.Uint
}

func (t tradeStub) Size() int64 {
	return t.size
}

func (t tradeStub) Price() *num.Uint {
	return t.price.Clone()
}

func (t tradeStub) MarketPrice() *num.Uint {
	if t.marketPrice != nil {
		return t.marketPrice.Clone()
	}
	return t.price.Clone()
}

func (t tradeStub) ToVega(dp num.Decimal) vega.Trade {
	// dp = num.DecimalFromFloat(10).Pow(dp)
	// size, _ := num.DecimalFromInt64(t.size).Abs().Mul(dp).Float64()
	size := uint64(t.size)
	if t.size < 0 {
		size = uint64(-t.size)
	}
	return vega.Trade{
		Size:       size,
		Price:      t.price.String(),
		AssetPrice: t.price.String(),
	}
}

func TestCalculateOpenClosedVolume(t *testing.T) {
	open := int64(0)
	closed := int64(0)
	// no pending volume, new buy trade of 100, expect to open 100 close 0
	open, closed = entities.CalculateOpenClosedVolume(0, 100)
	require.Equal(t, int64(100), open)
	require.Equal(t, int64(0), closed)

	// no pending volume, new sell trade of 100, expect to open -100 close 0
	open, closed = entities.CalculateOpenClosedVolume(0, -100)
	require.Equal(t, int64(-100), open)
	require.Equal(t, int64(0), closed)

	// we have a pending open volume of 100 and we get a new buy trade of 50, expect to return opened 50, close 0
	open, closed = entities.CalculateOpenClosedVolume(100, 50)
	require.Equal(t, int64(50), open)
	require.Equal(t, int64(0), closed)

	// we have a pending open volume of -100 and we get a new sell trade of 50, expect to return opened -50, close 0
	open, closed = entities.CalculateOpenClosedVolume(-100, -50)
	require.Equal(t, int64(-50), open)
	require.Equal(t, int64(0), closed)

	// we have a pending open volume of 100 and we get a new sell trade of 50, expect to return opened 0, close 50
	open, closed = entities.CalculateOpenClosedVolume(100, -50)
	require.Equal(t, int64(0), open)
	require.Equal(t, int64(50), closed)

	// we have a pending open volume of -100 and we get a new buy trade of 50, expect to return opened 0, close -50
	open, closed = entities.CalculateOpenClosedVolume(-100, 50)
	require.Equal(t, int64(0), open)
	require.Equal(t, int64(-50), closed)

	// we have a pending open volume of 100 and we get a new sell trade of 150, expect to return opened -50, close 100
	open, closed = entities.CalculateOpenClosedVolume(100, -150)
	require.Equal(t, int64(-50), open)
	require.Equal(t, int64(100), closed)

	// we have a pending open volume of -100 and we get a new buy trade of 150, expect to return opened 50, close -100
	open, closed = entities.CalculateOpenClosedVolume(-100, 150)
	require.Equal(t, int64(50), open)
	require.Equal(t, int64(-100), closed)
}

func TestWithDifferingMarketAssetPrecision(t *testing.T) {
	ctx := context.Background()
	market := "market-id"
	party := "party1"
	position := entities.NewEmptyPosition(entities.MarketID(market), entities.PartyID(party))
	ps := events.NewSettlePositionEvent(ctx, party, market, num.NewUint(1000), []events.TradeSettlement{
		tradeStub{
			size:        -5,
			price:       num.NewUint(10000000),
			marketPrice: num.NewUint(1000),
		},
		tradeStub{
			size:        -5,
			price:       num.NewUint(10000000),
			marketPrice: num.NewUint(1000),
		},
	}, 1, num.DecimalFromFloat(1))
	position.UpdateWithPositionSettlement(ps)
	pp := position.ToProto()

	// average entry price should be 1k in market precision
	assert.Equal(t, ps.Price().String(), pp.AverageEntryPrice)
	assert.Equal(t, "1000", position.AverageEntryMarketPrice.String())
	assert.Equal(t, "10000000", position.AverageEntryPrice.String())

	// now update with a trade
	trade := vega.Trade{
		Price:      "2000",
		AssetPrice: "20000000",
		Size:       10,
	}
	position.UpdateWithTrade(trade, true, num.DecimalOne())
	assert.Equal(t, "1500", position.PendingAverageEntryMarketPrice.String())
	assert.Equal(t, "15000000", position.PendingAverageEntryPrice.String())
	assert.Equal(t, int64(-20), position.PendingOpenVolume)

	trade = vega.Trade{
		Price:      "1000",
		AssetPrice: "10000000",
		Size:       5,
	}
	position.UpdateWithTrade(trade, false, num.DecimalOne())
	assert.Equal(t, "1500", position.PendingAverageEntryMarketPrice.String())
	assert.Equal(t, "15000000", position.PendingAverageEntryPrice.String())
	assert.Equal(t, int64(-15), position.PendingOpenVolume)
	assert.Equal(t, "25000000", position.PendingRealisedPnl.String())
	assert.Equal(t, "75000000", position.PendingUnrealisedPnl.String())
}
