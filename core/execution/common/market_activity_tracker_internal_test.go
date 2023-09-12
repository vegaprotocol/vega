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
	data := make([]*num.Decimal, maxWindowSize)
	for i := int64(0); i < maxWindowSize; i++ {
		d := num.DecimalFromInt64(i)
		data[i] = &d
	}
	for idx := 0; idx < len(data); idx++ {
		require.Equal(t, num.DecimalFromInt64(4950), calcTotalForWindowD(data, idx, maxWindowSize))
	}

	windowSize := 5
	idx := 3 // meaning we should get 2+1+0+99+98=200
	require.Equal(t, num.DecimalFromInt64(200), calcTotalForWindowD(data, idx, windowSize))

	idx = 0 // meaning we should get 99+98+97+96+95=485
	require.Equal(t, num.DecimalFromInt64(485), calcTotalForWindowD(data, idx, windowSize))

	idx = 50 // somewhere in the middle - we should get 49+48+47+46+45=235
	require.Equal(t, num.DecimalFromInt64(235), calcTotalForWindowD(data, idx, windowSize))

	windowSize = 2
	idx = 1 // meaning we should get 0+99=99
	require.Equal(t, num.DecimalFromInt64(99), calcTotalForWindowD(data, idx, windowSize))

	idx = 0 // meaning we should get 99+98=197
	require.Equal(t, num.DecimalFromInt64(197), calcTotalForWindowD(data, idx, windowSize))

	idx = 50 // somewhere in the middle - we should get 49+48=97
	require.Equal(t, num.DecimalFromInt64(97), calcTotalForWindowD(data, idx, windowSize))
}

func TestCalcTotalForWindowU(t *testing.T) {
	data := make([]*num.Uint, maxWindowSize)
	for i := uint64(5); i < maxWindowSize; i++ {
		data[i] = num.NewUint(i)
	}
	for idx := 0; idx < len(data); idx++ {
		require.Equal(t, num.DecimalFromInt64(4940), calcTotalForWindowU(data, idx, maxWindowSize))
	}

	windowSize := 5
	idx := 3 // meaning we should get nil+nil+nil+99+98=197
	require.Equal(t, num.DecimalFromInt64(197), calcTotalForWindowU(data, idx, windowSize))

	idx = 0 // meaning we should get 99+98+97+96+95=485
	require.Equal(t, num.DecimalFromInt64(485), calcTotalForWindowU(data, idx, windowSize))

	idx = 50 // somewhere in the middle - we should get 49+48+47+46+45=235
	require.Equal(t, num.DecimalFromInt64(235), calcTotalForWindowU(data, idx, windowSize))

	windowSize = 2
	idx = 1 // meaning we should get nil+99=99
	require.Equal(t, num.DecimalFromInt64(99), calcTotalForWindowU(data, idx, windowSize))

	idx = 0 // meaning we should get 99+98=197
	require.Equal(t, num.DecimalFromInt64(197), calcTotalForWindowU(data, idx, windowSize))

	idx = 50 // somewhere in the middle - we should get 49+48=97
	require.Equal(t, num.DecimalFromInt64(97), calcTotalForWindowU(data, idx, windowSize))
}

func TestGetTotalFees(t *testing.T) {
	fd := &feeData{runningTotal: num.UintZero(), previousEpochs: make([]*num.Uint, maxWindowSize), previousEpochsIdx: 0}
	for i := uint64(5); i < maxWindowSize; i++ {
		fd.previousEpochs[i] = num.NewUint(i)
	}

	windowSize := 5
	fd.previousEpochsIdx = 3 // meaning we should get nil+nil+nil+99+98=197
	require.Equal(t, num.DecimalFromInt64(197), getTotalFees(fd, windowSize))

	fd.previousEpochsIdx = 0 // meaning we should get 99+98+97+96+95=485
	require.Equal(t, num.DecimalFromInt64(485), getTotalFees(fd, windowSize))

	fd.previousEpochsIdx = 50 // somewhere in the middle - we should get 49+48+47+46+45=235
	require.Equal(t, num.DecimalFromInt64(235), getTotalFees(fd, windowSize))

	windowSize = 2
	fd.previousEpochsIdx = 1 // meaning we should get nil+99=99
	require.Equal(t, num.DecimalFromInt64(99), getTotalFees(fd, windowSize))

	fd.previousEpochsIdx = 0 // meaning we should get 99+98=197
	require.Equal(t, num.DecimalFromInt64(197), getTotalFees(fd, windowSize))

	fd.previousEpochsIdx = 50 // somewhere in the middle - we should get 49+48=97
	require.Equal(t, num.DecimalFromInt64(97), getTotalFees(fd, windowSize))
}

func TestGetFees(t *testing.T) {
	fd1 := &feeData{runningTotal: num.UintZero(), previousEpochs: make([]*num.Uint, maxWindowSize), previousEpochsIdx: 3}
	for i := uint64(5); i < maxWindowSize; i++ {
		fd1.previousEpochs[i] = num.NewUint(i)
	}
	fd2 := &feeData{runningTotal: num.UintZero(), previousEpochs: make([]*num.Uint, maxWindowSize), previousEpochsIdx: 1}
	for i := uint64(0); i < maxWindowSize-5; i++ {
		fd2.previousEpochs[i] = num.NewUint(100 - i)
	}
	pfd := map[string]*feeData{"p1": fd1, "p2": fd2}
	windowSize := 5
	// party has no fee data
	require.Equal(t, num.DecimalZero(), getFees(pfd, "p3", windowSize))
	// party1 has 197 in window
	require.Equal(t, num.DecimalFromInt64(197), getFees(pfd, "p1", windowSize))
	// party2 has 100 in window (100, nil,nil, nil,nil)
	require.Equal(t, num.DecimalFromInt64(100), getFees(pfd, "p2", windowSize))
}

