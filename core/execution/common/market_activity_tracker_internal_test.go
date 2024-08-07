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

package common

import (
	"context"
	"errors"
	"testing"
	"time"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/execution/common/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	vgproto "code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestExcludePartiesInTeams(t *testing.T) {
	allParties := map[string]struct{}{"p1": {}, "p2": {}, "p3": {}, "p4": {}}
	partiesInTeam := []string{"p1", "p4"}
	remaining := excludePartiesInTeams(allParties, partiesInTeam)
	require.Equal(t, 2, len(remaining))
	_, ok := remaining["p1"]
	require.False(t, ok)
	_, ok = remaining["p4"]
	require.False(t, ok)
	_, ok = remaining["p2"]
	require.True(t, ok)
	_, ok = remaining["p3"]
	require.True(t, ok)
}

func TestSortedK(t *testing.T) {
	expected := []string{"p1", "p2", "p3", "p4", "p5", "p6", "p7"}
	m := map[string]int{"p2": 2, "p5": 5, "p7": 7, "p1": 1, "p4": 4, "p3": 3, "p6": 6}
	for i := 0; i < 10; i++ {
		sorted := sortedK(m)
		require.Equal(t, 7, len(sorted))
		for j := 0; j < len(sorted); j++ {
			require.Equal(t, expected[j], sorted[j])
		}
	}
}

func TestCalcTotalForWindowD(t *testing.T) {
	data := []map[string]num.Decimal{}
	for i := int64(0); i < maxWindowSize; i++ {
		data = append(data, map[string]num.Decimal{})
		data[i]["party1"] = num.DecimalFromInt64(i)
	}
	for idx := 0; idx < len(data); idx++ {
		tot, _ := calcTotalForWindowD("party1", data, maxWindowSize)
		require.Equal(t, num.DecimalFromInt64(4950), tot)
	}

	windowSize := 5
	tot, _ := calcTotalForWindowD("party1", data, windowSize)
	require.Equal(t, num.DecimalFromInt64(485), tot)

	windowSize = 2
	tot, _ = calcTotalForWindowD("party1", data, windowSize)
	require.Equal(t, num.DecimalFromInt64(197), tot)

	tot, _ = calcTotalForWindowD("party2", data, windowSize)
	require.Equal(t, num.DecimalZero(), tot)
}

func TestGetTotalFees(t *testing.T) {
	fd := make([]*num.Uint, maxWindowSize)
	for i := uint64(5); i < maxWindowSize; i++ {
		fd[i] = num.NewUint(i)
	}

	windowSize := 5
	require.Equal(t, num.DecimalFromInt64(485), getTotalFees(fd, windowSize))

	windowSize = 1
	require.Equal(t, num.DecimalFromInt64(99), getTotalFees(fd, windowSize))
}

func TestGetFees(t *testing.T) {
	feeHistory := []map[string]*num.Uint{{"p1": num.NewUint(100)}, {"p1": num.NewUint(97), "p2": num.NewUint(200)}}
	windowSize := 5
	// party has no fee data
	tot, ok := getFees(feeHistory, "p3", windowSize)
	require.Equal(t, "0", tot.String())
	require.False(t, ok)
	// party1 has 197 in window (100, 97)
	tot, ok = getFees(feeHistory, "p1", windowSize)
	require.Equal(t, num.DecimalFromInt64(197), tot)
	require.True(t, ok)
	// party2 has 100 in window (0, 200)
	tot, ok = getFees(feeHistory, "p2", windowSize)
	require.Equal(t, num.DecimalFromInt64(200), tot)
	require.True(t, ok)
}

func getDefaultTracker(t *testing.T) *marketTracker {
	t.Helper()
	return &marketTracker{
		asset:                    "asset",
		proposer:                 "proposer",
		proposersPaid:            map[string]struct{}{},
		readyToDelete:            false,
		valueTraded:              num.UintZero(),
		makerFeesReceived:        map[string]*num.Uint{},
		makerFeesPaid:            map[string]*num.Uint{},
		lpFees:                   map[string]*num.Uint{},
		totalMakerFeesReceived:   num.UintZero(),
		totalMakerFeesPaid:       num.UintZero(),
		totalLpFees:              num.UintZero(),
		twPosition:               map[string]*twPosition{},
		partyM2M:                 map[string]num.Decimal{},
		partyRealisedReturn:      map[string]num.Decimal{},
		twNotional:               map[string]*twNotional{},
		epochPartyM2M:            []map[string]num.Decimal{},
		epochPartyRealisedReturn: []map[string]num.Decimal{},
	}
}

func TestGetRelativeReturnMetricTotal(t *testing.T) {
	tracker := getDefaultTracker(t)

	for i := int64(0); i < maxWindowSize; i++ {
		d := num.DecimalFromInt64(i)
		m := map[string]num.Decimal{}
		m["p1"] = d
		tracker.epochPartyM2M = append(tracker.epochPartyM2M, m)
	}
	// nothing for party2
	tot, ok := tracker.getRelativeReturnMetricTotal("p2", 5)
	require.False(t, ok)
	require.Equal(t, num.DecimalZero(), tot)

	tot, ok = tracker.getRelativeReturnMetricTotal("p1", 5)
	require.True(t, ok)
	require.Equal(t, num.DecimalFromInt64(485), tot)
}

func TestGetPositionMetricTotal(t *testing.T) {
	tracker := getDefaultTracker(t)
	position := &twPosition{position: 0, t: time.Now(), currentEpochTWPosition: 42}
	tracker.twPosition["p1"] = position

	for i := uint64(0); i < maxWindowSize; i++ {
		tracker.epochTimeWeightedPosition = append(tracker.epochTimeWeightedPosition, map[string]uint64{"p1": i})
	}

	// nothing for party2
	tot, ok := tracker.getPositionMetricTotal("p2", 5)
	require.False(t, ok)
	require.Equal(t, uint64(0), tot)
	// 99+98+97+96+95 for party1
	tot, ok = tracker.getPositionMetricTotal("p1", 5)
	require.True(t, ok)
	require.Equal(t, uint64(485), tot)
}

func TestReturns(t *testing.T) {
	tracker := getDefaultTracker(t)

	tracker.recordM2M("p1", num.DecimalFromInt64(100))
	tracker.recordPosition("p1", 10, num.DecimalOne(), time.Unix(1, 0), time.Unix(0, 0))
	tracker.recordM2M("p1", num.DecimalFromInt64(-200))
	tracker.recordM2M("p2", num.DecimalFromInt64(-100))
	tracker.recordPosition("p2", 20, num.DecimalOne(), time.Unix(1, 0), time.Unix(0, 0))
	tracker.recordM2M("p3", num.DecimalFromInt64(200))
	tracker.recordPosition("p3", 20, num.DecimalOne(), time.Unix(1, 0), time.Unix(0, 0))

	tracker.processPositionEndOfEpoch(time.Unix(0, 0), time.Unix(2, 0))
	tracker.processM2MEndOfEpoch()

	ret1, ok := tracker.getReturns("p1", 1)
	require.True(t, ok)
	require.Equal(t, 1, len(ret1))

	ret1, ok = tracker.getReturns("p1", 100)
	require.True(t, ok)
	require.Equal(t, 100, len(ret1))

	// -100/5 = -20
	require.Equal(t, "-20", ret1[0].String())
	ret2, ok := tracker.getReturns("p2", 1)
	require.True(t, ok)
	require.Equal(t, 1, len(ret2))
	// -100/10 = -10
	require.Equal(t, "-10", ret2[0].String())
	ret3, ok := tracker.getReturns("p3", 1)
	require.True(t, ok)
	require.Equal(t, 1, len(ret3))
	// 200/10 = 20
	require.Equal(t, "20", ret3[0].String())
	_, ok = tracker.getReturns("p4", 1)
	require.False(t, ok)
}

func TestRealisedReturns(t *testing.T) {
	tracker := getDefaultTracker(t)

	tracker.recordFundingPayment("p1", num.DecimalFromInt64(100))
	tracker.recordRealisedPosition("p1", num.DecimalFromInt64(-50))
	tracker.recordFundingPayment("p1", num.DecimalFromInt64(-200))
	tracker.recordRealisedPosition("p1", num.DecimalFromInt64(20))
	tracker.recordFundingPayment("p2", num.DecimalFromInt64(-100))
	tracker.recordRealisedPosition("p2", num.DecimalFromInt64(-10))
	tracker.recordRealisedPosition("p2", num.DecimalFromInt64(20))
	tracker.recordFundingPayment("p3", num.DecimalFromInt64(200))

	tracker.processPartyRealisedReturnOfEpoch()

	ret1, ok := tracker.getRealisedReturnMetricTotal("p1", 1)
	require.Equal(t, "-130", ret1.String())
	require.True(t, ok)
	ret2, ok := tracker.getRealisedReturnMetricTotal("p2", 1)
	require.Equal(t, "-90", ret2.String())
	require.True(t, ok)
	ret3, ok := tracker.getRealisedReturnMetricTotal("p3", 1)
	require.Equal(t, "200", ret3.String())
	require.True(t, ok)

	tracker.recordFundingPayment("p1", num.DecimalFromInt64(-30))
	tracker.recordRealisedPosition("p2", num.DecimalFromInt64(70))
	tracker.recordRealisedPosition("p2", num.DecimalFromInt64(80))
	tracker.recordRealisedPosition("p3", num.DecimalFromInt64(-50))

	tracker.processPartyRealisedReturnOfEpoch()
	ret1, ok = tracker.getRealisedReturnMetricTotal("p1", 1)
	require.Equal(t, "-30", ret1.String())
	require.True(t, ok)
	ret2, ok = tracker.getRealisedReturnMetricTotal("p2", 1)
	require.Equal(t, "150", ret2.String())
	require.True(t, ok)
	ret3, ok = tracker.getRealisedReturnMetricTotal("p3", 1)
	require.Equal(t, "-50", ret3.String())
	require.True(t, ok)

	ret1, ok = tracker.getRealisedReturnMetricTotal("p1", 2)
	require.Equal(t, "-160", ret1.String())
	require.True(t, ok)
	ret2, ok = tracker.getRealisedReturnMetricTotal("p2", 2)
	require.Equal(t, "60", ret2.String())
	require.True(t, ok)
	ret3, ok = tracker.getRealisedReturnMetricTotal("p3", 2)
	require.Equal(t, "150", ret3.String())
	require.True(t, ok)
}

