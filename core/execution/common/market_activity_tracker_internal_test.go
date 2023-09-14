package common

import (
	"context"
	"errors"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/execution/common/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
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
		require.Equal(t, num.DecimalFromInt64(4950), calcTotalForWindowD("party1", data, maxWindowSize))
	}

	windowSize := 5
	require.Equal(t, num.DecimalFromInt64(485), calcTotalForWindowD("party1", data, windowSize))

	windowSize = 2
	require.Equal(t, num.DecimalFromInt64(197), calcTotalForWindowD("party1", data, windowSize))

	require.Equal(t, num.DecimalZero(), calcTotalForWindowD("party2", data, windowSize))
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
	require.Equal(t, "0", getFees(feeHistory, "p3", windowSize).String())
	// party1 has 197 in window (100, 97)
	require.Equal(t, num.DecimalFromInt64(197), getFees(feeHistory, "p1", windowSize))
	// party2 has 100 in window (0, 200)
	require.Equal(t, num.DecimalFromInt64(200), getFees(feeHistory, "p2", windowSize))
}

func getDefaultTracker(t *testing.T) *marketTracker {
	t.Helper()
	return &marketTracker{
		asset:                  "asset",
		proposer:               "proposer",
		proposersPaid:          map[string]struct{}{},
		readyToDelete:          false,
		valueTraded:            num.UintZero(),
		makerFeesReceived:      map[string]*num.Uint{},
		makerFeesPaid:          map[string]*num.Uint{},
		lpFees:                 map[string]*num.Uint{},
		totalMakerFeesReceived: num.UintZero(),
		totalMakerFeesPaid:     num.UintZero(),
		totalLpFees:            num.UintZero(),
		twPosition:             map[string]*twPosition{},
		partyM2M:               map[string]num.Decimal{},
		twNotional:             map[string]*twNotional{},
		epochPartyM2M:          []map[string]num.Decimal{},
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
	require.Equal(t, num.DecimalZero(), tracker.getRelativeReturnMetricTotal("p2", 5))

	require.Equal(t, num.DecimalFromInt64(485), tracker.getRelativeReturnMetricTotal("p1", 5))
}

func TestGetPositionMetricTotal(t *testing.T) {
	tracker := getDefaultTracker(t)
	position := &twPosition{position: 0, t: time.Now(), currentEpochTWPosition: 42}
	tracker.twPosition["p1"] = position

	for i := uint64(0); i < maxWindowSize; i++ {
		tracker.epochTimeWeightedPosition = append(tracker.epochTimeWeightedPosition, map[string]uint64{"p1": i})
	}

	// nothing for party2
	require.Equal(t, uint64(0), tracker.getPositionMetricTotal("p2", 5))
	// 99+98+97+96+95 for party1
	require.Equal(t, uint64(485), tracker.getPositionMetricTotal("p1", 5))
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
	// max(-100/5 = -20,0)=0
	require.Equal(t, "0", ret1[0].String())
	ret2, ok := tracker.getReturns("p2", 1)
	require.True(t, ok)
	require.Equal(t, 1, len(ret2))
	// max(-100/10 = -10,0)=0
	require.Equal(t, "0", ret2[0].String())
	ret3, ok := tracker.getReturns("p3", 1)
	require.True(t, ok)
	require.Equal(t, 1, len(ret3))
	// 200/10 = 20
	require.Equal(t, "20", ret3[0].String())
	_, ok = tracker.getReturns("p4", 1)
	require.False(t, ok)
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
	println(tracker.getPositionMetricTotal("p1", 1))
	require.Equal(t, uint64(191666658), tracker.getPositionMetricTotal("p1", 1))

	// epoch 2
	// 191666658 * ( 10000000 - 10000000 ) + ( 300000000 * 10000000 ) / 10000000 = 300000000
	tracker.recordPosition("p1", 10, num.DecimalOne(), time.Unix(90, 0), time.Unix(60, 0))
	// 300000000 * ( 10000000 - 5000000 ) + ( 100000000 * 5000000 ) / 10000000 = 200000000
	tracker.processPositionEndOfEpoch(time.Unix(60, 0), time.Unix(120, 0))
	require.Equal(t, uint64(200000000), tracker.getPositionMetricTotal("p1", 1))
	require.Equal(t, uint64(391666658), tracker.getPositionMetricTotal("p1", 2))

	// epoch 3
	// no position changes over the epoch
	// 200000000 * ( 10000000 - 10000000 ) + ( 100000000 * 10000000 ) / 10000000 = 100000000
	tracker.processPositionEndOfEpoch(time.Unix(120, 0), time.Unix(180, 0))
	require.Equal(t, uint64(100000000), tracker.getPositionMetricTotal("p1", 1))
	require.Equal(t, uint64(300000000), tracker.getPositionMetricTotal("p1", 2))
	require.Equal(t, uint64(491666658), tracker.getPositionMetricTotal("p1", 3))
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

	// epoch 2
	// (( 358 * 0 ) + ( 600 * 10000000 )) / 10000000 = 600
	tracker.recordNotional("p1", num.NewUint(300), time.Unix(90, 0), time.Unix(60, 0))
	require.Equal(t, "600", tracker.twNotional["p1"].currentEpochTWNotional.String())

	// (( 600 * 5000000 ) + ( 300 * 5000000 )) / 10000000 = 450
	tracker.processNotionalEndOfEpoch(time.Unix(60, 0), time.Unix(120, 0))
	require.Equal(t, "450", tracker.twNotional["p1"].currentEpochTWNotional.String())

	// epoch 3
	// no position changes over the epoch
	// (( 450 * 0 ) + ( 300 * 10000000 )) / 10000000 = 300
	tracker.processNotionalEndOfEpoch(time.Unix(120, 0), time.Unix(180, 0))
	require.Equal(t, "300", tracker.twNotional["p1"].currentEpochTWNotional.String())
}

func TestCalculateMetricForIndividualsAvePosition(t *testing.T) {
	epochService := &DummyEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	tracker := NewMarketActivityTracker(logging.NewTestLogger(), epochService, teams, balanceChecker)
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

	// end epoch1
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(0, 0), EndTime: time.Unix(60, 0)})

	// get metrics for market m1 with window size=1
	metrics := tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.UintZero(), 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "9.166666", metrics[0].Score.String())
	require.Equal(t, "75", metrics[1].Score.String())

	// get metrics for market m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.UintZero(), 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "13.333332", metrics[0].Score.String())
	require.Equal(t, "116.66666", metrics[1].Score.String())

	// get metrics for market m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.UintZero(), 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "15", metrics[0].Score.String())
	require.Equal(t, "75", metrics[1].Score.String())

	// get metrics for market m1,m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.UintZero(), 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "22.499998", metrics[0].Score.String())
	require.Equal(t, "191.66666", metrics[1].Score.String())

	// get metrics for all market window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.UintZero(), 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "37.499998", metrics[0].Score.String())
	require.Equal(t, "266.66666", metrics[1].Score.String())

	// start epoch2
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(60, 0)})
	tracker.RecordPosition("a1", "p1", "m1", 20, num.NewUint(5), num.DecimalOne(), time.Unix(90, 0))
	tracker.RecordPosition("a1", "p2", "m2", 10, num.NewUint(10), num.DecimalOne(), time.Unix(75, 0))
	tracker.RecordPosition("a2", "p1", "m3", 20, num.NewUint(5), num.DecimalOne(), time.Unix(90, 0))
	tracker.RecordPosition("a2", "p2", "m2", 10, num.NewUint(10), num.DecimalOne(), time.Unix(75, 0))
	// end epoch2
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(60, 0), EndTime: time.Unix(120, 0)})

	// get metrics for market m1 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.UintZero(), 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "15", metrics[0].Score.String())
	require.Equal(t, "100", metrics[1].Score.String())

	// get metrics for market m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.UintZero(), 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "20", metrics[0].Score.String())
	require.Equal(t, "57.5", metrics[1].Score.String())

	// get metrics for market m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.UintZero(), 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "30", metrics[0].Score.String())
	require.Equal(t, "300", metrics[1].Score.String())

	// get metrics for market m1,m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.UintZero(), 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "35", metrics[0].Score.String())
	require.Equal(t, "157.5", metrics[1].Score.String())

	// get metrics for all market window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.UintZero(), 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "65", metrics[0].Score.String())
	require.Equal(t, "457.5", metrics[1].Score.String())

	// calc with window size = 2
	// get metrics for market m1 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.UintZero(), 2)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "12.083333", metrics[0].Score.String())
	require.Equal(t, "87.5", metrics[1].Score.String())

	// get metrics for market m2 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.UintZero(), 2)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "16.666666", metrics[0].Score.String())
	require.Equal(t, "87.08333", metrics[1].Score.String())

	// get metrics for market m2 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.UintZero(), 2)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "22.5", metrics[0].Score.String())
	require.Equal(t, "187.5", metrics[1].Score.String())

	// get metrics for market m1,m2 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.UintZero(), 2)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "28.749999", metrics[0].Score.String())
	require.Equal(t, "174.58333", metrics[1].Score.String())

	// get metrics for all market window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.UintZero(), 2)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "51.249999", metrics[0].Score.String())
	require.Equal(t, "362.08333", metrics[1].Score.String())

	// now make p2 not eligible via not having sufficient governance token
	balanceChecker.EXPECT().GetAvailableBalance("p1").Return(num.NewUint(2), nil).Times(1)
	balanceChecker.EXPECT().GetAvailableBalance("p2").Return(nil, errors.New("some error")).Times(1)
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.NewUint(1), num.UintZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)

	// repeat now p2 has balance just not enough
	balanceChecker.EXPECT().GetAvailableBalance("p1").Return(num.NewUint(2), nil).Times(1)
	balanceChecker.EXPECT().GetAvailableBalance("p2").Return(num.NewUint(1), nil).Times(1)
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.NewUint(2), num.UintZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
}

