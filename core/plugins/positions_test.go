// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package plugins_test

// No race condition checks on these tests, the channels are buffered to avoid actual issues
// we are aware that the tests themselves can be written in an unsafe way, but that's the tests
// not the code itsel. The behaviour of the tests is 100% reliable.
import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/plugins"
	"code.vegaprotocol.io/vega/types/num"

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
	lsevt := events.NewLossSocializationEvent(position.ctx, "party1", market, num.NewUint(300), true, 1)
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
	lsevt := events.NewLossSocializationEvent(position.ctx, "party1", market, num.NewUint(300), true, 1)
	position.Push(lsevt)
	pp, err = position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// with the changes, the RealisedPNL should be 700
	assert.Equal(t, "-300", pp[0].RealisedPnl.String())
	assert.Equal(t, "-600", pp[0].UnrealisedPnl.String())
	// now assume this party is distressed, and we've taken all their funds
	sde := events.NewSettleDistressed(position.ctx, "party1", market, num.Zero(), num.NewUint(100), 1)
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
	lsevt := events.NewLossSocializationEvent(position.ctx, "party1", market, num.NewUint(300), true, 1)
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
