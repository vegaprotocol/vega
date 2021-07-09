package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/stretchr/testify/assert"
)

func TestOrderBookSimple_CancelWrongOrderIncorrectOrderID(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Type:        types.Order_TYPE_LIMIT,
		Id:          "v0000000000000-0000001",
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Type:        types.Order_TYPE_LIMIT,
		Id:          "v0000000000000-0000002", // Invalid, must match original
	}
	_, err = book.CancelOrder(&order2)
	assert.Error(t, err, types.ErrOrderRemovalFailure)
	assert.Equal(t, book.getNumberOfBuyLevels(), 1)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
}

func TestOrderBookSimple_CancelWrongOrderIncorrectMarketID(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Type:        types.Order_TYPE_LIMIT,
		Id:          "v0000000000000-0000001",
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		MarketId:    "incorrectMarket", // Invalid, must match original
		PartyId:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Type:        types.Order_TYPE_LIMIT,
		Id:          "v0000000000000-0000001",
	}
	_, err = book.CancelOrder(&order2)
	assert.Error(t, err, types.ErrOrderRemovalFailure)
	assert.Equal(t, book.getNumberOfBuyLevels(), 1)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
}

func TestOrderBookSimple_CancelWrongOrderIncorrectSide(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Type:        types.Order_TYPE_LIMIT,
		Id:          "v0000000000000-0000001",
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_SELL, // Invalid, must match original
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Type:        types.Order_TYPE_LIMIT,
		Id:          "v0000000000000-0000001",
	}
	_, err = book.CancelOrder(&order2)
	assert.Error(t, err, types.ErrOrderRemovalFailure)
	assert.Equal(t, book.getNumberOfBuyLevels(), 1)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
}

func TestOrderBookSimple_CancelWrongOrderIncorrectPrice(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Type:        types.Order_TYPE_LIMIT,
		Id:          "v0000000000000-0000001",
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       num.NewUint(101), // Invalid, must match original
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Type:        types.Order_TYPE_LIMIT,
		Id:          "v0000000000000-0000001",
	}
	_, err = book.CancelOrder(&order2)
	assert.Error(t, err, types.ErrOrderRemovalFailure)
	assert.Equal(t, book.getNumberOfBuyLevels(), 1)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
}

func TestOrderBookSimple_CancelOrderIncorrectNonCriticalFields(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Type:        types.Order_TYPE_LIMIT,
		Id:          "v0000000000000-0000001",
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))

	order2 := types.Order{
		MarketId:    market,                        // Must match
		PartyId:     "B",                           // Does not matter
		Side:        types.Side_SIDE_BUY,           // Must match
		Price:       num.NewUint(100),              // Must match
		Size:        10,                            // Does not matter
		Remaining:   10,                            // Does not matter
		TimeInForce: types.Order_TIME_IN_FORCE_GTC, // Does not matter
		Type:        types.Order_TYPE_LIMIT,        // Does not matter
		Id:          "v0000000000000-0000001",      // Must match
	}
	_, err = book.CancelOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, book.getNumberOfBuyLevels(), 0)
	assert.Equal(t, book.getNumberOfSellLevels(), 0)
}
