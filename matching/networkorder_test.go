package matching_test

import (
	"testing"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/assert"
)

// reproducing bug from https://github.com/vegaprotocol/vega/issues/2180

func TestNetworkOrder_ValidAveragedPrice(t *testing.T) {
	market := "testMarket"
	book := getTestOrderBook(t, market)
	defer book.Finish()

	orders := []types.Order{
		{
			MarketID:    market,
			Status:      types.OrderStatusActive,
			Party:       "A",
			Side:        types.SideBuy,
			Price:       num.NewUint(100),
			Size:        4,
			Remaining:   4,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
			ID:          "v0000000000000-0000001",
		},
		{
			MarketID:    market,
			Status:      types.OrderStatusActive,
			Party:       "B",
			Side:        types.SideBuy,
			Price:       num.NewUint(75),
			Size:        4,
			Remaining:   4,
			TimeInForce: types.OrderTimeInForceGTC,
			Type:        types.OrderTypeLimit,
			ID:          "v0000000000000-0000002",
		},
	}

	var (
		totalSize                 uint64
		totalPrice, expectedPrice = num.Zero(), num.NewUint(0)
	)
	for _, v := range orders {
		v := v
		_, err := book.ob.SubmitOrder(&v)
		assert.NoError(t, err)
		// totalPrice += v.Price * v.Size
		totalPrice.Add(
			totalPrice,
			num.Zero().Mul(v.Price, num.NewUint(v.Size)),
		)
		totalSize += v.Size
	}
	expectedPrice.Div(totalPrice, num.NewUint(totalSize))
	assert.Equal(t, uint64(87), expectedPrice.Uint64())

	// now let's place the network order and validate it's price
	netorder := types.Order{
		MarketID:    market,
		Size:        8,
		Remaining:   8,
		Status:      types.OrderStatusActive,
		Party:       "network",
		Side:        types.SideSell,
		Price:       num.Zero(),
		CreatedAt:   0,
		TimeInForce: types.OrderTimeInForceFOK,
		Type:        types.OrderTypeNetwork,
	}

	_, err := book.ob.SubmitOrder(&netorder)
	assert.NoError(t, err)
	// now we expect the price of the order to be updated
	assert.Equal(t, expectedPrice.Uint64(), netorder.Price.Uint64())
}
