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

package products_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
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
	tmocks "code.vegaprotocol.io/vega/core/vegatime/mocks"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeriodicSettlement(t *testing.T) {
	t.Run("incoming data ignored before leaving opening auction", testIncomingExternalDataIgnoredBeforeLeavingOpeningAuction)
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
	t.Run("margin increase, negative payment", testGetMarginIncreaseNegativePayment)
	t.Run("test pathological case with out of order points", testOutOfOrderPointsBeforePeriodStart)
	t.Run("test update perpetual", testUpdatePerpetual)
	t.Run("test terminate trading coincides with time trigger", testTerminateTradingCoincidesTimeTrigger)
	t.Run("test funding-payment on start boundary", testFundingPaymentOnStartBoundary)
	t.Run("test data point is before the first point", TestPrependPoint)
}

func TestRealData(t *testing.T) {
	tcs := []struct {
		name    string
		reverse bool
	}{
		{
			"in order",
			false,
		},
		{
			"out of order",
			false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			perp := testPerpetual(t)
			defer perp.ctrl.Finish()

			ctx := context.Background()
			tstData, err := getGQLData()
			require.NoError(t, err)
			data := tstData.GetDataPoints(false)

			// want to start the period from before the point with the smallest time
			seq := math.MaxInt
			st := data[0].t
			nd := data[0].t
			for i := 0; i < len(data); i++ {
				if data[i].t < st {
					st = data[i].t
				}
				if data[i].t > nd {
					nd = data[i].t
				}
				seq = num.MinV(seq, data[i].seq)
			}

			perp.perpetual.SetSettlementListener(func(context.Context, *num.Numeric) {})
			// leave opening auction
			whenLeaveOpeningAuction(t, perp, st-1)

			perp.broker.EXPECT().Send(gomock.Any()).AnyTimes()
			perp.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()

			// set the first internal data-point
			for _, dp := range data {
				if dp.seq > seq {
					perp.perpetual.PromptSettlementCue(ctx, dp.t)
					seq = dp.seq
				}
				perp.perpetual.AddTestExternalPoint(ctx, dp.price, dp.t)
				perp.perpetual.SubmitDataPoint(ctx, num.UintZero().Add(dp.price, num.NewUint(100)), dp.t)
			}
			d := perp.perpetual.GetData(nd).Data.(*types.PerpetualData)
			assert.Equal(t, "29124220000", d.ExternalTWAP)
			assert.Equal(t, "29124220100", d.InternalTWAP)
			assert.Equal(t, "100", d.FundingPayment)
		})
	}
}

func testIncomingExternalDataIgnoredBeforeLeavingOpeningAuction(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()

	ctx := context.Background()

	// no error because its really a callback from the oracle engine, but we expect no events
	perp.perpetual.AddTestExternalPoint(ctx, num.UintOne(), 2000)
	data := perp.perpetual.GetData(2000)
	require.Nil(t, data)

	// internal data point recevied without error
	perp.broker.EXPECT().Send(gomock.AssignableToTypeOf(&events.FundingPeriodDataPoint{})).Times(1)
	err := perp.perpetual.SubmitDataPoint(ctx, num.UintOne(), 2000)
	data = perp.perpetual.GetData(2000)
	assert.NoError(t, err)
	require.Nil(t, data)

	// check that settlement cues are ignored, we expect no events when it is
	perp.perpetual.PromptSettlementCue(ctx, 4000)
}

func testPeriodEndWithNoDataPoints(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()

	ctx := context.Background()
	now := time.Unix(1, 0)

	// funding payment will be zero because there are no data points
	var called bool
	fn := func(context.Context, *num.Numeric) {
		called = true
	}
	perp.perpetual.SetSettlementListener(fn)

	whenLeaveOpeningAuction(t, perp, now.UnixNano())

	perp.broker.EXPECT().Send(gomock.Any()).Times(2)
	perp.perpetual.PromptSettlementCue(ctx, now.Add(40*time.Second).UnixNano())

	// we had no points to check we didn't call into listener
	assert.False(t, called)
}

