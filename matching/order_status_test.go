package matching_test

import (
	"testing"

	types "code.vegaprotocol.io/vega/proto"
	"github.com/stretchr/testify/assert"
)

func TestOrderStatuses(t *testing.T) {
	t.Run("FOK - stopped", testFOKStopped)
	t.Run("FOK - filled", testFOKFilled)

	t.Run("IOC - stopped", testIOCStopped)
	t.Run("IOC - partially filled", testIOCPartiallyFilled)
	t.Run("IOC - filled", testIOCFilled)
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
