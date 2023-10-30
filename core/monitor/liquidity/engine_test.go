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

package liquidity_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/monitor/liquidity"
	"code.vegaprotocol.io/vega/core/monitor/liquidity/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/golang/mock/gomock"
)

type testHarness struct {
	AuctionState          *mocks.MockAuctionState
	TargetStakeCalculator *mocks.MockTargetStakeCalculator
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()
	ctrl := gomock.NewController(t)
	return &testHarness{
		AuctionState:          mocks.NewMockAuctionState(ctrl),
		TargetStakeCalculator: mocks.NewMockTargetStakeCalculator(ctrl),
	}
}

func (h *testHarness) WhenInOpeningAuction() *testHarness {
	h.AuctionState.EXPECT().IsLiquidityAuction().AnyTimes().Return(false)
	h.AuctionState.EXPECT().IsLiquidityExtension().AnyTimes().Return(false)
	h.AuctionState.EXPECT().IsOpeningAuction().AnyTimes().Return(true)
	return h
}

func (h *testHarness) WhenInLiquidityAuction(v bool) *testHarness {
	h.AuctionState.EXPECT().IsLiquidityAuction().AnyTimes().Return(v)
	h.AuctionState.EXPECT().IsLiquidityExtension().AnyTimes().Return(false)
	h.AuctionState.EXPECT().IsOpeningAuction().AnyTimes().Return(false)
	return h
}

func TestEngineWhenInLiquidityAuction(t *testing.T) {
	now := time.Now()

	tests := []struct {
		desc string
		// when
		current             *num.Uint
		target              *num.Uint
		bestStaticBidVolume uint64
		bestStaticAskVolume uint64
		// expect
		auctionShouldEnd bool
	}{
		{"Current >  Target", num.NewUint(20), num.NewUint(15), 1, 1, true},
		{"Current == Target", num.NewUint(15), num.NewUint(15), 1, 1, true},
		{"Current <  Target", num.NewUint(14), num.NewUint(15), 1, 1, false},
		{"Current >  Target, no best bid", num.NewUint(20), num.NewUint(15), 0, 1, false},
		{"Current == Target, no best ask", num.NewUint(15), num.NewUint(15), 1, 0, false},
		{"Current == Target, no best bid and ask", num.NewUint(15), num.NewUint(15), 0, 0, false},
	}

	h := newTestHarness(t).WhenInLiquidityAuction(true)
	exp := now.Add(-1 * time.Second)
	keep := now.Add(time.Second)
	mon := liquidity.NewMonitor(h.TargetStakeCalculator, &types.LiquidityMonitoringParameters{
		TriggeringRatio: num.DecimalFromFloat(.7),
	})
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			var trades []*types.Trade
			rf := types.RiskFactor{}
			markPrice := num.NewUint(100)
			if test.auctionShouldEnd {
				h.AuctionState.EXPECT().SetReadyToLeave().Times(1)
				h.AuctionState.EXPECT().ExpiresAt().Times(1).Return(&exp)
				h.TargetStakeCalculator.EXPECT().GetTheoreticalTargetStake(rf, now, markPrice.Clone(), trades).Times(1).Return(test.target)
			} else {
				h.AuctionState.EXPECT().ExpiresAt().Times(1).Return(&keep)
			}

			mon.CheckLiquidity(h.AuctionState, now, test.current, trades, rf, markPrice.Clone(), test.bestStaticBidVolume, test.bestStaticAskVolume, true)
		})
	}
}