func TestCalculateMetricForPartyAvePosition(t *testing.T) {
	epochService := &DummyEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	tracker := NewMarketActivityTracker(logging.NewTestLogger(), epochService, teams, balanceChecker)
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
	require.Equal(t, "9.166666", tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p1 with scope=[m2] for window size=1
	// 0*(1-0.6666666666666667)+20*0.6666666666666667 = 13.3333333333
	require.Equal(t, "13.333332", tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p1 with scope=[m3] for window size=1
	// 0*(1-0.5)+30*0.5 = 15
	require.Equal(t, "15", tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p1 with scope=[m1, m2] for window size=1
	require.Equal(t, "22.499998", tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p1 with no scope for window size=1
	require.Equal(t, "37.499998", tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())

	// calculate metric for p2 with scope=[m1] for window size=1
	// 0*(1-0.75)+100*0.75 = 75
	require.Equal(t, "75", tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p2 with scope=[m2] for window size=1
	// 0*(1-0.5833333333333333)+200*0.5833333333333333
	require.Equal(t, "116.66666", tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p2 with scope=[m3] for window size=1
	// 0*(1-0.25)+300*0.25
	require.Equal(t, "75", tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p2 with scope=[m1, m3] for window size=1
	require.Equal(t, "150", tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p2 with no scope for window size=1
	require.Equal(t, "266.66666", tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())

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
	require.Equal(t, "15", tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p1 with scope=[m2] for window size=1
	// 13.333333333333334*(1-1)+20*1
	require.Equal(t, "20", tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p1 with scope=[m3] for window size=1
	// 15*(1-1)+30*1 = 30
	require.Equal(t, "30", tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p1 with scope=[m1, m3] for window size=1
	require.Equal(t, "35", tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p1 with no scope for window size=1
	require.Equal(t, "65", tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())

	// calculate metric for p2 with scope=[m1] for window size=1
	// 75*(1-1)+100*1
	require.Equal(t, "100", tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p1 with scope=[m2] for window size=1
	// 200*(1-0.75)+10*0.75
	require.Equal(t, "57.5", tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p1 with scope=[m3] for window size=1
	// 75*(1-1)+300*1
	require.Equal(t, "300", tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p1 with scope=[m1, m3] for window size=1
	require.Equal(t, "157.5", tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p1 with no scope for window size=1
	require.Equal(t, "457.5", tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())

	// now calculate for window size=2

	// calculate metric for p1 with scope=[m1] for window size=2
	// (15 + 9.166666666666667)/2
	require.Equal(t, "12.083333", tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 2).String())
	// calculate metric for p1 with scope=[m2] for window size=2
	// (13.333333333333334" + 20)/2
	require.Equal(t, "16.666666", tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 2).String())
	// calculate metric for p1 with scope=[m3] for window size=2
	// (15 + 30)/2
	require.Equal(t, "22.5", tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 2).String())
	// calculate metric for p1 with scope=[m1, m3] for window size=2
	require.Equal(t, "28.749999", tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 2).String())
	// calculate metric for p1 with no scope for window size=2
	require.Equal(t, "51.249999", tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 2).String())

	// calculate metric for p2 with scope=[m1] for window size=2
	// (100 + 75)/2
	require.Equal(t, "87.5", tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 2).String())
	// calculate metric for p2 with scope=[m2] for window size=2
	// (116.66666666666666 + 57.5)/2
	require.Equal(t, "87.08333", tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 2).String())
	// calculate metric for p2 with scope=[m3] for window size=2
	// (300 + 75)/2
	require.Equal(t, "187.5", tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 2).String())
	// calculate metric for p2 with scope=[m1, m3] for window size=2
	require.Equal(t, "275", tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 2).String())
	// calculate metric for p2 with no scope for window size=2
	require.Equal(t, "362.08333", tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 2).String())

	// start epoch3
	epochService.target(context.Background(), types.Epoch{Seq: 3, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(120, 0)})
	// end epoch3
	epochService.target(context.Background(), types.Epoch{Seq: 3, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(120, 0), EndTime: time.Unix(180, 0)})
	// calculate metric for p1 with scope=[m1] for window size=1
	// 15*(1-1)+20*1
	require.Equal(t, "20", tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p1 with scope=[m2] for window size=1
	// 20*(1-1)+20*1
	require.Equal(t, "20", tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p1 with scope=[m3] for window size=1
	// 30*(1-1)+30*1 = 30
	require.Equal(t, "30", tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p1 with scope=[m1, m3] for window size=1
	require.Equal(t, "40", tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p1 with no scope for window size=1
	require.Equal(t, "70", tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())

	// calculate metric for p2 with scope=[m1] for window size=1
	// 100*(1-1)+100*1
	require.Equal(t, "100", tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p2 with scope=[m2] for window size=1
	// 57.5*(1-1)+10*1
	require.Equal(t, "10", tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p2 with scope=[m3] for window size=1
	// 300*(1-1)+300*1
	require.Equal(t, "300", tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p2 with scope=[m1, m3] for window size=1
	require.Equal(t, "110", tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p2 with no scope for window size=1
	require.Equal(t, "410", tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())

	// now calculate for window size=3

	// calculate metric for p1 with scope=[m1] for window size=3
	// (9.166666666666667 + 15 + 20)/3
	require.Equal(t, "14.722222", tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 3).String())
	// calculate metric for p1 with scope=[m2] for window size=3
	// (13.333333333333334" + 20 + 20)/3
	require.Equal(t, "17.7777773333333333", tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 3).String())
	// calculate metric for p1 with scope=[m3] for window size=3
	// (15 + 30 + 30)/3
	require.Equal(t, "25", tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 3).String())
	// calculate metric for p1 with scope=[m1, m3] for window size=3
	require.Equal(t, "32.4999993333333333", tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 3).String())
	// calculate metric for p1 with no scope for window size=3
	require.Equal(t, "57.4999993333333333", tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 3).String())

	// calculate metric for p2 with scope=[m1] for window size=3
	// (100 + 75+100)/3
	require.Equal(t, "91.6666666666666667", tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 3).String())
	// calculate metric for p2 with scope=[m2] for window size=3
	// (116.66666666666666 + 57.5 + 10)/3
	require.Equal(t, "61.3888866666666667", tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 3).String())
	// calculate metric for p2 with scope=[m3] for window size=3
	// (300 + 75 + 300)/3
	require.Equal(t, "225", tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 3).String())
	// calculate metric for p2 with scope=[m1, m3] for window size=3
	require.Equal(t, "316.6666666666666667", tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 3).String())
	// calculate metric for p2 with no scope for window size=3
	require.Equal(t, "378.0555533333333333", tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 3).String())
}

func TestCalculateMetricForIndividualReturnVolatility(t *testing.T) {
	epochService := &DummyEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	tracker := NewMarketActivityTracker(logging.NewTestLogger(), epochService, teams, balanceChecker)
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

	// get metrics for market m1 with window size=1
	metrics := tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.UintZero(), 1)
	// only one sample (window size=1) variance=0 by definition
	require.Equal(t, 0, len(metrics))

	// get metrics for market m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.UintZero(), 1)
	// only one sample (window size=1) variance=0 by definition
	require.Equal(t, 0, len(metrics))

	// get metrics for market m3 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.UintZero(), 1)
	// only one sample (window size=1) variance=0 by definition
	require.Equal(t, 0, len(metrics))

	// get metrics for market m1,m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.UintZero(), 1)
	// only one sample (window size=1) variance=0 by definition
	require.Equal(t, 0, len(metrics))

	// get metrics for all market window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.UintZero(), 1)
	// only one sample (window size=1) variance=0 by definition
	require.Equal(t, 0, len(metrics))

	// // start epoch2
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

	// calc with window size = 2
	// get metrics for market m1 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.UintZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	// variance(30, 16.3636363636)
	require.Equal(t, "46.4875951915850383", metrics[0].Score.String())

	// get metrics for market m2 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.UintZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p2", metrics[0].Party)
	// variance(1.7391304347826087,0.4285714285714286)
	require.Equal(t, "0.4293912111426469", metrics[0].Score.String())

	// get metrics for market m3 with window size=2
	// variance(6.6666666666)
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.UintZero(), 2)
	require.Equal(t, 0, len(metrics))

	// get metrics for market m1,m2 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.UintZero(), 2)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	// variance(30, 16.363636363636363)
	require.Equal(t, "46.4875951915850383", metrics[0].Score.String())
	// variance(1.7391304347826087,0.4285714285714286)
	require.Equal(t, "0.4293912111426469", metrics[1].Score.String())

	// get metrics for all market window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.UintZero(), 2)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	// variance(30, 23.0303030303030297)
	require.Equal(t, "12.144164815093132", metrics[0].Score.String())
	// variance(1.7391304347826087,0.4285714285714286)
	require.Equal(t, "0.4293912111426469", metrics[1].Score.String())

	// now make p2 not eligible via not having sufficient governance token
	balanceChecker.EXPECT().GetAvailableBalance("p1").Return(num.NewUint(2), nil).Times(1)
	balanceChecker.EXPECT().GetAvailableBalance("p2").Return(nil, errors.New("some error")).Times(1)
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.NewUint(1), num.UintZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)

	// repeat now p2 has balance just not enough
	balanceChecker.EXPECT().GetAvailableBalance("p1").Return(num.NewUint(2), nil).Times(1)
	balanceChecker.EXPECT().GetAvailableBalance("p2").Return(num.NewUint(1), nil).Times(1)
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.NewUint(2), num.UintZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
}

func TestCalculateMetricForIndividualsRelativeReturn(t *testing.T) {
	epochService := &DummyEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	tracker := NewMarketActivityTracker(logging.NewTestLogger(), epochService, teams, balanceChecker)
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

	// get metrics for market m1 with window size=1
	metrics := tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 1)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "16.3636375537190948", metrics[0].Score.String())

	// get metrics for market m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 1)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p2", metrics[0].Party)
	require.Equal(t, "0.4285714530612259", metrics[0].Score.String())

	// get metrics for market m3 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 1)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "6.6666666666666667", metrics[0].Score.String())

	// get metrics for market m1,m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 1)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "12.6136371787", metrics[0].Score.StringFixed(10))

	// get metrics for all market window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 1)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "19.2803038454", metrics[0].Score.StringFixed(10))

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
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 1)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "30", metrics[0].Score.String())

	// get metrics for market m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 1)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p2", metrics[0].Party)
	require.Equal(t, "1.7391304347826087", metrics[0].Score.String())

	// get metrics for market m3 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 1)
	require.Equal(t, 0, len(metrics))

	// get metrics for market m1,m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 1)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "25", metrics[0].Score.String())

	// get metrics for all market window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 1)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "25", metrics[0].Score.String())

	// calc with window size = 2
	// get metrics for market m1 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "23.1818187768595474", metrics[0].Score.String())

	// get metrics for market m2 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p2", metrics[0].Party)
	require.Equal(t, "1.0838509439219173", metrics[0].Score.String())

	// get metrics for market m3 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "3.3333333333333334", metrics[0].Score.String())

	// get metrics for market m1,m2 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "18.8068185894", metrics[0].Score.StringFixed(10))

	// get metrics for all market window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.UintZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "22.1401519227", metrics[0].Score.StringFixed(10))

	// now make p2 not eligible via not having sufficient governance token
	balanceChecker.EXPECT().GetAvailableBalance("p1").Return(num.NewUint(2), nil).Times(1)
	balanceChecker.EXPECT().GetAvailableBalance("p2").Return(nil, errors.New("some error")).Times(1)
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.NewUint(1), num.UintZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)

	// repeat now p2 has balance just not enough
	balanceChecker.EXPECT().GetAvailableBalance("p1").Return(num.NewUint(2), nil).Times(1)
	balanceChecker.EXPECT().GetAvailableBalance("p2").Return(num.NewUint(1), nil).Times(1)
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.NewUint(2), num.UintZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
}

