// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package products_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/datasource"
	dscommon "code.vegaprotocol.io/vega/core/datasource/common"
	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/external/signedoracle"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/products"
	"code.vegaprotocol.io/vega/core/products/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeriodicSettlement(t *testing.T) {
	t.Run("incoming data ignored before leaving opening auction", testIncomingDataIgnoredBeforeLeavingOpeningAuction)
	t.Run("period end with no data point", testPeriodEndWithNoDataPoints)
	t.Run("equal internal and external prices", testEqualInternalAndExternalPrices)
	t.Run("constant difference long pays short", testConstantDifferenceLongPaysShort)
	t.Run("data points outside of period", testDataPointsOutsidePeriod)
	t.Run("data points not on boundary", testDataPointsNotOnBoundary)
	t.Run("matching data points outside of period through callbacks", testRegisteredCallbacks)
	t.Run("non-matching data points outside of period through callbacks", testRegisteredCallbacksWithDifferentData)
	t.Run("funding payments with interest rate", testFundingPaymentsWithInterestRate)
	t.Run("funding payments with interest rate clamped", testFundingPaymentsWithInterestRateClamped)
	t.Run("terminate perps market test", testTerminateTrading)
	t.Run("margin increase", testGetMarginIncrease)
}

func TestExternalDataPointTWAPInSequence(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()

	ctx := context.Background()
	tstData, err := getGQLData()
	require.NoError(t, err)
	data := tstData.GetDataPoints()
	perp.broker.EXPECT().Send(gomock.Any()).Times(1)
	// leave opening auction
	perp.perpetual.OnLeaveOpeningAuction(ctx, data[0].t-1)

	seq := data[0].seq
	// set the first internal data-point
	// perp.broker.EXPECT().Send(gomock.Any()).Times(1)
	// perp.perpetual.SubmitDataPoint(ctx, data[0].price.Clone(), data[0].t)
	for i, dp := range data {
		if dp.seq > seq {
			perp.broker.EXPECT().Send(gomock.Any()).Times(2)
			if dp.seq == 2 {
				perp.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(func(evts []events.Event) {
					fmt.Printf("SEQ: %d\n%#v\n", dp.seq, evts)
				})
			}
			perp.perpetual.PromptSettlementCue(ctx, dp.t)
			seq = dp.seq
		}
		check := func(e events.Event) {
			de, ok := e.(*events.FundingPeriodDataPoint)
			require.True(t, ok)
			dep := de.Proto()
			if dep.Twap == "0" {
				return
			}
			require.Equal(t, dp.twap.String(), dep.Twap, fmt.Sprintf("IDX: %d\n%#v\n", i, dep))
		}
		perp.broker.EXPECT().Send(gomock.Any()).Times(1).Do(check)
		perp.perpetual.AddTestExternalPoint(ctx, dp.price, dp.t)
	}
}

func TestExternalDataPointTWAPOutSequence(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()

	ctx := context.Background()
	tstData, err := getGQLData()
	require.NoError(t, err)
	data := tstData.GetDataPoints()
	perp.broker.EXPECT().Send(gomock.Any()).Times(1)
	// leave opening auction
	perp.perpetual.OnLeaveOpeningAuction(ctx, data[0].t-1)

	seq := data[0].seq
	last := 0
	for i := 0; i < len(data); i++ {
		if data[i].seq != seq {
			break
		}
		last = i
	}
	perp.broker.EXPECT().Send(gomock.Any()).Times(1)
	// add the first (earliest) data-point first
	perp.perpetual.AddTestExternalPoint(ctx, data[0].price, data[0].t)
	// submit external data points in non-sequential order
	for j := last; j < 0; j-- {
		dp := data[j]
		if dp.seq > seq {
			// break
			perp.broker.EXPECT().Send(gomock.Any()).Times(2)
			perp.perpetual.PromptSettlementCue(ctx, dp.t)
		}
		check := func(e events.Event) {
			de, ok := e.(*events.FundingPeriodDataPoint)
			require.True(t, ok)
			dep := de.Proto()
			if dep.Twap == "0" {
				return
			}
			require.Equal(t, dp.twap.String(), dep.Twap, fmt.Sprintf("IDX: %d\n%#v\n", j, dep))
		}
		perp.broker.EXPECT().Send(gomock.Any()).Times(1).Do(check)
		perp.perpetual.AddTestExternalPoint(ctx, dp.price, dp.t)
	}
}

