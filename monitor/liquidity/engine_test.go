package liquidity

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/monitor/liquidity/mocks"
	"github.com/golang/mock/gomock"
)

type testHarness struct {
	AuctionState *mocks.MockAuctionState
}

func newTestHarness(t *testing.T) *testHarness {
	ctrl := gomock.NewController(t)
	return &testHarness{
		AuctionState: mocks.NewMockAuctionState(ctrl),
	}
}

func (h *testHarness) WhenInLiquidityAuction(v bool) *testHarness {
	h.AuctionState.EXPECT().IsLiquidityAuction().AnyTimes().Return(v)
	return h
}

func TestEngineWhenInLiquidityAuction(t *testing.T) {
	var constant = 0.7

	tests := []struct {
		desc string
		// when
		current             float64
		target              float64
		bestStaticBidVolume uint64
		bestStaticAskVolume uint64
		// expect
		auctionShouldEnd bool
	}{
		{"Current >  Target", 20, 15, 1, 1, true},
		{"Current == Target", 15, 15, 1, 1, true},
		{"Current <  Target", 14, 15, 1, 1, false},
		{"Current >  Target, no best bid", 20, 15, 0, 1, false},
		{"Current == Target, no best ask", 15, 15, 1, 0, false},
		{"Current == Target, no best bid and ask", 15, 15, 0, 0, false},
	}

	mon := NewMonitor()
	h := newTestHarness(t).WhenInLiquidityAuction(true)
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			if test.auctionShouldEnd {
				h.AuctionState.EXPECT().EndAuction().Times(1)
			}
			mon.CheckLiquidity(h.AuctionState, time.Now(), constant, test.current, test.target, test.bestStaticBidVolume, test.bestStaticAskVolume)
		})
	}
}

func TestEngineWhenNotInLiquidityAuction(t *testing.T) {
	var (
		constant = 0.5
		now      = time.Now()
	)

	tests := []struct {
		desc string
		// when
		current             float64
		target              float64
		bestStaticBidVolume uint64
		bestStaticAskVolume uint64
		// expect
		auctionShouldStart bool
	}{
		{"Current <  (Target * c1)", 10, 30, 1, 1, true},
		{"Current >  (Target * c1)", 15, 15, 1, 1, false},
		{"Current == (Target * c1)", 10, 20, 1, 1, false},
		{"Current >  (Target * c1), no best bid", 15, 15, 0, 1, true},
		{"Current == (Target * c1), no best ask", 10, 20, 1, 0, true},
		{"Current == (Target * c1), no best bid and ask", 10, 20, 0, 0, true},
	}

	mon := NewMonitor()
	h := newTestHarness(t).WhenInLiquidityAuction(false)
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			if test.auctionShouldStart {
				h.AuctionState.EXPECT().StartLiquidityAuction(now, nil).Times(1)
			}

			mon.CheckLiquidity(h.AuctionState, now, constant, test.current, test.target, test.bestStaticBidVolume, test.bestStaticAskVolume)
		})
	}
}