func TestPositions(t *testing.T) {
	tracker := getDefaultTracker(t)
	// epoch 1
	tracker.recordPosition("p1", 10, num.DecimalOne(), time.Unix(5, 0), time.Unix(0, 0))
	// 0 * ( 10000000 - 6666666 ) + ( 100000000 * 6666666 ) / 10000000= 66666660
	tracker.recordPosition("p1", 20, num.DecimalOne(), time.Unix(15, 0), time.Unix(0, 0))
	// 66666660 * ( 10000000 - 6666666 ) + ( 200000000 * 6666666 )/ 10000000 = 155555544
	tracker.recordPosition("p1", 30, num.DecimalOne(), time.Unix(45, 0), time.Unix(0, 0))
	// 155555544 * ( 10000000 - 2500000 ) + ( 300000000 * 2500000 ) / 10000000 = 191666658
	tracker.processPositionEndOfEpoch(time.Unix(0, 0), time.Unix(60, 0))
	tot, ok := tracker.getPositionMetricTotal("p1", 1)
	require.True(t, ok)
	require.Equal(t, uint64(191666658), tot)

	// epoch 2
	// 191666658 * ( 10000000 - 10000000 ) + ( 300000000 * 10000000 ) / 10000000 = 300000000
	tracker.recordPosition("p1", 10, num.DecimalOne(), time.Unix(90, 0), time.Unix(60, 0))
	// 300000000 * ( 10000000 - 5000000 ) + ( 100000000 * 5000000 ) / 10000000 = 200000000
	tracker.processPositionEndOfEpoch(time.Unix(60, 0), time.Unix(120, 0))
	tot, ok = tracker.getPositionMetricTotal("p1", 1)
	require.Equal(t, uint64(200000000), tot)
	require.True(t, ok)
	tot, ok = tracker.getPositionMetricTotal("p1", 2)
	require.Equal(t, uint64(391666658), tot)
	require.True(t, ok)

	// epoch 3
	// no position changes over the epoch
	// 200000000 * ( 10000000 - 10000000 ) + ( 100000000 * 10000000 ) / 10000000 = 100000000
	tracker.processPositionEndOfEpoch(time.Unix(120, 0), time.Unix(180, 0))
	tot, ok = tracker.getPositionMetricTotal("p1", 1)
	require.Equal(t, uint64(100000000), tot)
	require.True(t, ok)
	tot, ok = tracker.getPositionMetricTotal("p1", 2)
	require.Equal(t, uint64(300000000), tot)
	require.True(t, ok)
	tot, ok = tracker.getPositionMetricTotal("p1", 3)
	require.Equal(t, uint64(491666658), tot)
	require.True(t, ok)
}

func TestAverageNotional(t *testing.T) {
	tracker := getDefaultTracker(t)
	// epoch 1
	tracker.recordNotional("p1", num.NewUint(50), time.Unix(5, 0), time.Unix(0, 0))
	require.Equal(t, "0", tracker.twNotional["p1"].currentEpochTWNotional.String())

	// (( 0 * 3333334 ) + ( 50 * 6666666 )) / 10000000 = 33
	tracker.recordNotional("p1", num.NewUint(200), time.Unix(15, 0), time.Unix(0, 0))
	require.Equal(t, "33", tracker.twNotional["p1"].currentEpochTWNotional.String())

	// (( 33 * 5000000 ) + ( 200 * 5000000 )) / 10000000 = 116
	tracker.recordNotional("p1", num.NewUint(600), time.Unix(30, 0), time.Unix(0, 0))
	require.Equal(t, "116", tracker.twNotional["p1"].currentEpochTWNotional.String())

	// (( 116 * 5000000 ) + ( 600 * 5000000 )) / 10000000 = 358
	tracker.processNotionalEndOfEpoch(time.Unix(0, 0), time.Unix(60, 0))
	require.Equal(t, "358", tracker.twNotional["p1"].currentEpochTWNotional.String())
	require.Equal(t, "358", tracker.epochTimeWeightedNotional[len(tracker.epochTimeWeightedNotional)-1]["p1"].String())

	// epoch 2
	// (( 358 * 0 ) + ( 600 * 10000000 )) / 10000000 = 600
	tracker.recordNotional("p1", num.NewUint(300), time.Unix(90, 0), time.Unix(60, 0))
	require.Equal(t, "600", tracker.twNotional["p1"].currentEpochTWNotional.String())

	// (( 600 * 5000000 ) + ( 300 * 5000000 )) / 10000000 = 450
	tracker.processNotionalEndOfEpoch(time.Unix(60, 0), time.Unix(120, 0))
	require.Equal(t, "450", tracker.twNotional["p1"].currentEpochTWNotional.String())
	require.Equal(t, "450", tracker.epochTimeWeightedNotional[len(tracker.epochTimeWeightedNotional)-1]["p1"].String())

	// epoch 3
	// no position changes over the epoch
	// (( 450 * 0 ) + ( 300 * 10000000 )) / 10000000 = 300
	tracker.processNotionalEndOfEpoch(time.Unix(120, 0), time.Unix(180, 0))
	require.Equal(t, "300", tracker.twNotional["p1"].currentEpochTWNotional.String())
	require.Equal(t, "300", tracker.epochTimeWeightedNotional[len(tracker.epochTimeWeightedNotional)-1]["p1"].String())
}

func TestCalculateMetricForIndividualsAveNotional(t *testing.T) {
	ctx := context.Background()
	epochService := &DummyEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	collateralService := mocks.NewMockCollateral(ctrl)
	tracker := NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker, collateralService)
	epochService.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)
	tracker.SetEligibilityChecker(&DummyEligibilityChecker{})
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Time{}})

	// add some markets for 2 different assets
	tracker.MarketProposed("a1", "m1", "z1")
	tracker.MarketProposed("a1", "m2", "z2")
	tracker.MarketProposed("a1", "m3", "z3")

	// add some markets for 2 different assets
	tracker.MarketProposed("a1", "m1", "z1")
	tracker.MarketProposed("a1", "m2", "z2")
	tracker.MarketProposed("a1", "m3", "z3")

	// record some values for all metrics
	tracker.RecordPosition("a1", "p1", "m1", 10, num.NewUint(1), num.DecimalOne(), time.Unix(5, 0))
	tracker.RecordPosition("a1", "p1", "m2", 20, num.NewUint(2), num.DecimalOne(), time.Unix(20, 0))
	tracker.RecordPosition("a1", "p1", "m3", 30, num.NewUint(3), num.DecimalOne(), time.Unix(30, 0))
	tracker.RecordPosition("a1", "p2", "m1", 100, num.NewUint(10), num.DecimalOne(), time.Unix(15, 0))
	tracker.RecordPosition("a1", "p2", "m2", 200, num.NewUint(20), num.DecimalOne(), time.Unix(25, 0))
	tracker.RecordPosition("a1", "p2", "m3", 300, num.NewUint(30), num.DecimalOne(), time.Unix(45, 0))

	gameID := "game123"
	// end epoch1
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(0, 0), EndTime: time.Unix(60, 0)})

	// get metrics for market m1 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.NewUint(0), nil).Times(2)
	metrics := tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, num.UintZero(), num.UintZero(), 1, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0.0000009", metrics[0].Score.String())
	require.Equal(t, "0.000075", metrics[1].Score.String())

	// get metrics for market m2 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.NewUint(0), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, num.UintZero(), num.UintZero(), 1, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0.0000026", metrics[0].Score.String())
	require.Equal(t, "0.0002333", metrics[1].Score.String())

	// get metrics for market m3 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.NewUint(0), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, num.UintZero(), num.UintZero(), 1, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0.0000045", metrics[0].Score.String())
	require.Equal(t, "0.000225", metrics[1].Score.String())

	// get metrics for market m1,m2 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.NewUint(0), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, num.UintZero(), num.UintZero(), 1, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0.0000035", metrics[0].Score.String())
	require.Equal(t, "0.0003083", metrics[1].Score.String())

	// get metrics for all market window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.NewUint(0), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, num.UintZero(), num.UintZero(), 1, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0.000008", metrics[0].Score.String())
	require.Equal(t, "0.0005333", metrics[1].Score.String())

	// start epoch2
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(60, 0)})
	tracker.RecordPosition("a1", "p1", "m1", 20, num.NewUint(5), num.DecimalOne(), time.Unix(90, 0))
	tracker.RecordPosition("a1", "p2", "m2", 10, num.NewUint(10), num.DecimalOne(), time.Unix(75, 0))
	tracker.RecordPosition("a2", "p1", "m3", 20, num.NewUint(5), num.DecimalOne(), time.Unix(90, 0))
	tracker.RecordPosition("a2", "p2", "m2", 10, num.NewUint(10), num.DecimalOne(), time.Unix(75, 0))
	// end epoch2
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(60, 0), EndTime: time.Unix(120, 0)})

	// get metrics for market m1 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.NewUint(0), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, num.UintZero(), num.UintZero(), 1, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0.0000055", metrics[0].Score.String())
	require.Equal(t, "0.0001", metrics[1].Score.String())

	// get metrics for market m2 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.NewUint(0), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, num.UintZero(), num.UintZero(), 1, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0.000004", metrics[0].Score.String())
	require.Equal(t, "0.0001075", metrics[1].Score.String())

	// get metrics for market m2 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.NewUint(0), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, num.UintZero(), num.UintZero(), 1, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0.000009", metrics[0].Score.String())
	require.Equal(t, "0.0009", metrics[1].Score.String())

	// get metrics for market m1,m2 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.NewUint(0), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, num.UintZero(), num.UintZero(), 1, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0.0000095", metrics[0].Score.String())
	require.Equal(t, "0.0002075", metrics[1].Score.String())

	// get metrics for all market window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.NewUint(0), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, num.UintZero(), num.UintZero(), 1, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0.0000185", metrics[0].Score.String())
	require.Equal(t, "0.0011075", metrics[1].Score.String())

	// calc with window size = 2
	// get metrics for market m1 with window size=2
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.NewUint(0), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, num.UintZero(), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0.0000032", metrics[0].Score.String())
	require.Equal(t, "0.0000875", metrics[1].Score.String())

	// get metrics for market m2 with window size=2
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.NewUint(0), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, num.UintZero(), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0.0000033", metrics[0].Score.String())
	require.Equal(t, "0.0001704", metrics[1].Score.String())

	// get metrics for market m2 with window size=2
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.NewUint(0), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, num.UintZero(), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0.00000675", metrics[0].Score.String())
	require.Equal(t, "0.0005625", metrics[1].Score.String())

	// get metrics for market m1,m2 with window size=2
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.NewUint(0), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, num.UintZero(), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0.0000065", metrics[0].Score.String())
	require.Equal(t, "0.0002579", metrics[1].Score.String())

	// get metrics for all market window size=2
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.NewUint(0), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, num.UintZero(), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0.00001325", metrics[0].Score.String())
	require.Equal(t, "0.0008204", metrics[1].Score.String())

	// now make p2 not eligible via not having sufficient governance token
	balanceChecker.EXPECT().GetAvailableBalance("p1").Return(num.NewUint(2), nil).Times(1)
	balanceChecker.EXPECT().GetAvailableBalance("p2").Return(nil, errors.New("some error")).Times(1)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, num.NewUint(1), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, false, metrics[1].IsEligible)
	require.Equal(t, "0", metrics[1].StakingBalance.String())

	// repeat now p2 has balance just not enough
	balanceChecker.EXPECT().GetAvailableBalance("p1").Return(num.NewUint(2), nil).Times(1)
	balanceChecker.EXPECT().GetAvailableBalance("p2").Return(num.NewUint(1), nil).Times(1)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, num.NewUint(2), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "2", metrics[0].StakingBalance.String())
	require.Equal(t, true, metrics[0].IsEligible)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, false, metrics[1].IsEligible)
	require.Equal(t, "1", metrics[1].StakingBalance.String())
}

