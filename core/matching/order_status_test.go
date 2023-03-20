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

package matching_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/stretchr/testify/assert"
)

func TestOrderStatuses(t *testing.T) {
	t.Run("FOK - stopped", testFOKStopped)
	t.Run("FOK - filled", testFOKFilled)

	t.Run("IOC - stopped", testIOCStopped)
	t.Run("IOC - partially filled", testIOCPartiallyFilled)
	t.Run("IOC - filled", testIOCFilled)

	t.Run("GTC - active", testGTCActive)
	t.Run("GTC - stopped not filled", testGTCStoppedNotFilled)
	t.Run("GTC - cancelled not filled", testGTCCancelledNotFilled)
	t.Run("GTC - active partially filled", testGTCActivePartiallyFilled)
	t.Run("GTC - cancelled partially filled", testGTCCancelledPartiallyFilled)
	t.Run("GTC - stopped partially filled", testGTCStoppedPartiallyFilled)
	t.Run("GTC - filled", testGTCFilled)

	t.Run("GTT - active", testGTTActive)
	t.Run("GTT - cancelled not filled", testGTTCancelledNotFilled)
	t.Run("GTT - stopped not filled", testGTTStoppedNotFilled)
	t.Run("GTT - active partially filled", testGTTActivePartiallyFilled)
	t.Run("GTT - cancelled partially filled", testGTTCancelledPartiallyFilled)
	t.Run("GTT - stopped partially filled", testGTTStoppedPartiallyFilled)
	t.Run("GTT - filled", testGTTFilled)

	// the following test from the specs is not added as it is not possible to test through the order book.
	// and it's not possible for an order to become expired once it's been filled as the order is removed
	// from the book, and the book is setting up orders.
	// |      GTT      |   Yes   |   Yes   |         No        |         No        |      Filled      |
}

func testFOKStopped(t *testing.T) {
	market := "testMarket"
	partyID := "p1"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         partyID,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceFOK,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.ob.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, types.OrderStatusStopped, order.Status)
}

func testFOKFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book
	order1 := types.Order{
		ID:            "V0000000032-0000000009",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         partyID1,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	_, err := book.ob.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our fok order to be filled
	order := types.Order{
		ID:            "V0000000032-0000000010",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         partyID2,
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceFOK,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.ob.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, types.OrderStatusFilled, order.Status)
}

func testIOCStopped(t *testing.T) {
	market := "testMarket"
	partyID := "p1"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         partyID,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceIOC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.ob.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, types.OrderStatusStopped, order.Status)
}

func testIOCPartiallyFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book
	order1 := types.Order{
		ID:            "V0000000032-0000000009",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         partyID1,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	_, err := book.ob.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our IOC order to be filled
	order := types.Order{
		ID:            "V0000000032-0000000010",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         partyID2,
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          2,
		Remaining:     2,
		TimeInForce:   types.OrderTimeInForceIOC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.ob.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, types.OrderStatusPartiallyFilled, order.Status)
}

func testIOCFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book
	order1 := types.Order{
		ID:            "V0000000032-0000000009",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         partyID1,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	_, err := book.ob.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our fok order to be filled
	order := types.Order{
		ID:            "V0000000032-0000000010",
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         partyID2,
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceIOC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.ob.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, types.OrderStatusFilled, order.Status)
}

