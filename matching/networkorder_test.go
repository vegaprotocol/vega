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
			MarketId:    market,
			Status:      types.Order_STATUS_ACTIVE,
			PartyId:     "A",
			Side:        types.Side_SIDE_BUY,
			Price:       num.NewUint(100),
			Size:        4,
			Remaining:   4,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			Type:        types.Order_TYPE_LIMIT,
			Id:          "v0000000000000-0000001",
		},
		{
			MarketId:    market,
			Status:      types.Order_STATUS_ACTIVE,
			PartyId:     "B",
			Side:        types.Side_SIDE_BUY,
			Price:       num.NewUint(75),
			Size:        4,
			Remaining:   4,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			Type:        types.Order_TYPE_LIMIT,
			Id:          "v0000000000000-0000002",
		},
	}

	var (
		totalSize                 uint64
		totalPrice, expectedPrice = num.NewUint(0), num.NewUint(0)
	)
	for _, v := range orders {
		v := v
		_, err := book.ob.SubmitOrder(&v)
		assert.NoError(t, err)
		// totalPrice += v.Price * v.Size
		totalPrice.Add(
			totalPrice,
			num.NewUint(0).Mul(v.Price, num.NewUint(v.Size)),
		)
		totalSize += v.Size
	}
	expectedPrice.Div(totalPrice, num.NewUint(totalSize))
	assert.Equal(t, uint64(87), expectedPrice.Uint64())

	// now let's place the network order and validate it's price
	netorder := types.Order{
		MarketId:    market,
		Size:        8,
		Remaining:   8,
		Status:      types.Order_STATUS_ACTIVE,
		PartyId:     "network",
		Side:        types.Side_SIDE_SELL,
		Price:       num.NewUint(0),
		CreatedAt:   0,
		TimeInForce: types.Order_TIME_IN_FORCE_FOK,
		Type:        types.Order_TYPE_NETWORK,
	}

	_, err := book.ob.SubmitOrder(&netorder)
	assert.NoError(t, err)
	// now we expect the price of the order to be updated
	assert.Equal(t, expectedPrice.Uint64(), netorder.Price.Uint64())
}