func TestCalculateMetricForPartyAveNotional(t *testing.T) {
	epochService := &DummyEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	collateralService := mocks.NewMockCollateral(ctrl)
	tracker := NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker, collateralService)
	epochService.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)
	tracker.SetEligibilityChecker(&DummyEligibilityChecker{})
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Time{}})

	// add some markets for 2 different assets
	tracker.MarketProposed("a1", "m1", "z1")
	tracker.MarketProposed("a1", "m2", "z2")
	tracker.MarketProposed("a1", "m3", "z3")

	// record some values for all metrics
	tracker.RecordPosition("a1", "p1", "m1", 10, num.NewUint(1), num.DecimalOne(), time.Unix(5, 0))
	tracker.RecordPosition("a1", "p1", "m2", 20, num.NewUint(2), num.DecimalOne(), time.Unix(20, 0))
	tracker.RecordPosition("a1", "p1", "m3", 30, num.NewUint(3), num.DecimalOne(), time.Unix(30, 0))
	tracker.RecordPosition("a1", "p2", "m1", 100, num.NewUint(10), num.DecimalOne(), time.Unix(15, 0))
	tracker.RecordPosition("a1", "p2", "m2", 200, num.NewUint(20), num.DecimalOne(), time.Unix(25, 0))
	tracker.RecordPosition("a1", "p2", "m3", 300, num.NewUint(30), num.DecimalOne(), time.Unix(45, 0))

	// end epoch1
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(0, 0), EndTime: time.Unix(60, 0)})

	// calculate metric for p1 with scope=[m1] for window size=1
	// 0*(1-0.9166666666666667)+10*0.9166666666666667 = 9.1666666667
	score, _ := tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.0000009", score.String())
	// calculate metric for p1 with scope=[m2] for window size=1
	// 0*(1-0.6666666666666667)+20*0.6666666666666667 = 13.3333333333
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.0000026", score.String())
	// calculate metric for p1 with scope=[m3] for window size=1
	// 0*(1-0.5)+30*0.5 = 15
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.0000045", score.String())
	// calculate metric for p1 with scope=[m1, m2] for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.0000035", score.String())
	// calculate metric for p1 with no scope for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.000008", score.String())

	// calculate metric for p2 with scope=[m1] for window size=1
	// 0*(1-0.75)+100*0.75 = 75
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.000075", score.String())
	// calculate metric for p2 with scope=[m2] for window size=1
	// 0*(1-0.5833333333333333)+200*0.5833333333333333
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.0002333", score.String())
	// calculate metric for p2 with scope=[m3] for window size=1
	// 0*(1-0.25)+300*0.25
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.000225", score.String())
	// calculate metric for p2 with scope=[m1, m3] for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.0003", score.String())
	// calculate metric for p2 with no scope for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.0005333", score.String())

	// start epoch2
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(60, 0)})
	tracker.RecordPosition("a1", "p1", "m1", 20, num.NewUint(5), num.DecimalOne(), time.Unix(90, 0))
	tracker.RecordPosition("a1", "p2", "m2", 10, num.NewUint(10), num.DecimalOne(), time.Unix(75, 0))
	tracker.RecordPosition("a2", "p1", "m3", 20, num.NewUint(5), num.DecimalOne(), time.Unix(90, 0))
	tracker.RecordPosition("a2", "p2", "m2", 10, num.NewUint(10), num.DecimalOne(), time.Unix(75, 0))
	// end epoch2
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(60, 0), EndTime: time.Unix(120, 0)})

	// calculate metric for p1 with scope=[m1] for window size=1
	// 10*(1-0.5)+20*0.5
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.0000055", score.String())
	// calculate metric for p1 with scope=[m2] for window size=1
	// 13.333333333333334*(1-1)+20*1
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.000004", score.String())
	// calculate metric for p1 with scope=[m3] for window size=1
	// 15*(1-1)+30*1 = 30
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.000009", score.String())
	// calculate metric for p1 with scope=[m1, m3] for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.0000095", score.String())
	// calculate metric for p1 with no scope for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.0000185", score.String())

	// calculate metric for p2 with scope=[m1] for window size=1
	// 75*(1-1)+100*1
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.0001", score.String())
	// calculate metric for p1 with scope=[m2] for window size=1
	// 200*(1-0.75)+10*0.75
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.0001075", score.String())
	// calculate metric for p1 with scope=[m3] for window size=1
	// 75*(1-1)+300*1
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.0009", score.String())
	// calculate metric for p1 with scope=[m1, m3] for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.0002075", score.String())
	// calculate metric for p1 with no scope for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.0011075", score.String())

	// now calculate for window size=2

	// calculate metric for p1 with scope=[m1] for window size=2
	// (15 + 9.166666666666667)/2
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 2)
	require.Equal(t, "0.0000032", score.String())
	// calculate metric for p1 with scope=[m2] for window size=2
	// (13.333333333333334" + 20)/2
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 2)
	require.Equal(t, "0.0000033", score.String())
	// calculate metric for p1 with scope=[m3] for window size=2
	// (15 + 30)/2
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 2)
	require.Equal(t, "0.00000675", score.String())
	// calculate metric for p1 with scope=[m1, m3] for window size=2
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 2)
	require.Equal(t, "0.0000065", score.String())
	// calculate metric for p1 with no scope for window size=2
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 2)
	require.Equal(t, "0.00001325", score.String())

	// calculate metric for p2 with scope=[m1] for window size=2
	// (100 + 75)/2
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 2)
	require.Equal(t, "0.0000875", score.String())
	// calculate metric for p2 with scope=[m2] for window size=2
	// (116.66666666666666 + 57.5)/2
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 2)
	require.Equal(t, "0.0001704", score.String())
	// calculate metric for p2 with scope=[m3] for window size=2
	// (300 + 75)/2
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 2)
	require.Equal(t, "0.0005625", score.String())
	// calculate metric for p2 with scope=[m1, m3] for window size=2
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 2)
	require.Equal(t, "0.00065", score.String())
	// calculate metric for p2 with no scope for window size=2
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 2)
	require.Equal(t, "0.0008204", score.String())

	// start epoch3
	epochService.target(context.Background(), types.Epoch{Seq: 3, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(120, 0)})
	// end epoch3
	epochService.target(context.Background(), types.Epoch{Seq: 3, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(120, 0), EndTime: time.Unix(180, 0)})
	// calculate metric for p1 with scope=[m1] for window size=1
	// 15*(1-1)+20*1
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.00001", score.String())
	// calculate metric for p1 with scope=[m2] for window size=1
	// 20*(1-1)+20*1
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.000004", score.String())
	// calculate metric for p1 with scope=[m3] for window size=1
	// 30*(1-1)+30*1 = 30
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.000009", score.String())
	// calculate metric for p1 with scope=[m1, m3] for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.000014", score.String())
	// calculate metric for p1 with no scope for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.000023", score.String())

	// calculate metric for p2 with scope=[m1] for window size=1
	// 100*(1-1)+100*1
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.0001", score.String())
	// calculate metric for p2 with scope=[m2] for window size=1
	// 57.5*(1-1)+10*1
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.00001", score.String())
	// calculate metric for p2 with scope=[m3] for window size=1
	// 300*(1-1)+300*1
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.0009", score.String())
	// calculate metric for p2 with scope=[m1, m3] for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.00011", score.String())
	// calculate metric for p2 with no scope for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 1)
	require.Equal(t, "0.00101", score.String())

	// now calculate for window size=3

	// calculate metric for p1 with scope=[m1] for window size=3
	// (9.166666666666667 + 15 + 20)/3
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 3)
	require.Equal(t, "0.0000054666666667", score.String())
	// calculate metric for p1 with scope=[m2] for window size=3
	// (13.333333333333334" + 20 + 20)/3
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 3)
	require.Equal(t, "0.0000035333333333", score.String())
	// calculate metric for p1 with scope=[m3] for window size=3
	// (15 + 30 + 30)/3
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 3)
	require.Equal(t, "0.0000075", score.String())
	// calculate metric for p1 with scope=[m1, m3] for window size=3
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 3)
	require.Equal(t, "0.000009", score.String())
	// calculate metric for p1 with no scope for window size=3
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 3)
	require.Equal(t, "0.0000165", score.String())

	// calculate metric for p2 with scope=[m1] for window size=3
	// (100 + 75+100)/3
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 3)
	require.Equal(t, "0.0000916666666667", score.String())
	// calculate metric for p2 with scope=[m2] for window size=3
	// (116.66666666666666 + 57.5 + 10)/3
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 3)
	require.Equal(t, "0.0001169333333333", score.String())
	// calculate metric for p2 with scope=[m3] for window size=3
	// (300 + 75 + 300)/3
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 3)
	require.Equal(t, "0.000675", score.String())
	// calculate metric for p2 with scope=[m1, m3] for window size=3
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 3)
	require.Equal(t, "0.0007666666666667", score.String())
	// calculate metric for p2 with no scope for window size=3
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, 3)
	require.Equal(t, "0.0008836", score.String())
}

