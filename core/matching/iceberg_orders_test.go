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
		// book.SubmitIcebergOrder(o)
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

func submitCrossedWashOrder(t *testing.T, book *tstOB, size uint64) (*types.Order, *types.OrderConfirmation) {
	t.Helper()
	o := &types.Order{
		ID:            vgcrypto.RandomHash(),
		Status:        types.OrderStatusActive,
		MarketID:      book.marketID,
		Party:         "A",
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

func getTradesCrossedOrder(t *testing.T, book *tstOB, size uint64) []*types.Trade {
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
	trades, err := book.GetTrades(o)
	require.NoError(t, err)
	return trades
}

func TestIcebergsFakeUncross(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// submit an iceberg order that sits on the book
	iceberg, confirm := submitIcebergOrder(t, book, 100, 4, 2, false)
	assert.Equal(t, 0, len(confirm.Trades))

	// check it is now on the book
	_, err := book.GetOrderByID(iceberg.ID)
	assert.NoError(t, err)
	assert.Equal(t, uint64(100), book.getTotalBuyVolume())

	// check the peaks are proper
	assert.Equal(t, uint64(4), iceberg.Remaining)
	assert.Equal(t, uint64(96), iceberg.IcebergOrder.ReservedRemaining)

	// submit an order bigger than the peak
	trades := getTradesCrossedOrder(t, book, 10)
	assert.Equal(t, 1, len(trades))
	assert.Equal(t, uint64(10), trades[0].Size)

	// now submit it for real, and check refresh happens
	o, confirm := submitCrossedOrder(t, book, 10)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, uint64(10), trades[0].Size)
	assert.Equal(t, uint64(0), o.Remaining)
}

func TestIcebergFullPeakConsumedExactly(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// submit an iceberg order that sits on the book
	iceberg, confirm := submitIcebergOrder(t, book, 100, 40, 10, false)
	assert.Equal(t, 0, len(confirm.Trades))

	// check it is now on the book
	_, err := book.GetOrderByID(iceberg.ID)
	assert.NoError(t, err)
	assert.Equal(t, uint64(100), book.getTotalBuyVolume())

	trades := getTradesCrossedOrder(t, book, 40)
	assert.Equal(t, 1, len(trades))
	assert.Equal(t, uint64(40), trades[0].Size)

	// now submit it and check it gets filled
	o, confirm := submitCrossedOrder(t, book, 40)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, types.OrderStatusFilled, o.Status)

	// check that the iceberg has been refreshed and book volume is back at 40
	assert.Equal(t, 1, book.getNumberOfBuyLevels())
	assert.Equal(t, uint64(60), book.getTotalBuyVolume())
	assert.Equal(t, uint64(40), iceberg.Remaining)
	assert.Equal(t, uint64(20), iceberg.IcebergOrder.ReservedRemaining)
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
	assert.Equal(t, uint64(99), book.getTotalBuyVolume())

	// now submit another order that *will* remove the rest of the peak
	_, confirm = submitCrossedOrder(t, book, 3)
	assert.Equal(t, 1, len(confirm.Trades))

	assert.Equal(t, uint64(4), iceberg.Remaining)
	assert.Equal(t, uint64(92), iceberg.IcebergOrder.ReservedRemaining)
	assert.Equal(t, uint64(96), book.getTotalBuyVolume())
}

func TestIcebergAggressiveTakesAll(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	_, confirm := submitCrossedOrder(t, book, 10)
	assert.Equal(t, 0, len(confirm.Trades))

	// submit the iceberg as an aggressive order and more than its peak is consumed
	o, confirm := submitIcebergOrder(t, book, 50, 4, 2, false)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, uint64(10), confirm.Trades[0].Size)

	// now check iceberg sits on the book with the correct peaks
	assert.Equal(t, uint64(4), o.Remaining)
	assert.Equal(t, uint64(36), o.IcebergOrder.ReservedRemaining)
	assert.Equal(t, uint64(40), book.getTotalBuyVolume())
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
	assert.Equal(t, uint64(0), book.getTotalBuyVolume())
	assert.Equal(t, uint64(900), book.getTotalSellVolume())
}