func TestCalculateMetricForPartyRelativeReturn(t *testing.T) {
	epochService := &DummyEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	tracker := NewMarketActivityTracker(logging.NewTestLogger(), epochService, teams, balanceChecker)
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
	require.Equal(t, "16.3636375537190948", tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())
	// calculate metric for p1 with scope=[m2] for window size=1
	// max(0, -50 /13.333332)
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())
	// calculate metric for p1 with scope=[m3] for window size=1
	// 100 / 15
	require.Equal(t, "6.6666666666666667", tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())
	// calculate metric for p1 with scope=[m1, m2] for window size=1
	// 150 / 9.166666 - 50 /13.333332
	require.Equal(t, "12.6136371787", tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).StringFixed(10))
	// 150 / 9.166666 - 50 /13.333332 +100/15
	require.Equal(t, "19.2803038454", tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).StringFixed(10))

	// calculate metric for p2 with scope=[m1] for window size=1
	// max(-100 / 75, 0)
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())
	// calculate metric for p2 with scope=[m2] for window size=1
	// 50 / 116.66666
	require.Equal(t, "0.4285714530612259", tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())
	// calculate metric for p2 with scope=[m3] for window size=1
	// max(-100 / 75,0)
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())
	// calculate metric for p2 with scope=[m1, m2] for window size=1
	// max(0, -100/75+50/116.66666666666666)
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())
	// calculate metric for p2 with no scope for window size=1
	// max(-100/75+50/116.66666666666666-100/75, 0)
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())

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
	require.Equal(t, "30", tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())

	// calculate metric for p1 with scope=[m2] for window size=1
	// max(0, -100/20)
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())

	// calculate metric for p1 with scope=[m3] for window size=1
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())

	// calculate metric for p1 with scope=[m1, m2] for window size=1
	require.Equal(t, "25", tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())

	// calculate metric for p1 with no scope for window size=1
	require.Equal(t, "25", tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())

	// calculate metric for p2 with scope=[m1] for window size=1
	// max(0, -450/100)
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())

	// calculate metric for p1 with scope=[m2] for window size=1
	// 100/57.5
	require.Equal(t, "1.7391304347826087", tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())

	// calculate metric for p1 with scope=[m3] for window size=1
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())

	// calculate metric for p1 with scope=[m1, m3] for window size=1
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())

	// calculate metric for p1 with no scope for window size=1
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())

	// now calculate for window size=2

	// calculate metric for p1 with scope=[m1] for window size=2
	// (16.3636375537 + 30)/2
	require.Equal(t, "23.1818187768595474", tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2).String())
	// calculate metric for p1 with scope=[m2] for window size=2
	// (0 + 0)/2
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2).String())
	// calculate metric for p1 with scope=[m3] for window size=2
	// (6.6666666666666667 + 0)/2
	require.Equal(t, "3.3333333333333334", tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2).String())
	// calculate metric for p1 with scope=[m1, m3] for window size=2
	require.Equal(t, "18.8068185894", tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2).StringFixed(10))
	// calculate metric for p1 with no scope for window size=2
	require.Equal(t, "22.1401519227", tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2).StringFixed(10))

	// calculate metric for p2 with scope=[m1] for window size=2
	// (0 + 0)/2
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2).String())
	// calculate metric for p2 with scope=[m2] for window size=2
	// (0.4285714285714286 + 1.7391304347826087)/2
	require.Equal(t, "1.0838509439219173", tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2).String())
	// calculate metric for p2 with scope=[m3] for window size=2
	// (0 + 0)/2
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2).String())
	// calculate metric for p2 with scope=[m1, m3] for window size=2
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2).String())
	// calculate metric for p2 with no scope for window size=2
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2).String())

	// start epoch3
	epochService.target(context.Background(), types.Epoch{Seq: 3, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(120, 0)})
	// end epoch3
	epochService.target(context.Background(), types.Epoch{Seq: 3, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(120, 0), EndTime: time.Unix(180, 0)})

	// there was no m2m activity so all should be 0

	// calculate metric for p1 with scope=[m1] for window size=1
	// 0/20
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())
	// calculate metric for p1 with scope=[m2] for window size=1
	// 0/20
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())
	// calculate metric for p1 with scope=[m3] for window size=1
	// 0/30
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())
	// calculate metric for p1 with scope=[m1, m3] for window size=1
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())
	// calculate metric for p1 with no scope for window size=1
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())

	// calculate metric for p2 with scope=[m1] for window size=1
	// 0/100
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())
	// calculate metric for p2 with scope=[m2] for window size=1
	// 0/10
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())
	// calculate metric for p2 with scope=[m3] for window size=1
	// 0/300
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())
	// calculate metric for p2 with scope=[m1, m3] for window size=1
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())
	// calculate metric for p2 with no scope for window size=1
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())

	// // now calculate for window size=3

	// calculate metric for p1 with scope=[m1] for window size=3
	// (16.363636363636363 + 30)/3
	require.Equal(t, "15.4545458512396983", tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3).String())
	// calculate metric for p1 with scope=[m2] for window size=3
	// (0+0+0)/3
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3).String())
	// calculate metric for p1 with scope=[m3] for window size=3
	// (6.6666666666666667 + 0)/3
	require.Equal(t, "2.2222222222222222", tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3).String())
	// calculate metric for p1 with scope=[m1, m3] for window size=3
	require.Equal(t, "12.5378790596", tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3).StringFixed(10))
	// calculate metric for p1 with no scope for window size=3
	require.Equal(t, "14.7601012818", tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3).StringFixed(10))

	// calculate metric for p2 with scope=[m1] for window size=3
	// (0 +0+0)/3
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3).String())
	// calculate metric for p2 with scope=[m2] for window size=3
	// (0.4285714285714286 + 1.7391304347826087 + 0 )/3
	require.Equal(t, "0.7225672959479449", tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3).String())
	// calculate metric for p2 with scope=[m3] for window size=3
	// (0 +0+0)/3
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3).String())
	// calculate metric for p2 with scope=[m1, m3] for window size=3
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3).String())
	// calculate metric for p2 with no scope for window size=3
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3).String())
}