func TestCalculateMetricForIndividualReturnVolatility(t *testing.T) {
	ctx := context.Background()
	epochService := &DummyEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	collateralService := mocks.NewMockCollateral(ctrl)
	tracker := NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker, collateralService)
	epochService.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)
	tracker.SetEligibilityChecker(&DummyEligibilityChecker{})
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Time{}})

	// add some markets for 2 different assets
	tracker.MarketProposed("a1", "m1", "z1")
	tracker.MarketProposed("a1", "m2", "z2")
	tracker.MarketProposed("a1", "m3", "z3")

	// add some markets for 2 different assets
	tracker.MarketProposed("a1", "m1", "z1")
	tracker.MarketProposed("a1", "m2", "z2")
	tracker.MarketProposed("a1", "m3", "z3")

	// record some values for all metrics
	tracker.RecordPosition("a1", "p1", "m1", 10, num.NewUint(1), num.DecimalOne(), time.Unix(5, 0))
	tracker.RecordPosition("a1", "p1", "m2", 20, num.NewUint(2), num.DecimalOne(), time.Unix(20, 0))
	tracker.RecordPosition("a1", "p1", "m3", 30, num.NewUint(3), num.DecimalOne(), time.Unix(30, 0))
	tracker.RecordPosition("a1", "p2", "m1", 100, num.NewUint(10), num.DecimalOne(), time.Unix(15, 0))
	tracker.RecordPosition("a1", "p2", "m2", 200, num.NewUint(20), num.DecimalOne(), time.Unix(25, 0))
	tracker.RecordPosition("a1", "p2", "m3", 300, num.NewUint(30), num.DecimalOne(), time.Unix(45, 0))
	tracker.RecordPosition("a1", "p3", "m1", 10, num.NewUint(1), num.DecimalOne(), time.Unix(10, 0))
	tracker.RecordPosition("a1", "p3", "m2", 20, num.NewUint(2), num.DecimalOne(), time.Unix(10, 0))
	tracker.RecordPosition("a1", "p3", "m3", 30, num.NewUint(3), num.DecimalOne(), time.Unix(10, 0))

	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(80))
	tracker.RecordM2M("a1", "p2", "m1", num.DecimalFromInt64(20))
	tracker.RecordM2M("a1", "p3", "m1", num.DecimalFromInt64(-100))
	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(10))
	tracker.RecordM2M("a1", "p2", "m1", num.DecimalFromInt64(-10))
	tracker.RecordM2M("a1", "p1", "m2", num.DecimalFromInt64(50))
	tracker.RecordM2M("a1", "p2", "m2", num.DecimalFromInt64(-5))
	tracker.RecordM2M("a1", "p3", "m2", num.DecimalFromInt64(-45))
	tracker.RecordM2M("a1", "p1", "m3", num.DecimalFromInt64(-35))
	tracker.RecordM2M("a1", "p2", "m3", num.DecimalFromInt64(35))

	// end epoch1
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(0, 0), EndTime: time.Unix(60, 0)})

	gameID := "game123"

	// get metrics for market m1 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics := tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.UintZero(), 1, gameID, 1)
	// only one sample (window size=1) variance=0 by definition
	require.Equal(t, 2, len(metrics))
	require.Equal(t, false, metrics[0].IsEligible)
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "0", metrics[0].Score.String())
	require.Equal(t, false, metrics[1].IsEligible)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0", metrics[1].Score.String())

	// get metrics for market m2 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.UintZero(), 1, gameID, 1)
	// only one sample (window size=1) variance=0 by definition
	require.Equal(t, 2, len(metrics))
	require.Equal(t, false, metrics[0].IsEligible)
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "0", metrics[0].Score.String())
	require.Equal(t, false, metrics[1].IsEligible)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0", metrics[1].Score.String())

	// get metrics for market m3 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.UintZero(), 1, gameID, 1)
	// only one sample (window size=1) variance=0 by definition
	require.Equal(t, 2, len(metrics))
	require.Equal(t, false, metrics[0].IsEligible)
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "0", metrics[0].Score.String())
	require.Equal(t, false, metrics[1].IsEligible)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0", metrics[1].Score.String())

	// get metrics for market m1,m2 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.UintZero(), 1, gameID, 1)
	// only one sample (window size=1) variance=0 by definition
	require.Equal(t, 2, len(metrics))
	require.Equal(t, false, metrics[0].IsEligible)
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "0", metrics[0].Score.String())
	require.Equal(t, false, metrics[1].IsEligible)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0", metrics[1].Score.String())

	// get metrics for all market window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.UintZero(), 1, gameID, 1)
	// only one sample (window size=1) variance=0 by definition
	require.Equal(t, 2, len(metrics))
	require.Equal(t, false, metrics[0].IsEligible)
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "0", metrics[0].Score.String())
	require.Equal(t, false, metrics[1].IsEligible)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0", metrics[1].Score.String())

	// // start epoch2
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(60, 0)})
	tracker.RecordPosition("a1", "p1", "m1", 20, num.NewUint(5), num.DecimalOne(), time.Unix(90, 0))
	tracker.RecordPosition("a1", "p2", "m2", 10, num.NewUint(10), num.DecimalOne(), time.Unix(75, 0))
	tracker.RecordPosition("a1", "p3", "m1", 10, num.NewUint(10), num.DecimalOne(), time.Unix(75, 0))
	tracker.RecordPosition("a2", "p1", "m3", 20, num.NewUint(5), num.DecimalOne(), time.Unix(90, 0))
	tracker.RecordPosition("a2", "p2", "m2", 10, num.NewUint(10), num.DecimalOne(), time.Unix(75, 0))

	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(45))
	tracker.RecordM2M("a1", "p3", "m1", num.DecimalFromInt64(-45))
	tracker.RecordM2M("a1", "p1", "m2", num.DecimalFromInt64(-10))
	tracker.RecordM2M("a1", "p2", "m2", num.DecimalFromInt64(10))
	// nothing in m3

	// end epoch2
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(60, 0), EndTime: time.Unix(120, 0)})

	// calc with window size = 2
	// get metrics for market m1 with window size=2
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	// variance(3, 9.8181825322314569) => 11.6219032607065405 => 1/11.6219032607065405 = 0.08604442642
	require.Equal(t, "0.086044426422046", metrics[0].Score.String())
	require.Equal(t, false, metrics[1].IsEligible)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0", metrics[1].Score.String())

	// get metrics for market m2 with window size=2
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "0.2214532481172412", metrics[0].Score.String())
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "85.1257359604949139", metrics[1].Score.String())

	// get metrics for market m3 with window size=2
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, false, metrics[0].IsEligible)
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "0", metrics[0].Score.String())
	require.Equal(t, false, metrics[1].IsEligible)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0", metrics[1].Score.String())

	// get metrics for market m1,m2 with window size=2
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	// variance(2.5, 13.5681829072314944) = 30.6261682169828538 => 0.03265181569
	require.Equal(t, "0.0326518156928779", metrics[0].Score.String())
	// variance(0.1739130434782609,0.0904761880272107) = 0.0017404272118899 => 574.5715725245
	require.Equal(t, "574.5715725244936759", metrics[1].Score.String())

	// get metrics for all market window size=2
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	// variance(2.5, 11.2348495738981611) = 19.0743992696572216 => 0.05242629065
	require.Equal(t, "0.0524262906455334", metrics[0].Score.String())
	// variance(0.1739130434782609,0.5571428546938774) = 0.0367162720510893 => 27.2358805548
	require.Equal(t, "27.2358805547724978", metrics[1].Score.String())

	// now make p2 not eligible via not having sufficient governance token
	balanceChecker.EXPECT().GetAvailableBalance("p1").Return(num.NewUint(2), nil).Times(1)
	balanceChecker.EXPECT().GetAvailableBalance("p2").Return(nil, errors.New("some error")).Times(1)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.NewUint(1), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, true, metrics[0].IsEligible)
	require.Equal(t, false, metrics[1].IsEligible)
	require.Equal(t, "2", metrics[0].StakingBalance.String())
	require.Equal(t, "0", metrics[1].StakingBalance.String())

	// repeat now p2 has balance just not enough
	balanceChecker.EXPECT().GetAvailableBalance("p1").Return(num.NewUint(2), nil).Times(1)
	balanceChecker.EXPECT().GetAvailableBalance("p2").Return(num.NewUint(1), nil).Times(1)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.NewUint(2), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, true, metrics[0].IsEligible)
	require.Equal(t, false, metrics[1].IsEligible)
	require.Equal(t, "2", metrics[0].StakingBalance.String())
	require.Equal(t, "1", metrics[1].StakingBalance.String())
}

