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

package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func submitIcebergOrder(t *testing.T, book *tstOB, size, peak, minPeak uint64, addToBook bool) (*types.Order, *types.OrderConfirmation) {
	t.Helper()
	o := &types.Order{
		ID:            vgcrypto.RandomHash(),
		Status:        types.OrderStatusActive,
		MarketID:      book.marketID,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          size,
		Remaining:     size,
		TimeInForce:   types.OrderTimeInForceGTT,
		Type:          types.OrderTypeLimit,
		ExpiresAt:     10,
		IcebergOrder: &types.IcebergOrder{
			InitialPeakSize: peak,
			MinimumPeakSize: minPeak,
		},
	}
	confirm, err := book.SubmitOrder(o)
	require.NoError(t, err)

	if addToBook {
		// aggressive iceberg orders do not naturally sit on the book and are added a different way so
		// we do that here
		o.Remaining = peak
		o.IcebergOrder.ReservedRemaining = size - peak
		book.SubmitIcebergOrder(o)
	}
	return o, confirm
}

func submitCrossedOrder(t *testing.T, book *tstOB, size uint64) (*types.Order, *types.OrderConfirmation) {
	t.Helper()
	o := &types.Order{
		ID:            vgcrypto.RandomHash(),
		Status:        types.OrderStatusActive,
		MarketID:      book.marketID,
		Party:         "B",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          size,
		Remaining:     size,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(o)
	require.NoError(t, err)
	return o, confirm
}

func TestIcebergFullPeakConsumed(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// submit an iceberg order that sits on the book
	iceberg, confirm := submitIcebergOrder(t, book, 100, 4, 2, false)
	assert.Equal(t, 0, len(confirm.Trades))

	// it doesn't get added to the book because they are special
	_, err := book.GetOrderByID(iceberg.ID)
	assert.ErrorIs(t, err, ErrOrderDoesNotExist)

	// we add it in our special way after doing the peak shuffle
	iceberg.Remaining = 4
	iceberg.IcebergOrder.ReservedRemaining = 96
	book.SubmitIcebergOrder(iceberg)

	// check it is now on the book
	_, err = book.GetOrderByID(iceberg.ID)
	assert.NoError(t, err)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(4))

	// submit an order bigger than the peak
	o, confirm := submitCrossedOrder(t, book, 10)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, uint64(6), o.Remaining)

	// there will be no buy level or volume because the iceberg's peak has been consumed
	assert.Equal(t, 0, book.getNumberOfBuyLevels())
	assert.Equal(t, uint64(0), book.getTotalBuyVolume())
	assert.Equal(t, uint64(0), iceberg.Remaining)
	assert.Equal(t, uint64(96), iceberg.IcebergOrder.ReservedRemaining)

	// check the the iceberg is ready for a refresh and refresh it
	require.True(t, iceberg.NeedsRefreshing())

	// cancel and resubmit
	book.CancelOrder(iceberg)
	iceberg.Remaining = 4
	iceberg.IcebergOrder.ReservedRemaining = 92
	book.SubmitIcebergOrder(iceberg)

	// it is now on the book and the buy volume has increased by the peak
	assert.Equal(t, 1, book.getNumberOfBuyLevels())
	assert.Equal(t, book.getTotalBuyVolume(), uint64(4))
}

func TestIcebergPeakAboveMinimum(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	iceberg, confirm := submitIcebergOrder(t, book, 100, 4, 2, true)
	assert.Equal(t, 0, len(confirm.Trades))

	// submit order that only takes a little off the peak
	_, confirm = submitCrossedOrder(t, book, 1)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, 1, book.getNumberOfBuyLevels())
	assert.Equal(t, uint64(3), book.getTotalBuyVolume())

	// check a refresh is not needed
	require.False(t, iceberg.NeedsRefreshing())

	// now submit another order that *will* remove the peak
	_, confirm = submitCrossedOrder(t, book, 1000)
	assert.Equal(t, 1, len(confirm.Trades))

	assert.Equal(t, uint64(0), iceberg.Remaining)
	assert.Equal(t, uint64(96), iceberg.IcebergOrder.ReservedRemaining)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(0))
}

func TestIcebergAggressiveTakesAll(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	_, confirm := submitCrossedOrder(t, book, 10)
	assert.Equal(t, 0, len(confirm.Trades))

	// submit the iceberg as an aggressive order and more than its peak is consumed
	_, confirm = submitIcebergOrder(t, book, 50, 4, 2, false)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, uint64(10), confirm.Trades[0].Size)
}

func TestAggressiveIcebergFullyFilled(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	_, confirm := submitCrossedOrder(t, book, 1000)
	assert.Equal(t, 0, len(confirm.Trades))

	// submit the aggressice iceberg that will be fully filled
	iceberg, confirm := submitIcebergOrder(t, book, 100, 4, 2, false)
	assert.Equal(t, 1, len(confirm.Trades))

	// check that
	assert.Equal(t, uint64(0), iceberg.Remaining)
	assert.Equal(t, uint64(0), iceberg.IcebergOrder.ReservedRemaining)
	assert.Equal(t, types.OrderStatusFilled, iceberg.Status)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(0))
	assert.False(t, iceberg.NeedsRefreshing())
}

