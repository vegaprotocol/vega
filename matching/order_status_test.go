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
}

func testFOKStopped(t *testing.T) {
	market := "testMarket"
	partyID := "p1"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		PartyID:     partyID,
		Side:        types.Side_Sell,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_FOK,
		Type:        types.Order_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, types.Order_Stopped, order.Status)
}

func testFOKFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book
	order1 := types.Order{
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_Sell,
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
		MarketID:    market,
		PartyID:     partyID2,
		Side:        types.Side_Buy,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_FOK,
		Type:        types.Order_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, types.Order_Filled, order.Status)
}

func testIOCStopped(t *testing.T) {
	market := "testMarket"
	partyID := "p1"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		PartyID:     partyID,
		Side:        types.Side_Sell,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_IOC,
		Type:        types.Order_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, types.Order_Stopped, order.Status)
}

func testIOCPartiallyFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book
	order1 := types.Order{
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_Sell,
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
		MarketID:    market,
		PartyID:     partyID2,
		Side:        types.Side_Buy,
		Price:       100,
		Size:        2,
		Remaining:   2,
		TimeInForce: types.Order_IOC,
		Type:        types.Order_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, types.Order_PartiallyFilled, order.Status)
}

func testIOCFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book
	order1 := types.Order{
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_Sell,
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
		MarketID:    market,
		PartyID:     partyID2,
		Side:        types.Side_Buy,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_IOC,
		Type:        types.Order_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, types.Order_Filled, order.Status)
}

func testGTCActive(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	orderID := "v0000000000000-0000001"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book, be partially filled, and stopped
	order1 := types.Order{
		Id:          orderID,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_Sell,
		Price:       100,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	_, err := book.SubmitOrder(&order1)
	assert.NoError(t, err)
	assert.Equal(t, types.Order_Active, order1.Status)
}

func testGTCStoppedNotFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	orderID := "v0000000000000-0000001"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book, be partially filled, and stopped
	order1 := types.Order{
		Id:          orderID,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_Sell,
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
	assert.Equal(t, types.Order_Stopped, rmOrders[0].Status)
}

func testGTCCancelledNotFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	orderID := "v0000000000000-0000001"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book, be partially filled, and stopped
	order1 := types.Order{
		Id:          orderID,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_Sell,
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
	assert.Equal(t, types.Order_Cancelled, confirm.Order.Status)
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
		Id:          orderID,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_Sell,
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
		MarketID:    market,
		PartyID:     partyID2,
		Side:        types.Side_Buy,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Len(t, confirm.PassiveOrdersAffected, 1)
	assert.Equal(t, types.Order_Active, confirm.PassiveOrdersAffected[0].Status)
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
		Id:          orderID,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_Sell,
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
		MarketID:    market,
		PartyID:     partyID2,
		Side:        types.Side_Buy,
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
	assert.Equal(t, types.Order_PartiallyFilled, confirm.Order.Status)
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
		Id:          orderID,
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_Sell,
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
		MarketID:    market,
		PartyID:     partyID2,
		Side:        types.Side_Buy,
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
	assert.Equal(t, types.Order_Stopped, rmOrders[0].Status)
}

func testGTCFilled(t *testing.T) {
	market := "testMarket"
	partyID1 := "p1"
	partyID2 := "p2"

	book := getTestOrderBook(t, market)
	defer book.Finish()

	// place a first order to sit in the book
	order1 := types.Order{
		MarketID:    market,
		PartyID:     partyID1,
		Side:        types.Side_Sell,
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
		MarketID:    market,
		PartyID:     partyID2,
		Side:        types.Side_Buy,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, types.Order_Filled, order.Status)
}

type marketPositionFake struct {
	party string
}

func (m marketPositionFake) Party() string { return m.party }
func (m marketPositionFake) Size() int64   { return 0 }
func (m marketPositionFake) Buy() int64    { return 0 }
func (m marketPositionFake) Sell() int64   { return 0 }
func (m marketPositionFake) Price() uint64 { return 0 }