func TestCalculateMetricForIndividualsRelativeReturn(t *testing.T) {
	ctx := context.Background()
	epochService := &DummyEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	collateralService := mocks.NewMockCollateral(ctrl)
	tracker := NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker, collateralService)
	epochService.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)
	tracker.SetEligibilityChecker(&DummyEligibilityChecker{})
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Time{}})

	// add some markets for 2 different assets
	tracker.MarketProposed("a1", "m1", "z1")
	tracker.MarketProposed("a1", "m2", "z2")
	tracker.MarketProposed("a1", "m3", "z3")

	// add some markets for 2 different assets
	tracker.MarketProposed("a1", "m1", "z1")
	tracker.MarketProposed("a1", "m2", "z2")
	tracker.MarketProposed("a1", "m3", "z3")

	// record some values for all metrics
	tracker.RecordPosition("a1", "p1", "m1", 10, num.NewUint(1), num.DecimalOne(), time.Unix(5, 0))
	tracker.RecordPosition("a1", "p1", "m2", 20, num.NewUint(2), num.DecimalOne(), time.Unix(20, 0))
	tracker.RecordPosition("a1", "p1", "m3", 30, num.NewUint(3), num.DecimalOne(), time.Unix(30, 0))
	tracker.RecordPosition("a1", "p2", "m1", 100, num.NewUint(10), num.DecimalOne(), time.Unix(15, 0))
	tracker.RecordPosition("a1", "p2", "m2", 200, num.NewUint(20), num.DecimalOne(), time.Unix(25, 0))
	tracker.RecordPosition("a1", "p2", "m3", 300, num.NewUint(30), num.DecimalOne(), time.Unix(45, 0))

	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(-100))
	tracker.RecordM2M("a1", "p2", "m1", num.DecimalFromInt64(100))
	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(250))
	tracker.RecordM2M("a1", "p2", "m1", num.DecimalFromInt64(-250))
	tracker.RecordM2M("a1", "p1", "m2", num.DecimalFromInt64(-50))
	tracker.RecordM2M("a1", "p2", "m2", num.DecimalFromInt64(50))
	tracker.RecordM2M("a1", "p1", "m3", num.DecimalFromInt64(100))
	tracker.RecordM2M("a1", "p2", "m3", num.DecimalFromInt64(-100))

	// end epoch1
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(0, 0), EndTime: time.Unix(60, 0)})

	gameID := "game123"

	// get metrics for market m1 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics := tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 1, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "16.3636375537", metrics[0].Score.StringFixed(10))
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "-2", metrics[1].Score.String())

	// get metrics for market m2 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 1, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "-3.7500003750", metrics[0].Score.StringFixed(10))
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "0.4285714531", metrics[1].Score.StringFixed(10))

	// get metrics for market m3 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 1, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "6.6666666667", metrics[0].Score.StringFixed(10))
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "-1.3333333333", metrics[1].Score.StringFixed(10))

	// get metrics for market m1,m2 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 1, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "12.6136371787", metrics[0].Score.StringFixed(10))
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "-1.5714285469", metrics[1].Score.StringFixed(10))

	// get metrics for all market window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 1, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "19.2803038454", metrics[0].Score.StringFixed(10))
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "-2.9047618803", metrics[1].Score.StringFixed(10))

	// start epoch2
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(60, 0)})
	tracker.RecordPosition("a1", "p1", "m1", 20, num.NewUint(5), num.DecimalOne(), time.Unix(90, 0))
	tracker.RecordPosition("a1", "p2", "m2", 10, num.NewUint(10), num.DecimalOne(), time.Unix(75, 0))
	tracker.RecordPosition("a2", "p1", "m3", 20, num.NewUint(5), num.DecimalOne(), time.Unix(90, 0))
	tracker.RecordPosition("a2", "p2", "m2", 10, num.NewUint(10), num.DecimalOne(), time.Unix(75, 0))

	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(450))
	tracker.RecordM2M("a1", "p2", "m1", num.DecimalFromInt64(-450))
	tracker.RecordM2M("a1", "p1", "m2", num.DecimalFromInt64(-100))
	tracker.RecordM2M("a1", "p2", "m2", num.DecimalFromInt64(100))
	// nothing in m3

	// end epoch2
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(60, 0), EndTime: time.Unix(120, 0)})

	// get metrics for market m1 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 1, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "30", metrics[0].Score.String())
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "-4.5", metrics[1].Score.String())

	// get metrics for market m2 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 1, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "-5", metrics[0].Score.String())
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "1.7391304348", metrics[1].Score.StringFixed(10))

	// get metrics for market m3 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 1, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, false, metrics[0].IsEligible)
	require.Equal(t, false, metrics[1].IsEligible)

	// get metrics for market m1,m2 with window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 1, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "25", metrics[0].Score.String())
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "-2.7608695652", metrics[1].Score.StringFixed(10))

	// get metrics for all market window size=1
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 1, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "25", metrics[0].Score.String())
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "-2.7608695652", metrics[1].Score.StringFixed(10))

	// calc with window size = 2
	// get metrics for market m1 with window size=2
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "23.1818187769", metrics[0].Score.StringFixed(10))
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "-3.25", metrics[1].Score.String())

	// get metrics for market m2 with window size=2
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "-4.3750001875", metrics[0].Score.StringFixed(10))
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "1.0838509439", metrics[1].Score.StringFixed(10))

	// get metrics for market m3 with window size=2
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "3.3333333333", metrics[0].Score.StringFixed(10))
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "-0.6666666667", metrics[1].Score.StringFixed(10))

	// get metrics for market m1,m2 with window size=2
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "18.8068185894", metrics[0].Score.StringFixed(10))
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "-2.1661490561", metrics[1].Score.StringFixed(10))

	// get metrics for all market window size=2
	balanceChecker.EXPECT().GetAvailableBalance(gomock.Any()).Return(num.UintZero(), nil).Times(2)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "22.1401519227", metrics[0].Score.StringFixed(10))
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "-2.8328157227", metrics[1].Score.StringFixed(10))

	// now make p2 not eligible via not having sufficient governance token
	balanceChecker.EXPECT().GetAvailableBalance("p1").Return(num.NewUint(2), nil).Times(1)
	balanceChecker.EXPECT().GetAvailableBalance("p2").Return(nil, errors.New("some error")).Times(1)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.NewUint(1), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, true, metrics[0].IsEligible)
	require.Equal(t, "2", metrics[0].StakingBalance.String())
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, false, metrics[1].IsEligible)
	require.Equal(t, "0", metrics[1].StakingBalance.String())

	// repeat now p2 has balance just not enough
	balanceChecker.EXPECT().GetAvailableBalance("p1").Return(num.NewUint(2), nil).Times(1)
	balanceChecker.EXPECT().GetAvailableBalance("p2").Return(num.NewUint(1), nil).Times(1)
	metrics = tracker.calculateMetricForIndividuals(ctx, "a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.NewUint(2), num.UintZero(), 2, gameID, 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, true, metrics[0].IsEligible)
	require.Equal(t, "2", metrics[0].StakingBalance.String())
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, false, metrics[1].IsEligible)
	require.Equal(t, "1", metrics[1].StakingBalance.String())
}

