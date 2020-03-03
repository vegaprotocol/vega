// +build !race ignore

package plugins_test

// No race condition checks on these tests, the channels are buffered to avoid actual issues
// we are aware that the tests themselves can be written in an unsafe way, but that's the tests
// not the code itsel. The behaviour of the tests is 100% reliable
import (
	"context"
	"sync"
	"testing"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/plugins"
	"code.vegaprotocol.io/vega/plugins/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type posStub struct {
	mID, party      string
	size, buy, sell int64
	price           uint64
	trades          []events.TradeSettlement
	margin          uint64
	hasMargin       bool
}

type tradeStub struct {
	size  int64
	price uint64
}

type posPluginTst struct {
	*plugins.Positions
	ctrl  *gomock.Controller
	ctx   context.Context
	cfunc context.CancelFunc
	pos   *mocks.MockPosBuffer
	ls    *mocks.MockLossSocializationBuffer
}

func TestStartStop(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	// make buffered channel. We're not going to be waiting on anything from here anyway
	// if it's not buffered the select-case might be blocking
	ch := make(chan []events.SettlePosition, 1)
	lsch := make(chan []events.LossSocialization)
	ref := 0
	position.pos.EXPECT().Subscribe().Times(1).Return(ch, ref)
	position.ls.EXPECT().Subscribe().Times(1).Return(lsch, ref)
	// will be called by Stop(), might be called when ctx is cancelled
	position.pos.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
		if ch != nil {
			close(ch)
			ch = nil
		}
	})
	position.ls.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
		if ch != nil {
			close(ch)
			ch = nil
		}
	})
	position.Start(position.ctx)
	position.Stop()
}

func TestStartCtxCancel(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	// make buffered channel. We're not going to be waiting on anything from here anyway
	// if it's not buffered the select-case might be blocking
	ch := make(chan []events.SettlePosition, 1)
	lsch := make(chan []events.LossSocialization)
	ref := 0
	wg := sync.WaitGroup{}
	wg.Add(1)
	position.pos.EXPECT().Subscribe().Times(1).Return(ch, ref)
	position.ls.EXPECT().Subscribe().Times(1).Return(lsch, ref)
	// unsubscribe should be called only on ctx cancel
	position.pos.EXPECT().Unsubscribe(ref).MinTimes(1).DoAndReturn(func(_ int) {
		if ch != nil {
			close(ch)
			ch = nil
			wg.Done()
		}
	})
	position.ls.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
		if ch != nil {
			close(ch)
			ch = nil
		}
	})
	position.Start(position.ctx)
	position.cfunc()
	wg.Wait() // wait for ctx cancel to have had its effect
}

func TestMultipleTradesOfSameSize(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	ch := make(chan []events.SettlePosition)
	ref := 1
	lsch := make(chan []events.LossSocialization)
	market := "market-id"
	position.pos.EXPECT().Subscribe().Times(1).Return(ch, ref)
	position.ls.EXPECT().Subscribe().Times(1).Return(lsch, ref)
	position.pos.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
		if ch != nil {
			close(ch)
			ch = nil
		}
	})
	position.ls.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
		if ch != nil {
			close(ch)
			ch = nil
		}
	})
	position.Start(position.ctx)
	ps := posStub{
		mID:   market,
		party: "trader1",
		size:  -2,
		price: 1000,
		trades: []events.TradeSettlement{
			tradeStub{
				size:  -1,
				price: 1000,
			},
			tradeStub{
				size:  -1,
				price: 1000,
			},
		},
	}
	ch <- []events.SettlePosition{ps}
	ch <- []events.SettlePosition{}
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// average entry price should be 1k
	assert.Equal(t, ps.price, pp[0].AverageEntryPrice)
}