func testIncomingDataIgnoredBeforeLeavingOpeningAuction(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()

	ctx := context.Background()

	// no error because its really a callback from the oracle engine, but we expect no events
	perp.perpetual.AddTestExternalPoint(ctx, num.UintOne(), 2000)

	err := perp.perpetual.SubmitDataPoint(ctx, num.UintOne(), 2000)
	assert.ErrorIs(t, err, products.ErrInitialPeriodNotStarted)

	// check that settlement cues are ignored, we expect no events when it is
	perp.perpetual.PromptSettlementCue(ctx, 4000)
}

func testPeriodEndWithNoDataPoints(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()

	ctx := context.Background()

	// funding payment will be zero because there are no data points
	var called bool
	fn := func(context.Context, *num.Numeric) {
		called = true
	}
	perp.perpetual.SetSettlementListener(fn)

	perp.broker.EXPECT().Send(gomock.Any()).Times(2)
	perp.perpetual.OnLeaveOpeningAuction(ctx, 1000)

	perp.broker.EXPECT().Send(gomock.Any()).Times(1)
	perp.perpetual.PromptSettlementCue(ctx, 1040)

	// we had no points to check we didn't call into listener
	assert.False(t, called)
}

func testEqualInternalAndExternalPrices(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()
	ctx := context.Background()

	// set of the data points such that difference in averages is 0
	points := getTestDataPoints(t)

	// tell the perpetual that we are ready to accept settlement stuff
	perp.broker.EXPECT().Send(gomock.Any()).Times(1)
	perp.perpetual.OnLeaveOpeningAuction(ctx, points[0].t)

	// send in some data points
	perp.broker.EXPECT().Send(gomock.Any()).Times(len(points) * 2)
	for _, p := range points {
		// send in an external and a matching internal
		require.NoError(t, perp.perpetual.SubmitDataPoint(ctx, p.price, p.t))
		perp.perpetual.AddTestExternalPoint(ctx, p.price, p.t)
	}

	// ask for the funding payment
	var fundingPayment *num.Numeric
	fn := func(_ context.Context, fp *num.Numeric) {
		fundingPayment = fp
	}
	perp.perpetual.SetSettlementListener(fn)

	perp.broker.EXPECT().Send(gomock.Any()).Times(2)
	perp.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	perp.perpetual.PromptSettlementCue(ctx, points[len(points)-1].t)
	assert.NotNil(t, fundingPayment)
	assert.True(t, fundingPayment.IsInt())
	assert.Equal(t, "0", fundingPayment.String())
}

func testConstantDifferenceLongPaysShort(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()
	ctx := context.Background()

	// test data
	points := getTestDataPoints(t)

	// when: the funding period starts at 1000
	perp.broker.EXPECT().Send(gomock.Any()).Times(1)
	perp.perpetual.OnLeaveOpeningAuction(ctx, 1000)

	// and: the difference in external/internal prices are a constant -10
	submitDataWithDifference(t, perp, points, -10)

	// funding payment will be zero so no transfers
	var fundingPayment *num.Numeric
	fn := func(_ context.Context, fp *num.Numeric) {
		fundingPayment = fp
	}
	perp.perpetual.SetSettlementListener(fn)

	perp.broker.EXPECT().Send(gomock.Any()).Times(2)
	perp.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	perp.perpetual.PromptSettlementCue(ctx, points[len(points)-1].t)
	assert.NotNil(t, fundingPayment)
	assert.True(t, fundingPayment.IsInt())
	assert.Equal(t, "-10", fundingPayment.String())
}