func testGTCActive(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	orderID := "v0000000000000-0000001"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	order1 := types.Order{
		Status:        types.OrderStatusActive,
		ID:            orderID,
		MarketID:      market,
		Party:         partyID1,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	_, err := book.ob.SubmitOrder(&order1)
	assert.NoError(t, err)
	assert.Equal(t, types.OrderStatusActive, order1.Status)
}

func testGTCStoppedNotFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	orderID := vgcrypto.RandomHash()

	book := getTestOrderBook(t, market)
	defer book.Finish()

	order1 := types.Order{
		Status:        types.OrderStatusActive,
		ID:            orderID,
		MarketID:      market,
		Party:         partyID1,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	_, err := book.ob.SubmitOrder(&order1)
	assert.NoError(t, err)

	// then stop the order
	rmOrders, err := book.ob.RemoveDistressedOrders([]events.MarketPosition{marketPositionFake{partyID1}})
	assert.NoError(t, err)
	assert.Len(t, rmOrders, 1)
	assert.Equal(t, types.OrderStatusStopped, rmOrders[0].Status)
}

func testGTCCancelledNotFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	orderID := vgcrypto.RandomHash()

	book := getTestOrderBook(t, market)
	defer book.Finish()

	order1 := types.Order{
		Status:        types.OrderStatusActive,
		ID:            orderID,
		MarketID:      market,
		Party:         partyID1,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	_, err := book.ob.SubmitOrder(&order1)
	assert.NoError(t, err)

	// then stop the order
	confirm, err := book.ob.CancelOrder(&order1)
	assert.NoError(t, err)
	assert.Equal(t, types.OrderStatusCancelled, confirm.Order.Status)
}

func testGTCActivePartiallyFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"
	orderID := vgcrypto.RandomHash()

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book, be partially filled, and stopped
	order1 := types.Order{
		Status:        types.OrderStatusActive,
		ID:            orderID,
		MarketID:      market,
		Party:         partyID1,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	_, err := book.ob.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our order which will consume some of the first order
	order := types.Order{
		ID:            vgcrypto.RandomHash(),
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         partyID2,
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.ob.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Len(t, confirm.PassiveOrdersAffected, 1)
	assert.Equal(t, types.OrderStatusActive, confirm.PassiveOrdersAffected[0].Status)
}

func testGTCCancelledPartiallyFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"
	orderID := vgcrypto.RandomHash()

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book, be partially filled, and stopped
	order1 := types.Order{
		Status:        types.OrderStatusActive,
		ID:            orderID,
		MarketID:      market,
		Party:         partyID1,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	_, err := book.ob.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our order which will consume some of the first order
	order := types.Order{
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         partyID2,
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	_, err = book.ob.SubmitOrder(&order)
	assert.NoError(t, err)

	// then stop the order
	confirm, err := book.ob.CancelOrder(&order1)
	assert.NoError(t, err)
	assert.NoError(t, err)
	assert.Equal(t, types.OrderStatusCancelled, confirm.Order.Status)
}

func testGTCStoppedPartiallyFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"
	orderID := vgcrypto.RandomHash()

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book, be partially filled, and stopped
	order1 := types.Order{
		Status:        types.OrderStatusActive,
		ID:            orderID,
		MarketID:      market,
		Party:         partyID1,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	_, err := book.ob.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our order which will consume some of the first order
	order := types.Order{
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         partyID2,
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	_, err = book.ob.SubmitOrder(&order)
	assert.NoError(t, err)

	// then stop the order
	rmOrders, err := book.ob.RemoveDistressedOrders([]events.MarketPosition{marketPositionFake{partyID1}})
	assert.NoError(t, err)
	assert.Len(t, rmOrders, 1)
	assert.Equal(t, types.OrderStatusStopped, rmOrders[0].Status)
}

func testGTCFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book
	order1 := types.Order{
		ID:            vgcrypto.RandomHash(),
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         partyID1,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	_, err := book.ob.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our GTC order to be filled
	order := types.Order{
		ID:            vgcrypto.RandomHash(),
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         partyID2,
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.ob.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, types.OrderStatusFilled, order.Status)
}

func testGTTActive(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	orderID := vgcrypto.RandomHash()

	book := getTestOrderBook(t, market)
	defer book.Finish()

	order1 := types.Order{
		Status:        types.OrderStatusActive,
		ID:            orderID,
		MarketID:      market,
		Party:         partyID1,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTT,
		Type:          types.OrderTypeLimit,
		ExpiresAt:     10,
	}
	_, err := book.ob.SubmitOrder(&order1)
	assert.NoError(t, err)
	assert.Equal(t, types.OrderStatusActive, order1.Status)
}

func testGTTStoppedNotFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	orderID := vgcrypto.RandomHash()

	book := getTestOrderBook(t, market)
	defer book.Finish()

	order1 := types.Order{
		Status:        types.OrderStatusActive,
		ID:            orderID,
		MarketID:      market,
		Party:         partyID1,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTT,
		Type:          types.OrderTypeLimit,
		ExpiresAt:     10,
	}
	_, err := book.ob.SubmitOrder(&order1)
	assert.NoError(t, err)

	// then stop the order
	rmOrders, err := book.ob.RemoveDistressedOrders([]events.MarketPosition{marketPositionFake{partyID1}})
	assert.NoError(t, err)
	assert.Len(t, rmOrders, 1)
	assert.Equal(t, types.OrderStatusStopped, rmOrders[0].Status)
}

func testGTTCancelledNotFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	orderID := vgcrypto.RandomHash()

	book := getTestOrderBook(t, market)
	defer book.Finish()

	order1 := types.Order{
		Status:        types.OrderStatusActive,
		ID:            orderID,
		MarketID:      market,
		Party:         partyID1,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTT,
		Type:          types.OrderTypeLimit,
		ExpiresAt:     10,
	}
	_, err := book.ob.SubmitOrder(&order1)
	assert.NoError(t, err)

	// then stop the order
	confirm, err := book.ob.CancelOrder(&order1)
	assert.NoError(t, err)
	assert.Equal(t, types.OrderStatusCancelled, confirm.Order.Status)
}

func testGTTActivePartiallyFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"
	orderID := vgcrypto.RandomHash()

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book, be partially filled
	order1 := types.Order{
		Status:        types.OrderStatusActive,
		ID:            orderID,
		MarketID:      market,
		Party:         partyID1,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTT,
		Type:          types.OrderTypeLimit,
		ExpiresAt:     10,
	}
	_, err := book.ob.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our order which will consume some of the first order
	order := types.Order{
		ID:            vgcrypto.RandomHash(),
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         partyID2,
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTT,
		Type:          types.OrderTypeLimit,
		ExpiresAt:     10,
	}
	confirm, err := book.ob.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Len(t, confirm.PassiveOrdersAffected, 1)
	assert.Equal(t, types.OrderStatusActive, confirm.PassiveOrdersAffected[0].Status)
}

func testGTTCancelledPartiallyFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"
	orderID := vgcrypto.RandomHash()

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book, be partially filled, and cancelled
	order1 := types.Order{
		Status:        types.OrderStatusActive,
		ID:            orderID,
		MarketID:      market,
		Party:         partyID1,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTT,
		Type:          types.OrderTypeLimit,
		ExpiresAt:     10,
	}
	_, err := book.ob.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our order which will consume some of the first order
	order := types.Order{
		ID:            vgcrypto.RandomHash(),
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         partyID2,
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTT,
		Type:          types.OrderTypeLimit,
		ExpiresAt:     10,
	}
	_, err = book.ob.SubmitOrder(&order)
	assert.NoError(t, err)

	// then stop the order
	confirm, err := book.ob.CancelOrder(&order1)
	assert.NoError(t, err)
	assert.NoError(t, err)
	assert.Equal(t, types.OrderStatusCancelled, confirm.Order.Status)
}

func testGTTStoppedPartiallyFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"
	orderID := vgcrypto.RandomHash()

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book, be partially filled, and stopped
	order1 := types.Order{
		Status:        types.OrderStatusActive,
		ID:            orderID,
		MarketID:      market,
		Party:         partyID1,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTT,
		Type:          types.OrderTypeLimit,
		ExpiresAt:     10,
	}
	_, err := book.ob.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our order which will consume some of the first order
	order := types.Order{
		ID:            vgcrypto.RandomHash(),
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         partyID2,
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTT,
		Type:          types.OrderTypeLimit,
		ExpiresAt:     10,
	}
	_, err = book.ob.SubmitOrder(&order)
	assert.NoError(t, err)

	// then stop the order
	rmOrders, err := book.ob.RemoveDistressedOrders([]events.MarketPosition{marketPositionFake{partyID1}})
	assert.NoError(t, err)
	assert.Len(t, rmOrders, 1)
	assert.Equal(t, types.OrderStatusStopped, rmOrders[0].Status)
}

func testGTTFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book
	order1 := types.Order{
		ID:            vgcrypto.RandomHash(),
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         partyID1,
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTT,
		Type:          types.OrderTypeLimit,
		ExpiresAt:     10,
	}
	_, err := book.ob.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our GTT order to be filled
	order := types.Order{
		ID:            vgcrypto.RandomHash(),
		Status:        types.OrderStatusActive,
		MarketID:      market,
		Party:         partyID2,
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTT,
		Type:          types.OrderTypeLimit,
		ExpiresAt:     10,
	}
	confirm, err := book.ob.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, types.OrderStatusFilled, order.Status)
}

type marketPositionFake struct {
	party string
}

func (m marketPositionFake) Party() string             { return m.party }
func (m marketPositionFake) Size() int64               { return 0 }
func (m marketPositionFake) Buy() int64                { return 0 }
func (m marketPositionFake) Sell() int64               { return 0 }
func (m marketPositionFake) Price() *num.Uint          { return num.UintZero() }
func (m marketPositionFake) BuySumProduct() *num.Uint  { return num.UintZero() }
func (m marketPositionFake) SellSumProduct() *num.Uint { return num.UintZero() }
func (m marketPositionFake) VWBuy() *num.Uint          { return num.UintZero() }
func (m marketPositionFake) VWSell() *num.Uint         { return num.UintZero() }
