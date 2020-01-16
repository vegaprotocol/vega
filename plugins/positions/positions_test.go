package positions_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/buffer"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/plugins/positions"
	"code.vegaprotocol.io/vega/plugins/positions/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type tstPos struct {
	*positions.Pos
	ctrl  *gomock.Controller
	ctx   context.Context
	cfunc context.CancelFunc
	sub   *mocks.MockSubscriber
	buf   *mocks.MockPlugBuffer
	ch    chan []events.SettlePosition
}

type testData struct {
	evt                        posStub
	realised, open, unrealised int64
	aep                        uint64
	trader, market             string
}

var (
	rawCfg = map[string]interface{}{
		positions.PluginName: positions.DefaultConfig(),
	}
)

func TestSetup(t *testing.T) {
	pos := getTestPos(t)
	defer pos.Finish()
	_ = pos.Start()
	pos.cfunc()
}

// this test needs to get the buffer mocked passed in, still
// @TODO work out what to do with the config here
func testNew(t *testing.T) {
	pos := getTestPos(t)
	defer pos.Finish()
	pos.buf.EXPECT().PositionsSub(gomock.Any()).AnyTimes().Return(&buffer.SettleSub{})
	//@TODO this rawCfg is wrong, aparently it's a map[string]map[string]interface{}
	// not a map[string]interface{} where the interface{} values are config objects?
	p2, err := pos.New(logging.NewTestLogger(), pos.ctx, pos.buf, nil, rawCfg[positions.PluginName])
	assert.NoError(t, err)
	// ensure we get the same type back
	assert.IsType(t, pos, p2)
}

func TestLongPositions(t *testing.T) {
	mkt := "test-market"
	data := map[string][]testData{
		"long gets more long": []testData{
			{
				evt: posStub{
					mID:   mkt,
					size:  100,
					price: 5,
					trades: []events.TradeSettlement{
						tradeStub{
							size:  100,
							price: 5,
						},
					},
				},
				realised:   0,
				open:       100,
				unrealised: 0,
				aep:        5,
				trader:     "trader-1",
				market:     mkt,
			},
			{
				evt: posStub{
					mID:   mkt,
					size:  125,
					price: 10,
					trades: []events.TradeSettlement{
						tradeStub{
							size:  25,
							price: 10,
						},
					},
				},
				realised:   0,
				open:       125,
				unrealised: 500,
				aep:        6,
				trader:     "trader-1",
				market:     mkt,
			},
		},
		"long gets less long": []testData{
			{
				evt: posStub{
					mID:   mkt,
					size:  100,
					price: 5,
					trades: []events.TradeSettlement{
						tradeStub{
							size:  100,
							price: 5,
						},
					},
				},
				realised:   0,
				open:       100,
				unrealised: 0,
				aep:        5,
				trader:     "trader-2",
				market:     mkt,
			},
			{
				evt: posStub{
					mID:   mkt,
					size:  75,
					price: 10,
					trades: []events.TradeSettlement{
						tradeStub{
							size:  -25,
							price: 10,
						},
					},
				},
				realised:   125,
				open:       75,
				unrealised: 375,
				aep:        5,
				trader:     "trader-2",
				market:     mkt,
			},
		},
		"long gets closed": []testData{
			{
				evt: posStub{
					mID:   mkt,
					size:  100,
					price: 5,
					trades: []events.TradeSettlement{
						tradeStub{
							size:  100,
							price: 5,
						},
					},
				},
				realised:   0,
				open:       100,
				unrealised: 0,
				aep:        5,
				trader:     "trader-3",
				market:     mkt,
			},
			{
				evt: posStub{
					mID:   mkt,
					size:  0,
					price: 10,
					trades: []events.TradeSettlement{
						tradeStub{
							size:  -100,
							price: 10,
						},
					},
				},
				realised:   0,
				open:       0,
				unrealised: 0,
				aep:        0,
				trader:     "trader-3",
				market:     mkt,
			},
		},
		"long gets turned short": []testData{
			{
				evt: posStub{
					mID:   mkt,
					size:  100,
					price: 5,
					trades: []events.TradeSettlement{
						tradeStub{
							size:  100,
							price: 5,
						},
					},
				},
				realised:   0,
				open:       100,
				unrealised: 0,
				aep:        5,
				trader:     "trader-4",
				market:     mkt,
			},
			{
				evt: posStub{
					mID:   mkt,
					size:  -25,
					price: 10,
					trades: []events.TradeSettlement{
						tradeStub{
							size:  -125,
							price: 10,
						},
					},
				},
				realised:   500,
				open:       -25,
				unrealised: -75,
				aep:        7,
				trader:     "trader-4",
				market:     mkt,
			},
		},
	}
	pos := getTestPos(t)
	_ = pos.Start()
	defer pos.Finish()
	for testSet, set := range data {
		for _, evt := range set {
			evt.evt.party = evt.trader
			evt.evt.mID = evt.market
			pos.ch <- []events.SettlePosition{evt.evt}
			pos.ch <- nil // this blocks test until event has updated data
			p, err := pos.GetPositionsByMarketAndParty(evt.market, evt.trader)
			assert.NoError(t, err, testSet)
			assert.Equal(t, evt.open, p.OpenVolume, testSet)
			assert.Equal(t, evt.realised, p.RealisedPNL, testSet)
			assert.Equal(t, evt.unrealised, p.UnrealisedPNL, testSet)
			assert.Equal(t, evt.aep, p.AverageEntryPrice, testSet)
		}
	}
}