func getDefaultTracker(t *testing.T) *marketTracker {
	t.Helper()
	return &marketTracker{
		asset:                  "asset",
		proposer:               "proposer",
		proposersPaid:          map[string]struct{}{},
		readyToDelete:          false,
		valueTraded:            num.UintZero(),
		makerFeesReceived:      map[string]*feeData{},
		makerFeesPaid:          map[string]*feeData{},
		lpFees:                 map[string]*feeData{},
		totalMakerFeesReceived: &feeData{runningTotal: num.UintZero(), previousEpochs: make([]*num.Uint, maxWindowSize), previousEpochsIdx: 0},
		totalMakerFeesPaid:     &feeData{runningTotal: num.UintZero(), previousEpochs: make([]*num.Uint, maxWindowSize), previousEpochsIdx: 0},
		totalLpFees:            &feeData{runningTotal: num.UintZero(), previousEpochs: make([]*num.Uint, maxWindowSize), previousEpochsIdx: 0},
		timeWeightedPosition:   map[string]*twPosition{},
		partyM2M:               map[string]*m2mData{},
		twNotionalPosition:     map[string]*twNotionalPosition{},
	}
}

func TestGetPartiesForMetric(t *testing.T) {
	tracker := getDefaultTracker(t)

	tracker.makerFeesReceived["mfr1"] = &feeData{}
	tracker.makerFeesReceived["mfr2"] = &feeData{}
	tracker.makerFeesReceived["mfr3"] = &feeData{}
	tracker.makerFeesPaid["mfp1"] = &feeData{}
	tracker.makerFeesPaid["mfp2"] = &feeData{}
	tracker.makerFeesPaid["mfp3"] = &feeData{}
	tracker.lpFees["lpf1"] = &feeData{}
	tracker.lpFees["lpf2"] = &feeData{}
	tracker.lpFees["lpf3"] = &feeData{}
	tracker.timeWeightedPosition["twp1"] = &twPosition{}
	tracker.timeWeightedPosition["twp2"] = &twPosition{}
	tracker.timeWeightedPosition["twp3"] = &twPosition{}
	tracker.partyM2M["m2m1"] = &m2mData{}
	tracker.partyM2M["m2m2"] = &m2mData{}
	tracker.partyM2M["m2m3"] = &m2mData{}

	require.Equal(t, 0, len(tracker.getPartiesForMetric(vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE)))
	require.Equal(t, 0, len(tracker.getPartiesForMetric(vega.DispatchMetric_DISPATCH_METRIC_VALIDATOR_RANKING)))
	mfr := tracker.getPartiesForMetric(vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED)
	require.Equal(t, 3, len(mfr))
	require.Equal(t, "mfr1", mfr[0])
	require.Equal(t, "mfr2", mfr[1])
	require.Equal(t, "mfr3", mfr[2])

	mfp := tracker.getPartiesForMetric(vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID)
	require.Equal(t, 3, len(mfp))
	require.Equal(t, "mfp1", mfp[0])
	require.Equal(t, "mfp2", mfp[1])
	require.Equal(t, "mfp3", mfp[2])

	lpf := tracker.getPartiesForMetric(vega.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED)
	require.Equal(t, 3, len(lpf))
	require.Equal(t, "lpf1", lpf[0])
	require.Equal(t, "lpf2", lpf[1])
	require.Equal(t, "lpf3", lpf[2])

	twp := tracker.getPartiesForMetric(vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION)
	require.Equal(t, 3, len(twp))
	require.Equal(t, "twp1", twp[0])
	require.Equal(t, "twp2", twp[1])
	require.Equal(t, "twp3", twp[2])

	m2m := tracker.getPartiesForMetric(vega.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN)
	require.Equal(t, 3, len(m2m))
	require.Equal(t, "m2m1", m2m[0])
	require.Equal(t, "m2m2", m2m[1])
	require.Equal(t, "m2m3", m2m[2])

	m2m = tracker.getPartiesForMetric(vega.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY)
	require.Equal(t, 3, len(m2m))
	require.Equal(t, "m2m1", m2m[0])
	require.Equal(t, "m2m2", m2m[1])
	require.Equal(t, "m2m3", m2m[2])
}

func TestGetRelativeReturnMetricTotal(t *testing.T) {
	tracker := getDefaultTracker(t)
	m2m := &m2mData{runningTotal: num.DecimalZero(), previousEpochs: make([]*num.Decimal, maxWindowSize), previousEpochsIdx: 3}
	tracker.partyM2M["p1"] = m2m

	for i := int64(0); i < maxWindowSize; i++ {
		d := num.DecimalFromInt64(i)
		m2m.previousEpochs[i] = &d
	}
	// nothing for party2
	require.Equal(t, num.DecimalZero(), tracker.getRelativeReturnMetricTotal("p2", 5))
	// 2+1+0+99+98=200 for party1
	require.Equal(t, num.DecimalFromInt64(200), tracker.getRelativeReturnMetricTotal("p1", 5))
}

