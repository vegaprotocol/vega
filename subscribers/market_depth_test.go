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
		Status:      types.Order_STATUS_ACTIVE,
		MarketID:    "M",
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

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 4)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 4)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 102), uint64(7))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 102), uint64(1))

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 101), uint64(8))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 101), uint64(1))

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 100), uint64(9))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 100), uint64(1))

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 99), uint64(10))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 99), uint64(1))
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

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 4)
	assert.Equal(t, mdb.GetOrderCount("M"), 4)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_SELL, 102), uint64(7))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_SELL, 102), uint64(1))

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_SELL, 101), uint64(8))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_SELL, 101), uint64(1))

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_SELL, 100), uint64(9))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_SELL, 100), uint64(1))

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_SELL, 99), uint64(10))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_SELL, 99), uint64(1))
}

func TestAddOrderToEmptyBook(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event := events.NewOrderEvent(ctx, order)
	mdb.Push(event)

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 1)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 1)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 100), uint64(10))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 100), uint64(1))
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

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 0)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
}

func TestStoppedOrder(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event := events.NewOrderEvent(ctx, order)
	mdb.Push(event)

	cancelorder := *order
	cancelorder.Status = types.Order_STATUS_STOPPED
	event2 := events.NewOrderEvent(ctx, &cancelorder)
	mdb.Push(event2)

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 0)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
}

func TestExpiredOrder(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event := events.NewOrderEvent(ctx, order)
	mdb.Push(event)

	cancelorder := *order
	cancelorder.Status = types.Order_STATUS_EXPIRED
	event2 := events.NewOrderEvent(ctx, &cancelorder)
	mdb.Push(event2)

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 0)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
}

func TestAmendOrderPrice(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event := events.NewOrderEvent(ctx, order)
	mdb.Push(event)

	// Amend the price to force a change in price level
	amendorder := *order
	amendorder.Price = 90
	event2 := events.NewOrderEvent(ctx, &amendorder)
	mdb.Push(event2)

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 1)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 1)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 90), uint64(10))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 90), uint64(1))
}

func TestAmendOrderVolumeUp(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event := events.NewOrderEvent(ctx, order)
	mdb.Push(event)

	// Amend the price to force a change in price level
	amendorder := *order
	amendorder.Size = 20
	amendorder.Remaining = 20
	event2 := events.NewOrderEvent(ctx, &amendorder)
	mdb.Push(event2)

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 1)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 1)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 100), uint64(20))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 100), uint64(1))
}

func TestAmendOrderVolumeDown(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event := events.NewOrderEvent(ctx, order)
	mdb.Push(event)

	// Amend the price to force a change in price level
	amendorder := *order
	amendorder.Size = 5
	amendorder.Remaining = 5
	event2 := events.NewOrderEvent(ctx, &amendorder)
	mdb.Push(event2)

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 1)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 1)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 100), uint64(5))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 100), uint64(1))
}

func TestPartialFill(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event := events.NewOrderEvent(ctx, order)
	mdb.Push(event)

	// Amend the price to force a change in price level
	pforder := *order
	pforder.Remaining = 5
	event2 := events.NewOrderEvent(ctx, &pforder)
	mdb.Push(event2)

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 1)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 1)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 100), uint64(5))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 100), uint64(1))
}

func TestFullyFill(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event := events.NewOrderEvent(ctx, order)
	mdb.Push(event)

	// Amend the price to force a change in price level
	fforder := *order
	fforder.Remaining = 0
	fforder.Status = types.Order_STATUS_FILLED
	event2 := events.NewOrderEvent(ctx, &fforder)
	mdb.Push(event2)

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 0)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
}

func TestMarketOrder(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	mdb := getTestMDB(t, ctx, true)

	// market orders should not stay on the book
	marketorder := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_MARKET, 100, 10, 10)
	event1 := events.NewOrderEvent(ctx, marketorder)
	mdb.Push(event1)

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 0)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
}

func TestFOKOrder(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	mdb := getTestMDB(t, ctx, true)

	// FOK orders do not stay on the book
	fokorder := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	fokorder.TimeInForce = types.Order_TIF_FOK
	event := events.NewOrderEvent(ctx, fokorder)
	mdb.Push(event)

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 0)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
}

func TestIOCOrder(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	mdb := getTestMDB(t, ctx, true)

	// IOC orders do not stay on the book
	iocorder := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	iocorder.TimeInForce = types.Order_TIF_IOC
	event := events.NewOrderEvent(ctx, iocorder)
	mdb.Push(event)

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 0)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
}

func TestRejectedOrder(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	mdb := getTestMDB(t, ctx, true)

	// Rejected orders should be ignored
	order := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	order.Status = types.Order_STATUS_REJECTED
	event := events.NewOrderEvent(ctx, order)
	mdb.Push(event)

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 0)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
}

func TestInvalidOrder(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	mdb := getTestMDB(t, ctx, true)

	// Invalid orders should be ignored
	order := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	order.Status = types.Order_STATUS_INVALID
	event := events.NewOrderEvent(ctx, order)
	mdb.Push(event)

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 0)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
}
