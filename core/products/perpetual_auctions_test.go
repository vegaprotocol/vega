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
	t.Run("data points in auction received out of order", testDataPointsInAuctionOutOfOrder)
	t.Run("auction preserved when period resets", testAuctionFundingPeriodReset)
	t.Run("funding data in auction period start", testFundingDataAtInAuctionPeriodStart)
	t.Run("past funding payment calculation", testPastFundingPayment)
	t.Run("past funding payment calculation in auction", testPastFundingPaymentInAuction)
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
	auctionStart := points[0].t + int64(time.Second)
	whenAuctionStateChanges(t, perp, auctionStart, true)

	// submit a crazy point difference, then a normal point
	submitPointWithDifference(t, perp, points[1], -9999999)
	submitPointWithDifference(t, perp, points[2], expectedTWAP)

	// now we leave auction and the crazy point difference will not affect the TWAP because it was in an auction period
	auctionEnd := points[2].t + int64(time.Second)
	whenAuctionStateChanges(t, perp, auctionEnd, false)

	currentPeriodLength := float64(points[len(points)-1].t - points[0].t)
	timeInAuction := float64(auctionEnd - auctionStart)
	periodFractionOutsideAuction := 1 - timeInAuction/currentPeriodLength

	fundingPayment := whenTheFundingPeriodEnds(t, perp, points[len(points)-1].t)
	assert.Equal(t, int64(periodFractionOutsideAuction*float64(expectedTWAP)), fundingPayment.Int64())
}

func testDataPointsInAuctionOutOfOrder(t *testing.T) {
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

	currentPeriodLength := float64(points[len(points)-1].t - points[0].t)
	timeInAuction := float64(points[1].t - points[0].t + points[3].t - points[2].t)
	periodFractionOutsideAuction := num.DecimalOne().Sub(num.DecimalFromFloat(timeInAuction).Div(num.DecimalFromFloat(currentPeriodLength)))
	// funding payment will be the constant diff in the first point
	expected, _ := num.IntFromDecimal(periodFractionOutsideAuction.Mul(num.DecimalFromInt64(100)))
	assert.Equal(t, num.IntToString(expected), getFundingPayment(t, perp, nd))

	// now submit a point that is mid the auction period
	submitPointWithDifference(t, perp, points[2], 200)

	expected, _ = num.IntFromDecimal(periodFractionOutsideAuction.Mul(num.DecimalFromInt64(150)))
	assert.Equal(t, num.IntToString(expected), getFundingPayment(t, perp, nd))

	// now submit a point also in before the previous point, also in an auction period
	// and its contribution should be ignored.
	crazy := &testDataPoint{t: between(a1, points[1].t), price: num.NewUint(1000)}
	submitPointWithDifference(t, perp, crazy, 9999999)
	assert.Equal(t, "49", getFundingPayment(t, perp, nd))
}

func testAuctionFundingPeriodReset(t *testing.T) {
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
	periodFractionOutsideAuction := 0.5
	assert.Equal(t, int64(periodFractionOutsideAuction*float64(expectedTWAP)), fundingPayment.Int64())

	// should still be on an auction to ending another funding period should give 0
	submitPointWithDifference(t, perp, points[1], -999999)
	fundingPayment = whenTheFundingPeriodEnds(t, perp, points[2].t)
	assert.Equal(t, int64(0), fundingPayment.Int64())

	// submit a point and leave auction
	submitPointWithDifference(t, perp, points[2], expectedTWAP)
	whenAuctionStateChanges(t, perp, between(points[2].t, points[3].t), false)

	fundingPayment = whenTheFundingPeriodEnds(t, perp, points[3].t)
	assert.Equal(t, int64(periodFractionOutsideAuction*100), fundingPayment.Int64())

	// now we're not in an auction, ending the period again will preserve that
	fundingPayment = whenTheFundingPeriodEnds(t, perp, points[3].t+int64(time.Hour))
	assert.Equal(t, int64(100), fundingPayment.Int64())
}

func testFundingDataAtInAuctionPeriodStart(t *testing.T) {
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

	end := points[0].t + int64(2*time.Second)
	fundingPayment := whenTheFundingPeriodEnds(t, perp, end)
	periodFractionOutsideAuction := 0.5
	assert.Equal(t, int64(periodFractionOutsideAuction*float64(expectedTWAP)), fundingPayment.Int64())

	// but if we query the funding payment right now it'll be zero because this 0 length, just started
	// funding period is all in auction
	fp := getFundingPayment(t, perp, end)
	assert.Equal(t, "0", fp)
}