func testDataPointsOutsidePeriod(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()
	ctx := context.Background()

	// set of the data points such that difference in averages is 0
	points := getTestDataPoints(t)

	// tell the perpetual that we are ready to accept settlement stuff
	perp.broker.EXPECT().Send(gomock.Any()).Times(1)
	perp.perpetual.OnLeaveOpeningAuction(ctx, points[0].t)

	// add data-points from the past, they will just be ignored
	perp.broker.EXPECT().Send(gomock.Any()).Times(2)
	require.NoError(t, perp.perpetual.SubmitDataPoint(ctx, num.UintOne(), points[0].t-int64(time.Hour)))
	perp.perpetual.AddTestExternalPoint(ctx, num.UintZero(), points[0].t-int64(time.Hour))

	// send in some data points
	perp.broker.EXPECT().Send(gomock.Any()).Times(len(points) * 2)
	for _, p := range points {
		// send in an external and a matching internal
		require.NoError(t, perp.perpetual.SubmitDataPoint(ctx, p.price, p.t))
		perp.perpetual.AddTestExternalPoint(ctx, p.price, p.t)
	}

	// add some data-points in the future from when we will cue the end of the funding period
	// they should not affect the funding payment of this period
	lastPoint := points[len(points)-1]
	perp.broker.EXPECT().Send(gomock.Any()).Times(2)
	require.NoError(t, perp.perpetual.SubmitDataPoint(ctx, num.UintOne(), lastPoint.t+int64(time.Hour)))
	perp.perpetual.AddTestExternalPoint(ctx, num.UintZero(), lastPoint.t+int64(time.Hour))

	// ask for the funding payment
	var fundingPayment *num.Numeric
	fn := func(_ context.Context, fp *num.Numeric) {
		fundingPayment = fp
	}
	perp.perpetual.SetSettlementListener(fn)

	// 6 times because: end + start of the period, plus 2 carry over points for external + internal (total 4)
	perp.broker.EXPECT().Send(gomock.Any()).Times(2)
	perp.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(func(evts []events.Event) {
		require.Equal(t, 4, len(evts)) // 4 carry over points
	})
	perp.perpetual.PromptSettlementCue(ctx, lastPoint.t)
	assert.NotNil(t, fundingPayment)
	assert.True(t, fundingPayment.IsInt())
	assert.Equal(t, "0", fundingPayment.String())
}

func testDataPointsNotOnBoundary(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()
	ctx := context.Background()

	// set of the data points such that difference in averages is 0
	points := getTestDataPoints(t)

	// start time is *after* our first data points
	perp.broker.EXPECT().Send(gomock.Any()).Times(1)
	perp.perpetual.OnLeaveOpeningAuction(ctx, 1005)

	// send in some data points
	submitDataWithDifference(t, perp, points, 10)

	// ask for the funding payment
	var fundingPayment *num.Numeric
	fn := func(_ context.Context, fp *num.Numeric) {
		fundingPayment = fp
	}
	perp.perpetual.SetSettlementListener(fn)

	// period end is *after* our last point
	perp.broker.EXPECT().Send(gomock.Any()).Times(2)
	perp.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	perp.perpetual.PromptSettlementCue(ctx, points[len(points)-1].t+int64(time.Hour))
	assert.NotNil(t, fundingPayment)
	assert.True(t, fundingPayment.IsInt())
	assert.Equal(t, "10", fundingPayment.String())
}

