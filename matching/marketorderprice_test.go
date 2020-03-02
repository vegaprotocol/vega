package matching

import (
	"testing"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func TestOrderBook_marketOrderPriceEmptyBook(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	// If the book is empty then we use the default open price of 100
	price := book.MarketOrderPrice(types.Side_Buy)
	assert.Equal(t, uint64(100), price)

	price = book.MarketOrderPrice(types.Side_Sell)
	assert.Equal(t, uint64(100), price)
}

func TestOrderBook_marketOrderPriceBuy(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_Buy,
		Price:       50,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	// Normal case
	price := book.MarketOrderPrice(types.Side_Buy)
	assert.Equal(t, uint64(100), price)

	price = book.MarketOrderPrice(types.Side_Sell)
	assert.Equal(t, uint64(50), price)
}

func TestOrderBook_marketOrderPriceSell(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_Sell,
		Price:       200,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	// Normal case
	price := book.MarketOrderPrice(types.Side_Buy)
	assert.Equal(t, uint64(200), price)

	price = book.MarketOrderPrice(types.Side_Sell)
	assert.Equal(t, uint64(100), price)
}

func TestOrderBook_marketOrderPriceBuys(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_Buy,
		Price:       50,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_Buy,
		Price:       1,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	// Normal case
	price := book.MarketOrderPrice(types.Side_Buy)
	assert.Equal(t, uint64(100), price)

	price = book.MarketOrderPrice(types.Side_Sell)
	assert.Equal(t, uint64(1), price)
}

func TestOrderBook_marketOrderPriceSells(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_Sell,
		Price:       200,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_Sell,
		Price:       1000,
		Size:        100,
		Remaining:   100,
		TimeInForce: types.Order_GTC,
		Type:        types.Order_LIMIT,
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	// Normal case
	price := book.MarketOrderPrice(types.Side_Buy)
	assert.Equal(t, uint64(1000), price)

	price = book.MarketOrderPrice(types.Side_Sell)
	assert.Equal(t, uint64(100), price)
}