func testPastFundingPaymentInAuction(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()
	expectedTWAP := 100000000000
	// set of the data points such that difference in averages is 0
	points := getTestDataPoints(t)

	// tell the perpetual that we are ready to accept settlement stuff
	whenLeaveOpeningAuction(t, perp, points[0].t)

	// submit the first point and enter an auction
	submitPointWithDifference(t, perp, points[0], expectedTWAP)

	// funding period ends so we have a carry-over
	end := points[0].t + int64(2*time.Second)
	fundingPayment := whenTheFundingPeriodEnds(t, perp, end)
	assert.Equal(t, int64(expectedTWAP), fundingPayment.Int64())

	whenAuctionStateChanges(t, perp, points[1].t, true)

	// now add another just an internal point
	perp.broker.EXPECT().Send(gomock.Any()).Times(1)
	require.NoError(t, perp.perpetual.SubmitDataPoint(context.Background(), points[2].price, points[2].t))

	endPrev := end
	end = points[2].t - int64(500*time.Millisecond)
	fundingPayment = whenTheFundingPeriodEnds(t, perp, end)

	currentPeriodLength := float64(end - endPrev)
	timeInAuction := float64(end - points[1].t)
	periodFractionOutsideAuction := 1 - timeInAuction/currentPeriodLength

	assert.Equal(t, int64(periodFractionOutsideAuction*float64(expectedTWAP)), fundingPayment.Int64())
}

func testPastFundingPayment(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()
	expectedTWAP := 100000000000
	// set of the data points such that difference in averages is 0
	points := getTestDataPoints(t)

	// tell the perpetual that we are ready to accept settlement stuff
	whenLeaveOpeningAuction(t, perp, points[0].t)

	// submit the first point and enter an auction
	submitPointWithDifference(t, perp, points[0], expectedTWAP)

	// funding period ends so we have a carry-over
	end := points[0].t + int64(2*time.Second)
	fundingPayment := whenTheFundingPeriodEnds(t, perp, end)
	assert.Equal(t, int64(expectedTWAP), fundingPayment.Int64())

	// now add another just an internal point
	perp.broker.EXPECT().Send(gomock.Any()).Times(1)
	require.NoError(t, perp.perpetual.SubmitDataPoint(context.Background(), points[2].price, points[2].t))

	end = points[2].t - int64(500*time.Millisecond)
	fundingPayment = whenTheFundingPeriodEnds(t, perp, end)
	assert.Equal(t, int64(expectedTWAP), fundingPayment.Int64())
}

func TestZeroLengthAuctionPeriods(t *testing.T) {
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

	// but then enter again straight away
	whenAuctionStateChanges(t, perp, points[len(points)-1].t, true)

	fundingPayment := whenTheFundingPeriodEnds(t, perp, points[len(points)-1].t)
	assert.Equal(t, "0", fundingPayment.String())
}

func TestFairgroundPanic(t *testing.T) {
	perp := testPerpetual(t)
	defer perp.ctrl.Finish()

	// tell the perpetual that we are ready to accept settlement stuff
	whenLeaveOpeningAuction(t, perp, 1708097537000000000)

	ctx := context.Background()

	// submit the first point and enter an auction
	perp.broker.EXPECT().Send(gomock.Any()).Times(4)
	perp.perpetual.AddTestExternalPoint(ctx, num.NewUint(2375757190), 1706655048000000000)
	perp.perpetual.SubmitDataPoint(ctx, num.NewUint(2375757190), 1706655048000000000)

	// enter an auction
	whenAuctionStateChanges(t, perp, 1708098633000000000, true)

	// core asks for margin increase
	perp.perpetual.GetMarginIncrease(1708098634815117249)

	// then we leave auction in the same block
	whenAuctionStateChanges(t, perp, 1708098634815117249, false)

	// add a point
	perp.perpetual.AddTestExternalPoint(ctx, num.NewUint(2375757190), 1708098648000000000)

	// then add a older point
	perp.perpetual.AddTestExternalPoint(ctx, num.NewUint(2375757190), 1708098612000000000)
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
