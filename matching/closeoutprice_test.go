package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/stretchr/testify/assert"
)

func TestOrderBook_closeOutPriceBuy(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	tradedOrder1 := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideSell,
		Price:       num.NewUint(100),
		Size:        100,
		Remaining:   100,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&tradedOrder1)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	tradedOrder2 := types.Order{
		MarketID:    market,
		Party:       "B",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        100,
		Remaining:   100,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&tradedOrder2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))

	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        100,
		Remaining:   100,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	// Normal case
	price, err := book.GetCloseoutPrice(10, types.SideBuy)
	assert.NoError(t, err)
	assert.Equal(t, price.Uint64(), uint64(100))

	// Incorrect size
	price, err = book.GetCloseoutPrice(0, types.SideBuy)
	assert.Error(t, err, ErrInvalidVolume)
	assert.Equal(t, price.Uint64(), uint64(0))

	// Not enough on the book
	price, err = book.GetCloseoutPrice(200, types.SideBuy)
	assert.Error(t, err, ErrNotEnoughOrders)
	assert.Equal(t, price.Uint64(), uint64(100))

	// Wrong side
	price, err = book.GetCloseoutPrice(10, types.SideSell)
	assert.Error(t, err, ErrNotEnoughOrders)
	assert.Equal(t, price.Uint64(), uint64(100))
}

func TestOrderBook_closeOutPriceSell(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	tradedOrder1 := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideSell,
		Price:       num.NewUint(100),
		Size:        100,
		Remaining:   100,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&tradedOrder1)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	tradedOrder2 := types.Order{
		MarketID:    market,
		Party:       "B",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        100,
		Remaining:   100,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&tradedOrder2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))

	untradedOrder := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideSell,
		Price:       num.NewUint(100),
		Size:        100,
		Remaining:   100,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&untradedOrder)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	// Normal case
	price, err := book.GetCloseoutPrice(10, types.SideSell)
	assert.NoError(t, err)
	assert.Equal(t, price.Uint64(), uint64(100))

	// Incorrect size
	price, err = book.GetCloseoutPrice(0, types.SideSell)
	assert.Error(t, err, ErrInvalidVolume)
	assert.Equal(t, price.Uint64(), uint64(0))

	// Not enough on the book
	price, err = book.GetCloseoutPrice(200, types.SideSell)
	assert.Error(t, err, ErrNotEnoughOrders)
	assert.Equal(t, price.Uint64(), uint64(100))

	// Wrong side
	price, err = book.GetCloseoutPrice(10, types.SideBuy)
	assert.Error(t, err, ErrNotEnoughOrders)
	assert.Equal(t, price.Uint64(), uint64(100))
}

func TestOrderBook_closeOutPriceBuy2(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        100,
		Remaining:   100,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(90),
		Size:        100,
		Remaining:   100,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order3 := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(80),
		Size:        100,
		Remaining:   100,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&order3)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	// Normal case
	price, err := book.GetCloseoutPrice(100, types.SideBuy)
	assert.NoError(t, err)
	assert.Equal(t, price.Uint64(), uint64(100))

	// Normal case
	price, err = book.GetCloseoutPrice(200, types.SideBuy)
	assert.NoError(t, err)
	assert.Equal(t, price.Uint64(), uint64(95))

	// Normal case
	price, err = book.GetCloseoutPrice(300, types.SideBuy)
	assert.NoError(t, err)
	assert.Equal(t, price.Uint64(), uint64(90))
}

func TestOrderBook_closeOutPriceSell2(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideSell,
		Price:       num.NewUint(100),
		Size:        100,
		Remaining:   100,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideSell,
		Price:       num.NewUint(110),
		Size:        100,
		Remaining:   100,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order3 := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideSell,
		Price:       num.NewUint(120),
		Size:        100,
		Remaining:   100,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&order3)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	// Normal case
	price, err := book.GetCloseoutPrice(100, types.SideSell)
	assert.NoError(t, err)
	assert.Equal(t, price.Uint64(), uint64(100))

	// Normal case
	price, err = book.GetCloseoutPrice(200, types.SideSell)
	assert.NoError(t, err)
	assert.Equal(t, price.Uint64(), uint64(105))

	// Normal case
	price, err = book.GetCloseoutPrice(300, types.SideSell)
	assert.NoError(t, err)
	assert.Equal(t, price.Uint64(), uint64(110))
}
