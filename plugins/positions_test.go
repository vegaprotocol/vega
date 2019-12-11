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
}

func TestStartStop(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	// make buffered channel. We're not going to be waiting on anything from here anyway
	// if it's not buffered the select-case might be blocking
	ch := make(chan []events.SettlePosition, 1)
	ref := 0
	position.pos.EXPECT().Subscribe().Times(1).Return(ch, ref)
	// will be called by Stop(), might be called when ctx is cancelled
	position.pos.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
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
	ref := 0
	wg := sync.WaitGroup{}
	wg.Add(1)
	position.pos.EXPECT().Subscribe().Times(1).Return(ch, ref)
	// unsubscribe should be called only on ctx cancel
	position.pos.EXPECT().Unsubscribe(ref).MinTimes(1).DoAndReturn(func(_ int) {
		if ch != nil {
			close(ch)
			ch = nil
			wg.Done()
		}
	})
	position.Start(position.ctx)
	position.cfunc()
	wg.Wait() // wait for ctx cancel to have had its effect
}

func TestProcessBufferData(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	// ch := make(chan []events.SettlePosition, 1)
	ch := make(chan []events.SettlePosition)
	ref := 1
	position.pos.EXPECT().Subscribe().Times(1).Return(ch, ref)
	position.pos.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
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
		wg.Done()
	}()
	ch <- []events.SettlePosition{ps}
	ps.party = "trader2"
	ps.size = -10
	ps.trades[0] = tradeStub{
		size:  -10,
		price: 1000,
	}
	go func() {
		ch <- []events.SettlePosition{ps}
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
	p := plugins.NewPositions(pos)
	tst := posPluginTst{
		Positions: p,
		pos:       pos,
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

func (t tradeStub) Size() int64 {
	return t.size
}

func (t tradeStub) Price() uint64 {
	return t.price
}