func TestEngineWhenNotInLiquidityAuction(t *testing.T) {
	now := time.Now()

	tests := []struct {
		desc string
		// when
		current             *num.Uint
		target              *num.Uint
		bestStaticBidVolume uint64
		bestStaticAskVolume uint64
		// expect
		auctionTrigger types.AuctionTrigger
	}{
		{"Current <  (Target * c1)", num.NewUint(10), num.NewUint(30), 1, 1, types.AuctionTriggerLiquidityTargetNotMet},
		{"Current >  (Target * c1)", num.NewUint(15), num.NewUint(15), 1, 1, types.AuctionTriggerUnspecified},
		{"Current == (Target * c1)", num.NewUint(10), num.NewUint(20), 1, 1, types.AuctionTriggerUnspecified},
	}

	h := newTestHarness(t).WhenInLiquidityAuction(false)
	mon := liquidity.NewMonitor(h.TargetStakeCalculator, &types.LiquidityMonitoringParameters{
		TriggeringRatio: num.DecimalFromFloat(.5),
	})
	h.AuctionState.EXPECT().ExpiresAt().Times(len(tests)).Return(nil)
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			if test.auctionTrigger == types.AuctionTriggerLiquidityTargetNotMet {
				h.AuctionState.EXPECT().StartLiquidityAuctionUnmetTarget(now, gomock.Any()).Times(1)
			}
			var trades []*types.Trade
			rf := types.RiskFactor{}
			markPrice := num.NewUint(100)
			h.TargetStakeCalculator.EXPECT().GetTheoreticalTargetStake(rf, now, markPrice.Clone(), trades).Return(test.target)
			mon.CheckLiquidity(h.AuctionState, now, test.current, trades, rf, markPrice.Clone(), test.bestStaticBidVolume, test.bestStaticAskVolume, true)
		})
	}
}

func TestEngineInOpeningAuction(t *testing.T) {
	// these are the same tests as above (not in liq auction), but instead of start liquidity auction
	// we expect the opening auction to be extended
	now := time.Now()

	tests := []struct {
		desc string
		// when
		current             *num.Uint
		target              *num.Uint
		bestStaticBidVolume uint64
		bestStaticAskVolume uint64
		// expect
		auctionTrigger types.AuctionTrigger
	}{
		{"Current <  (Target)", num.NewUint(10), num.NewUint(30), 1, 1, types.AuctionTriggerLiquidityTargetNotMet},
		{"Current >  (Target)", num.NewUint(15), num.NewUint(15), 1, 1, types.AuctionTriggerUnspecified},
		{"Current == (Target * C1)", num.NewUint(10), num.NewUint(20), 1, 1, types.AuctionTriggerLiquidityTargetNotMet},
		{"Current == (Target)", num.NewUint(20), num.NewUint(20), 1, 1, types.AuctionTriggerUnspecified},
	}

	h := newTestHarness(t).WhenInOpeningAuction()
	mon := liquidity.NewMonitor(h.TargetStakeCalculator, &types.LiquidityMonitoringParameters{
		TriggeringRatio: num.DecimalFromFloat(.5),
	})
	h.AuctionState.EXPECT().ExpiresAt().Times(len(tests)).Return(nil)
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			if test.auctionTrigger == types.AuctionTriggerLiquidityTargetNotMet {
				h.AuctionState.EXPECT().ExtendAuctionLiquidityUnmetTarget(gomock.Any()).Times(1)
			} else {
				// opening auciton is flagged as ready to leave
				h.AuctionState.EXPECT().SetReadyToLeave().Times(1)
			}
			var trades []*types.Trade
			rf := types.RiskFactor{}
			markPrice := num.NewUint(100)
			h.TargetStakeCalculator.EXPECT().GetTheoreticalTargetStake(rf, now, markPrice.Clone(), trades).Return(test.target)
			mon.CheckLiquidity(h.AuctionState, now, test.current, trades, rf, markPrice.Clone(), test.bestStaticBidVolume, test.bestStaticAskVolume, true)
		})
	}
}