func testRegisteredCallbacks(t *testing.T) {
	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	oe := mocks.NewMockOracleEngine(ctrl)
	broker := mocks.NewMockBroker(ctrl)
	exp := &num.Numeric{}
	exp.SetUint(num.UintZero())
	ctx := context.Background()
	received := false
	points := getTestDataPoints(t)
	marketSettle := func(_ context.Context, data *num.Numeric) {
		received = true
		require.Equal(t, exp.String(), data.String())
	}
	var settle, period spec.OnMatchedData
	oe.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).Times(2).DoAndReturn(func(_ context.Context, s spec.Spec, cb spec.OnMatchedData) (spec.SubscriptionID, spec.Unsubscriber, error) {
		filters := s.OriginalSpec.GetDefinition().DataSourceType.GetFilters()
		for _, f := range filters {
			if f.Key.Type == datapb.PropertyKey_TYPE_INTEGER || f.Key.Type == datapb.PropertyKey_TYPE_DECIMAL {
				settle = cb
				return spec.SubscriptionID(1), func(_ context.Context, _ spec.SubscriptionID) {}, nil
			}
		}
		period = cb
		return spec.SubscriptionID(1), func(_ context.Context, _ spec.SubscriptionID) {}, nil
	})
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	perp := getTestPerpProd(t)
	perpetual, err := products.NewPerpetual(context.Background(), log, perp, "", oe, broker, 1)
	require.NoError(t, err)
	require.NotNil(t, settle)
	require.NotNil(t, period)
	// register the callback
	perpetual.NotifyOnSettlementData(marketSettle)

	perpetual.OnLeaveOpeningAuction(ctx, points[0].t)

	for _, p := range points {
		// send in an external and a matching internal
		require.NoError(t, perpetual.SubmitDataPoint(ctx, p.price, p.t))
		settle(ctx, dscommon.Data{
			Data: map[string]string{
				perp.DataSourceSpecBinding.SettlementDataProperty: p.price.String(),
			},
			MetaData: map[string]string{
				"eth-block-time": fmt.Sprintf("%d", time.Unix(0, p.t).Unix()),
			},
		})
	}
	// add some data-points in the future from when we will cue the end of the funding period
	// they should not affect the funding payment of this period
	lastPoint := points[len(points)-1]
	require.NoError(t, perpetual.SubmitDataPoint(ctx, num.UintOne(), lastPoint.t+int64(time.Hour)))
	settle(ctx, dscommon.Data{
		Data: map[string]string{
			perp.DataSourceSpecBinding.SettlementDataProperty: "1",
		},
		MetaData: map[string]string{
			"eth-block-time": fmt.Sprintf("%d", time.Unix(0, lastPoint.t+int64(time.Hour)).Unix()),
		},
	})
	// make sure the data-point outside of the period doesn't trigger the schedule callback
	// that has to come from the oracle, too
	assert.False(t, received)

	// end period
	period(ctx, dscommon.Data{
		Data: map[string]string{
			perp.DataSourceSpecBinding.SettlementScheduleProperty: fmt.Sprintf("%d", time.Unix(0, lastPoint.t).Unix()),
		},
	})

	assert.True(t, received)
}

func testRegisteredCallbacksWithDifferentData(t *testing.T) {
	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	oe := mocks.NewMockOracleEngine(ctrl)
	broker := mocks.NewMockBroker(ctrl)
	exp := &num.Numeric{}
	// should be 2
	res, _ := num.IntFromString("-4", 10)
	exp.SetInt(res)
	ctx := context.Background()
	received := false
	points := getTestDataPoints(t)
	marketSettle := func(_ context.Context, data *num.Numeric) {
		received = true
		require.Equal(t, exp.String(), data.String())
	}
	var settle, period spec.OnMatchedData
	oe.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).Times(2).DoAndReturn(func(_ context.Context, s spec.Spec, cb spec.OnMatchedData) (spec.SubscriptionID, spec.Unsubscriber, error) {
		filters := s.OriginalSpec.GetDefinition().DataSourceType.GetFilters()
		for _, f := range filters {
			if f.Key.Type == datapb.PropertyKey_TYPE_INTEGER || f.Key.Type == datapb.PropertyKey_TYPE_DECIMAL {
				settle = cb
				return spec.SubscriptionID(1), func(_ context.Context, _ spec.SubscriptionID) {}, nil
			}
		}
		period = cb
		return spec.SubscriptionID(1), func(_ context.Context, _ spec.SubscriptionID) {}, nil
	})
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	perp := getTestPerpProd(t)
	perpetual, err := products.NewPerpetual(context.Background(), log, perp, "", oe, broker, 1)
	require.NoError(t, err)
	require.NotNil(t, settle)
	require.NotNil(t, period)
	// register the callback
	perpetual.NotifyOnSettlementData(marketSettle)

	// start the funding period
	perpetual.OnLeaveOpeningAuction(ctx, points[0].t)

	// send data in from before the start of the period, it should not affect the result
	require.NoError(t, perpetual.SubmitDataPoint(ctx, num.UintOne(), points[0].t-int64(time.Hour)))
	// callback to receive settlement data
	settle(ctx, dscommon.Data{
		Data: map[string]string{
			perp.DataSourceSpecBinding.SettlementDataProperty: "1",
		},
		MetaData: map[string]string{
			"eth-block-time": fmt.Sprintf("%d", time.Unix(0, points[0].t-int64(time.Hour)).Unix()),
		},
	})

	// send all external points, but not all internal ones and have their price
	// be one less. This means external twap > internal tswap so we expect a negative funding rate
	for i, p := range points {
		if i%2 == 0 {
			ip := num.UintZero().Sub(p.price, num.UintOne())
			require.NoError(t, perpetual.SubmitDataPoint(ctx, ip, p.t))
		}
		settle(ctx, dscommon.Data{
			Data: map[string]string{
				perp.DataSourceSpecBinding.SettlementDataProperty: p.price.String(),
			},
			MetaData: map[string]string{
				"eth-block-time": fmt.Sprintf("%d", time.Unix(0, p.t).Unix()),
			},
		})
	}

	// add some data-points in the future from when we will cue the end of the funding period
	// they should not affect the funding payment of this period
	lastPoint := points[len(points)-1]
	require.NoError(t, perpetual.SubmitDataPoint(ctx, num.UintOne(), lastPoint.t+int64(time.Hour)))
	settle(ctx, dscommon.Data{
		Data: map[string]string{
			perp.DataSourceSpecBinding.SettlementDataProperty: "1",
		},
		MetaData: map[string]string{
			"eth-block-time": fmt.Sprintf("%d", time.Unix(0, lastPoint.t+int64(time.Hour)).Unix()),
		},
	})

	// end period
	period(ctx, dscommon.Data{
		Data: map[string]string{
			perp.DataSourceSpecBinding.SettlementScheduleProperty: fmt.Sprintf("%d", time.Unix(0, lastPoint.t).Unix()),
		},
	})

	assert.True(t, received)
}