func TestIcebergPeakAboveMinimumTradeToZero(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	iceberg, confirm := submitIcebergOrder(t, book, 100, 4, 2, true)
	assert.Equal(t, 0, len(confirm.Trades))

	// submit an order that takes the berg below its minimum peak, but is not zero
	_, confirm = submitCrossedOrder(t, book, 3)
	assert.Equal(t, 1, len(confirm.Trades))

	assert.Equal(t, types.OrderStatusActive, iceberg.Status)
	assert.True(t, iceberg.NeedsRefreshing())
	assert.Equal(t, uint64(1), iceberg.Remaining)
	assert.Equal(t, uint64(96), iceberg.IcebergOrder.ReservedRemaining)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(1))

	// without refreshing submit another order to take the rest
	_, confirm = submitCrossedOrder(t, book, 10)
	assert.Equal(t, 1, len(confirm.Trades))

	// check it still needs a refresh
	assert.Equal(t, uint64(0), iceberg.Remaining)
	assert.Equal(t, uint64(96), iceberg.IcebergOrder.ReservedRemaining)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(0))
	assert.True(t, iceberg.NeedsRefreshing())

	// send in *another* order and check it doesn't trade since the iceberg
	// is hidden until a refresh
	_, confirm = submitCrossedOrder(t, book, 100)
	assert.Equal(t, 0, len(confirm.Trades))
}

func TestIcebergRefreshToPartialPeak(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// submit an iceberg order that sits on the book with a big peak
	iceberg, confirm := submitIcebergOrder(t, book, 100, 90, 2, true)
	assert.Equal(t, 0, len(confirm.Trades))

	// expect the volume to be the peak size
	assert.Equal(t, book.getTotalBuyVolume(), uint64(90))

	// submit an order that takes almost the full peak
	_, confirm = submitCrossedOrder(t, book, 89)
	assert.Equal(t, 1, len(confirm.Trades))

	// remaining + reserver < initial peak
	assert.Equal(t, uint64(1), iceberg.Remaining)
	assert.Equal(t, uint64(10), iceberg.IcebergOrder.ReservedRemaining)

	// check the the iceberg is ready for a refresh and refresh it
	require.True(t, iceberg.NeedsRefreshing())

	book.CancelOrder(iceberg)
	iceberg.Remaining = 11
	iceberg.IcebergOrder.ReservedRemaining = 0
	book.SubmitIcebergOrder(iceberg)

	// check volume change
	assert.Equal(t, book.getTotalBuyVolume(), uint64(11))
}

func TestIcebergTimePriorityLostOnRefresh(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// submit an iceberg order that sits on the book with a big peak
	iceberg, confirm := submitIcebergOrder(t, book, 100, 10, 8, true)
	assert.Equal(t, 0, len(confirm.Trades))

	// now submit a second order that will be next in line
	iceberg2, confirm := submitIcebergOrder(t, book, 100, 100, 1, true)
	assert.Equal(t, 0, len(confirm.Trades))

	// expect the volume to be the peak size
	assert.Equal(t, book.getTotalBuyVolume(), uint64(110))

	// submit a order that will take out some of the peak of the first iceberg
	_, confirm = submitCrossedOrder(t, book, 5)
	assert.Equal(t, 1, len(confirm.Trades))

	assert.Equal(t, uint64(5), iceberg.Remaining)
	assert.Equal(t, uint64(90), iceberg.IcebergOrder.ReservedRemaining)

	// check the the iceberg is ready for a refresh and refresh it
	require.True(t, iceberg.NeedsRefreshing())

	// submit a small order and check it still trades with iceberg1
	_, confirm = submitCrossedOrder(t, book, 1)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, iceberg.ID, confirm.PassiveOrdersAffected[0].ID)

	// refresh the first iceberg
	book.CancelOrder(iceberg)
	iceberg.Remaining = 10
	iceberg.IcebergOrder.ReservedRemaining = 85
	book.SubmitIcebergOrder(iceberg)

	// a new small order will match with the second iceberg
	_, confirm = submitCrossedOrder(t, book, 1)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, iceberg2.ID, confirm.PassiveOrdersAffected[0].ID)
}

func TestAmendIceberg(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// submit an iceberg order that sits on the book with a big peak
	iceberg, confirm := submitIcebergOrder(t, book, 100, 10, 8, true)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, book.getTotalBuyVolume(), uint64(10))

	// amend iceberg such that the size is increased and the reserve is increased
	amend := iceberg.Clone()
	amend.Size = 150
	amend.IcebergOrder.ReservedRemaining = 140
	err := book.AmendOrder(iceberg, amend)
	require.NoError(t, err)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(10))

	// amend iceberg such that the volume is decreased but not enough to eat into the peak
	amend = iceberg.Clone()
	amend.Size = 140
	amend.IcebergOrder.ReservedRemaining = 130
	err = book.AmendOrder(iceberg, amend)
	require.NoError(t, err)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(10))

	// decrease again such that reserved is 0 and peak is reduced
	amend = iceberg.Clone()
	amend.Size = 5
	amend.Remaining = 5
	amend.IcebergOrder.ReservedRemaining = 0
	err = book.AmendOrder(iceberg, amend)
	require.NoError(t, err)
	assert.Equal(t, book.getTotalBuyVolume(), uint64(5))
}