func TestGetPositionMetricTotal(t *testing.T) {
	tracker := getDefaultTracker(t)
	position := &twPosition{position: num.DecimalZero(), t: time.Now(), currentEpochTWPosition: num.DecimalE(), previousEpochs: make([]*num.Decimal, maxWindowSize), previousEpochsIdx: 3}
	tracker.timeWeightedPosition["p1"] = position

	for i := int64(0); i < maxWindowSize; i++ {
		d := num.DecimalFromInt64(i)
		position.previousEpochs[i] = &d
	}
	// nothing for party2
	require.Equal(t, num.DecimalZero(), tracker.getPositionMetricTotal("p2", 5))
	// 2+1+0+99+98=200 for party1
	require.Equal(t, num.DecimalFromInt64(200), tracker.getPositionMetricTotal("p1", 5))
}

func TestReturns(t *testing.T) {
	tracker := getDefaultTracker(t)

	tracker.recordM2M("p1", num.DecimalFromInt64(100))
	tracker.recordPosition("p1", num.DecimalFromInt64(10), time.Unix(1, 0), time.Unix(0, 0))
	tracker.recordM2M("p1", num.DecimalFromInt64(-200))
	tracker.recordM2M("p2", num.DecimalFromInt64(-100))
	tracker.recordPosition("p2", num.DecimalFromInt64(20), time.Unix(1, 0), time.Unix(0, 0))
	tracker.recordM2M("p3", num.DecimalFromInt64(200))
	tracker.recordPosition("p3", num.DecimalFromInt64(20), time.Unix(1, 0), time.Unix(0, 0))

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
	// (45/60)*(10*10/15*(15/45)+30/45*20)+15/60*30
	tracker.recordPosition("p1", num.DecimalFromInt64(10), time.Unix(5, 0), time.Unix(0, 0))
	tracker.recordPosition("p1", num.DecimalFromInt64(20), time.Unix(15, 0), time.Unix(0, 0))
	tracker.recordPosition("p1", num.DecimalFromInt64(30), time.Unix(45, 0), time.Unix(0, 0))

	tracker.processPositionEndOfEpoch(time.Unix(0, 0), time.Unix(60, 0))
	require.Equal(t, num.MustDecimalFromString("19.16666666666667").String(), tracker.getPositionMetricTotal("p1", 1).StringFixed(14))

	// epoch 2
	// (30 + 10)/2
	tracker.recordPosition("p1", num.DecimalFromInt64(-10), time.Unix(90, 0), time.Unix(60, 0))
	tracker.processPositionEndOfEpoch(time.Unix(60, 0), time.Unix(120, 0))
	require.Equal(t, num.MustDecimalFromString("20").String(), tracker.getPositionMetricTotal("p1", 1).String())
	require.Equal(t, num.MustDecimalFromString("39.16666666666667").String(), tracker.getPositionMetricTotal("p1", 2).StringFixed(14))

	// epoch 3
	// no position changes over the epoch
	tracker.processPositionEndOfEpoch(time.Unix(120, 0), time.Unix(180, 0))
	require.Equal(t, num.MustDecimalFromString("10").String(), tracker.getPositionMetricTotal("p1", 1).String())
	require.Equal(t, num.MustDecimalFromString("30").String(), tracker.getPositionMetricTotal("p1", 2).String())
	require.Equal(t, num.MustDecimalFromString("49.16666666666667").String(), tracker.getPositionMetricTotal("p1", 3).StringFixed(14))
}

