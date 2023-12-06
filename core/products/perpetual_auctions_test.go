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
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPerpetualsWithAuctions(t *testing.T) {
	t.Run("funding period is all an auction", testFundingPeriodIsAllAnAuction)
	t.Run("data point in auction is ignored", testDataPointInAuctionIgnored)
	t.Run("data points in auction received out of order", TestDataPointsInAuctionOutOfOrder)
	t.Run("auction preserved when period resets", TestAuctionFundingPeriodReset)
}

func testFundingPeriodIsAllAnAuction(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()

	// set of the data points such that difference in averages is 0
	points := getTestDataPoints(t)

	// tell the perpetual that we are ready to accept settlement stuff
	whenLeaveOpeningAuction(t, perp, points[0].t)

	// enter auction
	whenAuctionStateChanges(t, perp, points[0].t, true)

	// send in some data points with a TWAP difference
	submitDataWithDifference(t, perp, points, 10)

	// leave auction
	whenAuctionStateChanges(t, perp, points[len(points)-1].t, false)

	fundingPayment := whenTheFundingPeriodEnds(t, perp, points[len(points)-1].t)
	assert.Equal(t, "0", fundingPayment.String())
}

func testDataPointInAuctionIgnored(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()
	expectedTWAP := 100
	// set of the data points such that difference in averages is 0
	points := getTestDataPoints(t)
	require.GreaterOrEqual(t, 4, len(points))

	// tell the perpetual that we are ready to accept settlement stuff
	whenLeaveOpeningAuction(t, perp, points[0].t)

	// submit the first point then enter an auction
	submitPointWithDifference(t, perp, points[0], expectedTWAP)
	whenAuctionStateChanges(t, perp, points[0].t+int64(time.Second), true)

	// submit a crazy point difference, then a normal point
	submitPointWithDifference(t, perp, points[1], -9999999)
	submitPointWithDifference(t, perp, points[2], expectedTWAP)

	// now we leave auction and the crazy point difference will not affect the TWAP because it was in an auction period
	whenAuctionStateChanges(t, perp, points[2].t+int64(time.Second), false)

	fundingPayment := whenTheFundingPeriodEnds(t, perp, points[len(points)-1].t)
	assert.Equal(t, int64(expectedTWAP), fundingPayment.Int64())
}

func TestDataPointsInAuctionOutOfOrder(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()
	expectedTWAP := 100
	// set of the data points such that difference in averages is 0
	points := getTestDataPoints(t)

	// tell the perpetual that we are ready to accept settlement stuff
	st := points[0].t
	nd := points[3].t
	a1 := between(points[0].t, points[1].t)
	a2 := between(points[2].t, points[3].t)

	whenLeaveOpeningAuction(t, perp, st)

	// submit the first point and enter an auction
	submitPointWithDifference(t, perp, points[0], expectedTWAP)
	whenAuctionStateChanges(t, perp, a1, true)
	whenAuctionStateChanges(t, perp, a2, false)

	// funding payment will be the constant diff in the first point
	assert.Equal(t, "100", getFundingPayment(t, perp, nd))

	// now submit a point that is mid the auction period
	submitPointWithDifference(t, perp, points[2], 200)
	assert.Equal(t, "150", getFundingPayment(t, perp, nd))

	// now submit a point also in before the previous point, also in an auction period
	// and its contribution should be ignored.
	crazy := &testDataPoint{t: between(a1, points[1].t), price: num.NewUint(1000)}
	submitPointWithDifference(t, perp, crazy, 9999999)
	assert.Equal(t, "150", getFundingPayment(t, perp, nd))
}

func TestAuctionFundingPeriodReset(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()
	expectedTWAP := 100
	// set of the data points such that difference in averages is 0
	points := getTestDataPoints(t)

	// tell the perpetual that we are ready to accept settlement stuff
	whenLeaveOpeningAuction(t, perp, points[0].t)

	// submit the first point and enter an auction
	submitPointWithDifference(t, perp, points[0], expectedTWAP)
	whenAuctionStateChanges(t, perp, points[0].t+int64(time.Second), true)

	fundingPayment := whenTheFundingPeriodEnds(t, perp, points[0].t+int64(2*time.Second))
	assert.Equal(t, int64(expectedTWAP), fundingPayment.Int64())

	// should still be on an auction to ending another funding period should give 0
	submitPointWithDifference(t, perp, points[1], -999999)
	fundingPayment = whenTheFundingPeriodEnds(t, perp, points[2].t)
	assert.Equal(t, int64(0), fundingPayment.Int64())

	// submit a point and leave auction
	submitPointWithDifference(t, perp, points[2], expectedTWAP)
	whenAuctionStateChanges(t, perp, between(points[2].t, points[3].t), false)

	fundingPayment = whenTheFundingPeriodEnds(t, perp, points[3].t)
	assert.Equal(t, int64(100), fundingPayment.Int64())

	// now we're not in an auction, ending the period again will preserve that
	fundingPayment = whenTheFundingPeriodEnds(t, perp, points[3].t+int64(time.Hour))
	assert.Equal(t, int64(100), fundingPayment.Int64())
}

func whenTheFundingPeriodEnds(t *testing.T, perp *tstPerp, now int64) *num.Int {
	t.Helper()
	ctx := context.Background()
	var fundingPayment *num.Numeric
	fn := func(_ context.Context, fp *num.Numeric) {
		fundingPayment = fp
	}
	perp.perpetual.SetSettlementListener(fn)
	perp.broker.EXPECT().Send(gomock.Any()).Times(2)
	perp.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	perp.perpetual.PromptSettlementCue(ctx, now)
	require.NotNil(t, fundingPayment)
	require.True(t, fundingPayment.IsInt())
	return fundingPayment.Int()
}

func getFundingPayment(t *testing.T, perp *tstPerp, now int64) string {
	t.Helper()
	data := perp.perpetual.GetData(now).Data.(*types.PerpetualData)
	return data.FundingPayment
}

func getFundingRate(t *testing.T, perp *tstPerp, now int64) string {
	t.Helper()
	data := perp.perpetual.GetData(now).Data.(*types.PerpetualData)
	return data.FundingRate
}

func between(p, q int64) int64 {
	return (p + q) / 2
}
