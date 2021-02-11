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
		current float64
		target  float64
		// expect
		auctionShouldEnd bool
	}{
		{"Current >  Target", 20, 15, true},
		{"Current == Target", 15, 15, true},
		{"Current <  Target", 14, 15, false},
	}

	mon := NewMonitor()
	h := newTestHarness(t).WhenInLiquidityAuction(true)
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			if test.auctionShouldEnd {
				h.AuctionState.EXPECT().AuctionEnd().Times(1)
			}
			mon.CheckLiquidity(h.AuctionState, time.Now(), constant, test.current, test.target)
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
		current float64
		target  float64
		// expect
		auctionShouldStart bool
	}{
		{"Current <  (Target * c1)", 10, 30, true},
		{"Current >  (Target * c1)", 15, 15, false},
		{"Current == (Target * c1)", 10, 20, false},
	}

	mon := NewMonitor()
	h := newTestHarness(t).WhenInLiquidityAuction(false)
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			if test.auctionShouldStart {
				h.AuctionState.EXPECT().StartLiquidityAuction(now, nil).Times(1)
			}

			mon.CheckLiquidity(h.AuctionState, now, constant, test.current, test.target)
		})
	}
}