func TestCalculateMetricForParty(t *testing.T) {
	epochService := &DummyEpochEngine{}
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	balanceChecker := mocks.NewMockAccountBalanceChecker(ctrl)
	tracker := NewMarketActivityTracker(logging.NewTestLogger(), epochService, teams, balanceChecker)
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
		require.Equal(t, num.DecimalZero(), tracker.calculateMetricForParty("a1", "p1", []string{}, dm, 1))
	}
}

func TestCalculateMetricForTeamUtil(t *testing.T) {
	isEligible := func(asset, party string, markets []string, minStakingBalanceRequired *num.Uint, notionalTimeWeightedAveragePositionRequired *num.Uint) bool {
		if party == "party1" || party == "party2" || party == "party3" || party == "party4" {
			return true
		}
		return false
	}
	calculateMetricForParty := func(asset, party string, marketsInScope []string, metric vega.DispatchMetric, windowSize int) num.Decimal {
		if party == "party1" {
			return num.DecimalFromFloat(1.5)
		}
		if party == "party2" {
			return num.DecimalFromFloat(2)
		}
		if party == "party3" {
			return num.DecimalFromFloat(0.5)
		}
		if party == "party4" {
			return num.DecimalFromFloat(2.5)
		}
		if party == "party5" {
			return num.DecimalFromFloat(0.8)
		}
		return num.DecimalZero()
	}

	teamScore, partyScores := calculateMetricForTeamUtil("asset1", []string{"party1", "party2", "party3", "party4", "party5"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintOne(), num.UintOne(), 5, num.DecimalFromFloat(0.5), isEligible, calculateMetricForParty)
	// we're indicating the the score of the team need to be the mean of the top 0.5 * number of participants = floor(0.5*5) = 2
	// the top scores are 2.5 and 2 => team score should be 2.25
	// 4 party scores expected (1-4) as party5 is not eligible
	require.Equal(t, "2.25", teamScore.String())
	require.Equal(t, 4, len(partyScores))
	require.Equal(t, "party4", partyScores[0].Party)
	require.Equal(t, "2.5", partyScores[0].Score.String())
	require.Equal(t, "party2", partyScores[1].Party)
	require.Equal(t, "2", partyScores[1].Score.String())
	require.Equal(t, "party1", partyScores[2].Party)
	require.Equal(t, "1.5", partyScores[2].Score.String())
	require.Equal(t, "party3", partyScores[3].Party)
	require.Equal(t, "0.5", partyScores[3].Score.String())

	// lets repeat the check when there is no one eligible
	teamScore, partyScores = calculateMetricForTeamUtil("asset1", []string{"party5"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintOne(), num.UintOne(), 5, num.DecimalFromFloat(0.5), isEligible, calculateMetricForParty)
	require.Equal(t, "0", teamScore.String())
	require.Equal(t, 0, len(partyScores))
}

type DummyEpochEngine struct {
	target func(context.Context, types.Epoch)
}

func (e *DummyEpochEngine) NotifyOnEpoch(f func(context.Context, types.Epoch), _ func(context.Context, types.Epoch)) {
	e.target = f
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