func TestShortPositions(t *testing.T) {
	mkt := "test-market"
	data := map[string][]testData{
		"short gets more short": []testData{
			{
				evt: posStub{
					mID:   mkt,
					size:  -100,
					price: 5,
					trades: []events.TradeSettlement{
						tradeStub{
							size:  -100,
							price: 5,
						},
					},
				},
				realised:   0,
				open:       -100,
				unrealised: 0,
				aep:        5,
				trader:     "trader-1",
				market:     mkt,
			},
			{
				evt: posStub{
					mID:   mkt,
					size:  -125,
					price: 10,
					trades: []events.TradeSettlement{
						tradeStub{
							size:  -25,
							price: 10,
						},
					},
				},
				realised:   0,
				open:       -125,
				unrealised: -500,
				aep:        6,
				trader:     "trader-1",
				market:     mkt,
			},
		},
		"short gets less short": []testData{
			{
				evt: posStub{
					mID:   mkt,
					size:  -100,
					price: 5,
					trades: []events.TradeSettlement{
						tradeStub{
							size:  -100,
							price: 5,
						},
					},
				},
				realised:   0,
				open:       -100,
				unrealised: 0,
				aep:        5,
				trader:     "trader-2",
				market:     mkt,
			},
			{
				evt: posStub{
					mID:   mkt,
					size:  -75,
					price: 10,
					trades: []events.TradeSettlement{
						tradeStub{
							size:  25,
							price: 10,
						},
					},
				},
				realised:   -125,
				open:       -75,
				unrealised: -375,
				aep:        5,
				trader:     "trader-2",
				market:     mkt,
			},
		},
		"short gets closed": []testData{
			{
				evt: posStub{
					mID:   mkt,
					size:  -100,
					price: 5,
					trades: []events.TradeSettlement{
						tradeStub{
							size:  -100,
							price: 5,
						},
					},
				},
				realised:   0,
				open:       -100,
				unrealised: 0,
				aep:        5,
				trader:     "trader-3",
				market:     mkt,
			},
			{
				evt: posStub{
					mID:   mkt,
					size:  0,
					price: 10,
					trades: []events.TradeSettlement{
						tradeStub{
							size:  100,
							price: 10,
						},
					},
				},
				realised:   0,
				open:       0,
				unrealised: 0,
				aep:        0,
				trader:     "trader-3",
				market:     mkt,
			},
		},
		"short gets turned long": []testData{
			{
				evt: posStub{
					mID:   mkt,
					size:  -100,
					price: 5,
					trades: []events.TradeSettlement{
						tradeStub{
							size:  -100,
							price: 5,
						},
					},
				},
				realised:   0,
				open:       -100,
				unrealised: 0,
				aep:        5,
				trader:     "trader-4",
				market:     mkt,
			},
			{
				evt: posStub{
					mID:   mkt,
					size:  25,
					price: 10,
					trades: []events.TradeSettlement{
						tradeStub{
							size:  125,
							price: 10,
						},
					},
				},
				realised:   -500,
				open:       25,
				unrealised: 75,
				aep:        7,
				trader:     "trader-4",
				market:     mkt,
			},
		},
	}
	pos := getTestPos(t)
	_ = pos.Start()
	defer pos.Finish()
	for testSet, set := range data {
		for _, evt := range set {
			evt.evt.party = evt.trader
			evt.evt.mID = evt.market
			pos.ch <- []events.SettlePosition{evt.evt}
			pos.ch <- nil // this blocks test until event has updated data
			p, err := pos.GetPositionsByMarketAndParty(evt.market, evt.trader)
			assert.NoError(t, err, testSet)
			assert.Equal(t, evt.open, p.OpenVolume, testSet)
			assert.Equal(t, evt.realised, p.RealisedPNL, testSet)
			assert.Equal(t, evt.unrealised, p.UnrealisedPNL, testSet)
			assert.Equal(t, evt.aep, p.AverageEntryPrice, testSet)
		}
	}
}

