package matching

import (
	"testing"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func TestOrderBook_closeOutPriceBuy(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	// Normal case
	price, err := book.GetCloseoutPrice(10, types.Side_SIDE_BUY)
	assert.NoError(t, err)
	assert.Equal(t, price, uint64(100))

	// Incorrect size
	price, err = book.GetCloseoutPrice(0, types.Side_SIDE_BUY)
	assert.Error(t, err, ErrInvalidVolume)
	assert.Equal(t, price, uint64(0))

	// Not enough on the book
	price, err = book.GetCloseoutPrice(200, types.Side_SIDE_BUY)
	assert.Error(t, err, ErrNotEnoughOrders)
	assert.Equal(t, price, uint64(100))

	// Wrong side
	price, err = book.GetCloseoutPrice(10, types.Side_SIDE_SELL)
	assert.Error(t, err, ErrNotEnoughOrders)
	assert.Equal(t, price, uint64(100))
}

func TestOrderBook_closeOutPriceSell(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	// Normal case
	price, err := book.GetCloseoutPrice(10, types.Side_SIDE_SELL)
	assert.NoError(t, err)
	assert.Equal(t, price, uint64(100))

	// Incorrect size
	price, err = book.GetCloseoutPrice(0, types.Side_SIDE_SELL)
	assert.Error(t, err, ErrInvalidVolume)
	assert.Equal(t, price, uint64(0))

	// Not enough on the book
	price, err = book.GetCloseoutPrice(200, types.Side_SIDE_SELL)
	assert.Error(t, err, ErrNotEnoughOrders)
	assert.Equal(t, price, uint64(100))

	// Wrong side
	price, err = book.GetCloseoutPrice(10, types.Side_SIDE_BUY)
	assert.Error(t, err, ErrNotEnoughOrders)
	assert.Equal(t, price, uint64(100))
}

func TestOrderBook_closeOutPriceBuy2(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       90,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order3 := types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       80,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err = book.SubmitOrder(&order3)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	// Normal case
	price, err := book.GetCloseoutPrice(100, types.Side_SIDE_BUY)
	assert.NoError(t, err)
	assert.Equal(t, price, uint64(100))

	// Normal case
	price, err = book.GetCloseoutPrice(200, types.Side_SIDE_BUY)
	assert.NoError(t, err)
	assert.Equal(t, price, uint64(95))

	// Normal case
	price, err = book.GetCloseoutPrice(300, types.Side_SIDE_BUY)
	assert.NoError(t, err)
	assert.Equal(t, price, uint64(90))
}

func TestOrderBook_closeOutPriceSell2(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       110,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order3 := types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       120,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err = book.SubmitOrder(&order3)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	// Normal case
	price, err := book.GetCloseoutPrice(100, types.Side_SIDE_SELL)
	assert.NoError(t, err)
	assert.Equal(t, price, uint64(100))

	// Normal case
	price, err = book.GetCloseoutPrice(200, types.Side_SIDE_SELL)
	assert.NoError(t, err)
	assert.Equal(t, price, uint64(105))

	// Normal case
	price, err = book.GetCloseoutPrice(300, types.Side_SIDE_SELL)
	assert.NoError(t, err)
	assert.Equal(t, price, uint64(110))
}