func TestCalculateMetricForPartyRelativeReturn(t *testing.T) {
	epochService := &DummyEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	collateralService := mocks.NewMockCollateral(ctrl)
	tracker := NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker, collateralService)
	epochService.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)
	tracker.SetEligibilityChecker(&DummyEligibilityChecker{})
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Time{}})

	// add some markets for 2 different assets
	tracker.MarketProposed("a1", "m1", "z1")
	tracker.MarketProposed("a1", "m2", "z2")
	tracker.MarketProposed("a1", "m3", "z3")

	// record some values for all metrics
	tracker.RecordPosition("a1", "p1", "m1", 10, num.NewUint(1), num.DecimalOne(), time.Unix(5, 0))
	tracker.RecordPosition("a1", "p1", "m2", 20, num.NewUint(2), num.DecimalOne(), time.Unix(20, 0))
	tracker.RecordPosition("a1", "p1", "m3", 30, num.NewUint(3), num.DecimalOne(), time.Unix(30, 0))
	tracker.RecordPosition("a1", "p2", "m1", 100, num.NewUint(10), num.DecimalOne(), time.Unix(15, 0))
	tracker.RecordPosition("a1", "p2", "m2", 200, num.NewUint(20), num.DecimalOne(), time.Unix(25, 0))
	tracker.RecordPosition("a1", "p2", "m3", 300, num.NewUint(30), num.DecimalOne(), time.Unix(45, 0))

	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(-100))
	tracker.RecordM2M("a1", "p2", "m1", num.DecimalFromInt64(100))
	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(250))
	tracker.RecordM2M("a1", "p2", "m1", num.DecimalFromInt64(-250))
	tracker.RecordM2M("a1", "p1", "m2", num.DecimalFromInt64(-50))
	tracker.RecordM2M("a1", "p2", "m2", num.DecimalFromInt64(50))
	tracker.RecordM2M("a1", "p1", "m3", num.DecimalFromInt64(100))
	tracker.RecordM2M("a1", "p2", "m3", num.DecimalFromInt64(-100))

	// end epoch1
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(0, 0), EndTime: time.Unix(60, 0)})

	// calculate metric for p1 with scope=[m1] for window size=1
	// 150 / 9.166666
	score, _ := tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "16.3636375537190948", score.String())
	// calculate metric for p1 with scope=[m2] for window size=1
	// -50 /13.333332
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "-3.7500003750000375", score.String())
	// calculate metric for p1 with scope=[m3] for window size=1
	// 100 / 15
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "6.6666666666666667", score.String())
	// calculate metric for p1 with scope=[m1, m2] for window size=1
	// 150 / 9.166666 - 50 /13.333332
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "12.6136371787", score.StringFixed(10))
	// 150 / 9.166666 - 50 /13.333332 +100/15
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "19.2803038454", score.StringFixed(10))

	// calculate metric for p2 with scope=[m1] for window size=1
	// -150 / 75
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "-2", score.String())
	// calculate metric for p2 with scope=[m2] for window size=1
	// 50 / 116.66666
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "0.4285714531", score.StringFixed(10))
	// calculate metric for p2 with scope=[m3] for window size=1
	// -100 / 75
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "-1.3333333333", score.StringFixed(10))
	// calculate metric for p2 with scope=[m1, m2] for window size=1
	// -2 + 0.4285714531
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "-1.5714285469", score.StringFixed(10))
	// calculate metric for p2 with no scope for window size=1
	// -2+0.4285714531-1.3333333333
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "-2.9047618803", score.StringFixed(10))

	// start epoch2
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(60, 0)})
	tracker.RecordPosition("a1", "p1", "m1", 20, num.NewUint(5), num.DecimalOne(), time.Unix(90, 0))
	tracker.RecordPosition("a1", "p2", "m2", 10, num.NewUint(10), num.DecimalOne(), time.Unix(75, 0))
	tracker.RecordPosition("a2", "p1", "m3", 20, num.NewUint(5), num.DecimalOne(), time.Unix(90, 0))
	tracker.RecordPosition("a2", "p2", "m2", 10, num.NewUint(10), num.DecimalOne(), time.Unix(75, 0))

	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(450))
	tracker.RecordM2M("a1", "p2", "m1", num.DecimalFromInt64(-450))
	tracker.RecordM2M("a1", "p1", "m2", num.DecimalFromInt64(-100))
	tracker.RecordM2M("a1", "p2", "m2", num.DecimalFromInt64(100))
	// nothing in m3

	// end epoch2
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(60, 0), EndTime: time.Unix(120, 0)})

	// calculate metric for p1 with scope=[m1] for window size=1
	// 450/15=30
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "30", score.String())

	// calculate metric for p1 with scope=[m2] for window size=1
	// -100/20
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "-5", score.String())

	// calculate metric for p1 with scope=[m3] for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "0", score.String())

	// calculate metric for p1 with scope=[m1, m2] for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "25", score.String())

	// calculate metric for p1 with no scope for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "25", score.String())

	// calculate metric for p2 with scope=[m1] for window size=1
	// -450/100
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "-4.5", score.String())

	// calculate metric for p1 with scope=[m2] for window size=1
	// 100/57.5
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "1.7391304348", score.StringFixed(10))

	// calculate metric for p1 with scope=[m3] for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "0", score.String())

	// calculate metric for p1 with scope=[m1, m3] for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "-2.7608695652", score.StringFixed(10))

	// calculate metric for p1 with no scope for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "-2.7608695652", score.StringFixed(10))

	// now calculate for window size=2

	// calculate metric for p1 with scope=[m1] for window size=2
	// (16.3636375537 + 30)/2
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2)
	require.Equal(t, "23.1818187768595474", score.String())
	// calculate metric for p1 with scope=[m2] for window size=2
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2)
	require.Equal(t, "-4.3750001875", score.StringFixed(10))
	// calculate metric for p1 with scope=[m3] for window size=2
	// (6.6666666666666667 + 0)/2
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2)
	require.Equal(t, "3.3333333333", score.StringFixed(10))
	// calculate metric for p1 with scope=[m1, m3] for window size=2
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2)
	require.Equal(t, "18.8068185894", score.StringFixed(10))
	// calculate metric for p1 with no scope for window size=2
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2)
	require.Equal(t, "22.1401519227", score.StringFixed(10))

	// calculate metric for p2 with scope=[m1] for window size=2
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2)
	require.Equal(t, "-3.25", score.String())
	// calculate metric for p2 with scope=[m2] for window size=2
	// (0.4285714285714286 + 1.7391304347826087)/2
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2)
	require.Equal(t, "1.0838509439", score.StringFixed(10))
	// calculate metric for p2 with scope=[m3] for window size=2
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2)
	require.Equal(t, "-0.6666666667", score.StringFixed(10))
	// calculate metric for p2 with scope=[m1, m3] for window size=2
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2)
	require.Equal(t, "-3.9166666667", score.StringFixed(10))
	// calculate metric for p2 with no scope for window size=2
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2)
	require.Equal(t, "-2.8328157227", score.StringFixed(10))

	// start epoch3
	epochService.target(context.Background(), types.Epoch{Seq: 3, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(120, 0)})
	// end epoch3
	epochService.target(context.Background(), types.Epoch{Seq: 3, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(120, 0), EndTime: time.Unix(180, 0)})

	// there was no m2m activity so all should be 0

	// calculate metric for p1 with scope=[m1] for window size=1
	// 0/20
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "0", score.String())
	// calculate metric for p1 with scope=[m2] for window size=1
	// 0/20
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "0", score.String())
	// calculate metric for p1 with scope=[m3] for window size=1
	// 0/30
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "0", score.String())
	// calculate metric for p1 with scope=[m1, m3] for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "0", score.String())
	// calculate metric for p1 with no scope for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "0", score.String())

	// calculate metric for p2 with scope=[m1] for window size=1
	// 0/100
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "0", score.String())
	// calculate metric for p2 with scope=[m2] for window size=1
	// 0/10
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "0", score.String())
	// calculate metric for p2 with scope=[m3] for window size=1
	// 0/300
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "0", score.String())
	// calculate metric for p2 with scope=[m1, m3] for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "0", score.String())
	// calculate metric for p2 with no scope for window size=1
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1)
	require.Equal(t, "0", score.String())

	// // now calculate for window size=3

	// calculate metric for p1 with scope=[m1] for window size=3
	// (16.363636363636363 + 30)/3
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3)
	require.Equal(t, "15.4545458512", score.StringFixed(10))
	// calculate metric for p1 with scope=[m2] for window size=3
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3)
	require.Equal(t, "-2.9166667917", score.StringFixed(10))
	// calculate metric for p1 with scope=[m3] for window size=3
	// (6.6666666666666667 + 0)/3
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3)
	require.Equal(t, "2.2222222222", score.StringFixed(10))
	// calculate metric for p1 with scope=[m1, m3] for window size=3
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3)
	require.Equal(t, "12.5378790596", score.StringFixed(10))
	// calculate metric for p1 with no scope for window size=3
	score, _ = tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3)
	require.Equal(t, "14.7601012818", score.StringFixed(10))

	// calculate metric for p2 with scope=[m1] for window size=3
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3)
	require.Equal(t, "-2.1666666667", score.StringFixed(10))
	// calculate metric for p2 with scope=[m2] for window size=3
	// (0.4285714285714286 + 1.7391304347826087 + 0 )/3
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3)
	require.Equal(t, "0.7225672959", score.StringFixed(10))
	// calculate metric for p2 with scope=[m3] for window size=3
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3)
	require.Equal(t, "-0.4444444444", score.StringFixed(10))
	// calculate metric for p2 with scope=[m1, m3] for window size=3
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3)
	require.Equal(t, "-2.6111111111", score.StringFixed(10))
	// calculate metric for p2 with no scope for window size=3
	score, _ = tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3)
	require.Equal(t, "-1.8885438152", score.StringFixed(10))
}

