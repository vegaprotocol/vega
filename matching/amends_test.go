package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/stretchr/testify/assert"
)

func TestOrderBookAmends_simpleAmend(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          2,
		Remaining:     2,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.SideBuy))

	amend := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	err = book.AmendOrder(&order, &amend)
	assert.NoError(t, err)
	assert.Equal(t, 1, int(book.getVolumeAtLevel(100, types.SideBuy)))
}

func TestOrderBookAmends_invalidPartyID(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          2,
		Remaining:     2,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.SideBuy))

	amend := types.Order{
		MarketID:      market,
		Party:         "B",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	err = book.AmendOrder(&order, &amend)
	assert.Error(t, types.ErrOrderAmendFailure, err)
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.SideBuy))
}

func TestOrderBookAmends_invalidPriceAmend(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          2,
		Remaining:     2,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.SideBuy))

	amend := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(101),
		OriginalPrice: num.NewUint(101),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	err = book.AmendOrder(&order, &amend)
	assert.Error(t, types.ErrOrderAmendFailure, err)
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.SideBuy))
}

func TestOrderBookAmends_invalidSize(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          2,
		Remaining:     2,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.SideBuy))

	amend := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          5,
		Remaining:     5,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	err = book.AmendOrder(&order, &amend)
	assert.Error(t, types.ErrOrderAmendFailure, err)
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.SideBuy))
}

func TestOrderBookAmends_reduceToZero(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          2,
		Remaining:     2,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.SideBuy))

	amend := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          0,
		Remaining:     0,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	err = book.AmendOrder(&order, &amend)
	assert.Error(t, types.ErrOrderAmendFailure, err)
	assert.Equal(t, uint64(2), book.getVolumeAtLevel(100, types.SideBuy))
}

func TestOrderBookAmends_invalidSizeDueToPartialFill(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, uint64(10), book.getVolumeAtLevel(100, types.SideBuy))

	order2 := types.Order{
		MarketID:      market,
		Party:         "B",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          5,
		Remaining:     5,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, uint64(5), book.getVolumeAtLevel(100, types.SideBuy))

	amend := types.Order{
		MarketID:      market,
		Party:         "B",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          6,
		Remaining:     6,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	err = book.AmendOrder(&order, &amend)
	assert.Error(t, types.ErrOrderAmendFailure, err)
	assert.Equal(t, uint64(5), book.getVolumeAtLevel(100, types.SideBuy))
}

func TestOrderBookAmends_validSizeDueToPartialFill(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()
	order := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          10,
		Remaining:     10,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err := book.SubmitOrder(&order)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(confirm.Trades))
	assert.Equal(t, uint64(10), book.getVolumeAtLevel(100, types.SideBuy))

	order2 := types.Order{
		MarketID:      market,
		Party:         "B",
		Side:          types.SideSell,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          5,
		Remaining:     5,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	confirm, err = book.SubmitOrder(&order2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(confirm.Trades))
	assert.Equal(t, uint64(5), book.getVolumeAtLevel(100, types.SideBuy))

	amend := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          3,
		Remaining:     3,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	err = book.AmendOrder(&order, &amend)
	assert.Error(t, types.ErrOrderAmendFailure, err)
	assert.Equal(t, uint64(3), book.getVolumeAtLevel(100, types.SideBuy))
}

func TestOrderBookAmends_noOrderToAmend(t *testing.T) {
	market := "market"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	amend := types.Order{
		MarketID:      market,
		Party:         "A",
		Side:          types.SideBuy,
		Price:         num.NewUint(100),
		OriginalPrice: num.NewUint(100),
		Size:          1,
		Remaining:     1,
		TimeInForce:   types.OrderTimeInForceGTC,
		Type:          types.OrderTypeLimit,
	}
	err := book.AmendOrder(nil, &amend)
	assert.Error(t, types.ErrOrderNotFound, err)
}