func TestComplexScenario(t *testing.T) {
	mkt := "test-market"
	data := map[string][]testData{
		"long up and down": []testData{
			{
				evt: posStub{
					mID:   mkt,
					size:  100,
					price: 100,
					trades: []events.TradeSettlement{
						tradeStub{
							size:  100,
							price: 100,
						},
					},
				},
				realised:   0,
				open:       100,
				unrealised: 0,
				aep:        100,
				trader:     "trader-1",
				market:     mkt,
			},
			{
				evt: posStub{
					mID:   mkt,
					size:  75,
					price: 25,
					trades: []events.TradeSettlement{
						tradeStub{
							size:  -25,
							price: 25,
						},
					},
				},
				realised:   -1875,
				open:       75,
				unrealised: -5625,
				aep:        100,
				trader:     "trader-1",
				market:     mkt,
			},
			{
				evt: posStub{
					mID:   mkt,
					size:  125,
					price: 50,
					trades: []events.TradeSettlement{
						tradeStub{
							size:  50,
							price: 50,
						},
					},
				},
				realised:   -1875,
				open:       125,
				unrealised: -3750,
				aep:        80,
				trader:     "trader-1",
				market:     mkt,
			},
			{
				evt: posStub{
					mID:   mkt,
					size:  25,
					price: 75,
					trades: []events.TradeSettlement{
						tradeStub{
							size:  -100,
							price: 75,
						},
					},
				},
				realised:   -2375,
				open:       25,
				unrealised: -125,
				aep:        80,
				trader:     "trader-1",
				market:     mkt,
			},
		},
	}
	pos := getTestPos(t)
	_ = pos.Start()
	defer pos.Finish()
	for testSet, set := range data {
		for _, evt := range set {
			evt.evt.party = evt.trader
			evt.evt.mID = evt.market
			pos.ch <- []events.SettlePosition{evt.evt}
			pos.ch <- nil // this blocks test until event has updated data
			p, err := pos.GetPositionsByMarketAndParty(evt.market, evt.trader)
			assert.NoError(t, err, testSet)
			assert.Equal(t, evt.open, p.OpenVolume, testSet)
			assert.Equal(t, evt.realised, p.RealisedPNL, testSet)
			assert.Equal(t, evt.unrealised, p.UnrealisedPNL, testSet)
			assert.Equal(t, evt.aep, p.AverageEntryPrice, testSet)
		}
	}
}

func getTestPos(t *testing.T) *tstPos {
	ctrl := gomock.NewController(t)
	sub := mocks.NewMockSubscriber(ctrl)
	ch := make(chan []events.SettlePosition) // do not buffer channel, ensuring the test values have been read
	buf := mocks.NewMockPlugBuffer(ctrl)
	ctx, cfunc := context.WithCancel(context.Background())
	sub.EXPECT().Recv().AnyTimes().Return(ch)
	sub.EXPECT().Done().AnyTimes().DoAndReturn(func() <-chan struct{} {
		return ctx.Done()
	})
	return &tstPos{
		Pos:   positions.New(ctx, sub, positions.NewPositionsStore(ctx)),
		ctrl:  ctrl,
		ctx:   ctx,
		cfunc: cfunc,
		sub:   sub,
		buf:   buf,
		ch:    ch,
	}
}

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

func (p *tstPos) Finish() {
	p.cfunc()
	p.ctrl.Finish()
	close(p.ch)
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