func TestIcebergPeakBelowMinimumNotZero(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	iceberg, confirm := submitIcebergOrder(t, book, 100, 4, 2, true)
	assert.Equal(t, 0, len(confirm.Trades))

	// submit an order that takes the berg below its minimum peak, but is not zero
	_, confirm = submitCrossedOrder(t, book, 3)
	assert.Equal(t, 1, len(confirm.Trades))

	// check it refreshes properly
	assert.Equal(t, types.OrderStatusActive, iceberg.Status)
	assert.Equal(t, uint64(4), iceberg.Remaining)
	assert.Equal(t, uint64(93), iceberg.IcebergOrder.ReservedRemaining)
	assert.Equal(t, uint64(97), book.getTotalBuyVolume())

	// put in another order which will eat into the remaining
	submitCrossedOrder(t, book, 10)
	assert.Equal(t, uint64(4), iceberg.Remaining)
	assert.Equal(t, uint64(83), iceberg.IcebergOrder.ReservedRemaining)
	assert.Equal(t, uint64(87), book.getTotalBuyVolume())
}

func TestIcebergRefreshToPartialPeak(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// submit an iceberg order that sits on the book with a big peak
	iceberg, confirm := submitIcebergOrder(t, book, 100, 90, 2, true)
	assert.Equal(t, 0, len(confirm.Trades))

	// expect the volume to be the peak size
	assert.Equal(t, uint64(100), book.getTotalBuyVolume())

	// submit an order that takes almost the full peak
	_, confirm = submitCrossedOrder(t, book, 89)
	assert.Equal(t, 1, len(confirm.Trades))

	// remaining + reserved < initial peak
	assert.Equal(t, uint64(11), iceberg.Remaining)
	assert.Equal(t, uint64(0), iceberg.IcebergOrder.ReservedRemaining)
	assert.Equal(t, uint64(11), book.getTotalBuyVolume())

	// check we can now fill it and the iceberg is removed
	_, confirm = submitCrossedOrder(t, book, 100)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, uint64(0), iceberg.Remaining)
	assert.Equal(t, uint64(0), iceberg.IcebergOrder.ReservedRemaining)
	assert.Equal(t, types.OrderStatusFilled, iceberg.Status)
	assert.Equal(t, uint64(0), book.getTotalBuyVolume())
}

func TestIcebergHiddenDistribution(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// submit 3 iceberg orders
	iceberg1, confirm := submitIcebergOrder(t, book, 300, 100, 2, false)
	assert.Equal(t, 0, len(confirm.Trades))

	iceberg2, confirm := submitIcebergOrder(t, book, 300, 200, 2, false)
	assert.Equal(t, 0, len(confirm.Trades))

	iceberg3, _ := submitIcebergOrder(t, book, 200, 100, 2, false)

	// submit a big order such that all three peaks are consumed (100 + 200 + 100 = 400)
	// and the left over is 300
	trades := getTradesCrossedOrder(t, book, 700)
	assert.Equal(t, 3, len(trades))
	assert.Equal(t, uint64(250), trades[0].Size)
	assert.Equal(t, uint64(275), trades[1].Size)
	assert.Equal(t, uint64(175), trades[2].Size)

	// now submit it for real
	o, confirm := submitCrossedOrder(t, book, 700)
	assert.Equal(t, 3, len(confirm.Trades))
	assert.Equal(t, types.OrderStatusFilled, o.Status)

	// check iceberg one has been refresh properly
	assert.Equal(t, uint64(50), iceberg1.Remaining)
	assert.Equal(t, uint64(0), iceberg1.IcebergOrder.ReservedRemaining)

	assert.Equal(t, uint64(25), iceberg2.Remaining)
	assert.Equal(t, uint64(0), iceberg2.IcebergOrder.ReservedRemaining)

	assert.Equal(t, uint64(25), iceberg3.Remaining)
	assert.Equal(t, uint64(0), iceberg3.IcebergOrder.ReservedRemaining)
}