func TestPrependPoint(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()

	ctx := context.Background()
	now := time.Unix(1000, 0)
	whenLeaveOpeningAuction(t, perp, now.UnixNano())

	perp.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// we'll use this point to check that we do not lose a later point when we recalc when earlier points come in
	err := perp.perpetual.SubmitDataPoint(ctx, num.NewUint(10), time.Unix(5000, 0).UnixNano())
	perp.perpetual.AddTestExternalPoint(ctx, num.NewUint(9), time.Unix(5000, 0).UnixNano())
	require.NoError(t, err)
	require.Equal(t, "1", getFundingPayment(t, perp, time.Unix(5000, 0).UnixNano()))

	// first point is after the start of the period
	err = perp.perpetual.SubmitDataPoint(ctx, num.NewUint(10), time.Unix(2000, 0).UnixNano())
	require.NoError(t, err)
	require.Equal(t, "1", getFundingPayment(t, perp, time.Unix(5000, 0).UnixNano()))

	// now another one comes in before this, but also after the start of the period
	err = perp.perpetual.SubmitDataPoint(ctx, num.NewUint(50), time.Unix(1500, 0).UnixNano())
	require.NoError(t, err)
	require.Equal(t, "6", getFundingPayment(t, perp, time.Unix(5000, 0).UnixNano()))

	// now one comes in before the start of the period
	err = perp.perpetual.SubmitDataPoint(ctx, num.NewUint(50), time.Unix(500, 0).UnixNano())
	require.NoError(t, err)
	require.Equal(t, "11", getFundingPayment(t, perp, time.Unix(5000, 0).UnixNano()))

	// now one comes in before this point
	err = perp.perpetual.SubmitDataPoint(ctx, num.UintOne(), time.Unix(250, 0).UnixNano())
	require.ErrorIs(t, err, products.ErrDataPointIsTooOld)
	require.Equal(t, "11", getFundingPayment(t, perp, time.Unix(5000, 0).UnixNano()))

	// now one comes in after the first point, but before the period start
	err = perp.perpetual.SubmitDataPoint(ctx, num.UintOne(), time.Unix(500, 0).UnixNano())
	require.ErrorIs(t, err, products.ErrDataPointAlreadyExistsAtTime)
	require.Equal(t, "11", getFundingPayment(t, perp, time.Unix(5000, 0).UnixNano()))

	// now one comes in after the first point, but before the period start
	err = perp.perpetual.SubmitDataPoint(ctx, num.NewUint(50), time.Unix(750, 0).UnixNano())
	require.NoError(t, err)
	require.Equal(t, "11", getFundingPayment(t, perp, time.Unix(5000, 0).UnixNano()))

	// now one comes that equals period start
	err = perp.perpetual.SubmitDataPoint(ctx, num.NewUint(50), time.Unix(1000, 0).UnixNano())
	require.NoError(t, err)
	require.Equal(t, "11", getFundingPayment(t, perp, time.Unix(5000, 0).UnixNano()))

	err = perp.perpetual.SubmitDataPoint(ctx, num.NewUint(100000), time.Unix(750, 0).UnixNano())
	require.ErrorIs(t, err, products.ErrDataPointIsTooOld)
}

func testEqualInternalAndExternalPrices(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()
	ctx := context.Background()

	// set of the data points such that difference in averages is 0
	points := getTestDataPoints(t)

	// tell the perpetual that we are ready to accept settlement stuff
	whenLeaveOpeningAuction(t, perp, points[0].t)

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
	whenLeaveOpeningAuction(t, perp, points[0].t)

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

	productData := perp.perpetual.GetData(points[len(points)-1].t)
	perpData, ok := productData.Data.(*types.PerpetualData)
	assert.True(t, ok)

	perp.perpetual.PromptSettlementCue(ctx, points[len(points)-1].t)
	assert.NotNil(t, fundingPayment)
	assert.True(t, fundingPayment.IsInt())
	assert.Equal(t, "-10", fundingPayment.String())
	assert.Equal(t, "-10", perpData.FundingPayment)
	assert.Equal(t, "116", perpData.ExternalTWAP)
	assert.Equal(t, "106", perpData.InternalTWAP)
	assert.Equal(t, "-0.0862068965517241", perpData.FundingRate)
	assert.Equal(t, uint64(0), perpData.SeqNum)
	assert.Equal(t, int64(3600000000000), perpData.StartTime)
}

