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
	"fmt"
	"testing"

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
	perp.perpetual.OnLeaveOpeningAuction(ctx, 1000)

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
	perp.perpetual.PromptSettlementCue(ctx, 1040)
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
	perp.perpetual.PromptSettlementCue(ctx, 1040)
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
	perp.perpetual.OnLeaveOpeningAuction(ctx, 1000)

	// add data-points from the past, they will just be ignored
	perp.broker.EXPECT().Send(gomock.Any()).Times(2)
	require.NoError(t, perp.perpetual.SubmitDataPoint(ctx, num.UintOne(), 890))
	perp.perpetual.AddTestExternalPoint(ctx, num.UintZero(), 900)

	// send in some data points
	perp.broker.EXPECT().Send(gomock.Any()).Times(len(points) * 2)
	for _, p := range points {
		// send in an external and a matching internal
		require.NoError(t, perp.perpetual.SubmitDataPoint(ctx, p.price, p.t))
		perp.perpetual.AddTestExternalPoint(ctx, p.price, p.t)
	}

	// add some data-points in the future from when we will cue the end of the funding period
	// they should not affect the funding payment of this period
	perp.broker.EXPECT().Send(gomock.Any()).Times(2)
	require.NoError(t, perp.perpetual.SubmitDataPoint(ctx, num.UintOne(), 2000))
	perp.perpetual.AddTestExternalPoint(ctx, num.UintZero(), 2020)

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
	perp.perpetual.PromptSettlementCue(ctx, 1040)
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
	perp.perpetual.PromptSettlementCue(ctx, 1050)
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
	perpetual, err := products.NewPerpetual(context.Background(), log, perp, oe, broker, 1)
	require.NoError(t, err)
	require.NotNil(t, settle)
	require.NotNil(t, period)
	// register the callback
	perpetual.NotifyOnSettlementData(marketSettle)

	perpetual.OnLeaveOpeningAuction(ctx, scaleToNano(t, 1000))

	require.NoError(t, perpetual.SubmitDataPoint(ctx, num.UintOne(), scaleToNano(t, 890)))
	// callback to receive settlement data
	settle(ctx, dscommon.Data{
		Data: map[string]string{
			perp.DataSourceSpecBinding.SettlementDataProperty: "1",
		},
		MetaData: map[string]string{
			"eth-block-time": "900",
		},
	})

	for _, p := range points {
		// send in an external and a matching internal
		require.NoError(t, perpetual.SubmitDataPoint(ctx, p.price, scaleToNano(t, p.t)))
		settle(ctx, dscommon.Data{
			Data: map[string]string{
				perp.DataSourceSpecBinding.SettlementDataProperty: p.price.String(),
			},
			MetaData: map[string]string{
				"eth-block-time": fmt.Sprintf("%d", p.t),
			},
		})
	}

	// add some data-points in the future from when we will cue the end of the funding period
	// they should not affect the funding payment of this period
	require.NoError(t, perpetual.SubmitDataPoint(ctx, num.UintOne(), scaleToNano(t, 2000)))
	settle(ctx, dscommon.Data{
		Data: map[string]string{
			perp.DataSourceSpecBinding.SettlementDataProperty: "1",
		},
		MetaData: map[string]string{
			"eth-block-time": "2020",
		},
	})
	// make sure the data-point outside of the period doesn't trigger the schedule callback
	// that has to come from the oracle, too
	assert.False(t, received)

	// end period
	period(ctx, dscommon.Data{
		Data: map[string]string{
			perp.DataSourceSpecBinding.SettlementScheduleProperty: "1040",
		},
		MetaData: map[string]string{
			"eth-block-time": "1040", // this isn't used currently
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
	exp.SetUint(num.Sum(num.UintOne(), num.UintOne()))
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
	perpetual, err := products.NewPerpetual(context.Background(), log, perp, oe, broker, 1)
	require.NoError(t, err)
	require.NotNil(t, settle)
	require.NotNil(t, period)
	// register the callback
	perpetual.NotifyOnSettlementData(marketSettle)

	perpetual.OnLeaveOpeningAuction(ctx, scaleToNano(t, 1000))

	require.NoError(t, perpetual.SubmitDataPoint(ctx, num.UintOne(), scaleToNano(t, 890)))
	// callback to receive settlement data
	settle(ctx, dscommon.Data{
		Data: map[string]string{
			perp.DataSourceSpecBinding.SettlementDataProperty: "1",
		},
		MetaData: map[string]string{
			"eth-block-time": "900",
		},
	})

	// send all external points, but not all internal ones are matching
	for i, p := range points {
		if i%2 == 0 {
			ip := num.UintZero().Sub(p.price, num.UintOne())
			require.NoError(t, perpetual.SubmitDataPoint(ctx, ip, scaleToNano(t, p.t)))
		}
		settle(ctx, dscommon.Data{
			Data: map[string]string{
				perp.DataSourceSpecBinding.SettlementDataProperty: p.price.String(),
			},
			MetaData: map[string]string{
				"eth-block-time": fmt.Sprintf("%d", p.t),
			},
		})
	}

	// add some data-points in the future from when we will cue the end of the funding period
	// they should not affect the funding payment of this period
	require.NoError(t, perpetual.SubmitDataPoint(ctx, num.UintOne(), scaleToNano(t, 2000)))
	settle(ctx, dscommon.Data{
		Data: map[string]string{
			perp.DataSourceSpecBinding.SettlementDataProperty: "1",
		},
		MetaData: map[string]string{
			"eth-block-time": "2020",
		},
	})

	// end period
	period(ctx, dscommon.Data{
		Data: map[string]string{
			perp.DataSourceSpecBinding.SettlementScheduleProperty: "1040",
		},
		MetaData: map[string]string{
			"eth-block-time": "1040", // this isn't used currently
		},
	})

	assert.True(t, received)
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
	return []*testDataPoint{
		{
			price: num.NewUint(110),
			t:     1000,
		},
		{
			price: num.NewUint(120),
			t:     1010,
		},
		{
			price: num.NewUint(120),
			t:     1020,
		},
		{
			price: num.NewUint(100),
			t:     1030,
		},
	}
}

func scaleToNano(t *testing.T, secs int64) int64 {
	t.Helper()
	return secs * 1000000000
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

	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	oe := mocks.NewMockOracleEngine(ctrl)
	broker := mocks.NewMockBroker(ctrl)
	oe.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(spec.SubscriptionID(1), func(_ context.Context, _ spec.SubscriptionID) {}, nil)
	perp := getTestPerpProd(t)

	perpetual, err := products.NewPerpetual(context.Background(), log, perp, oe, broker, 1)
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