func TestCalculateMetricForParty(t *testing.T) {
	epochService := &DummyEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	collateralService := mocks.NewMockCollateral(ctrl)
	tracker := NewMarketActivityTracker(logging.NewTestLogger(), teams, balanceChecker, broker, collateralService)
	epochService.NotifyOnEpoch(tracker.OnEpochEvent, tracker.OnEpochRestore)
	tracker.SetEligibilityChecker(&DummyEligibilityChecker{})
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Time{}})

	// unsupported metric
	require.Panics(t, func() {
		tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE, 1)
	})
	require.Panics(t, func() {
		tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_VALIDATOR_RANKING, 1)
	})

	// no trackers per asset
	ds := make([]vgproto.DispatchMetric, 0, len(vgproto.DispatchMetric_name))
	for k := range vgproto.DispatchMetric_name {
		if vgproto.DispatchMetric(k) == vgproto.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE || vgproto.DispatchMetric(k) == vgproto.DispatchMetric_DISPATCH_METRIC_VALIDATOR_RANKING {
			continue
		}
		ds = append(ds, vgproto.DispatchMetric(k))
	}

	for _, dm := range ds {
		score, _ := tracker.calculateMetricForParty("a1", "p1", []string{}, dm, 1)
		require.Equal(t, num.DecimalZero(), score)
	}
}

