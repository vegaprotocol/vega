package matching_test

import (
	"testing"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
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
	t.Run("GTT - expired not filled", testGTTExpiredNotFilled)
	t.Run("GTT - cancelled not filled", testGTTCancelledNotFilled)
	t.Run("GTT - stopped not filled", testGTTStoppedNotFilled)
	t.Run("GTT - active partially filled", testGTTActivePartiallyFilled)
	t.Run("GTT - expired partially filled", testGTTExpiredPartiallyFilled)
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
		Status:      types.Order_STATUS_ACTIVE,
		MarketID:    market,
		PartyID:     partyID,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_FOK,
		Type:        types.Order_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, types.Order_STATUS_STOPPED, order.Status)
}

func testFOKFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book
	order1 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	_, err := book.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our fok order to be filled
	order := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		MarketID:    market,
		PartyID:     partyID2,
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_FOK,
		Type:        types.Order_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, types.Order_STATUS_FILLED, order.Status)
}

func testIOCStopped(t *testing.T) {
	market := "testMarket"
	partyID := "p1"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		MarketID:    market,
		PartyID:     partyID,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_IOC,
		Type:        types.Order_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, types.Order_STATUS_STOPPED, order.Status)
}

func testIOCPartiallyFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book
	order1 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	_, err := book.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our IOC order to be filled
	order := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		MarketID:    market,
		PartyID:     partyID2,
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        2,
		Remaining:   2,
		TimeInForce: types.Order_IOC,
		Type:        types.Order_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, types.Order_STATUS_PARTIALLY_FILLED, order.Status)
}

func testIOCFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book
	order1 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	_, err := book.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our fok order to be filled
	order := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		MarketID:    market,
		PartyID:     partyID2,
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_IOC,
		Type:        types.Order_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, types.Order_STATUS_FILLED, order.Status)
}

func testGTCActive(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	orderID := "v0000000000000-0000001"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	order1 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Id:          orderID,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	_, err := book.SubmitOrder(&order1)
	assert.NoError(t, err)
	assert.Equal(t, types.Order_STATUS_ACTIVE, order1.Status)
}

func testGTCStoppedNotFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	orderID := "v0000000000000-0000001"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	order1 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Id:          orderID,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	_, err := book.SubmitOrder(&order1)
	assert.NoError(t, err)

	// then stop the order
	rmOrders, err := book.RemoveDistressedOrders([]events.MarketPosition{marketPositionFake{partyID1}})
	assert.NoError(t, err)
	assert.Len(t, rmOrders, 1)
	assert.Equal(t, types.Order_STATUS_STOPPED, rmOrders[0].Status)
}

func testGTCCancelledNotFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	orderID := "v0000000000000-0000001"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	order1 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Id:          orderID,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	_, err := book.SubmitOrder(&order1)
	assert.NoError(t, err)

	// then stop the order
	confirm, err := book.CancelOrder(&order1)
	assert.NoError(t, err)
	assert.Equal(t, types.Order_STATUS_CANCELLED, confirm.Order.Status)
}

func testGTCActivePartiallyFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"
	orderID := "v0000000000000-0000001"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book, be partially filled, and stopped
	order1 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Id:          orderID,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	_, err := book.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our order which will consume some of the first order
	order := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		MarketID:    market,
		PartyID:     partyID2,
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Len(t, confirm.PassiveOrdersAffected, 1)
	assert.Equal(t, types.Order_STATUS_ACTIVE, confirm.PassiveOrdersAffected[0].Status)
}

func testGTCCancelledPartiallyFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"
	orderID := "v0000000000000-0000001"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book, be partially filled, and stopped
	order1 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Id:          orderID,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	_, err := book.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our order which will consume some of the first order
	order := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		MarketID:    market,
		PartyID:     partyID2,
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	_, err = book.SubmitOrder(&order)
	assert.NoError(t, err)

	// then stop the order
	confirm, err := book.CancelOrder(&order1)
	assert.NoError(t, err)
	assert.NoError(t, err)
	assert.Equal(t, types.Order_STATUS_CANCELLED, confirm.Order.Status)
}

func testGTCStoppedPartiallyFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"
	orderID := "v0000000000000-0000001"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book, be partially filled, and stopped
	order1 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Id:          orderID,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	_, err := book.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our order which will consume some of the first order
	order := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		MarketID:    market,
		PartyID:     partyID2,
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	_, err = book.SubmitOrder(&order)
	assert.NoError(t, err)

	// then stop the order
	rmOrders, err := book.RemoveDistressedOrders([]events.MarketPosition{marketPositionFake{partyID1}})
	assert.NoError(t, err)
	assert.Len(t, rmOrders, 1)
	assert.Equal(t, types.Order_STATUS_STOPPED, rmOrders[0].Status)
}

func testGTCFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book
	order1 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	_, err := book.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our GTC order to be filled
	order := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		MarketID:    market,
		PartyID:     partyID2,
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, types.Order_STATUS_FILLED, order.Status)
}

func testGTTActive(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	orderID := "v0000000000000-0000001"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	order1 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Id:          orderID,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_GTT,
		Type:        types.Order_LIMIT,
		ExpiresAt:   10,
	}
	_, err := book.SubmitOrder(&order1)
	assert.NoError(t, err)
	assert.Equal(t, types.Order_STATUS_ACTIVE, order1.Status)
}

func testGTTStoppedNotFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	orderID := "v0000000000000-0000001"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	order1 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Id:          orderID,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_GTT,
		Type:        types.Order_LIMIT,
		ExpiresAt:   10,
	}
	_, err := book.SubmitOrder(&order1)
	assert.NoError(t, err)

	// then stop the order
	rmOrders, err := book.RemoveDistressedOrders([]events.MarketPosition{marketPositionFake{partyID1}})
	assert.NoError(t, err)
	assert.Len(t, rmOrders, 1)
	assert.Equal(t, types.Order_STATUS_STOPPED, rmOrders[0].Status)
}

func testGTTCancelledNotFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	orderID := "v0000000000000-0000001"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	order1 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Id:          orderID,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_GTT,
		Type:        types.Order_LIMIT,
		ExpiresAt:   10,
	}
	_, err := book.SubmitOrder(&order1)
	assert.NoError(t, err)

	// then stop the order
	confirm, err := book.CancelOrder(&order1)
	assert.NoError(t, err)
	assert.Equal(t, types.Order_STATUS_CANCELLED, confirm.Order.Status)
}

func testGTTActivePartiallyFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"
	orderID := "v0000000000000-0000001"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book, be partially filled
	order1 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Id:          orderID,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_GTT,
		Type:        types.Order_LIMIT,
		ExpiresAt:   10,
	}
	_, err := book.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our order which will consume some of the first order
	order := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		MarketID:    market,
		PartyID:     partyID2,
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTT,
		Type:        types.Order_LIMIT,
		ExpiresAt:   10,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Len(t, confirm.PassiveOrdersAffected, 1)
	assert.Equal(t, types.Order_STATUS_ACTIVE, confirm.PassiveOrdersAffected[0].Status)
}

func testGTTCancelledPartiallyFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"
	orderID := "v0000000000000-0000001"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book, be partially filled, and cancelled
	order1 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Id:          orderID,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_GTT,
		Type:        types.Order_LIMIT,
		ExpiresAt:   10,
	}
	_, err := book.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our order which will consume some of the first order
	order := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		MarketID:    market,
		PartyID:     partyID2,
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTT,
		Type:        types.Order_LIMIT,
		ExpiresAt:   10,
	}
	_, err = book.SubmitOrder(&order)
	assert.NoError(t, err)

	// then stop the order
	confirm, err := book.CancelOrder(&order1)
	assert.NoError(t, err)
	assert.NoError(t, err)
	assert.Equal(t, types.Order_STATUS_CANCELLED, confirm.Order.Status)
}

func testGTTStoppedPartiallyFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"
	orderID := "v0000000000000-0000001"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book, be partially filled, and stopped
	order1 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Id:          orderID,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_GTT,
		Type:        types.Order_LIMIT,
		ExpiresAt:   10,
	}
	_, err := book.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our order which will consume some of the first order
	order := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		MarketID:    market,
		PartyID:     partyID2,
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTT,
		Type:        types.Order_LIMIT,
		ExpiresAt:   10,
	}
	_, err = book.SubmitOrder(&order)
	assert.NoError(t, err)

	// then stop the order
	rmOrders, err := book.RemoveDistressedOrders([]events.MarketPosition{marketPositionFake{partyID1}})
	assert.NoError(t, err)
	assert.Len(t, rmOrders, 1)
	assert.Equal(t, types.Order_STATUS_STOPPED, rmOrders[0].Status)
}

func testGTTFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book
	order1 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTT,
		Type:        types.Order_LIMIT,
		ExpiresAt:   10,
	}
	_, err := book.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our GTT order to be filled
	order := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		MarketID:    market,
		PartyID:     partyID2,
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTT,
		Type:        types.Order_LIMIT,
		ExpiresAt:   10,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, types.Order_STATUS_FILLED, order.Status)
}

func testGTTExpiredNotFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	orderID := "v0000000000000-0000001"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book, and expired
	order1 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Id:          orderID,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_GTT,
		Type:        types.Order_LIMIT,
		ExpiresAt:   10,
	}
	_, err := book.SubmitOrder(&order1)
	assert.NoError(t, err)

	// then remove expired, set 1 sec after order exp time.
	orders := book.RemoveExpiredOrders(11)
	assert.Len(t, orders, 1)
	assert.Equal(t, types.Order_STATUS_EXPIRED, orders[0].Status)
}

func testGTTExpiredPartiallyFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"
	orderID := "v0000000000000-0000001"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book, be partially filled, and expired
	order1 := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		Id:          orderID,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_GTT,
		Type:        types.Order_LIMIT,
		ExpiresAt:   10,
	}
	_, err := book.SubmitOrder(&order1)
	assert.NoError(t, err)

	// now place our order which will consume some of the first order
	order := types.Order{
		Status:      types.Order_STATUS_ACTIVE,
		MarketID:    market,
		PartyID:     partyID2,
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTT,
		Type:        types.Order_LIMIT,
		ExpiresAt:   10,
	}
	_, err = book.SubmitOrder(&order)
	assert.NoError(t, err)

	// then remove expired, set 1 sec after order exp time.
	orders := book.RemoveExpiredOrders(11)
	assert.Len(t, orders, 1)
	assert.Equal(t, types.Order_STATUS_EXPIRED, orders[0].Status)
}

type marketPositionFake struct {
	party string
}

func (m marketPositionFake) Party() string { return m.party }
func (m marketPositionFake) Size() int64   { return 0 }
func (m marketPositionFake) Buy() int64    { return 0 }
func (m marketPositionFake) Sell() int64   { return 0 }
func (m marketPositionFake) Price() uint64 { return 0 }
