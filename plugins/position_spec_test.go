// +build !race ignore

package plugins_test

import (
	"testing"

	"code.vegaprotocol.io/vega/events"
	"github.com/stretchr/testify/assert"
)

func TestPositionSpecSuite(t *testing.T) {
	t.Run("Long gets more long", testLongGetMoreLong)
	t.Run("Long gets less long", testLongGetsLessLong)
	t.Run("Long gets closed", testLongGetsClosed)
	t.Run("Long gets turned short", testLongGetsTurnedShort)
	t.Run("Short gets more short", testShortGetsMoreShort)
	t.Run("Short gets less short", testShortGetsLessShort)
	t.Run("Short gets turned long", testShortGetsTurnedLong)
	t.Run("Short gets closed", testShortGetsClosed)
}

func testLongGetMoreLong(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	ch := make(chan []events.SettlePosition)
	ref := 1
	market := "market-id"
	position.pos.EXPECT().Subscribe().Times(1).Return(ch, ref)
	position.pos.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
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
				size:  100,
				price: 50,
			},
			tradeStub{
				size:  25,
				price: 100,
			},
		},
	}
	ch <- []events.SettlePosition{ps}
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	// average entry price should be 1k
	assert.Equal(t, 60, int(pp[0].AverageEntryPrice))
	assert.Equal(t, 125, int(pp[0].OpenVolume))
	assert.Equal(t, 5000, int(pp[0].UnrealisedPNL))
	assert.Equal(t, 0, int(pp[0].RealisedPNL))
}

func testLongGetsLessLong(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	ch := make(chan []events.SettlePosition)
	ref := 1
	market := "market-id"
	position.pos.EXPECT().Subscribe().Times(1).Return(ch, ref)
	position.pos.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
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
				size:  100,
				price: 50,
			},
			tradeStub{
				size:  -25,
				price: 100,
			},
		},
	}
	ch <- []events.SettlePosition{ps}
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	assert.Equal(t, 50, int(pp[0].AverageEntryPrice))
	assert.Equal(t, 75, int(pp[0].OpenVolume))
	assert.Equal(t, 3750, int(pp[0].UnrealisedPNL))
	assert.Equal(t, 1250, int(pp[0].RealisedPNL))
}

func testLongGetsClosed(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	ch := make(chan []events.SettlePosition)
	ref := 1
	market := "market-id"
	position.pos.EXPECT().Subscribe().Times(1).Return(ch, ref)
	position.pos.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
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
				size:  100,
				price: 50,
			},
			tradeStub{
				size:  -100,
				price: 100,
			},
		},
	}
	ch <- []events.SettlePosition{ps}
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	assert.Equal(t, 0, int(pp[0].AverageEntryPrice))
	assert.Equal(t, 0, int(pp[0].OpenVolume))
	assert.Equal(t, 0, int(pp[0].UnrealisedPNL))
	assert.Equal(t, 5000, int(pp[0].RealisedPNL))
}

func testLongGetsTurnedShort(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	ch := make(chan []events.SettlePosition)
	ref := 1
	market := "market-id"
	position.pos.EXPECT().Subscribe().Times(1).Return(ch, ref)
	position.pos.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
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
				size:  100,
				price: 50,
			},
			tradeStub{
				size:  -125,
				price: 100,
			},
		},
	}
	ch <- []events.SettlePosition{ps}
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	assert.Equal(t, 100, int(pp[0].AverageEntryPrice))
	assert.Equal(t, -25, int(pp[0].OpenVolume))
	assert.Equal(t, 0, int(pp[0].UnrealisedPNL))
	assert.Equal(t, 5000, int(pp[0].RealisedPNL))
}

func testShortGetsMoreShort(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	ch := make(chan []events.SettlePosition)
	ref := 1
	market := "market-id"
	position.pos.EXPECT().Subscribe().Times(1).Return(ch, ref)
	position.pos.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
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
				size:  -100,
				price: 50,
			},
			tradeStub{
				size:  -25,
				price: 100,
			},
		},
	}
	ch <- []events.SettlePosition{ps}
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	assert.Equal(t, 60, int(pp[0].AverageEntryPrice))
	assert.Equal(t, -125, int(pp[0].OpenVolume))
	assert.Equal(t, -5000, int(pp[0].UnrealisedPNL))
	assert.Equal(t, 0, int(pp[0].RealisedPNL))
}

func testShortGetsLessShort(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	ch := make(chan []events.SettlePosition)
	ref := 1
	market := "market-id"
	position.pos.EXPECT().Subscribe().Times(1).Return(ch, ref)
	position.pos.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
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
				size:  -100,
				price: 50,
			},
			tradeStub{
				size:  25,
				price: 100,
			},
		},
	}
	ch <- []events.SettlePosition{ps}
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	assert.Equal(t, 50, int(pp[0].AverageEntryPrice))
	assert.Equal(t, -75, int(pp[0].OpenVolume))
	assert.Equal(t, -3750, int(pp[0].UnrealisedPNL))
	assert.Equal(t, -1250, int(pp[0].RealisedPNL))
}

func testShortGetsClosed(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	ch := make(chan []events.SettlePosition)
	ref := 1
	market := "market-id"
	position.pos.EXPECT().Subscribe().Times(1).Return(ch, ref)
	position.pos.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
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
				size:  -100,
				price: 50,
			},
			tradeStub{
				size:  100,
				price: 100,
			},
		},
	}
	ch <- []events.SettlePosition{ps}
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	assert.Equal(t, 0, int(pp[0].AverageEntryPrice))
	assert.Equal(t, 0, int(pp[0].OpenVolume))
	assert.Equal(t, 0, int(pp[0].UnrealisedPNL))
	assert.Equal(t, -5000, int(pp[0].RealisedPNL))
}

func testShortGetsTurnedLong(t *testing.T) {
	position := getPosPlugin(t)
	defer position.Finish()
	ch := make(chan []events.SettlePosition)
	ref := 1
	market := "market-id"
	position.pos.EXPECT().Subscribe().Times(1).Return(ch, ref)
	position.pos.EXPECT().Unsubscribe(ref).MinTimes(1).MaxTimes(2).DoAndReturn(func(_ int) {
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
				size:  -100,
				price: 50,
			},
			tradeStub{
				size:  125,
				price: 100,
			},
		},
	}
	ch <- []events.SettlePosition{ps}
	pp, err := position.GetPositionsByMarket(market)
	assert.NoError(t, err)
	assert.NotZero(t, len(pp))
	assert.Equal(t, 100, int(pp[0].AverageEntryPrice))
	assert.Equal(t, 25, int(pp[0].OpenVolume))
	assert.Equal(t, 0, int(pp[0].UnrealisedPNL))
	assert.Equal(t, -5000, int(pp[0].RealisedPNL))
}
