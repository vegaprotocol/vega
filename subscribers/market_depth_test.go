package subscribers_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/subscribers"
	"github.com/stretchr/testify/assert"
)

func getTestMDB(t *testing.T, ctx context.Context, ack bool) *subscribers.MarketDepthBuilder {
	return subscribers.NewMarketDepthBuilder(ctx, true)
}

func buildOrder(id string, side types.Side, orderType types.Order_Type, price uint64, size uint64, remaining uint64) *types.Order {
	order := &types.Order{
		Id:          id,
		Side:        side,
		Type:        orderType,
		Price:       price,
		Size:        size,
		Remaining:   remaining,
		TimeInForce: types.Order_TIF_GTC,
	}
	return order
}

func TestBuyPriceLevels(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	mdb := getTestMDB(t, ctx, true)

	order1 := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 9, 9)
	event1 := events.NewOrderEvent(ctx, order1)
	mdb.Push(event1)

	order2 := buildOrder("Order2", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 102, 7, 7)
	event2 := events.NewOrderEvent(ctx, order2)
	mdb.Push(event2)

	order3 := buildOrder("Order3", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 101, 8, 8)
	event3 := events.NewOrderEvent(ctx, order3)
	mdb.Push(event3)

	order4 := buildOrder("Order4", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 99, 10, 10)
	event4 := events.NewOrderEvent(ctx, order4)
	mdb.Push(event4)

	assert.Equal(t, mdb.GetBuyPriceLevels(), 4)
	assert.Equal(t, mdb.GetSellPriceLevels(), 0)
	assert.Equal(t, mdb.GetOrderCount(), 4)

	assert.Equal(t, mdb.GetVolumeAtPrice(types.Side_SIDE_BUY, 102), uint64(7))
	assert.Equal(t, mdb.GetOrderCountAtPrice(types.Side_SIDE_BUY, 102), uint64(1))

	assert.Equal(t, mdb.GetVolumeAtPrice(types.Side_SIDE_BUY, 101), uint64(8))
	assert.Equal(t, mdb.GetOrderCountAtPrice(types.Side_SIDE_BUY, 101), uint64(1))

	assert.Equal(t, mdb.GetVolumeAtPrice(types.Side_SIDE_BUY, 100), uint64(9))
	assert.Equal(t, mdb.GetOrderCountAtPrice(types.Side_SIDE_BUY, 100), uint64(1))

	assert.Equal(t, mdb.GetVolumeAtPrice(types.Side_SIDE_BUY, 99), uint64(10))
	assert.Equal(t, mdb.GetOrderCountAtPrice(types.Side_SIDE_BUY, 99), uint64(1))
}

func TestSellPriceLevels(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	mdb := getTestMDB(t, ctx, true)

	order1 := buildOrder("Order1", types.Side_SIDE_SELL, types.Order_TYPE_LIMIT, 100, 9, 9)
	event1 := events.NewOrderEvent(ctx, order1)
	mdb.Push(event1)

	order2 := buildOrder("Order2", types.Side_SIDE_SELL, types.Order_TYPE_LIMIT, 102, 7, 7)
	event2 := events.NewOrderEvent(ctx, order2)
	mdb.Push(event2)

	order3 := buildOrder("Order3", types.Side_SIDE_SELL, types.Order_TYPE_LIMIT, 101, 8, 8)
	event3 := events.NewOrderEvent(ctx, order3)
	mdb.Push(event3)

	order4 := buildOrder("Order4", types.Side_SIDE_SELL, types.Order_TYPE_LIMIT, 99, 10, 10)
	event4 := events.NewOrderEvent(ctx, order4)
	mdb.Push(event4)

	assert.Equal(t, mdb.GetBuyPriceLevels(), 0)
	assert.Equal(t, mdb.GetSellPriceLevels(), 4)
	assert.Equal(t, mdb.GetOrderCount(), 4)

	assert.Equal(t, mdb.GetVolumeAtPrice(types.Side_SIDE_SELL, 102), uint64(7))
	assert.Equal(t, mdb.GetOrderCountAtPrice(types.Side_SIDE_SELL, 102), uint64(1))

	assert.Equal(t, mdb.GetVolumeAtPrice(types.Side_SIDE_SELL, 101), uint64(8))
	assert.Equal(t, mdb.GetOrderCountAtPrice(types.Side_SIDE_SELL, 101), uint64(1))

	assert.Equal(t, mdb.GetVolumeAtPrice(types.Side_SIDE_SELL, 100), uint64(9))
	assert.Equal(t, mdb.GetOrderCountAtPrice(types.Side_SIDE_SELL, 100), uint64(1))

	assert.Equal(t, mdb.GetVolumeAtPrice(types.Side_SIDE_SELL, 99), uint64(10))
	assert.Equal(t, mdb.GetOrderCountAtPrice(types.Side_SIDE_SELL, 99), uint64(1))
}

func TestAddOrderToEmptyBook(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event := events.NewOrderEvent(ctx, order)
	mdb.Push(event)

	assert.Equal(t, mdb.GetBuyPriceLevels(), 1)
	assert.Equal(t, mdb.GetSellPriceLevels(), 0)
	assert.Equal(t, mdb.GetOrderCount(), 1)

	assert.Equal(t, mdb.GetVolumeAtPrice(types.Side_SIDE_BUY, 100), uint64(10))
	assert.Equal(t, mdb.GetOrderCountAtPrice(types.Side_SIDE_BUY, 100), uint64(1))
}

func TestCancelOrder(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event := events.NewOrderEvent(ctx, order)
	mdb.Push(event)

	cancelorder := *order
	cancelorder.Status = types.Order_STATUS_CANCELLED
	event2 := events.NewOrderEvent(ctx, &cancelorder)
	mdb.Push(event2)

	assert.Equal(t, mdb.GetBuyPriceLevels(), 0)
	assert.Equal(t, mdb.GetSellPriceLevels(), 0)
	assert.Equal(t, mdb.GetOrderCount(), 0)

	assert.Equal(t, mdb.GetVolumeAtPrice(types.Side_SIDE_BUY, 100), uint64(0))
	assert.Equal(t, mdb.GetOrderCountAtPrice(types.Side_SIDE_BUY, 100), uint64(0))
}