func TestEngineAfterParametersUpdate(t *testing.T) {
	h := newTestHarness(t).WhenInLiquidityAuction(false)

	now := time.Now()
	target := num.NewUint(100)
	bestStaticBidVolume := uint64(1)
	bestStaticAskVolume := uint64(1)
	var trades []*types.Trade
	rf := types.RiskFactor{}
	markPrice := num.NewUint(100)
	params := &types.LiquidityMonitoringParameters{
		TriggeringRatio:  num.DecimalFromFloat(.5),
		AuctionExtension: 40,
	}

	mon := liquidity.NewMonitor(h.TargetStakeCalculator, params)

	expiresAt := now.Add(-24 * time.Hour)
	h.AuctionState.EXPECT().ExpiresAt().Times(1).Return(&expiresAt)
	h.AuctionState.EXPECT().ExtendAuctionLiquidityUnmetTarget(types.AuctionDuration{
		Duration: params.AuctionExtension,
	}).Times(1)
	h.TargetStakeCalculator.EXPECT().GetTheoreticalTargetStake(rf, now, markPrice.Clone(), trades).Return(target)

	mon.CheckLiquidity(h.AuctionState, now, num.NewUint(40), trades, rf, markPrice.Clone(), bestStaticBidVolume, bestStaticAskVolume, true)

	updatedParams := &types.LiquidityMonitoringParameters{
		TriggeringRatio:  num.DecimalFromFloat(.8),
		AuctionExtension: 80,
	}

	mon.UpdateParameters(updatedParams)

	h.AuctionState.EXPECT().ExpiresAt().Times(1).Return(&expiresAt)
	// Verify the auction extension is called with update parameters.
	h.AuctionState.EXPECT().ExtendAuctionLiquidityUnmetTarget(types.AuctionDuration{
		Duration: updatedParams.AuctionExtension,
	}).Times(1)

	h.TargetStakeCalculator.EXPECT().GetTheoreticalTargetStake(rf, now, markPrice.Clone(), trades).Return(target)
	// Higher current stake to test the updated Triggering Ratio is reached.
	mon.CheckLiquidity(h.AuctionState, now, num.NewUint(70), nil, rf, markPrice.Clone(), bestStaticBidVolume, bestStaticAskVolume, true)
}

func TestEngineAfterParametersUpdateWithAuctionExtension0(t *testing.T) {
	h := newTestHarness(t).WhenInLiquidityAuction(false)

	now := time.Now()
	target := num.NewUint(100)
	bestStaticBidVolume := uint64(1)
	bestStaticAskVolume := uint64(1)
	var trades []*types.Trade
	rf := types.RiskFactor{}
	markPrice := num.NewUint(100)
	params := &types.LiquidityMonitoringParameters{
		TriggeringRatio:  num.DecimalFromFloat(.5),
		AuctionExtension: 0,
	}

	mon := liquidity.NewMonitor(h.TargetStakeCalculator, params)

	expiresAt := now.Add(-24 * time.Hour)
	h.AuctionState.EXPECT().ExpiresAt().Times(1).Return(&expiresAt)
	h.AuctionState.EXPECT().ExtendAuctionLiquidityUnmetTarget(types.AuctionDuration{
		Duration: 1, // to test the patch.
	}).Times(1)
	h.TargetStakeCalculator.EXPECT().GetTheoreticalTargetStake(rf, now, markPrice.Clone(), trades).Return(target)

	mon.CheckLiquidity(h.AuctionState, now, num.NewUint(40), trades, rf, markPrice.Clone(), bestStaticBidVolume, bestStaticAskVolume, true)

	updatedParams := &types.LiquidityMonitoringParameters{
		TriggeringRatio:  num.DecimalFromFloat(.8),
		AuctionExtension: 0,
	}

	mon.UpdateParameters(updatedParams)

	h.AuctionState.EXPECT().ExpiresAt().Times(1).Return(&expiresAt)
	// Verify the auction extension is called with update parameters.
	h.AuctionState.EXPECT().ExtendAuctionLiquidityUnmetTarget(types.AuctionDuration{
		Duration: 1, // to test the patch.
	}).Times(1)

	h.TargetStakeCalculator.EXPECT().GetTheoreticalTargetStake(rf, now, markPrice.Clone(), trades).Return(target)
	// Higher current stake to test the updated Triggering Ratio is reached.
	mon.CheckLiquidity(h.AuctionState, now, num.NewUint(70), nil, rf, markPrice.Clone(), bestStaticBidVolume, bestStaticAskVolume, true)
}
