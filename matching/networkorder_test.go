package matching_test

import (
	"testing"

	types "code.vegaprotocol.io/vega/proto"
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
			Price:       100,
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
			Price:       75,
			Size:        4,
			Remaining:   4,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			Type:        types.Order_TYPE_LIMIT,
			Id:          "v0000000000000-0000002",
		},
	}

	var totalPrice, totalSize, expectedPrice uint64
	for _, v := range orders {
		v := v
		_, err := book.ob.SubmitOrder(&v)
		assert.NoError(t, err)
		totalPrice += v.Price * v.Size
		totalSize += v.Size
	}
	expectedPrice = totalPrice / totalSize
	assert.Equal(t, 87, int(expectedPrice))

	// now let's place the network order and validate it's price
	netorder := types.Order{
		MarketId:    market,
		Size:        8,
		Remaining:   8,
		Status:      types.Order_STATUS_ACTIVE,
		PartyId:     "network",
		Side:        types.Side_SIDE_SELL,
		Price:       0,
		CreatedAt:   0,
		TimeInForce: types.Order_TIME_IN_FORCE_FOK,
		Type:        types.Order_TYPE_NETWORK,
	}

	_, err := book.ob.SubmitOrder(&netorder)
	assert.NoError(t, err)
	// now we expect the price of the order to be updated
	assert.Equal(t, int(expectedPrice), int(netorder.Price))
}
