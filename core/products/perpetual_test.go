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
	"testing"

	"code.vegaprotocol.io/vega/core/products"
	"code.vegaprotocol.io/vega/core/products/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeriodicSettlement(t *testing.T) {
	t.Run("cannot submit data-point before leaving opening auction", TestCannotSubmitDataPointBeforeOpeningAuction)
	t.Run("period end with no data point", TestPeriodEndWithNoDataPoints)
	t.Run("equal internal and external prices", TestEqualInternalAndExternalPrices)
	t.Run("constant difference long pays short", TestConstantDifferenceLongPaysShort)
	t.Run("data points outside of period", TestDataPointsOutsidePeriod)
	t.Run("data points not on boundary", TestDataPointsNotOnBoundary)
}

func TestCannotSubmitDataPointBeforeOpeningAuction(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()

	ctx := context.Background()

	// no error because its really a callback from the oracle engine, but we expect no events
	perp.perpetual.AddTestExternalPoint(ctx, num.UintOne(), 2000)

	err := perp.perpetual.SubmitDataPoint(ctx, num.UintOne(), 2000)
	assert.ErrorIs(t, err, products.ErrInitialPeriodNotStarted)
}

func TestPeriodEndWithNoDataPoints(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()

	ctx := context.Background()

	// funding payment will be zero because there are no data points
	var called bool
	fn := func(context.Context, *num.Numeric) {
		called = true
	}
	perp.perpetual.SetSettlementListener(fn)

	perp.broker.EXPECT().Send(gomock.Any()).Times(1)
	perp.perpetual.OnLeaveOpeningAuction(ctx, 1000)

	perp.broker.EXPECT().Send(gomock.Any()).Times(2)
	perp.perpetual.PromptSettlementCue(ctx, 1040)

	// we had no points to check we didn't call into listener
	assert.False(t, called)
}

func TestEqualInternalAndExternalPrices(t *testing.T) {
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

	perp.broker.EXPECT().Send(gomock.Any()).Times(4)
	perp.perpetual.PromptSettlementCue(ctx, 1040)
	assert.NotNil(t, fundingPayment)
	assert.True(t, fundingPayment.IsInt())
	assert.Equal(t, "0", fundingPayment.String())
}

func TestConstantDifferenceLongPaysShort(t *testing.T) {
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

	perp.broker.EXPECT().Send(gomock.Any()).Times(4)
	perp.perpetual.PromptSettlementCue(ctx, 1040)
	assert.NotNil(t, fundingPayment)
	assert.True(t, fundingPayment.IsInt())
	assert.Equal(t, "-10", fundingPayment.String())
}

func TestDataPointsOutsidePeriod(t *testing.T) {
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

	// 6 times because: end + start of the period, plus 2 carry over points for external + internal
	perp.broker.EXPECT().Send(gomock.Any()).Times(6)
	perp.perpetual.PromptSettlementCue(ctx, 1040)
	assert.NotNil(t, fundingPayment)
	assert.True(t, fundingPayment.IsInt())
	assert.Equal(t, "0", fundingPayment.String())
}

func TestDataPointsNotOnBoundary(t *testing.T) {
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
	perp.broker.EXPECT().Send(gomock.Any()).Times(4)
	perp.perpetual.PromptSettlementCue(ctx, 1050)
	assert.NotNil(t, fundingPayment)
	assert.True(t, fundingPayment.IsInt())
	assert.Equal(t, "10", fundingPayment.String())
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

type tstPerp struct {
	oe        *mocks.MockOracleEngine
	broker    *mocks.MockBroker
	perpetual *products.Perpetual
	ctrl      *gomock.Controller
}

func testPerpetual(t *testing.T) *tstPerp {
	t.Helper()

	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	oe := mocks.NewMockOracleEngine(ctrl)
	broker := mocks.NewMockBroker(ctrl)

	factor, _ := num.DecimalFromString("0.5")
	perp := &types.Perpetual{
		MarginFundingFactor: &factor,
	}

	perpetual, err := products.NewPerpetual(context.Background(), log, perp, oe, broker)
	if err != nil {
		t.Fatalf("couldn't create a Future for testing: %v", err)
	}
	return &tstPerp{
		perpetual: perpetual,
		oe:        oe,
		broker:    broker,
		ctrl:      ctrl,
	}
}
