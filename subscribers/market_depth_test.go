package subscribers_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/subscribers"
)

func getTestMDB(t *testing.T, ctx context.Context, ack bool) *subscribers.MarketDepthBuilder {
	return subscribers.NewMarketDepthBuilder(ctx, true)
}

func buildOrder(id string, side types.Side, orderType types.Order_Type, price uint64, size int64, remaining uint64) *types.Order {
	order := &types.Order{
		Id: "Hello",
	}
	return order
}

func TestAddOrderToEmptyBook(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event := events.NewOrderEvent(ctx, order)
	mdb.Push(event)

	/*	assert.Equal(t, mdb.GetBuyPriceLevels(), 1)
		assert.Equal(t, mdb.GetSellPriceLevels(), 0)
		assert.Equal(t, mdb.GetOrderCount(), 1)

		assert.Equal(t, mdb.GetVolumeAtPrice(100), 1)
		assert.Equal(t, mdb.GetOrderCountAtPrice(100), 1)*/
}