func testFundingPaymentsWithInterestRate(t *testing.T) {
	perp := testPerpetualWithOpts(t, "0.01", "-1", "1", "0")
	defer perp.ctrl.Finish()
	ctx := context.Background()

	// test data
	points := getTestDataPoints(t)
	lastPoint := points[len(points)-1]

	// when: the funding period starts
	perp.broker.EXPECT().Send(gomock.Any()).Times(1)
	perp.perpetual.OnLeaveOpeningAuction(ctx, points[0].t)

	// scale the price so that we have more precision to work with
	scale := num.UintFromUint64(100000000000)
	for _, p := range points {
		p.price = num.UintZero().Mul(p.price, scale)
	}

	// and: the difference in external/internal prices are a constant -10
	submitDataWithDifference(t, perp, points, -1000000000000)

	// Whats happening:
	// the fundingPayment without the interest terms will be -10000000000000
	//
	// interest will be (1 + r * t) * swap - fswap
	// where r = 0.01, t = 0.25, stwap = 11666666666666, ftwap = 10666666666656,
	// interest = (1 + 0.0025) * 11666666666666 - 10666666666656 = 1029166666666

	// since lower clamp <    interest   < upper clamp
	//   -11666666666666 < 1029166666666 < 11666666666666
	// there is no adjustment and so
	// funding payment = -10000000000000 + 1029166666666 = 29166666666

	var fundingPayment *num.Numeric
	fn := func(_ context.Context, fp *num.Numeric) {
		fundingPayment = fp
	}
	perp.perpetual.SetSettlementListener(fn)

	perp.broker.EXPECT().Send(gomock.Any()).Times(2)
	perp.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	perp.perpetual.PromptSettlementCue(ctx, lastPoint.t)
	assert.NotNil(t, fundingPayment)
	assert.True(t, fundingPayment.IsInt())
	assert.Equal(t, "29166666666", fundingPayment.String())
}

func testFundingPaymentsWithInterestRateClamped(t *testing.T) {
	perp := testPerpetualWithOpts(t, "0.5", "0.001", "0.002", "0")
	defer perp.ctrl.Finish()
	ctx := context.Background()

	// test data
	points := getTestDataPoints(t)

	// when: the funding period starts
	perp.broker.EXPECT().Send(gomock.Any()).Times(1)
	perp.perpetual.OnLeaveOpeningAuction(ctx, points[0].t)

	// scale the price so that we have more precision to work with
	scale := num.UintFromUint64(100000000000)
	for _, p := range points {
		p.price = num.UintZero().Mul(p.price, scale)
	}

	// and: the difference in external/internal prices are a constant -10
	submitDataWithDifference(t, perp, points, -10)

	// Whats happening:
	// the fundingPayment will be -10 without the interest terms
	//
	// interest will be (1 + r * t) * swap - fswap
	// where stwap=116, ftwap=106, r=0.5 t=0.25
	// interest = (1 + 0.125) * 11666666666666 - 11666666666656 = 1458333333343

	// if we consider the clamps:
	// lower clamp:   11666666666
	// interest:    1458333333343
	// upper clamp:   23333333333

	// so we have exceeded the upper clamp the the interest term is snapped to it and so:
	// funding payment = -10 + 23333333333 = 23333333323

	var fundingPayment *num.Numeric
	fn := func(_ context.Context, fp *num.Numeric) {
		fundingPayment = fp
	}
	perp.perpetual.SetSettlementListener(fn)

	perp.broker.EXPECT().Send(gomock.Any()).Times(2)
	perp.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	perp.perpetual.PromptSettlementCue(ctx, points[3].t)
	assert.NotNil(t, fundingPayment)
	assert.True(t, fundingPayment.IsInt())
	assert.Equal(t, "23333333323", fundingPayment.String())
}

