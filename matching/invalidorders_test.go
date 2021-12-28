package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/stretchr/testify/assert"
)

func TestOrderBookInvalid_emptyMarketID(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    "",
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        1,
		Remaining:   1,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidMarketID, err)
	assert.Equal(t, (*types.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_emptyPartyID(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        1,
		Remaining:   1,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidPartyID, err)
	assert.Equal(t, (*types.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_ZeroSize(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        0,
		Remaining:   0,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidRemainingSize, err)
	assert.Equal(t, (*types.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_ZeroPrice(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.Zero(),
		Size:        1,
		Remaining:   1,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), confirm.Order.Price.Uint64())
}

func TestOrderBookInvalid_RemainingTooBig(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   11,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidRemainingSize, err)
	assert.Equal(t, (*types.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_GTCMarket(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeMarket,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidPersistence, err)
	assert.Equal(t, (*types.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_GTCNetwork(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeNetwork,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidPersistence, err)
	assert.Equal(t, (*types.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_GTTMarket(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.OrderTimeInForceGTT,
		Type:        types.OrderTypeMarket,
		ExpiresAt:   1,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidPersistence, err)
	assert.Equal(t, (*types.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_GTTNetwork(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.OrderTimeInForceGTT,
		Type:        types.OrderTypeNetwork,
		ExpiresAt:   1,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidPersistence, err)
	assert.Equal(t, (*types.OrderConfirmation)(nil), confirm)
}

func TestOrderBookInvalid_IOCNetwork(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:    market,
		Party:       "A",
		Side:        types.SideBuy,
		Price:       num.NewUint(100),
		Size:        10,
		Remaining:   10,
		TimeInForce: types.OrderTimeInForceIOC,
		Type:        types.OrderTypeNetwork,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.Equal(t, types.ErrInvalidPersistence, err)
	assert.Equal(t, (*types.OrderConfirmation)(nil), confirm)
}