func TestIcebergHiddenDistributionCrumbs(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// submit 3 iceberg orders of equal sizes
	iceberg1, _ := submitIcebergOrder(t, book, 500, 100, 2, false)
	iceberg2, _ := submitIcebergOrder(t, book, 500, 100, 2, false)
	iceberg3, _ := submitIcebergOrder(t, book, 500, 100, 100, false)
	assert.Equal(t, uint64(1500), book.getTotalBuyVolume())

	// submit a big order such that all three peaks are consumed (100 + 100 + 100 = 300)
	// and the left over is 100 to be divided between three
	trades := getTradesCrossedOrder(t, book, 400)
	assert.Equal(t, 3, len(trades))
	assert.Equal(t, uint64(134), trades[0].Size)
	assert.Equal(t, uint64(133), trades[1].Size)
	assert.Equal(t, uint64(133), trades[2].Size)

	// now submit it for real
	o, confirm := submitCrossedOrder(t, book, 400)
	assert.Equal(t, 3, len(confirm.Trades))
	assert.Equal(t, types.OrderStatusFilled, o.Status)

	// check iceberg one has been refresh properly
	assert.Equal(t, uint64(100), iceberg1.Remaining)
	assert.Equal(t, uint64(266), iceberg1.IcebergOrder.ReservedRemaining)

	assert.Equal(t, uint64(100), iceberg2.Remaining)
	assert.Equal(t, uint64(267), iceberg2.IcebergOrder.ReservedRemaining)

	assert.Equal(t, uint64(100), iceberg3.Remaining)
	assert.Equal(t, uint64(267), iceberg3.IcebergOrder.ReservedRemaining)
}

func TestIcebergHiddenDistributionFullyConsumed(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// submit 3 iceberg orders
	iceberg1, confirm := submitIcebergOrder(t, book, 300, 100, 2, false)
	assert.Equal(t, 0, len(confirm.Trades))

	iceberg2, confirm := submitIcebergOrder(t, book, 300, 200, 2, false)
	assert.Equal(t, 0, len(confirm.Trades))

	iceberg3, _ := submitIcebergOrder(t, book, 200, 100, 2, false)

	// submit a big order such that all three peaks are consumed (100 + 200 + 100 = 400)
	// and all of the hidden volume (200 + 100 + 100 = 400)
	trades := getTradesCrossedOrder(t, book, 1000)
	assert.Equal(t, 3, len(trades))
	assert.Equal(t, uint64(300), trades[0].Size)
	assert.Equal(t, uint64(300), trades[1].Size)
	assert.Equal(t, uint64(200), trades[2].Size)

	// now submit it for real
	o, confirm := submitCrossedOrder(t, book, 1000)
	assert.Equal(t, 3, len(confirm.Trades))
	assert.Equal(t, types.OrderStatusActive, o.Status)
	assert.Equal(t, uint64(200), o.Remaining)

	// check iceberg one has been refresh properly
	assert.Equal(t, uint64(0), iceberg1.Remaining)
	assert.Equal(t, uint64(0), iceberg1.IcebergOrder.ReservedRemaining)
	assert.Equal(t, types.OrderStatusFilled, iceberg1.Status)

	assert.Equal(t, uint64(0), iceberg2.Remaining)
	assert.Equal(t, uint64(0), iceberg2.IcebergOrder.ReservedRemaining)
	assert.Equal(t, types.OrderStatusFilled, iceberg2.Status)

	assert.Equal(t, uint64(0), iceberg3.Remaining)
	assert.Equal(t, uint64(0), iceberg3.IcebergOrder.ReservedRemaining)
	assert.Equal(t, types.OrderStatusFilled, iceberg2.Status)
}