func TestMultipleTradesAndLossSocializationTraderNoOpenVolume(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	ch := make(chan []events.SettlePosition)
	ref := 1
	lsch := make(chan []events.LossSocialization)
	market := "market-id"
	position.pos.EXPECT().Subscribe().Times(1).Return(ch, ref)
	position.ls.EXPECT().Subscribe().Times(1).Return(lsch, ref)
	position.pos.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
		if ch != nil {
			close(ch)
			ch = nil
		}
	})
	position.ls.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
		if ch != nil {
			close(ch)
			ch = nil
		}
	})
	position.Start(position.ctx)
	ps := posStub{
		mID:   market,
		party: "trader1",
		size:  -2,
		price: 1000,
		trades: []events.TradeSettlement{
			tradeStub{
				size:  2,
				price: 1000,
			},
			tradeStub{
				size:  -2,
				price: 1500,
			},
		},
	}
	ch <- []events.SettlePosition{ps}
	ch <- []events.SettlePosition{}
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// average entry price should be 1k
	// initialy calculation say the RealisedPNL should be 1000
	assert.Equal(t, 1000, int(pp[0].RealisedPNL))

	// then we process the event for LossSocialization
	lsevt := lsStub{
		market:     market,
		party:      "trader1",
		amountLoss: -300,
		price:      1000,
	}
	lsch <- []events.LossSocialization{lsevt}
	lsch <- []events.LossSocialization{} // ensure previous event was processed
	pp, err = position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// with the changes, the RealisedPNL should be 700
	assert.Equal(t, 700, int(pp[0].RealisedPNL))
	assert.Equal(t, 0, int(pp[0].UnrealisedPNL))
}

func TestDistressedTraderUpdate(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	ch := make(chan []events.SettlePosition)
	ref := 1
	lsch := make(chan []events.LossSocialization)
	market := "market-id"
	position.pos.EXPECT().Subscribe().Times(1).Return(ch, ref)
	position.ls.EXPECT().Subscribe().Times(1).Return(lsch, ref)
	position.pos.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
		if ch != nil {
			close(ch)
			ch = nil
		}
	})
	position.ls.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
		if ch != nil {
			close(ch)
			ch = nil
		}
	})
	position.Start(position.ctx)
	ps := posStub{
		mID:   market,
		party: "trader1",
		size:  0,
		price: 1000,
		trades: []events.TradeSettlement{
			tradeStub{
				size:  2,
				price: 1000,
			},
			tradeStub{
				size:  3,
				price: 1200,
			},
		},
	}
	ch <- []events.SettlePosition{ps}
	ch <- []events.SettlePosition{}
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// average entry price should be 1k
	// initialy calculation say the RealisedPNL should be 1000
	assert.Equal(t, 0, int(pp[0].RealisedPNL))
	assert.Equal(t, -600, int(pp[0].UnrealisedPNL))

	// then we process the event for LossSocialization
	lsevt := lsStub{
		market:     market,
		party:      "trader1",
		amountLoss: -300,
		price:      1000,
	}
	lsch <- []events.LossSocialization{lsevt}
	lsch <- []events.LossSocialization{} // ensure the previous events were processed
	pp, err = position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// with the changes, the RealisedPNL should be 700
	assert.Equal(t, -300, int(pp[0].RealisedPNL))
	assert.Equal(t, -600, int(pp[0].UnrealisedPNL))
	// now assume this trader is distressed, and we've taken all their funds
	ps = posStub{
		mID:       market,
		party:     "trader1",
		size:      0,
		hasMargin: true,
		margin:    100,
	}
	ch <- []events.SettlePosition{ps}
	ch <- []events.SettlePosition{} // ensure the empty array is read. This ensures the actual event was processed
	pp, err = position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	assert.Equal(t, 0, int(pp[0].UnrealisedPNL))
	assert.Equal(t, -1000, int(pp[0].RealisedPNL))
}

