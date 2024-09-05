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

package plugins_test

// No race condition checks on these tests, the channels are buffered to avoid actual issues
// we are aware that the tests themselves can be written in an unsafe way, but that's the tests
// not the code itsel. The behaviour of the tests is 100% reliable.
import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/plugins"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type tradeStub struct {
	size  int64
	price *num.Uint
}

type posPluginTst struct {
	*plugins.Positions
	ctrl  *gomock.Controller
	ctx   context.Context
	cfunc context.CancelFunc
}

func TestMultipleTradesOfSameSize(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	market := "market-id"
	ps := events.NewSettlePositionEvent(position.ctx, "party1", market, num.NewUint(1000), []events.TradeSettlement{
		tradeStub{
			size:  -1,
			price: num.NewUint(1000),
		},
		tradeStub{
			size:  -1,
			price: num.NewUint(1000),
		},
	}, 1, num.DecimalFromFloat(1))
	position.Push(ps)
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// average entry price should be 1k
	assert.Equal(t, ps.Price(), pp[0].AverageEntryPrice)
}

func TestMultipleTradesAndLossSocializationPartyNoOpenVolume(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	market := "market-id"
	ps := events.NewSettlePositionEvent(position.ctx, "party1", market, num.NewUint(1000), []events.TradeSettlement{
		tradeStub{
			size:  2,
			price: num.NewUint(1000),
		},
		tradeStub{
			size:  -2,
			price: num.NewUint(1500),
		},
	}, 1, num.DecimalFromFloat(1))
	position.Push(ps)
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// average entry price should be 1k
	// initially calculation say the RealisedPNL should be 1000
	assert.Equal(t, "1000", pp[0].RealisedPnl.String())

	// then we process the event for LossSocialization
	lsevt := events.NewLossSocializationEvent(position.ctx, "party1", market, num.NewUint(300), true, 1, types.LossTypeUnspecified)
	position.Push(lsevt)
	pp, err = position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// with the changes, the RealisedPNL should be 700
	assert.Equal(t, "700", pp[0].RealisedPnl.String())
	assert.Equal(t, "0", pp[0].UnrealisedPnl.String())
}

func TestDistressedPartyUpdate(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	market := "market-id"
	ps := events.NewSettlePositionEvent(position.ctx, "party1", market, num.NewUint(1000), []events.TradeSettlement{
		tradeStub{
			size:  2,
			price: num.NewUint(1000),
		},
		tradeStub{
			size:  3,
			price: num.NewUint(1200),
		},
	}, 1, num.DecimalFromFloat(1))
	position.Push(ps)
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// average entry price should be 1k
	// initially calculation say the RealisedPNL should be 1000
	assert.Equal(t, "0", pp[0].RealisedPnl.String())
	assert.Equal(t, "-600", pp[0].UnrealisedPnl.String())

	// then we process the event for LossSocialization
	lsevt := events.NewLossSocializationEvent(position.ctx, "party1", market, num.NewUint(300), true, 1, types.LossTypeUnspecified)
	position.Push(lsevt)
	pp, err = position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// with the changes, the RealisedPNL should be 700
	assert.Equal(t, "-300", pp[0].RealisedPnl.String())
	assert.Equal(t, "-600", pp[0].UnrealisedPnl.String())
	// now assume this party is distressed, and we've taken all their funds
	sde := events.NewSettleDistressed(position.ctx, "party1", market, num.UintZero(), num.NewUint(100), 1)
	position.Push(sde)
	pp, err = position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	assert.Equal(t, "0", pp[0].UnrealisedPnl.String())
	assert.Equal(t, "-1000", pp[0].RealisedPnl.String())
}

func TestMultipleTradesAndLossSocializationPartyWithOpenVolume(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	market := "market-id"
	ps := events.NewSettlePositionEvent(position.ctx, "party1", market, num.NewUint(1000), []events.TradeSettlement{
		tradeStub{
			size:  2,
			price: num.NewUint(1000),
		},
		tradeStub{
			size:  3,
			price: num.NewUint(1200),
		},
	}, 1, num.DecimalFromFloat(1))
	position.Push(ps)
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// average entry price should be 1k
	// initially calculation say the RealisedPNL should be 1000
	assert.Equal(t, "0", pp[0].RealisedPnl.String())
	assert.Equal(t, "-600", pp[0].UnrealisedPnl.String())

	// then we process the event for LossSocialization
	lsevt := events.NewLossSocializationEvent(position.ctx, "party1", market, num.NewUint(300), true, 1, types.LossTypeUnspecified)
	position.Push(lsevt)
	pp, err = position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// with the changes, the RealisedPNL should be 700
	assert.Equal(t, "-300", pp[0].RealisedPnl.String())
	assert.Equal(t, "-600", pp[0].UnrealisedPnl.String())
}

func getPosPlugin(t *testing.T) *posPluginTst {
	t.Helper()
	ctrl := gomock.NewController(t)
	ctx, cfunc := context.WithCancel(context.Background())
	p := plugins.NewPositions(ctx)
	tst := posPluginTst{
		Positions: p,
		ctrl:      ctrl,
		ctx:       ctx,
		cfunc:     cfunc,
	}
	return &tst
}

func (p *posPluginTst) Finish() {
	p.cfunc() // cancel context
	defer p.ctrl.Finish()
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
