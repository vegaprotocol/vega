package liquidity_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/monitor/liquidity"
	"code.vegaprotocol.io/vega/monitor/liquidity/mocks"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/golang/mock/gomock"
)

type testHarness struct {
	AuctionState          *mocks.MockAuctionState
	TargetStakeCalculator *mocks.MockTargetStakeCalculator
}

func newTestHarness(t *testing.T) *testHarness {
	ctrl := gomock.NewController(t)
	return &testHarness{
		AuctionState:          mocks.NewMockAuctionState(ctrl),
		TargetStakeCalculator: mocks.NewMockTargetStakeCalculator(ctrl),
	}
}

func (h *testHarness) WhenInLiquidityAuction(v bool) *testHarness {
	h.AuctionState.EXPECT().IsLiquidityAuction().AnyTimes().Return(v)
	return h
}

func TestEngineWhenInLiquidityAuction(t *testing.T) {
	now := time.Now()

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

	h := newTestHarness(t).WhenInLiquidityAuction(true)
	exp := now.Add(-1 * time.Second)
	keep := now.Add(time.Second)
	mon := liquidity.NewMonitor(h.TargetStakeCalculator, &types.LiquidityMonitoringParameters{
		TriggeringRatio: .7,
	})
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			if test.auctionShouldEnd {
				h.AuctionState.EXPECT().EndAuction().Times(1)
				h.AuctionState.EXPECT().ExpiresAt().Times(1).Return(&exp)
			} else {
				h.AuctionState.EXPECT().ExpiresAt().Times(1).Return(&keep)
			}
			var trades []*types.Trade = nil
			var rf types.RiskFactor = types.RiskFactor{}
			var markPrice uint64 = 100
			h.TargetStakeCalculator.EXPECT().GetTheoreticalTargetStake(rf, now, markPrice, trades).Return(test.target)
			mon.CheckLiquidity(h.AuctionState, now, test.current, trades, rf, markPrice, test.bestStaticBidVolume, test.bestStaticAskVolume)
		})
	}
}

func TestEngineWhenNotInLiquidityAuction(t *testing.T) {
	now := time.Now()

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

	h := newTestHarness(t).WhenInLiquidityAuction(false)
	mon := liquidity.NewMonitor(h.TargetStakeCalculator, &types.LiquidityMonitoringParameters{
		TriggeringRatio: .5,
	})
	h.AuctionState.EXPECT().InAuction().Return(false).Times(len(tests))
	h.AuctionState.EXPECT().ExpiresAt().Times(len(tests)).Return(nil)
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			if test.auctionShouldStart {
				h.AuctionState.EXPECT().StartLiquidityAuction(now, gomock.Any()).Times(1)
			}
			var trades []*types.Trade = nil
			var rf types.RiskFactor = types.RiskFactor{}
			var markPrice uint64 = 100
			h.TargetStakeCalculator.EXPECT().GetTheoreticalTargetStake(rf, now, markPrice, trades).Return(test.target)
			mon.CheckLiquidity(h.AuctionState, now, test.current, trades, rf, markPrice, test.bestStaticBidVolume, test.bestStaticAskVolume)
		})
	}
}
