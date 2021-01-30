package matching

import (
	"testing"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func TestOrderBook_MarketOrderFOKNotFilledResponsePrice(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_BUY,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_TIME_IN_FORCE_FOK,
		Type:        types.Order_TYPE_MARKET,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)

	// Verify that the response price for the unfilled order is zero
	assert.NotEqual(t, (*types.OrderConfirmation)(nil), confirm)
	assert.Equal(t, uint64(0), confirm.Order.GetPrice())
}

func TestOrderBook_MarketOrderIOCNotFilledResponsePrice(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_BUY,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_TIME_IN_FORCE_IOC,
		Type:        types.Order_TYPE_MARKET,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)

	// Verify that the response price for the unfilled order is zero
	assert.NotEqual(t, (*types.OrderConfirmation)(nil), confirm)
	assert.Equal(t, uint64(0), confirm.Order.GetPrice())
}

func TestOrderBook_MarketOrderFOKPartiallyFilledResponsePrice(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        6,
		Remaining:   6,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	_, err := book.SubmitOrder(&order)
	assert.NoError(t, err)

	order = types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_BUY,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_TIME_IN_FORCE_FOK,
		Type:        types.Order_TYPE_MARKET,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)

	// Verify that the response price for the unfilled order is zero
	assert.NotEqual(t, (*types.OrderConfirmation)(nil), confirm)
	assert.Equal(t, uint64(0), confirm.Order.GetPrice())

	// Nothing was filled
	assert.Equal(t, uint64(10), confirm.Order.GetRemaining())

	// No orders
	assert.Nil(t, confirm.Trades)
	assert.Nil(t, confirm.PassiveOrdersAffected)
}

func TestOrderBook_MarketOrderIOCPartiallyFilledResponsePrice(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketId:    market,
		PartyId:     "A",
		Side:        types.Side_SIDE_SELL,
		Price:       100,
		Size:        6,
		Remaining:   6,
		TimeInForce: types.Order_TIME_IN_FORCE_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	_, err := book.SubmitOrder(&order)
	assert.NoError(t, err)

	order2 := types.Order{
		MarketId:    market,
		PartyId:     "B",
		Side:        types.Side_SIDE_BUY,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_TIME_IN_FORCE_IOC,
		Type:        types.Order_TYPE_MARKET,
	}
	confirm, err := book.SubmitOrder(&order2)
	assert.NoError(t, err)

	// Verify that the response price for the unfilled order is zero
	assert.NotEqual(t, (*types.OrderConfirmation)(nil), confirm)
	assert.Equal(t, uint64(0), confirm.Order.GetPrice())

	// Something was filled
	assert.Equal(t, uint64(4), confirm.Order.GetRemaining())

	// One order
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, 1, len(confirm.PassiveOrdersAffected))
}
