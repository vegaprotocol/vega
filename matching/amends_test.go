package matching

import (
	"testing"

	types "code.vegaprotocol.io/vega/proto/gen/golang"

	"github.com/stretchr/testify/assert"
)

func TestOrderBookAmends_simpleAmend(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        2,
		Remaining:   2,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.Side_SIDE_BUY))

	amend := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	err = book.AmendOrder(&order, &amend)
	assert.NoError(t, err)
	assert.Equal(t, 1, int(book.getVolumeAtLevel(100, types.Side_SIDE_BUY)))
}

func TestOrderBookAmends_invalidPartyID(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        2,
		Remaining:   2,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.Side_SIDE_BUY))

	amend := types.Order{
		MarketID:    market,
		PartyID:     "B",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	err = book.AmendOrder(&order, &amend)
	assert.Error(t, types.ErrOrderAmendFailure, err)
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.Side_SIDE_BUY))
}

func TestOrderBookAmends_invalidPriceAmend(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        2,
		Remaining:   2,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.Side_SIDE_BUY))

	amend := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       101,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	err = book.AmendOrder(&order, &amend)
	assert.Error(t, types.ErrOrderAmendFailure, err)
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.Side_SIDE_BUY))
}

func TestOrderBookAmends_invalidSize(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        2,
		Remaining:   2,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.Side_SIDE_BUY))

	amend := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        5,
		Remaining:   5,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	err = book.AmendOrder(&order, &amend)
	assert.Error(t, types.ErrOrderAmendFailure, err)
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.Side_SIDE_BUY))
}

func TestOrderBookAmends_reduceToZero(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        2,
		Remaining:   2,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.Side_SIDE_BUY))

	amend := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        0,
		Remaining:   0,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	err = book.AmendOrder(&order, &amend)
	assert.Error(t, types.ErrOrderAmendFailure, err)
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.Side_SIDE_BUY))
}

func TestOrderBookAmends_invalidSizeDueToPartialFill(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, uint64(10), book.getVolumeAtLevel(100, types.Side_SIDE_BUY))

	order2 := types.Order{
		MarketID:    market,
		PartyID:     "B",
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        5,
		Remaining:   5,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, uint64(5), book.getVolumeAtLevel(100, types.Side_SIDE_BUY))

	amend := types.Order{
		MarketID:    market,
		PartyID:     "B",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        6,
		Remaining:   6,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	err = book.AmendOrder(&order, &amend)
	assert.Error(t, types.ErrOrderAmendFailure, err)
	assert.Equal(t, uint64(5), book.getVolumeAtLevel(100, types.Side_SIDE_BUY))
}

func TestOrderBookAmends_validSizeDueToPartialFill(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, uint64(10), book.getVolumeAtLevel(100, types.Side_SIDE_BUY))

	order2 := types.Order{
		MarketID:    market,
		PartyID:     "B",
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        5,
		Remaining:   5,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, uint64(5), book.getVolumeAtLevel(100, types.Side_SIDE_BUY))

	amend := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        3,
		Remaining:   3,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	err = book.AmendOrder(&order, &amend)
	assert.Error(t, types.ErrOrderAmendFailure, err)
	assert.Equal(t, uint64(3), book.getVolumeAtLevel(100, types.Side_SIDE_BUY))
}

func TestOrderBookAmends_noOrderToAmend(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	amend := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	err := book.AmendOrder(nil, &amend)
	assert.Error(t, types.ErrOrderNotFound, err)
}

func TestOrderBookAmends_FlipToGTT(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		Id:          "ORDER000001",
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        2,
		Remaining:   2,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	originalOrder := order
	// Create the original GTC order
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.Side_SIDE_BUY))
	assert.Equal(t, book.expiringOrders.orders.Len(), 0)

	amend := types.Order{
		Id:          "ORDER000001",
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        2,
		Remaining:   2,
		TimeInForce: types.Order_TIF_GTT,
		Type:        types.Order_TYPE_LIMIT,
		ExpiresAt:   1000000,
	}
	// Amend the order to be a GTT
	err = book.AmendOrder(&originalOrder, &amend)
	assert.Error(t, types.ErrOrderAmendFailure, err)
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.Side_SIDE_BUY))
	assert.Equal(t, book.expiringOrders.orders.Len(), 1)

	postAmendOrder := order
	amend2 := types.Order{
		Id:          "ORDER000001",
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        2,
		Remaining:   2,
		TimeInForce: types.Order_TIF_GTT,
		Type:        types.Order_TYPE_LIMIT,
		ExpiresAt:   2000000,
	}
	// Amend the expiry time
	err = book.AmendOrder(&postAmendOrder, &amend2)
	assert.Error(t, types.ErrOrderAmendFailure, err)
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.Side_SIDE_BUY))
	assert.Equal(t, book.expiringOrders.orders.Len(), 1)

	postAmendOrder2 := order
	amend3 := types.Order{
		Id:          "ORDER000001",
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        2,
		Remaining:   2,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}

	// Amend back to be a GTC
	err = book.AmendOrder(&postAmendOrder2, &amend3)
	assert.Error(t, types.ErrOrderAmendFailure, err)
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.Side_SIDE_BUY))
	assert.Equal(t, book.expiringOrders.orders.Len(), 0)
}