func testTerminateTrading(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()
	ctx := context.Background()

	// set of the data points such that difference in averages is 0
	points := getTestDataPoints(t)

	// tell the perpetual that we are ready to accept settlement stuff
	perp.broker.EXPECT().Send(gomock.Any()).Times(1)
	perp.perpetual.OnLeaveOpeningAuction(ctx, points[0].t)

	// send in some data points
	perp.broker.EXPECT().Send(gomock.Any()).Times(len(points) * 2)
	for _, p := range points {
		// send in an external and a matching internal
		require.NoError(t, perp.perpetual.SubmitDataPoint(ctx, p.price, p.t))
		perp.perpetual.AddTestExternalPoint(ctx, p.price, p.t)
	}

	// ask for the funding payment
	var fundingPayment *num.Numeric
	fn := func(_ context.Context, fp *num.Numeric) {
		fundingPayment = fp
	}
	perp.perpetual.SetSettlementListener(fn)

	perp.broker.EXPECT().Send(gomock.Any()).Times(2)
	perp.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	perp.perpetual.UnsubscribeTradingTerminated(ctx)
	assert.NotNil(t, fundingPayment)
	assert.True(t, fundingPayment.IsInt())
	assert.Equal(t, "0", fundingPayment.String())
}

func testGetMarginIncrease(t *testing.T) {
	// margin factor is 0.5
	perp := testPerpetualWithOpts(t, "0", "0", "0", "0.5")
	defer perp.ctrl.Finish()
	ctx := context.Background()

	// test data
	points := getTestDataPoints(t)

	// before we've started the first funding interval margin increase is 0
	inc := perp.perpetual.GetMarginIncrease(points[0].t)
	assert.Equal(t, "0", inc.String())

	// start funding period
	perp.broker.EXPECT().Send(gomock.Any()).Times(1)
	perp.perpetual.OnLeaveOpeningAuction(ctx, 1000)

	// started interval, but not points, margin increase is 0
	inc = perp.perpetual.GetMarginIncrease(points[0].t)
	assert.Equal(t, "0", inc.String())

	// and: the difference in external/internal prices are is 10
	submitDataWithDifference(t, perp, points, 10)

	lastPoint := points[len(points)-1]
	inc = perp.perpetual.GetMarginIncrease(lastPoint.t)
	// margin increase is margin_factor * funding-payment = 0.5 * 10
	assert.Equal(t, "5", inc.String())
}

// submits the given data points as both external and interval but with the given different added to the internal price.
func submitDataWithDifference(t *testing.T, perp *tstPerp, points []*testDataPoint, diff int) {
	t.Helper()
	ctx := context.Background()

	var internalPrice *num.Uint
	perp.broker.EXPECT().Send(gomock.Any()).Times(len(points) * 2)
	for _, p := range points {
		perp.perpetual.AddTestExternalPoint(ctx, p.price, p.t)

		if diff > 0 {
			internalPrice = num.UintZero().Add(p.price, num.NewUint(uint64(diff)))
		}
		if diff < 0 {
			internalPrice = num.UintZero().Sub(p.price, num.NewUint(uint64(-diff)))
		}
		require.NoError(t, perp.perpetual.SubmitDataPoint(ctx, internalPrice, p.t))
	}
}

type testDataPoint struct {
	price *num.Uint
	t     int64
}