func testDataPointsOutsidePeriod(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()
	ctx := context.Background()

	// set of the data points such that difference in averages is 0
	points := getTestDataPoints(t)

	// tell the perpetual that we are ready to accept settlement stuff
	whenLeaveOpeningAuction(t, perp, points[0].t)

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
	whenLeaveOpeningAuction(t, perp, points[0].t+int64(time.Second))

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

func testOutOfOrderPointsBeforePeriodStart(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()
	ctx := context.Background()

	// start time will be after the *second* data point
	whenLeaveOpeningAuction(t, perp, 1693398617000000000)

	price := num.NewUint(100000000)
	timestamps := []int64{
		1693398614000000000,
		1693398615000000000,
		1693398616000000000,
		1693398618000000000,
		1693398617000000000,
	}

	perp.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	perp.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	for _, tt := range timestamps {
		perp.perpetual.AddTestExternalPoint(ctx, price, tt)
		perp.perpetual.SubmitDataPoint(ctx, num.UintZero().Add(price, num.NewUint(100000000)), tt)
	}

	// ask for the funding payment
	var fundingPayment *num.Numeric
	fn := func(_ context.Context, fp *num.Numeric) {
		fundingPayment = fp
	}
	perp.perpetual.SetSettlementListener(fn)

	// period end is *after* our last point
	perp.perpetual.PromptSettlementCue(ctx, 1693398617000000000+int64(time.Hour))
	assert.NotNil(t, fundingPayment)
	assert.True(t, fundingPayment.IsInt())
	assert.Equal(t, "100000000", fundingPayment.String())
}

func testRegisteredCallbacks(t *testing.T) {
	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	oe := mocks.NewMockOracleEngine(ctrl)
	broker := mocks.NewMockBroker(ctrl)
	ts := tmocks.NewMockTimeService(ctrl)
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
	perpetual, err := products.NewPerpetual(context.Background(), log, perp, "", ts, oe, broker, 1)
	require.NoError(t, err)
	require.NotNil(t, settle)
	require.NotNil(t, period)
	// register the callback
	perpetual.NotifyOnSettlementData(marketSettle)
	ts.EXPECT().GetTimeNow().Times(1).Return(time.Unix(0, points[0].t))
	perpetual.UpdateAuctionState(ctx, false)

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
	ts := tmocks.NewMockTimeService(ctrl)
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
	perpetual, err := products.NewPerpetual(context.Background(), log, perp, "", ts, oe, broker, 1)
	require.NoError(t, err)
	require.NotNil(t, settle)
	require.NotNil(t, period)
	// register the callback
	perpetual.NotifyOnSettlementData(marketSettle)

	// start the funding period
	ts.EXPECT().GetTimeNow().Times(1).Return(time.Unix(0, points[0].t))
	perpetual.UpdateAuctionState(ctx, false)

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
	perp := testPerpetual(t)
	perp.perp.InterestRate = num.DecimalFromFloat(0.01)
	perp.perp.ClampLowerBound = num.DecimalFromInt64(-1)
	perp.perp.ClampUpperBound = num.DecimalFromInt64(1)

	defer perp.ctrl.Finish()
	ctx := context.Background()

	// test data
	points := getTestDataPoints(t)
	lastPoint := points[len(points)-1]

	// when: the funding period starts
	whenLeaveOpeningAuction(t, perp, points[0].t)

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
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()
	perp.perp.InterestRate = num.DecimalFromFloat(0.5)
	perp.perp.ClampLowerBound = num.DecimalFromFloat(0.001)
	perp.perp.ClampUpperBound = num.DecimalFromFloat(0.002)
	ctx := context.Background()

	// test data
	points := getTestDataPoints(t)

	// when: the funding period starts
	whenLeaveOpeningAuction(t, perp, points[0].t)

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
	whenLeaveOpeningAuction(t, perp, points[0].t)

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

	perp.ts.EXPECT().GetTimeNow().Times(1).Return(time.Unix(10, points[len(points)-1].t))
	perp.broker.EXPECT().Send(gomock.Any()).Times(1)
	perp.perpetual.UnsubscribeTradingTerminated(ctx)
	assert.NotNil(t, fundingPayment)
	assert.True(t, fundingPayment.IsInt())
	assert.Equal(t, "0", fundingPayment.String())
}

func testTerminateTradingCoincidesTimeTrigger(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()
	ctx := context.Background()

	// set of the data points such that difference in averages is 0
	points := getTestDataPoints(t)

	// tell the perpetual that we are ready to accept settlement stuff
	whenLeaveOpeningAuction(t, perp, points[0].t)

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

	// do a normal settlement cue end time
	endTime := time.Unix(10, points[len(points)-1].t).Truncate(time.Second)
	perp.broker.EXPECT().Send(gomock.Any()).Times(2)
	perp.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	perp.perpetual.PromptSettlementCue(ctx, endTime.UnixNano())
	assert.NotNil(t, fundingPayment)
	assert.True(t, fundingPayment.IsInt())
	assert.Equal(t, "0", fundingPayment.String())

	// now terminate the market at the same time, we expect no funding payment, and just an event
	// to say the period has ended, with no start period.
	fundingPayment = nil
	perp.ts.EXPECT().GetTimeNow().Times(1).Return(time.Unix(10, points[len(points)-1].t))
	perp.broker.EXPECT().Send(gomock.Any()).Times(1)
	perp.perpetual.UnsubscribeTradingTerminated(ctx)
	assert.Nil(t, fundingPayment)
}

func testGetMarginIncrease(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()
	perp.perp.MarginFundingFactor = num.DecimalFromFloat(0.5)

	// test data
	points := getTestDataPoints(t)

	// before we've started the first funding interval margin increase is 0
	inc := perp.perpetual.GetMarginIncrease(points[0].t)
	assert.Equal(t, "0", inc.String())

	// start funding period
	whenLeaveOpeningAuction(t, perp, points[0].t)

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

func testGetMarginIncreaseNegativePayment(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()
	perp.perp.MarginFundingFactor = num.DecimalFromFloat(0.5)

	// test data
	points := getTestDataPoints(t)

	// start funding period
	whenLeaveOpeningAuction(t, perp, points[0].t)

	// and: the difference in external/internal prices are is 10
	submitDataWithDifference(t, perp, points, -10)

	lastPoint := points[len(points)-1]
	inc := perp.perpetual.GetMarginIncrease(lastPoint.t)
	// margin increase is margin_factor * funding-payment = 0.5 * 10
	assert.Equal(t, "-5", inc.String())
}

func testUpdatePerpetual(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()
	perp.perp.MarginFundingFactor = num.DecimalFromFloat(0.5)
	ctx := context.Background()

	// test data
	points := getTestDataPoints(t)
	whenLeaveOpeningAuction(t, perp, points[0].t)
	submitDataWithDifference(t, perp, points, 10)

	// query margin factor before update
	lastPoint := points[len(points)-1]
	inc := perp.perpetual.GetMarginIncrease(lastPoint.t)
	assert.Equal(t, "5", inc.String())

	// do the perps update with a new margin factor
	update := getTestPerpProd(t)
	update.MarginFundingFactor = num.DecimalFromFloat(1)
	err := perp.perpetual.Update(ctx, &types.InstrumentPerps{Perps: update}, perp.oe)
	require.NoError(t, err)

	// expect two unsubscriptions
	assert.Equal(t, perp.unsub, 2)

	// margin increase should now be double, which means the data-points were preserved
	inc = perp.perpetual.GetMarginIncrease(lastPoint.t)
	assert.Equal(t, "10", inc.String())

	// now submit a data point and check it is expected i.e the funding period is still active
	perp.broker.EXPECT().Send(gomock.Any()).Times(1)
	assert.NoError(t, perp.perpetual.SubmitDataPoint(ctx, num.NewUint(123), lastPoint.t+int64(time.Hour)))
}

func testFundingPaymentOnStartBoundary(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()
	// set of the data points such that difference in averages is 0
	points := getTestDataPoints(t)

	// tell the perpetual that we are ready to accept settlement stuff
	st := points[0].t
	whenLeaveOpeningAuction(t, perp, st)

	expectedTWAP := 100
	// send in data points at this time
	submitPointWithDifference(t, perp, points[0], expectedTWAP)

	// now get the funding-payment at this time
	fundingPayment := getFundingPayment(t, perp, st)
	assert.Equal(t, "100", fundingPayment)
}

func TestFundingPaymentModifiers(t *testing.T) {
	cases := []struct {
		twapDifference         int
		scalingFactor          *num.Decimal
		upperBound             *num.Decimal
		lowerBound             *num.Decimal
		expectedFundingPayment string
		expectedFundingRate    string
	}{
		{
			twapDifference:         220,
			scalingFactor:          ptr.From(num.DecimalFromFloat(0.5)),
			expectedFundingPayment: "110",
			expectedFundingRate:    "1",
		},
		{
			twapDifference:         1100,
			scalingFactor:          ptr.From(num.DecimalFromFloat(1.5)),
			expectedFundingPayment: "1650",
			expectedFundingRate:    "15",
		},
		{
			twapDifference:         100,
			upperBound:             ptr.From(num.DecimalFromFloat(0.5)),
			expectedFundingPayment: "55", // 0.5 * external-twap < diff, so snap to 0.5
			expectedFundingRate:    "0.5",
		},
		{
			twapDifference:         5,
			lowerBound:             ptr.From(num.DecimalFromFloat(0.5)),
			expectedFundingPayment: "55", // 0.5 * external-twap > 5, so snap to 0.5
			expectedFundingRate:    "0.5",
		},
		{
			twapDifference:         1100,
			scalingFactor:          ptr.From(num.DecimalFromFloat(1.5)),
			upperBound:             ptr.From(num.DecimalFromFloat(0.5)),
			expectedFundingPayment: "55",
			expectedFundingRate:    "0.5",
		},
	}

	for _, c := range cases {
		perp := testPerpetual(t)
		defer perp.ctrl.Finish()

		// set modifiers
		perp.perp.FundingRateScalingFactor = c.scalingFactor
		perp.perp.FundingRateLowerBound = c.lowerBound
		perp.perp.FundingRateUpperBound = c.upperBound

		// tell the perpetual that we are ready to accept settlement stuff
		points := getTestDataPoints(t)
		whenLeaveOpeningAuction(t, perp, points[0].t)
		submitPointWithDifference(t, perp, points[0], c.twapDifference)

		// check the goods
		fundingPayment := getFundingPayment(t, perp, points[0].t)
		assert.Equal(t, c.expectedFundingPayment, fundingPayment)

		fundingRate := getFundingRate(t, perp, points[0].t)
		assert.Equal(t, c.expectedFundingRate, fundingRate)
	}
}

// submits the given data points as both external and interval but with the given different added to the internal price.
func submitDataWithDifference(t *testing.T, perp *tstPerp, points []*testDataPoint, diff int) {
	t.Helper()
	for _, p := range points {
		submitPointWithDifference(t, perp, p, diff)
	}
}

// submits the single data point as both external and internal but with a differece in price.
func submitPointWithDifference(t *testing.T, perp *tstPerp, p *testDataPoint, diff int) {
	t.Helper()
	ctx := context.Background()

	var internalPrice *num.Uint
	perp.broker.EXPECT().Send(gomock.Any()).Times(2)
	perp.perpetual.AddTestExternalPoint(ctx, p.price, p.t)

	if diff > 0 {
		internalPrice = num.UintZero().Add(p.price, num.NewUint(uint64(diff)))
	}
	if diff < 0 {
		internalPrice = num.UintZero().Sub(p.price, num.NewUint(uint64(-diff)))
	}
	require.NoError(t, perp.perpetual.SubmitDataPoint(ctx, internalPrice, p.t))
}

func whenLeaveOpeningAuction(t *testing.T, perp *tstPerp, now int64) {
	t.Helper()
	perp.broker.EXPECT().Send(gomock.Any()).Times(1)
	whenAuctionStateChanges(t, perp, now, false)
}

func whenAuctionStateChanges(t *testing.T, perp *tstPerp, now int64, enter bool) {
	t.Helper()
	perp.ts.EXPECT().GetTimeNow().Times(1).Return(time.Unix(0, now))
	perp.perpetual.UpdateAuctionState(context.Background(), enter)
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
	ts        *tmocks.MockTimeService
	broker    *mocks.MockBroker
	perpetual *products.Perpetual
	ctrl      *gomock.Controller
	perp      *types.Perps

	unsub int
}

func (tp *tstPerp) unsubscribe(_ context.Context, _ spec.SubscriptionID) {
	tp.unsub++
}

func testPerpetual(t *testing.T) *tstPerp {
	t.Helper()

	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	oe := mocks.NewMockOracleEngine(ctrl)
	broker := mocks.NewMockBroker(ctrl)
	ts := tmocks.NewMockTimeService(ctrl)
	perp := getTestPerpProd(t)

	tp := &tstPerp{
		ts:     ts,
		oe:     oe,
		broker: broker,
		ctrl:   ctrl,
		perp:   perp,
	}
	tp.oe.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(spec.SubscriptionID(1), tp.unsubscribe, nil)

	perpetual, err := products.NewPerpetual(context.Background(), log, perp, "", ts, oe, broker, 1)
	if err != nil {
		t.Fatalf("couldn't create a perp for testing: %v", err)
	}

	tp.perpetual = perpetual
	return tp
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
	return &ret, nil
}

func (g *GQL) Sort(reverse bool) {
	// group by sequence
	sort.SliceStable(g.Data.FundingDataPoints.Edges, func(i, j int) bool {
		if g.Data.FundingDataPoints.Edges[i].Node.Seq == g.Data.FundingDataPoints.Edges[j].Node.Seq {
			if reverse {
				return g.Data.FundingDataPoints.Edges[i].Node.Timestamp.UnixNano() > g.Data.FundingDataPoints.Edges[j].Node.Timestamp.UnixNano()
			}
			return g.Data.FundingDataPoints.Edges[i].Node.Timestamp.UnixNano() < g.Data.FundingDataPoints.Edges[j].Node.Timestamp.UnixNano()
		}

		return g.Data.FundingDataPoints.Edges[i].Node.Seq < g.Data.FundingDataPoints.Edges[j].Node.Seq
	})
}

func (g *GQL) GetDataPoints(reverse bool) []DataPoint {
	g.Sort(reverse)
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