func TestAverageNotional(t *testing.T) {
	tracker := getDefaultTracker(t)
	// epoch 1
	tracker.recordNotional("p1", num.DecimalFromInt64(10), num.NewUint(5), time.Unix(5, 0), time.Unix(0, 0))
	require.Equal(t, "0", tracker.twNotionalPosition["p1"].currentEpochTWNotional.String())

	// 0*(1/3)+10*2/3*5 = 33.33333333333334
	tracker.recordNotional("p1", num.DecimalFromInt64(20), num.NewUint(10), time.Unix(15, 0), time.Unix(0, 0))
	require.Equal(t, "33.33333333333334", tracker.twNotionalPosition["p1"].currentEpochTWNotional.StringFixed(14))

	// 33.333333333333335*(1-0.5)+20*0.5*10 = 116.6666666667
	tracker.recordNotional("p1", num.DecimalFromInt64(30), num.NewUint(20), time.Unix(30, 0), time.Unix(0, 0))
	require.Equal(t, "116.66666666666667", tracker.twNotionalPosition["p1"].currentEpochTWNotional.StringFixed(14))

	// 116.6666666666666675*(1-0.5)+30*0.5*20 = 358.3333333333
	tracker.processNotionalEndOfEpoch(time.Unix(0, 0), time.Unix(60, 0))
	require.Equal(t, "358.33333333333333", tracker.twNotionalPosition["p1"].currentEpochTWNotional.StringFixed(14))

	// epoch 2
	// 358.33333333333333375*(1-1)+30*1*20 = 600
	tracker.recordNotional("p1", num.DecimalFromInt64(-10), num.NewUint(30), time.Unix(90, 0), time.Unix(60, 0))
	require.Equal(t, "600", tracker.twNotionalPosition["p1"].currentEpochTWNotional.String())

	// 600*(1-0.5)+10*0.5*30 = 450
	tracker.processNotionalEndOfEpoch(time.Unix(60, 0), time.Unix(120, 0))
	require.Equal(t, "450", tracker.twNotionalPosition["p1"].currentEpochTWNotional.String())

	// epoch 3
	// no position changes over the epoch
	// 450*(1-1)+10*1*30 =
	tracker.processNotionalEndOfEpoch(time.Unix(120, 0), time.Unix(180, 0))
	require.Equal(t, "300", tracker.twNotionalPosition["p1"].currentEpochTWNotional.String())
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
	tracker.RecordPosition("a1", "p1", "m1", num.DecimalFromInt64(10), num.NewUint(1), time.Unix(5, 0))
	tracker.RecordPosition("a1", "p1", "m2", num.DecimalFromInt64(20), num.NewUint(2), time.Unix(20, 0))
	tracker.RecordPosition("a1", "p1", "m3", num.DecimalFromInt64(30), num.NewUint(3), time.Unix(30, 0))
	tracker.RecordPosition("a1", "p2", "m1", num.DecimalFromInt64(100), num.NewUint(10), time.Unix(15, 0))
	tracker.RecordPosition("a1", "p2", "m2", num.DecimalFromInt64(200), num.NewUint(20), time.Unix(25, 0))
	tracker.RecordPosition("a1", "p2", "m3", num.DecimalFromInt64(300), num.NewUint(30), time.Unix(45, 0))

	// end epoch1
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(0, 0), EndTime: time.Unix(60, 0)})

	// get metrics for market m1 with window size=1
	metrics := tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.DecimalZero(), 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "9.166666666666667", metrics[0].Score.String())
	require.Equal(t, "75", metrics[1].Score.String())

	// get metrics for market m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.DecimalZero(), 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "13.333333333333334", metrics[0].Score.String())
	require.Equal(t, "116.66666666666666", metrics[1].Score.String())

	// get metrics for market m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.DecimalZero(), 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "15", metrics[0].Score.String())
	require.Equal(t, "75", metrics[1].Score.String())

	// get metrics for market m1,m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.DecimalZero(), 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "22.5000000000", metrics[0].Score.StringFixed(10))
	require.Equal(t, "191.66666666666666", metrics[1].Score.String())

	// get metrics for all market window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.DecimalZero(), 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "37.5000000000", metrics[0].Score.StringFixed(10))
	require.Equal(t, "266.66666666666666", metrics[1].Score.String())

	// start epoch2
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(60, 0)})
	tracker.RecordPosition("a1", "p1", "m1", num.DecimalFromInt64(20), num.NewUint(5), time.Unix(90, 0))
	tracker.RecordPosition("a1", "p2", "m2", num.DecimalFromInt64(10), num.NewUint(10), time.Unix(75, 0))
	tracker.RecordPosition("a2", "p1", "m3", num.DecimalFromInt64(20), num.NewUint(5), time.Unix(90, 0))
	tracker.RecordPosition("a2", "p2", "m2", num.DecimalFromInt64(10), num.NewUint(10), time.Unix(75, 0))
	// end epoch2
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(60, 0), EndTime: time.Unix(120, 0)})

	// get metrics for market m1 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.DecimalZero(), 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "15", metrics[0].Score.String())
	require.Equal(t, "100", metrics[1].Score.String())

	// get metrics for market m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.DecimalZero(), 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "20", metrics[0].Score.String())
	require.Equal(t, "57.5", metrics[1].Score.String())

	// get metrics for market m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.DecimalZero(), 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "30", metrics[0].Score.String())
	require.Equal(t, "300", metrics[1].Score.String())

	// get metrics for market m1,m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.DecimalZero(), 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "35", metrics[0].Score.String())
	require.Equal(t, "157.5", metrics[1].Score.String())

	// get metrics for all market window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.DecimalZero(), 1)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "65", metrics[0].Score.String())
	require.Equal(t, "457.5", metrics[1].Score.String())

	// calc with window size = 2
	// get metrics for market m1 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.DecimalZero(), 2)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "12.0833333333333335", metrics[0].Score.String())
	require.Equal(t, "87.5", metrics[1].Score.String())

	// get metrics for market m2 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.DecimalZero(), 2)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "16.666666666666667", metrics[0].Score.String())
	require.Equal(t, "87.08333333333333", metrics[1].Score.String())

	// get metrics for market m2 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.DecimalZero(), 2)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "22.5", metrics[0].Score.String())
	require.Equal(t, "187.5", metrics[1].Score.String())

	// get metrics for market m1,m2 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.DecimalZero(), 2)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "28.7500000000", metrics[0].Score.StringFixed(10))
	require.Equal(t, "174.58333333333333", metrics[1].Score.String())

	// get metrics for all market window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintZero(), num.DecimalZero(), 2)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	require.Equal(t, "51.2500000000", metrics[0].Score.StringFixed(10))
	require.Equal(t, "362.08333333333333", metrics[1].Score.String())

	// now make p2 not eligible via not having sufficient governance token
	balanceChecker.EXPECT().GetAvailableBalance("p1").Return(num.NewUint(2), nil).Times(1)
	balanceChecker.EXPECT().GetAvailableBalance("p2").Return(nil, errors.New("some error")).Times(1)
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.NewUint(1), num.DecimalZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)

	// repeat now p2 has balance just not enough
	balanceChecker.EXPECT().GetAvailableBalance("p1").Return(num.NewUint(2), nil).Times(1)
	balanceChecker.EXPECT().GetAvailableBalance("p2").Return(num.NewUint(1), nil).Times(1)
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.NewUint(2), num.DecimalZero(), 2)
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
	tracker.RecordPosition("a1", "p1", "m1", num.DecimalFromInt64(10), num.NewUint(1), time.Unix(5, 0))
	tracker.RecordPosition("a1", "p1", "m2", num.DecimalFromInt64(20), num.NewUint(2), time.Unix(20, 0))
	tracker.RecordPosition("a1", "p1", "m3", num.DecimalFromInt64(30), num.NewUint(3), time.Unix(30, 0))
	tracker.RecordPosition("a1", "p2", "m1", num.DecimalFromInt64(100), num.NewUint(10), time.Unix(15, 0))
	tracker.RecordPosition("a1", "p2", "m2", num.DecimalFromInt64(200), num.NewUint(20), time.Unix(25, 0))
	tracker.RecordPosition("a1", "p2", "m3", num.DecimalFromInt64(300), num.NewUint(30), time.Unix(45, 0))

	// end epoch1
	epochService.target(context.Background(), types.Epoch{Seq: 1, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(0, 0), EndTime: time.Unix(60, 0)})

	// calculate metric for p1 with scope=[m1] for window size=1
	// 0*(1-0.9166666666666667)+10*0.9166666666666667 = 9.1666666667
	require.Equal(t, "9.166666666666667", tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p1 with scope=[m2] for window size=1
	// 0*(1-0.6666666666666667)+20*0.6666666666666667 = 13.3333333333
	require.Equal(t, "13.333333333333334", tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p1 with scope=[m3] for window size=1
	// 0*(1-0.5)+30*0.5 = 15
	require.Equal(t, "15", tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p1 with scope=[m1, m2] for window size=1
	require.Equal(t, "22.5000000000", tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).StringFixed(10))
	// calculate metric for p1 with no scope for window size=1
	require.Equal(t, "37.5000000000", tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).StringFixed(10))

	// calculate metric for p2 with scope=[m1] for window size=1
	// 0*(1-0.75)+100*0.75 = 75
	require.Equal(t, "75", tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p2 with scope=[m2] for window size=1
	// 0*(1-0.5833333333333333)+200*0.5833333333333333
	require.Equal(t, "116.66666666666666", tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p2 with scope=[m3] for window size=1
	// 0*(1-0.25)+300*0.25
	require.Equal(t, "75", tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p2 with scope=[m1, m3] for window size=1
	require.Equal(t, "150", tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())
	// calculate metric for p2 with no scope for window size=1
	require.Equal(t, "266.66666666666666", tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 1).String())

	// start epoch2
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(60, 0)})
	tracker.RecordPosition("a1", "p1", "m1", num.DecimalFromInt64(20), num.NewUint(5), time.Unix(90, 0))
	tracker.RecordPosition("a1", "p2", "m2", num.DecimalFromInt64(10), num.NewUint(10), time.Unix(75, 0))
	tracker.RecordPosition("a2", "p1", "m3", num.DecimalFromInt64(20), num.NewUint(5), time.Unix(90, 0))
	tracker.RecordPosition("a2", "p2", "m2", num.DecimalFromInt64(10), num.NewUint(10), time.Unix(75, 0))
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
	require.Equal(t, "12.0833333333333335", tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 2).String())
	// calculate metric for p1 with scope=[m2] for window size=2
	// (13.333333333333334" + 20)/2
	require.Equal(t, "16.666666666666667", tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 2).String())
	// calculate metric for p1 with scope=[m3] for window size=2
	// (15 + 30)/2
	require.Equal(t, "22.5", tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 2).String())
	// calculate metric for p1 with scope=[m1, m3] for window size=2
	require.Equal(t, "28.7500000000", tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 2).StringFixed(10))
	// calculate metric for p1 with no scope for window size=2
	require.Equal(t, "51.2500000000", tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 2).StringFixed(10))

	// calculate metric for p2 with scope=[m1] for window size=2
	// (100 + 75)/2
	require.Equal(t, "87.5", tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 2).String())
	// calculate metric for p2 with scope=[m2] for window size=2
	// (116.66666666666666 + 57.5)/2
	require.Equal(t, "87.08333333333333", tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 2).String())
	// calculate metric for p2 with scope=[m3] for window size=2
	// (300 + 75)/2
	require.Equal(t, "187.5", tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 2).String())
	// calculate metric for p2 with scope=[m1, m3] for window size=2
	require.Equal(t, "275", tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 2).String())
	// calculate metric for p2 with no scope for window size=2
	require.Equal(t, "362.08333333333333", tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 2).String())

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
	require.Equal(t, "14.7222222222222223", tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 3).String())
	// calculate metric for p1 with scope=[m2] for window size=3
	// (13.333333333333334" + 20 + 20)/3
	require.Equal(t, "17.777777777777778", tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 3).String())
	// calculate metric for p1 with scope=[m3] for window size=3
	// (15 + 30 + 30)/3
	require.Equal(t, "25", tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 3).String())
	// calculate metric for p1 with scope=[m1, m3] for window size=3
	require.Equal(t, "32.5000000000", tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 3).StringFixed(10))
	// calculate metric for p1 with no scope for window size=3
	require.Equal(t, "57.5000000000", tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 3).StringFixed(10))

	// calculate metric for p2 with scope=[m1] for window size=3
	// (100 + 75+100)/3
	require.Equal(t, "91.6666666666666667", tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 3).String())
	// calculate metric for p2 with scope=[m2] for window size=3
	// (116.66666666666666 + 57.5 + 10)/3
	require.Equal(t, "61.3888888888888867", tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 3).String())
	// calculate metric for p2 with scope=[m3] for window size=3
	// (300 + 75 + 300)/3
	require.Equal(t, "225", tracker.calculateMetricForParty("a1", "p2", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 3).String())
	// calculate metric for p2 with scope=[m1, m3] for window size=3
	require.Equal(t, "316.66666666666667", tracker.calculateMetricForParty("a1", "p2", []string{"m1", "m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 3).StringFixed(14))
	// calculate metric for p2 with no scope for window size=3
	require.Equal(t, "378.05555555555555", tracker.calculateMetricForParty("a1", "p2", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, 3).StringFixed(14))
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
	tracker.RecordPosition("a1", "p1", "m1", num.DecimalFromInt64(10), num.NewUint(1), time.Unix(5, 0))
	tracker.RecordPosition("a1", "p1", "m2", num.DecimalFromInt64(20), num.NewUint(2), time.Unix(20, 0))
	tracker.RecordPosition("a1", "p1", "m3", num.DecimalFromInt64(30), num.NewUint(3), time.Unix(30, 0))
	tracker.RecordPosition("a1", "p2", "m1", num.DecimalFromInt64(100), num.NewUint(10), time.Unix(15, 0))
	tracker.RecordPosition("a1", "p2", "m2", num.DecimalFromInt64(200), num.NewUint(20), time.Unix(25, 0))
	tracker.RecordPosition("a1", "p2", "m3", num.DecimalFromInt64(300), num.NewUint(30), time.Unix(45, 0))

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
	metrics := tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.DecimalZero(), 1)
	// only one sample (window size=1) variance=0 by definition
	require.Equal(t, 0, len(metrics))

	// get metrics for market m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.DecimalZero(), 1)
	// only one sample (window size=1) variance=0 by definition
	require.Equal(t, 0, len(metrics))

	// get metrics for market m3 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.DecimalZero(), 1)
	// only one sample (window size=1) variance=0 by definition
	require.Equal(t, 0, len(metrics))

	// get metrics for market m1,m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.DecimalZero(), 1)
	// only one sample (window size=1) variance=0 by definition
	require.Equal(t, 0, len(metrics))

	// get metrics for all market window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.DecimalZero(), 1)
	// only one sample (window size=1) variance=0 by definition
	require.Equal(t, 0, len(metrics))

	// // start epoch2
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(60, 0)})
	tracker.RecordPosition("a1", "p1", "m1", num.DecimalFromInt64(20), num.NewUint(5), time.Unix(90, 0))
	tracker.RecordPosition("a1", "p2", "m2", num.DecimalFromInt64(10), num.NewUint(10), time.Unix(75, 0))
	tracker.RecordPosition("a2", "p1", "m3", num.DecimalFromInt64(20), num.NewUint(5), time.Unix(90, 0))
	tracker.RecordPosition("a2", "p2", "m2", num.DecimalFromInt64(10), num.NewUint(10), time.Unix(75, 0))

	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(450))
	tracker.RecordM2M("a1", "p2", "m1", num.DecimalFromInt64(-450))
	tracker.RecordM2M("a1", "p1", "m2", num.DecimalFromInt64(-100))
	tracker.RecordM2M("a1", "p2", "m2", num.DecimalFromInt64(100))
	// nothing in m3

	// end epoch2
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(60, 0), EndTime: time.Unix(120, 0)})

	// calc with window size = 2
	// get metrics for market m1 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.DecimalZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	// variance(30, 16.3636363636)
	require.Equal(t, "46.4876033057851283", metrics[0].Score.String())

	// get metrics for market m2 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.DecimalZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p2", metrics[0].Party)
	// variance(1.7391304347826087,0.4285714285714286)
	require.Equal(t, "0.429391227190309", metrics[0].Score.String())

	// get metrics for market m3 with window size=2
	// variance(6.6666666666)
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.DecimalZero(), 2)
	require.Equal(t, 0, len(metrics))

	// get metrics for market m1,m2 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.DecimalZero(), 2)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	// variance(30, 16.363636363636363)
	require.Equal(t, "46.4876033057851283", metrics[0].Score.String())
	// variance(1.7391304347826087,0.4285714285714286)
	require.Equal(t, "0.429391227190309", metrics[1].Score.String())

	// get metrics for all market window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.UintZero(), num.DecimalZero(), 2)
	require.Equal(t, 2, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "p2", metrics[1].Party)
	// variance(30, 23.0303030303030297)
	require.Equal(t, "12.1441689623507826", metrics[0].Score.String())
	// variance(1.7391304347826087,0.4285714285714286)
	require.Equal(t, "0.429391227190309", metrics[1].Score.String())

	// now make p2 not eligible via not having sufficient governance token
	balanceChecker.EXPECT().GetAvailableBalance("p1").Return(num.NewUint(2), nil).Times(1)
	balanceChecker.EXPECT().GetAvailableBalance("p2").Return(nil, errors.New("some error")).Times(1)
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.NewUint(1), num.DecimalZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)

	// repeat now p2 has balance just not enough
	balanceChecker.EXPECT().GetAvailableBalance("p1").Return(num.NewUint(2), nil).Times(1)
	balanceChecker.EXPECT().GetAvailableBalance("p2").Return(num.NewUint(1), nil).Times(1)
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY, num.NewUint(2), num.DecimalZero(), 2)
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
	tracker.RecordPosition("a1", "p1", "m1", num.DecimalFromInt64(10), num.NewUint(1), time.Unix(5, 0))
	tracker.RecordPosition("a1", "p1", "m2", num.DecimalFromInt64(20), num.NewUint(2), time.Unix(20, 0))
	tracker.RecordPosition("a1", "p1", "m3", num.DecimalFromInt64(30), num.NewUint(3), time.Unix(30, 0))
	tracker.RecordPosition("a1", "p2", "m1", num.DecimalFromInt64(100), num.NewUint(10), time.Unix(15, 0))
	tracker.RecordPosition("a1", "p2", "m2", num.DecimalFromInt64(200), num.NewUint(20), time.Unix(25, 0))
	tracker.RecordPosition("a1", "p2", "m3", num.DecimalFromInt64(300), num.NewUint(30), time.Unix(45, 0))

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
	metrics := tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.DecimalZero(), 1)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "16.363636363636363", metrics[0].Score.String())

	// get metrics for market m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.DecimalZero(), 1)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p2", metrics[0].Party)
	require.Equal(t, "0.4285714285714286", metrics[0].Score.String())

	// get metrics for market m3 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.DecimalZero(), 1)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "6.6666666666666667", metrics[0].Score.String())

	// get metrics for market m1,m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.DecimalZero(), 1)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "12.6136363636", metrics[0].Score.StringFixed(10))

	// get metrics for all market window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.DecimalZero(), 1)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "19.2803030303", metrics[0].Score.StringFixed(10))

	// start epoch2
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_START, StartTime: time.Unix(60, 0)})
	tracker.RecordPosition("a1", "p1", "m1", num.DecimalFromInt64(20), num.NewUint(5), time.Unix(90, 0))
	tracker.RecordPosition("a1", "p2", "m2", num.DecimalFromInt64(10), num.NewUint(10), time.Unix(75, 0))
	tracker.RecordPosition("a2", "p1", "m3", num.DecimalFromInt64(20), num.NewUint(5), time.Unix(90, 0))
	tracker.RecordPosition("a2", "p2", "m2", num.DecimalFromInt64(10), num.NewUint(10), time.Unix(75, 0))

	tracker.RecordM2M("a1", "p1", "m1", num.DecimalFromInt64(450))
	tracker.RecordM2M("a1", "p2", "m1", num.DecimalFromInt64(-450))
	tracker.RecordM2M("a1", "p1", "m2", num.DecimalFromInt64(-100))
	tracker.RecordM2M("a1", "p2", "m2", num.DecimalFromInt64(100))
	// nothing in m3

	// end epoch2
	epochService.target(context.Background(), types.Epoch{Seq: 2, Action: vgproto.EpochAction_EPOCH_ACTION_END, StartTime: time.Unix(60, 0), EndTime: time.Unix(120, 0)})

	// get metrics for market m1 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.DecimalZero(), 1)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "30", metrics[0].Score.String())

	// get metrics for market m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.DecimalZero(), 1)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p2", metrics[0].Party)
	require.Equal(t, "1.7391304347826087", metrics[0].Score.String())

	// get metrics for market m3 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.DecimalZero(), 1)
	require.Equal(t, 0, len(metrics))

	// get metrics for market m1,m2 with window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.DecimalZero(), 1)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "25", metrics[0].Score.String())

	// get metrics for all market window size=1
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.DecimalZero(), 1)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "25", metrics[0].Score.String())

	// calc with window size = 2
	// get metrics for market m1 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.DecimalZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "23.1818181818181815", metrics[0].Score.String())

	// get metrics for market m2 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.DecimalZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p2", metrics[0].Party)
	require.Equal(t, "1.0838509316770187", metrics[0].Score.String())

	// get metrics for market m3 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.DecimalZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "3.3333333333333334", metrics[0].Score.String())

	// get metrics for market m1,m2 with window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.DecimalZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "18.8068181818", metrics[0].Score.StringFixed(10))

	// get metrics for all market window size=2
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.UintZero(), num.DecimalZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)
	require.Equal(t, "22.1401515152", metrics[0].Score.StringFixed(10))

	// now make p2 not eligible via not having sufficient governance token
	balanceChecker.EXPECT().GetAvailableBalance("p1").Return(num.NewUint(2), nil).Times(1)
	balanceChecker.EXPECT().GetAvailableBalance("p2").Return(nil, errors.New("some error")).Times(1)
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.NewUint(1), num.DecimalZero(), 2)
	require.Equal(t, 1, len(metrics))
	require.Equal(t, "p1", metrics[0].Party)

	// repeat now p2 has balance just not enough
	balanceChecker.EXPECT().GetAvailableBalance("p1").Return(num.NewUint(2), nil).Times(1)
	balanceChecker.EXPECT().GetAvailableBalance("p2").Return(num.NewUint(1), nil).Times(1)
	metrics = tracker.calculateMetricForIndividuals("a1", []string{"p1", "p2"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, num.NewUint(2), num.DecimalZero(), 2)
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
	tracker.RecordPosition("a1", "p1", "m1", num.DecimalFromInt64(10), num.NewUint(1), time.Unix(5, 0))
	tracker.RecordPosition("a1", "p1", "m2", num.DecimalFromInt64(20), num.NewUint(2), time.Unix(20, 0))
	tracker.RecordPosition("a1", "p1", "m3", num.DecimalFromInt64(30), num.NewUint(3), time.Unix(30, 0))
	tracker.RecordPosition("a1", "p2", "m1", num.DecimalFromInt64(100), num.NewUint(10), time.Unix(15, 0))
	tracker.RecordPosition("a1", "p2", "m2", num.DecimalFromInt64(200), num.NewUint(20), time.Unix(25, 0))
	tracker.RecordPosition("a1", "p2", "m3", num.DecimalFromInt64(300), num.NewUint(30), time.Unix(45, 0))

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
	// 150 / 9.166666666666667
	require.Equal(t, "16.363636363636363", tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())
	// calculate metric for p1 with scope=[m2] for window size=1
	// max(0, -50 /13.333333333333334)
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())
	// calculate metric for p1 with scope=[m3] for window size=1
	// 100 / 15
	require.Equal(t, "6.6666666666666667", tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())
	// calculate metric for p1 with scope=[m1, m2] for window size=1
	// 150 / 9.166666666666667 - 50 /13.333333333333334
	require.Equal(t, "12.6136363636", tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).StringFixed(10))
	// 150 / 9.166666666666667 - 50 /13.333333333333334 +100/15
	require.Equal(t, "19.2803030303", tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).StringFixed(10))

	// calculate metric for p2 with scope=[m1] for window size=1
	// max(-100 / 75, 0)
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())
	// calculate metric for p2 with scope=[m2] for window size=1
	// 50 / 116.66666666666666
	require.Equal(t, "0.4285714285714286", tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 1).String())
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
	tracker.RecordPosition("a1", "p1", "m1", num.DecimalFromInt64(20), num.NewUint(5), time.Unix(90, 0))
	tracker.RecordPosition("a1", "p2", "m2", num.DecimalFromInt64(10), num.NewUint(10), time.Unix(75, 0))
	tracker.RecordPosition("a2", "p1", "m3", num.DecimalFromInt64(20), num.NewUint(5), time.Unix(90, 0))
	tracker.RecordPosition("a2", "p2", "m2", num.DecimalFromInt64(10), num.NewUint(10), time.Unix(75, 0))

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
	// (16.363636363636363 + 30)/2
	require.Equal(t, "23.1818181818181815", tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2).String())
	// calculate metric for p1 with scope=[m2] for window size=2
	// (0 + 0)/2
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2).String())
	// calculate metric for p1 with scope=[m3] for window size=2
	// (6.6666666666666667 + 0)/2
	require.Equal(t, "3.3333333333333334", tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2).String())
	// calculate metric for p1 with scope=[m1, m3] for window size=2
	require.Equal(t, "18.8068181818", tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2).StringFixed(10))
	// calculate metric for p1 with no scope for window size=2
	require.Equal(t, "22.1401515152", tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2).StringFixed(10))

	// calculate metric for p2 with scope=[m1] for window size=2
	// (0 + 0)/2
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2).String())
	// calculate metric for p2 with scope=[m2] for window size=2
	// (0.4285714285714286 + 1.7391304347826087)/2
	require.Equal(t, "1.0838509316770187", tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 2).String())
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
	require.Equal(t, "15.4545454545454543", tracker.calculateMetricForParty("a1", "p1", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3).String())
	// calculate metric for p1 with scope=[m2] for window size=3
	// (0+0+0)/3
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p1", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3).String())
	// calculate metric for p1 with scope=[m3] for window size=3
	// (6.6666666666666667 + 0)/3
	require.Equal(t, "2.2222222222222222", tracker.calculateMetricForParty("a1", "p1", []string{"m3"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3).String())
	// calculate metric for p1 with scope=[m1, m3] for window size=3
	require.Equal(t, "12.5378787879", tracker.calculateMetricForParty("a1", "p1", []string{"m1", "m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3).StringFixed(10))
	// calculate metric for p1 with no scope for window size=3
	require.Equal(t, "14.7601010101", tracker.calculateMetricForParty("a1", "p1", []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3).StringFixed(10))

	// calculate metric for p2 with scope=[m1] for window size=3
	// (0 +0+0)/3
	require.Equal(t, "0", tracker.calculateMetricForParty("a1", "p2", []string{"m1"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3).String())
	// calculate metric for p2 with scope=[m2] for window size=3
	// (0.4285714285714286 + 1.7391304347826087 + 0 )/3
	require.Equal(t, "0.7225672877846791", tracker.calculateMetricForParty("a1", "p2", []string{"m2"}, vgproto.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN, 3).String())
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
	isEligible := func(asset, party string, markets []string, minStakingBalanceRequired *num.Uint, notionalTimeWeightedAveragePositionRequired num.Decimal) bool {
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

	teamScore, partyScores := calculateMetricForTeamUtil("asset1", []string{"party1", "party2", "party3", "party4", "party5"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintOne(), num.DecimalOne(), 5, num.DecimalFromFloat(0.5), isEligible, calculateMetricForParty)
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
	teamScore, partyScores = calculateMetricForTeamUtil("asset1", []string{"party5"}, []string{}, vgproto.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION, num.UintOne(), num.DecimalOne(), 5, num.DecimalFromFloat(0.5), isEligible, calculateMetricForParty)
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