func getTestDataPoints(t *testing.T) []*testDataPoint {
	t.Helper()

	// interest-rates are daily so we want the time of the data-points to be of that scale
	// so we make them over 6 hours, a quarter of a day.

	year := 31536000000000000
	month := int64(year / 12)
	st := int64(time.Hour)

	return []*testDataPoint{
		{
			price: num.NewUint(110),
			t:     st,
		},
		{
			price: num.NewUint(120),
			t:     st + month,
		},
		{
			price: num.NewUint(120),
			t:     st + (month * 2),
		},
		{
			price: num.NewUint(100),
			t:     st + (month * 3),
		},
	}
}

type tstPerp struct {
	oe        *mocks.MockOracleEngine
	broker    *mocks.MockBroker
	perpetual *products.Perpetual
	ctrl      *gomock.Controller
	perp      *types.Perps
}

func testPerpetual(t *testing.T) *tstPerp {
	t.Helper()
	return testPerpetualWithOpts(t, "0", "0", "0", "0")
}

func testPerpetualWithOpts(t *testing.T, interestRate, clampLowerBound, clampUpperBound, marginFactor string) *tstPerp {
	t.Helper()

	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	oe := mocks.NewMockOracleEngine(ctrl)
	broker := mocks.NewMockBroker(ctrl)
	oe.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(spec.SubscriptionID(1), func(_ context.Context, _ spec.SubscriptionID) {}, nil)
	perp := getTestPerpProd(t)

	perpetual, err := products.NewPerpetual(context.Background(), log, perp, "", oe, broker, 1)
	perp.InterestRate = num.MustDecimalFromString(interestRate)
	perp.ClampLowerBound = num.MustDecimalFromString(clampLowerBound)
	perp.ClampUpperBound = num.MustDecimalFromString(clampUpperBound)
	perp.MarginFundingFactor = num.MustDecimalFromString(marginFactor)

	if err != nil {
		t.Fatalf("couldn't create a perp for testing: %v", err)
	}
	return &tstPerp{
		perpetual: perpetual,
		oe:        oe,
		broker:    broker,
		ctrl:      ctrl,
		perp:      perp,
	}
}

func getTestPerpProd(t *testing.T) *types.Perps {
	t.Helper()
	dp := uint32(1)
	pubKeys := []*dstypes.Signer{
		dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey),
	}

	factor, _ := num.DecimalFromString("0.5")
	settlementSrc := &datasource.Spec{
		Data: datasource.NewDefinition(
			datasource.ContentTypeOracle,
		).SetOracleConfig(
			&signedoracle.SpecConfiguration{
				Signers: pubKeys,
				Filters: []*dstypes.SpecFilter{
					{
						Key: &dstypes.SpecPropertyKey{
							Name:                "foo",
							Type:                datapb.PropertyKey_TYPE_INTEGER,
							NumberDecimalPlaces: ptr.From(uint64(dp)),
						},
						Conditions: nil,
					},
				},
			},
		),
	}

	scheduleSrc := &datasource.Spec{
		Data: datasource.NewDefinition(
			datasource.ContentTypeOracle,
		).SetOracleConfig(&signedoracle.SpecConfiguration{
			Signers: pubKeys,
			Filters: []*dstypes.SpecFilter{
				{
					Key: &dstypes.SpecPropertyKey{
						Name: "bar",
						Type: datapb.PropertyKey_TYPE_TIMESTAMP,
					},
					Conditions: nil,
				},
			},
		}),
	}

	return &types.Perps{
		MarginFundingFactor:                 factor,
		DataSourceSpecForSettlementData:     settlementSrc,
		DataSourceSpecForSettlementSchedule: scheduleSrc,
		DataSourceSpecBinding: &datasource.SpecBindingForPerps{
			SettlementDataProperty:     "foo",
			SettlementScheduleProperty: "bar",
		},
	}
}

type DataPoint struct {
	price *num.Uint
	t     int64
	seq   int
	twap  *num.Uint
}

type FundingNode struct {
	Timestamp time.Time `json:"timestamp"`
	Seq       int       `json:"seq"`
	Price     string    `json:"price"`
	TWAP      string    `json:"twap"`
	Source    string    `json:"dataPointSource"`
}

type Edge struct {
	Node FundingNode `json:"node"`
}

type FundingDataPoints struct {
	Edges []Edge `json:"edges"`
}

type GQLData struct {
	FundingDataPoints FundingDataPoints `json:"fundingPeriodDataPoints"`
}

type GQL struct {
	Data GQLData `json:"data"`
}

