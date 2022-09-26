// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package entities_test

// No race condition checks on these tests, the channels are buffered to avoid actual issues
// we are aware that the tests themselves can be written in an unsafe way, but that's the tests
// not the code itsel. The behaviour of the tests is 100% reliable.
import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/assert"
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
	lsevt := events.NewLossSocializationEvent(ctx, party, market, num.NewUint(300), true, 1)
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
	lsevt := events.NewLossSocializationEvent(ctx, party, market, num.NewUint(300), true, 1)
	position.UpdateWithLossSocialization(lsevt)
	pp = position.ToProto()
	assert.Equal(t, "-300", pp.RealisedPnl)
	assert.Equal(t, "-600", pp.UnrealisedPnl)

	// now assume this party is distressed, and we've taken all their funds
	sde := events.NewSettleDistressed(ctx, party, market, num.UintZero(), num.NewUint(100), 1)
	position.UpdateWithSettleDistressed(sde)
	pp = position.ToProto()
	assert.Equal(t, "0", pp.UnrealisedPnl)
	assert.Equal(t, "-1000", pp.RealisedPnl)
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
	lsevt := events.NewLossSocializationEvent(ctx, party, market, num.NewUint(300), true, 1)
	position.UpdateWithLossSocialization(lsevt)
	pp = position.ToProto()
	assert.Equal(t, "-300", pp.RealisedPnl)
	assert.Equal(t, "-600", pp.UnrealisedPnl)
}

type tradeStub struct {
	size  int64
	price *num.Uint
}

func (t tradeStub) Size() int64 {
	return t.size
}

func (t tradeStub) Price() *num.Uint {
	return t.price.Clone()
}

func (t tradeStub) MarketPrice() *num.Uint {
	return t.price.Clone()
}