func TestIcebergHiddenDistributionPrimeHidden(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// submit 3 iceberg orders
	iceberg1, confirm := submitIcebergOrder(t, book, 300, 43, 2, false)
	assert.Equal(t, 0, len(confirm.Trades))

	iceberg2, confirm := submitIcebergOrder(t, book, 300, 67, 2, false)
	assert.Equal(t, 0, len(confirm.Trades))

	iceberg3, _ := submitIcebergOrder(t, book, 200, 9, 2, false)

	// submit a big order such that all three peaks are consumed (43 + 67 + 9 = 119)
	// and the individual remaining hidden volumes are prime (257, 233, 191) such
	// that the distributed amount will make nasty precision crumbs
	trades := getTradesCrossedOrder(t, book, 250)
	assert.Equal(t, 3, len(trades))
	assert.Equal(t, uint64(94), trades[0].Size)
	assert.Equal(t, uint64(111), trades[1].Size)
	assert.Equal(t, uint64(45), trades[2].Size)

	// now submit it for real
	o, confirm := submitCrossedOrder(t, book, 250)
	assert.Equal(t, 3, len(confirm.Trades))
	assert.Equal(t, types.OrderStatusFilled, o.Status)
	assert.Equal(t, uint64(0), o.Remaining)

	// check iceberg one has been refresh properly
	assert.Equal(t, uint64(43), iceberg1.Remaining)
	assert.Equal(t, uint64(163), iceberg1.IcebergOrder.ReservedRemaining)
	assert.Equal(t, types.OrderStatusActive, iceberg1.Status)

	assert.Equal(t, uint64(67), iceberg2.Remaining)
	assert.Equal(t, uint64(122), iceberg2.IcebergOrder.ReservedRemaining)
	assert.Equal(t, types.OrderStatusActive, iceberg2.Status)

	assert.Equal(t, uint64(9), iceberg3.Remaining)
	assert.Equal(t, uint64(146), iceberg3.IcebergOrder.ReservedRemaining)
	assert.Equal(t, types.OrderStatusActive, iceberg2.Status)

	assert.Equal(t, uint64(550), book.getTotalBuyVolume())
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
	assert.Equal(t, uint64(200), book.getTotalBuyVolume())

	// submit a order that will take out some of the peak of the first iceberg, check its refreshed
	_, confirm = submitCrossedOrder(t, book, 5)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, uint64(10), iceberg.Remaining)
	assert.Equal(t, uint64(85), iceberg.IcebergOrder.ReservedRemaining)

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
	assert.Equal(t, uint64(100), book.getTotalBuyVolume())

	// amend iceberg such that the size is increased and the reserve is increased
	amend := iceberg.Clone()
	amend.Size = 150
	amend.IcebergOrder.ReservedRemaining = 140
	err := book.AmendOrder(iceberg, amend)
	require.NoError(t, err)
	assert.Equal(t, uint64(150), book.getTotalBuyVolume())

	// amend iceberg such that the volume is decreased but not enough to eat into the peak
	amend = iceberg.Clone()
	amend.Size = 140
	amend.IcebergOrder.ReservedRemaining = 130
	err = book.AmendOrder(iceberg, amend)
	require.NoError(t, err)
	assert.Equal(t, uint64(140), book.getTotalBuyVolume())

	// decrease again such that reserved is 0 and peak is reduced
	amend = iceberg.Clone()
	amend.Size = 5
	amend.Remaining = 5
	amend.IcebergOrder.ReservedRemaining = 0
	err = book.AmendOrder(iceberg, amend)
	require.NoError(t, err)
	assert.Equal(t, uint64(5), book.getTotalBuyVolume())
}

func TestIcebergWashTradePassiveIceberg(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// submit an iceberg order that sits on the book
	_, confirm := submitIcebergOrder(t, book, 100, 10, 8, true)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, uint64(100), book.getTotalBuyVolume())

	// same party submits and order which trades with themselves
	o, confirm := submitCrossedWashOrder(t, book, 10)
	assert.Equal(t, types.OrderStatusStopped, o.Status)
	assert.Equal(t, 0, len(confirm.Trades))
}

func TestIcebergWashTradeAggressiveIceberg(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// party submits an order that sits on the book
	_, confirm := submitCrossedWashOrder(t, book, 10)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, uint64(10), book.getTotalSellVolume())

	// same party submit an aggressive iceberg order that trades with themselves
	iceberg, confirm := submitIcebergOrder(t, book, 100, 10, 8, true)
	assert.Equal(t, types.OrderStatusStopped, iceberg.Status)
	assert.Equal(t, 0, len(confirm.Trades))
}

func TestIcebergWashTradeAggressiveIcebergPartialFill(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	_, confirm := submitCrossedOrder(t, book, 5)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, uint64(5), book.getTotalSellVolume())

	_, confirm = submitCrossedWashOrder(t, book, 10)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, uint64(15), book.getTotalSellVolume())

	// submit an iceberg order that partially trades, then causes a wash trade and so
	// is not put on the book
	iceberg, confirm := submitIcebergOrder(t, book, 100, 10, 8, true)
	assert.Equal(t, types.OrderStatusPartiallyFilled, iceberg.Status)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, uint64(10), book.getTotalSellVolume())
}