const testData = `{
  "data": {
    "fundingPeriodDataPoints": {
      "edges": [
        {
          "node": {
            "timestamp": "2023-08-16T13:52:00Z",
            "seq": 6,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:51:36Z",
            "seq": 6,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:51:00Z",
            "seq": 6,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:50:36Z",
            "seq": 6,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:50:00Z",
            "seq": 6,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:49:36Z",
            "seq": 6,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:49:00Z",
            "seq": 5,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:48:36Z",
            "seq": 5,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:48:12Z",
            "seq": 5,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:47:36Z",
            "seq": 5,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:47:00Z",
            "seq": 5,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:46:36Z",
            "seq": 5,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:46:00Z",
            "seq": 5,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:45:36Z",
            "seq": 5,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:45:00Z",
            "seq": 5,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:44:36Z",
            "seq": 5,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:44:00Z",
            "seq": 4,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:43:36Z",
            "seq": 4,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:43:00Z",
            "seq": 4,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:42:36Z",
            "seq": 4,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:42:00Z",
            "seq": 4,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:41:36Z",
            "seq": 4,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:41:24Z",
            "seq": 4,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:40:36Z",
            "seq": 4,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:40:00Z",
            "seq": 4,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:39:36Z",
            "seq": 4,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:39:12Z",
            "seq": 4,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:39:12Z",
            "seq": 3,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:38:48Z",
            "seq": 4,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:38:12Z",
            "seq": 3,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:37:36Z",
            "seq": 3,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:37:00Z",
            "seq": 3,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:36:36Z",
            "seq": 3,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:36:00Z",
            "seq": 3,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:35:36Z",
            "seq": 3,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:35:00Z",
            "seq": 3,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:34:36Z",
            "seq": 3,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:34:00Z",
            "seq": 2,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:33:36Z",
            "seq": 2,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:33:00Z",
            "seq": 2,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:32:36Z",
            "seq": 2,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:32:00Z",
            "seq": 2,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:31:48Z",
            "seq": 2,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:31:00Z",
            "seq": 2,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:30:36Z",
            "seq": 2,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:30:00Z",
            "seq": 2,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:29:36Z",
            "seq": 2,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:29:00Z",
            "seq": 1,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:28:36Z",
            "seq": 1,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:28:00Z",
            "seq": 1,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:27:36Z",
            "seq": 1,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:27:12Z",
            "seq": 1,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:26:36Z",
            "seq": 1,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:26:00Z",
            "seq": 1,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:25:36Z",
            "seq": 1,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:25:12Z",
            "seq": 1,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:24:48Z",
            "seq": 1,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        },
        {
          "node": {
            "timestamp": "2023-08-16T13:24:00Z",
            "seq": 1,
            "price": "29124220000",
            "twap": "29124220000",
            "dataPointSource": "SOURCE_EXTERNAL"
          }
        }
      ]
    }
  }
}`

func getGQLData() (*GQL, error) {
	ret := GQL{}
	if err := json.Unmarshal([]byte(testData), &ret); err != nil {
		return nil, err
	}
	ret.Sort()
	return &ret, nil
}

func (g *GQL) Sort() {
	// group by sequence
	sort.SliceStable(g.Data.FundingDataPoints.Edges, func(i, j int) bool {
		return g.Data.FundingDataPoints.Edges[i].Node.Seq < g.Data.FundingDataPoints.Edges[j].Node.Seq
	})
	for i, j := 0, len(g.Data.FundingDataPoints.Edges)-1; i < j; i, j = i+1, j-1 {
		g.Data.FundingDataPoints.Edges[i], g.Data.FundingDataPoints.Edges[j] = g.Data.FundingDataPoints.Edges[j], g.Data.FundingDataPoints.Edges[i]
	}
}

func (g *GQL) GetDataPoints() []DataPoint {
	ret := make([]DataPoint, 0, len(g.Data.FundingDataPoints.Edges))
	for _, n := range g.Data.FundingDataPoints.Edges {
		p, _ := num.UintFromString(n.Node.Price, 10)
		twap, _ := num.UintFromString(n.Node.TWAP, 10)
		ret = append(ret, DataPoint{
			price: p,
			t:     n.Node.Timestamp.UnixNano(),
			seq:   n.Node.Seq,
			twap:  twap,
		})
	}
	return ret
}
