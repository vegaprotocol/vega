package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/proto"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func TestOrderBookInvalid_emptyMarketID(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    "",
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidMarketID, err)
	assert.Equal(t, (*proto.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_emptyPartyID(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		PartyID:     "",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidPartyID, err)
	assert.Equal(t, (*proto.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_ZeroSize(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        0,
		Remaining:   0,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidRemainingSize, err)
	assert.Equal(t, (*proto.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_ZeroPrice(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       0,
		Size:        1,
		Remaining:   1,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), confirm.GetOrder().GetPrice())
}

func TestOrderBookInvalid_RemainingTooBig(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        10,
		Remaining:   11,
		TimeInForce: types.Order_TIF_GTC,
		Type:        types.Order_TYPE_LIMIT,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidRemainingSize, err)
	assert.Equal(t, (*proto.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_GTCMarket(t *testing.T) {
	market := "testMarket"
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
		Type:        types.Order_TYPE_MARKET,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidPersistence, err)
	assert.Equal(t, (*proto.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_GTCNetwork(t *testing.T) {
	market := "testMarket"
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
		Type:        types.Order_TYPE_NETWORK,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidPersistence, err)
	assert.Equal(t, (*proto.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_GTTMarket(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_TIF_GTT,
		Type:        types.Order_TYPE_MARKET,
		ExpiresAt:   1,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidPersistence, err)
	assert.Equal(t, (*proto.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_GTTNetwork(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_TIF_GTT,
		Type:        types.Order_TYPE_NETWORK,
		ExpiresAt:   1,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidPersistence, err)
	assert.Equal(t, (*proto.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_IOCNetwork(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		PartyID:     "A",
		Side:        types.Side_SIDE_BUY,
		Price:       100,
		Size:        10,
		Remaining:   10,
		TimeInForce: types.Order_TIF_IOC,
		Type:        types.Order_TYPE_NETWORK,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidPersistence, err)
	assert.Equal(t, (*proto.OrderConfirmation)(nil), confirm)
}