func TestMultipleTradesAndLossSocializationTraderWithOpenVolume(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	ch := make(chan []events.SettlePosition)
	ref := 1
	lsch := make(chan []events.LossSocialization)
	market := "market-id"
	position.pos.EXPECT().Subscribe().Times(1).Return(ch, ref)
	position.ls.EXPECT().Subscribe().Times(1).Return(lsch, ref)
	position.pos.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
		if ch != nil {
			close(ch)
			ch = nil
		}
	})
	position.ls.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
		if ch != nil {
			close(ch)
			ch = nil
		}
	})
	position.Start(position.ctx)
	ps := posStub{
		mID:   market,
		party: "trader1",
		size:  0,
		price: 1000,
		trades: []events.TradeSettlement{
			tradeStub{
				size:  2,
				price: 1000,
			},
			tradeStub{
				size:  3,
				price: 1200,
			},
		},
	}
	ch <- []events.SettlePosition{ps}
	ch <- []events.SettlePosition{}
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// average entry price should be 1k
	// initialy calculation say the RealisedPNL should be 1000
	assert.Equal(t, 0, int(pp[0].RealisedPNL))
	assert.Equal(t, -600, int(pp[0].UnrealisedPNL))

	// then we process the event for LossSocialization
	lsevt := lsStub{
		market:     market,
		party:      "trader1",
		amountLoss: -300,
		price:      1000,
	}
	lsch <- []events.LossSocialization{lsevt}
	lsch <- []events.LossSocialization{} // when this is read, the actual event was processed
	pp, err = position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// with the changes, the RealisedPNL should be 700
	assert.Equal(t, -300, int(pp[0].RealisedPNL))
	assert.Equal(t, -600, int(pp[0].UnrealisedPNL))
}

func TestProcessBufferData(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	// ch := make(chan []events.SettlePosition, 1)
	ch := make(chan []events.SettlePosition)
	ref := 1
	lsch := make(chan []events.LossSocialization)
	position.pos.EXPECT().Subscribe().Times(1).Return(ch, ref)
	position.ls.EXPECT().Subscribe().Times(1).Return(lsch, ref)
	position.pos.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
		if ch != nil {
			close(ch)
			ch = nil
		}
	})
	position.ls.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
		if ch != nil {
			close(ch)
			ch = nil
		}
	})
	position.Start(position.ctx)
	market := "market-id"
	// set up a position or two:
	ps := posStub{
		mID:   market,
		party: "trader1",
		size:  10,
		price: 1000,
		trades: []events.TradeSettlement{
			tradeStub{
				size:  10,
				price: 1000,
			},
		},
	}
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		ch <- []events.SettlePosition{ps}
		ch <- []events.SettlePosition{}
		wg.Done()
	}()
	ch <- []events.SettlePosition{ps}
	ch <- []events.SettlePosition{}
	ps.party = "trader2"
	ps.size = -10
	ps.trades[0] = tradeStub{
		size:  -10,
		price: 1000,
	}
	go func() {
		ch <- []events.SettlePosition{ps}
		ch <- []events.SettlePosition{}
		wg.Done()
	}()
	wg.Wait()
	// position.Stop()
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
}

func getPosPlugin(t *testing.T) *posPluginTst {
	ctrl := gomock.NewController(t)
	pos := mocks.NewMockPosBuffer(ctrl)
	ls := mocks.NewMockLossSocializationBuffer(ctrl)
	p := plugins.NewPositions(pos, ls)
	tst := posPluginTst{
		Positions: p,
		pos:       pos,
		ls:        ls,
		ctrl:      ctrl,
	}
	tst.ctx, tst.cfunc = context.WithCancel(context.Background())
	return &tst
}

func (p *posPluginTst) Finish() {
	p.cfunc() // cancel context
	defer p.ctrl.Finish()
}

func (p posStub) MarketID() string {
	return p.mID
}

func (p posStub) Party() string {
	return p.party
}

func (p posStub) Size() int64 {
	return p.size
}

func (p posStub) Buy() int64 {
	return p.buy
}

func (p posStub) Sell() int64 {
	return p.sell
}

func (p posStub) Price() uint64 {
	return p.price
}

func (p posStub) Trades() []events.TradeSettlement {
	return p.trades
}

func (p posStub) Margin() (uint64, bool) {
	return p.margin, p.hasMargin
}

func (t tradeStub) Size() int64 {
	return t.size
}

func (t tradeStub) Price() uint64 {
	return t.price
}

type lsStub struct {
	market     string
	party      string
	amountLoss int64
	price      uint64
}

func (l lsStub) MarketID() string {
	return l.market
}

func (l lsStub) PartyID() string {
	return l.party
}

func (l lsStub) AmountLost() int64 {
	return l.amountLoss
}

func (l lsStub) Price() uint64 {
	return l.price
}