func TestCalculateMetricForTeamUtil(t *testing.T) {
	ctx := context.Background()
	isEligible := func(_ context.Context, asset, party string, markets []string, minStakingBalanceRequired *num.Uint, notionalTimeWeightedAveragePositionRequired *num.Uint, _ string) (bool, *num.Uint, *num.Uint) {
		if party == "party1" || party == "party2" || party == "party3" || party == "party4" {
			return true, num.NewUint(100), num.NewUint(200)
		}
		return false, num.UintZero(), num.UintZero()
	}
	calculateMetricForParty := func(asset, party string, marketsInScope []string, metric vgproto.DispatchMetric, windowSize int) (num.Decimal, bool) {
		if party == "party1" {
			return num.DecimalFromFloat(1.5), true
		}
		if party == "party2" {
			return num.DecimalFromFloat(2), true
		}
		if party == "party3" {
			return num.DecimalFromFloat(0.5), true
		}
		if party == "party4" {
			return num.DecimalFromFloat(2.5), true
		}
		if party == "party5" {
			return num.DecimalFromFloat(0.8), true
		}
		return num.DecimalZero(), false
	}

	gameID := "game123"
	teamScore, partyScores := calculateMetricForTeamUtil(ctx, "asset1", []string{"party1", "party2", "party3", "party4", "party5"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, num.UintOne(), num.UintOne(), int(5), num.DecimalFromFloat(0.5), isEligible, calculateMetricForParty, gameID, map[string]*num.Uint{}, map[string][]map[string]struct{}{})
	// we're indicating the the score of the team need to be the mean of the top 0.5 * number of participants = floor(0.5*5) = 2
	// the top scores are 2.5 and 2 => team score should be 2.25
	// 4 party scores expected (1-4) as party5 is not eligible
	require.Equal(t, "2.25", teamScore.String())
	require.Equal(t, 5, len(partyScores))
	require.Equal(t, "party4", partyScores[0].Party)
	require.Equal(t, "2.5", partyScores[0].Score.String())
	require.Equal(t, "party2", partyScores[1].Party)
	require.Equal(t, "2", partyScores[1].Score.String())
	require.Equal(t, "party1", partyScores[2].Party)
	require.Equal(t, "1.5", partyScores[2].Score.String())
	require.Equal(t, "party3", partyScores[3].Party)
	require.Equal(t, "0.5", partyScores[3].Score.String())
	require.Equal(t, "party5", partyScores[4].Party)
	require.Equal(t, false, partyScores[4].IsEligible)

	// lets repeat the check when there is no one eligible
	teamScore, partyScores = calculateMetricForTeamUtil(ctx, "asset1", []string{"party5"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL, num.UintOne(), num.UintOne(), 5, num.DecimalFromFloat(0.5), isEligible, calculateMetricForParty, gameID, map[string]*num.Uint{}, map[string][]map[string]struct{}{})
	require.Equal(t, "0", teamScore.String())
	require.Equal(t, 1, len(partyScores))
	require.Equal(t, "party5", partyScores[0].Party)
	require.Equal(t, false, partyScores[0].IsEligible)
}

type DummyEpochEngine struct {
	target func(context.Context, types.Epoch)
}

func (e *DummyEpochEngine) NotifyOnEpoch(f func(context.Context, types.Epoch), _ func(context.Context, types.Epoch)) {
	e.target = f
}

type DummyCollateralEngine struct{}

func (e DummyCollateralEngine) GetAssetQuantum(asset string) (num.Decimal, error) {
	return num.DecimalOne(), nil
}

type DummyEligibilityChecker struct{}

func (e *DummyEligibilityChecker) IsEligibleForProposerBonus(marketID string, volumeTraded *num.Uint) bool {
	return volumeTraded.GT(num.NewUint(5000))
}

func TestIntoProto(t *testing.T) {
	mt := &marketTracker{
		asset:                       "asset",
		proposer:                    "proposer",
		proposersPaid:               map[string]struct{}{"p1": {}},
		readyToDelete:               true,
		valueTraded:                 num.NewUint(1000),
		makerFeesReceived:           map[string]*num.Uint{"p1": num.NewUint(1), "p2": num.NewUint(2)},
		makerFeesPaid:               map[string]*num.Uint{"p3": num.NewUint(3), "p4": num.NewUint(4)},
		lpFees:                      map[string]*num.Uint{"p5": num.NewUint(5), "p6": num.NewUint(6)},
		totalMakerFeesReceived:      num.NewUint(3),
		totalMakerFeesPaid:          num.NewUint(7),
		totalLpFees:                 num.NewUint(11),
		twPosition:                  map[string]*twPosition{"p1": {t: time.Now(), position: 200, currentEpochTWPosition: 300}},
		partyM2M:                    map[string]num.Decimal{"p1": num.DecimalFromInt64(20)},
		twNotional:                  map[string]*twNotional{"p2": {t: time.Now(), notional: num.NewUint(50), currentEpochTWNotional: num.NewUint(55)}},
		epochTotalMakerFeesReceived: []*num.Uint{num.NewUint(3000), num.NewUint(7000)},
		epochTotalMakerFeesPaid:     []*num.Uint{num.NewUint(3300), num.NewUint(7700)},
		epochTotalLpFees:            []*num.Uint{num.NewUint(3600), num.NewUint(8400)},
		epochMakerFeesReceived:      []map[string]*num.Uint{{"p1": num.NewUint(1000), "p2": num.NewUint(2000)}, {"p1": num.NewUint(3000), "p3": num.NewUint(4000)}},
		epochMakerFeesPaid:          []map[string]*num.Uint{{"p1": num.NewUint(1100), "p2": num.NewUint(2200)}, {"p1": num.NewUint(3300), "p3": num.NewUint(4400)}},
		epochLpFees:                 []map[string]*num.Uint{{"p1": num.NewUint(1200), "p2": num.NewUint(2400)}, {"p1": num.NewUint(3600), "p3": num.NewUint(4800)}},
		epochPartyM2M:               []map[string]num.Decimal{{"p1": num.DecimalFromInt64(1000), "p2": num.DecimalFromInt64(2000)}, {"p2": num.DecimalFromInt64(5000), "p3": num.DecimalFromInt64(4000)}},
		epochTimeWeightedPosition:   []map[string]uint64{{"p1": 100, "p2": 200}, {"p3": 90, "p4": 80}},
		epochTimeWeightedNotional:   []map[string]*num.Uint{{"p1": num.NewUint(1000), "p2": num.NewUint(2000)}, {"p1": num.NewUint(3000), "p3": num.NewUint(4000)}},
		allPartiesCache:             map[string]struct{}{"p1": {}, "p2": {}, "p3": {}, "p4": {}, "p5": {}, "p6": {}},
	}

	mt1Proto := mt.IntoProto("market1")
	mt2 := marketTrackerFromProto(mt1Proto)
	mt2Proto := mt2.IntoProto("market1")
	require.Equal(t, mt1Proto.String(), mt2Proto.String())
}

func TestEpochTakerFeesToProto(t *testing.T) {
	for i := 0; i < 10; i++ {
		epoch1 := map[string]map[string]map[string]*num.Uint{}
		epoch1["asset1"] = map[string]map[string]*num.Uint{}
		epoch1["asset1"]["market1"] = map[string]*num.Uint{}
		epoch1["asset1"]["market1"]["party3"] = num.NewUint(3)
		epoch1["asset1"]["market1"]["party1"] = num.NewUint(1)
		epoch1["asset1"]["market1"]["party2"] = num.NewUint(2)

		epoch1["asset1"]["market2"] = map[string]*num.Uint{}
		epoch1["asset1"]["market2"]["party4"] = num.NewUint(6)
		epoch1["asset1"]["market2"]["party3"] = num.NewUint(5)
		epoch1["asset1"]["market2"]["party1"] = num.NewUint(4)

		epoch1["asset1"]["market3"] = map[string]*num.Uint{}
		epoch1["asset1"]["market3"]["party6"] = num.NewUint(8)
		epoch1["asset1"]["market3"]["party5"] = num.NewUint(7)

		epoch1["asset2"] = map[string]map[string]*num.Uint{}
		epoch1["asset2"]["market1"] = map[string]*num.Uint{}
		epoch1["asset2"]["market1"]["party1"] = num.NewUint(11)
		epoch1["asset2"]["market1"]["party2"] = num.NewUint(21)
		epoch1["asset2"]["market1"]["party3"] = num.NewUint(31)

		epoch1["asset2"]["market4"] = map[string]*num.Uint{}
		epoch1["asset2"]["market4"]["party5"] = num.NewUint(9)
		epoch1["asset2"]["market4"]["party6"] = num.NewUint(10)
		epoch1["asset2"]["market4"]["party7"] = num.NewUint(11)

		epoch2 := map[string]map[string]map[string]*num.Uint{}
		epoch2["asset1"] = map[string]map[string]*num.Uint{}
		epoch2["asset1"]["market1"] = map[string]*num.Uint{}
		epoch2["asset1"]["market1"]["party5"] = num.NewUint(15)
		epoch2["asset1"]["market1"]["party6"] = num.NewUint(16)
		epoch2["asset1"]["market2"] = map[string]*num.Uint{}
		epoch2["asset1"]["market2"]["party1"] = num.NewUint(17)
		epoch2["asset1"]["market2"]["party2"] = num.NewUint(18)
		epoch2["asset1"]["market3"] = map[string]*num.Uint{}
		epoch2["asset1"]["market3"]["party4"] = num.NewUint(20)
		epoch2["asset1"]["market3"]["party3"] = num.NewUint(19)

		epoch2["asset2"] = map[string]map[string]*num.Uint{}
		epoch2["asset2"]["market1"] = map[string]*num.Uint{}
		epoch2["asset2"]["market4"] = map[string]*num.Uint{}
		epoch2["asset2"]["market1"]["party7"] = num.NewUint(41)
		epoch2["asset2"]["market4"]["party6"] = num.NewUint(31)

		epochData := []map[string]map[string]map[string]*num.Uint{
			epoch1,
			epoch2,
		}

		res := epochTakerFeesToProto(epochData)
		require.Equal(t, 2, len(res))
		require.Equal(t, 5, len(res[0].EpochPartyTakerFeesPaid))
		require.Equal(t, 5, len(res[1].EpochPartyTakerFeesPaid))

		require.Equal(t, "asset1", res[0].EpochPartyTakerFeesPaid[0].Asset)
		require.Equal(t, "market1", res[0].EpochPartyTakerFeesPaid[0].Market)
		require.Equal(t, "party1", res[0].EpochPartyTakerFeesPaid[0].TakerFees[0].Party)
		require.Equal(t, "1", num.UintFromBytes(res[0].EpochPartyTakerFeesPaid[0].TakerFees[0].TakerFees).String())
		require.Equal(t, "party2", res[0].EpochPartyTakerFeesPaid[0].TakerFees[1].Party)
		require.Equal(t, "2", num.UintFromBytes(res[0].EpochPartyTakerFeesPaid[0].TakerFees[1].TakerFees).String())
		require.Equal(t, "party3", res[0].EpochPartyTakerFeesPaid[0].TakerFees[2].Party)
		require.Equal(t, "3", num.UintFromBytes(res[0].EpochPartyTakerFeesPaid[0].TakerFees[2].TakerFees).String())

		require.Equal(t, "asset1", res[0].EpochPartyTakerFeesPaid[1].Asset)
		require.Equal(t, "market2", res[0].EpochPartyTakerFeesPaid[1].Market)
		require.Equal(t, "party1", res[0].EpochPartyTakerFeesPaid[1].TakerFees[0].Party)
		require.Equal(t, "4", num.UintFromBytes(res[0].EpochPartyTakerFeesPaid[1].TakerFees[0].TakerFees).String())
		require.Equal(t, "party3", res[0].EpochPartyTakerFeesPaid[1].TakerFees[1].Party)
		require.Equal(t, "5", num.UintFromBytes(res[0].EpochPartyTakerFeesPaid[1].TakerFees[1].TakerFees).String())
		require.Equal(t, "party4", res[0].EpochPartyTakerFeesPaid[1].TakerFees[2].Party)
		require.Equal(t, "6", num.UintFromBytes(res[0].EpochPartyTakerFeesPaid[1].TakerFees[2].TakerFees).String())

		require.Equal(t, "asset1", res[0].EpochPartyTakerFeesPaid[2].Asset)
		require.Equal(t, "market3", res[0].EpochPartyTakerFeesPaid[2].Market)
		require.Equal(t, "party5", res[0].EpochPartyTakerFeesPaid[2].TakerFees[0].Party)
		require.Equal(t, "7", num.UintFromBytes(res[0].EpochPartyTakerFeesPaid[2].TakerFees[0].TakerFees).String())
		require.Equal(t, "party6", res[0].EpochPartyTakerFeesPaid[2].TakerFees[1].Party)
		require.Equal(t, "8", num.UintFromBytes(res[0].EpochPartyTakerFeesPaid[2].TakerFees[1].TakerFees).String())

		require.Equal(t, "asset2", res[0].EpochPartyTakerFeesPaid[3].Asset)
		require.Equal(t, "market1", res[0].EpochPartyTakerFeesPaid[3].Market)
		require.Equal(t, "party1", res[0].EpochPartyTakerFeesPaid[3].TakerFees[0].Party)
		require.Equal(t, "11", num.UintFromBytes(res[0].EpochPartyTakerFeesPaid[3].TakerFees[0].TakerFees).String())
		require.Equal(t, "party2", res[0].EpochPartyTakerFeesPaid[3].TakerFees[1].Party)
		require.Equal(t, "21", num.UintFromBytes(res[0].EpochPartyTakerFeesPaid[3].TakerFees[1].TakerFees).String())
		require.Equal(t, "party3", res[0].EpochPartyTakerFeesPaid[3].TakerFees[2].Party)
		require.Equal(t, "31", num.UintFromBytes(res[0].EpochPartyTakerFeesPaid[3].TakerFees[2].TakerFees).String())

		require.Equal(t, "asset2", res[0].EpochPartyTakerFeesPaid[4].Asset)
		require.Equal(t, "market4", res[0].EpochPartyTakerFeesPaid[4].Market)
		require.Equal(t, "party5", res[0].EpochPartyTakerFeesPaid[4].TakerFees[0].Party)
		require.Equal(t, "9", num.UintFromBytes(res[0].EpochPartyTakerFeesPaid[4].TakerFees[0].TakerFees).String())
		require.Equal(t, "party6", res[0].EpochPartyTakerFeesPaid[4].TakerFees[1].Party)
		require.Equal(t, "10", num.UintFromBytes(res[0].EpochPartyTakerFeesPaid[4].TakerFees[1].TakerFees).String())
		require.Equal(t, "party7", res[0].EpochPartyTakerFeesPaid[4].TakerFees[2].Party)
		require.Equal(t, "11", num.UintFromBytes(res[0].EpochPartyTakerFeesPaid[4].TakerFees[2].TakerFees).String())

		require.Equal(t, "asset1", res[1].EpochPartyTakerFeesPaid[0].Asset)
		require.Equal(t, "market1", res[1].EpochPartyTakerFeesPaid[0].Market)
		require.Equal(t, "party5", res[1].EpochPartyTakerFeesPaid[0].TakerFees[0].Party)
		require.Equal(t, "15", num.UintFromBytes(res[1].EpochPartyTakerFeesPaid[0].TakerFees[0].TakerFees).String())
		require.Equal(t, "party6", res[1].EpochPartyTakerFeesPaid[0].TakerFees[1].Party)
		require.Equal(t, "16", num.UintFromBytes(res[1].EpochPartyTakerFeesPaid[0].TakerFees[1].TakerFees).String())

		require.Equal(t, "asset1", res[1].EpochPartyTakerFeesPaid[1].Asset)
		require.Equal(t, "market2", res[1].EpochPartyTakerFeesPaid[1].Market)
		require.Equal(t, "party1", res[1].EpochPartyTakerFeesPaid[1].TakerFees[0].Party)
		require.Equal(t, "17", num.UintFromBytes(res[1].EpochPartyTakerFeesPaid[1].TakerFees[0].TakerFees).String())
		require.Equal(t, "party2", res[1].EpochPartyTakerFeesPaid[1].TakerFees[1].Party)
		require.Equal(t, "18", num.UintFromBytes(res[1].EpochPartyTakerFeesPaid[1].TakerFees[1].TakerFees).String())

		require.Equal(t, "asset1", res[1].EpochPartyTakerFeesPaid[2].Asset)
		require.Equal(t, "market3", res[1].EpochPartyTakerFeesPaid[2].Market)
		require.Equal(t, "party3", res[1].EpochPartyTakerFeesPaid[2].TakerFees[0].Party)
		require.Equal(t, "19", num.UintFromBytes(res[1].EpochPartyTakerFeesPaid[2].TakerFees[0].TakerFees).String())
		require.Equal(t, "party4", res[1].EpochPartyTakerFeesPaid[2].TakerFees[1].Party)
		require.Equal(t, "20", num.UintFromBytes(res[1].EpochPartyTakerFeesPaid[2].TakerFees[1].TakerFees).String())

		require.Equal(t, "asset2", res[1].EpochPartyTakerFeesPaid[3].Asset)
		require.Equal(t, "market1", res[1].EpochPartyTakerFeesPaid[3].Market)
		require.Equal(t, "party7", res[1].EpochPartyTakerFeesPaid[3].TakerFees[0].Party)
		require.Equal(t, "41", num.UintFromBytes(res[1].EpochPartyTakerFeesPaid[3].TakerFees[0].TakerFees).String())

		require.Equal(t, "asset2", res[1].EpochPartyTakerFeesPaid[4].Asset)
		require.Equal(t, "market4", res[1].EpochPartyTakerFeesPaid[4].Market)
		require.Equal(t, "party6", res[1].EpochPartyTakerFeesPaid[4].TakerFees[0].Party)
		require.Equal(t, "31", num.UintFromBytes(res[1].EpochPartyTakerFeesPaid[4].TakerFees[0].TakerFees).String())
	}
}
