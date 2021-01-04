package subscribers_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto/gen/golang"
	"code.vegaprotocol.io/vega/subscribers"
	"github.com/stretchr/testify/assert"
)

func getTestMDB(t *testing.T, ctx context.Context, ack bool) *subscribers.MarketDepthBuilder {
	return subscribers.NewMarketDepthBuilder(ctx, nil, true)
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
	ctx := context.Background()
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
	ctx := context.Background()
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
	ctx := context.Background()
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
	ctx := context.Background()
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
	ctx := context.Background()
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
	ctx := context.Background()
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
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event := events.NewOrderEvent(ctx, order)
	mdb.Push(event)

	order2 := buildOrder("Order2", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event2 := events.NewOrderEvent(ctx, order2)
	mdb.Push(event2)

	// Amend the price to force a change in price level
	amendorder := *order
	amendorder.Price = 90
	event3 := events.NewOrderEvent(ctx, &amendorder)
	mdb.Push(event3)

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 2)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 2)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 100), uint64(10))
	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 90), uint64(10))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 100), uint64(1))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 90), uint64(1))
}

func TestAmendOrderVolumeUp(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event := events.NewOrderEvent(ctx, order)
	mdb.Push(event)

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
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event := events.NewOrderEvent(ctx, order)
	mdb.Push(event)

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

func TestAmendOrderVolumeDownToZero(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event := events.NewOrderEvent(ctx, order)
	mdb.Push(event)

	amendorder := *order
	amendorder.Size = 0
	amendorder.Remaining = 0
	event2 := events.NewOrderEvent(ctx, &amendorder)
	mdb.Push(event2)

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 0)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
}

func TestPartialFill(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event := events.NewOrderEvent(ctx, order)
	mdb.Push(event)

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

func TestIOCPartialFill(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 5)
	order.Status = types.Order_STATUS_PARTIALLY_FILLED
	order.TimeInForce = types.Order_TIF_IOC
	event := events.NewOrderEvent(ctx, order)
	mdb.Push(event)

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 0)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
}

func TestFullyFill(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event := events.NewOrderEvent(ctx, order)
	mdb.Push(event)

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
	ctx := context.Background()
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
	ctx := context.Background()
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
	ctx := context.Background()
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
	ctx := context.Background()
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
	ctx := context.Background()
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

func TestPartialMatchOrders(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order1 := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event1 := events.NewOrderEvent(ctx, order1)
	mdb.Push(event1)
	order2 := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 8)
	event2 := events.NewOrderEvent(ctx, order2)
	mdb.Push(event2)

	order3 := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 5)
	event3 := events.NewOrderEvent(ctx, order3)
	mdb.Push(event3)
	order4 := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 1)
	event4 := events.NewOrderEvent(ctx, order4)
	mdb.Push(event4)

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 1)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 1)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 100), uint64(1))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 100), uint64(1))
}

func TestFullyMatchOrders(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order1 := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event1 := events.NewOrderEvent(ctx, order1)
	mdb.Push(event1)
	order2 := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 8)
	event2 := events.NewOrderEvent(ctx, order2)
	mdb.Push(event2)

	order3 := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 5)
	event3 := events.NewOrderEvent(ctx, order3)
	mdb.Push(event3)
	order4 := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 0)
	order4.Status = types.Order_STATUS_FILLED
	event4 := events.NewOrderEvent(ctx, order4)
	mdb.Push(event4)

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 0)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 100), uint64(0))
}

func TestRemovingPriceLevels(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order1 := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 101, 10, 10)
	event1 := events.NewOrderEvent(ctx, order1)
	mdb.Push(event1)
	order2 := buildOrder("Order2", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 10, 10)
	event2 := events.NewOrderEvent(ctx, order2)
	mdb.Push(event2)
	order3 := buildOrder("Order3", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 102, 10, 10)
	event3 := events.NewOrderEvent(ctx, order3)
	mdb.Push(event3)

	order4 := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 101, 10, 0)
	order4.Status = types.Order_STATUS_FILLED
	event4 := events.NewOrderEvent(ctx, order4)
	mdb.Push(event4)

	assert.Equal(t, mdb.GetBuyPriceLevels("M"), 2)
	assert.Equal(t, mdb.GetSellPriceLevels("M"), 0)
	assert.Equal(t, mdb.GetOrderCount("M"), 2)

	assert.Equal(t, mdb.GetVolumeAtPrice("M", types.Side_SIDE_BUY, 101), uint64(0))
	assert.Equal(t, mdb.GetOrderCountAtPrice("M", types.Side_SIDE_BUY, 101), uint64(0))
}

func TestMarketDepthFields(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order1 := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 101, 10, 10)
	event1 := events.NewOrderEvent(ctx, order1)
	mdb.Push(event1)

	md, err := mdb.GetMarketDepth(ctx, "M", 0)
	assert.Nil(t, err)
	assert.NotNil(t, md)

	assert.Equal(t, md.MarketID, "M")
	assert.Equal(t, len(md.GetBuy()), 1)

	priceLevels := md.GetBuy()
	pl := priceLevels[0]
	assert.NotNil(t, pl)
	assert.Equal(t, pl.NumberOfOrders, uint64(1))
	assert.Equal(t, pl.Price, uint64(101))
	assert.Equal(t, pl.Volume, uint64(10))
}

func TestParkingOrder(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	// Create a valid and live pegged order
	order1 := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 101, 10, 10)
	order1.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1}
	event1 := events.NewOrderEvent(ctx, order1)
	mdb.Push(event1)

	// Park it
	order2 := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 0, 10, 10)
	order2.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1}
	order2.Status = types.Order_STATUS_PARKED
	event2 := events.NewOrderEvent(ctx, order2)
	mdb.Push(event2)

	md, err := mdb.GetMarketDepth(ctx, "M", 0)
	assert.Nil(t, err)
	assert.NotNil(t, md)

	assert.Equal(t, md.MarketID, "M")
	assert.Equal(t, len(md.GetBuy()), 0)
	assert.Equal(t, len(md.GetSell()), 0)

	// Unpark it
	order3 := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 101, 10, 10)
	order3.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1}
	order3.Status = types.Order_STATUS_ACTIVE
	event3 := events.NewOrderEvent(ctx, order3)
	mdb.Push(event3)

	md2, err := mdb.GetMarketDepth(ctx, "M", 0)
	assert.Nil(t, err)
	assert.NotNil(t, md2)

	assert.Equal(t, md2.MarketID, "M")
	assert.Equal(t, len(md2.GetBuy()), 1)
	assert.Equal(t, len(md2.GetSell()), 0)
}

func TestParkedOrder(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	// Create a parked pegged order which should not go on the depth book
	order1 := buildOrder("Order1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 101, 10, 10)
	order1.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1}
	order1.Status = types.Order_STATUS_PARKED
	event1 := events.NewOrderEvent(ctx, order1)
	mdb.Push(event1)

	md, err := mdb.GetMarketDepth(ctx, "M", 0)
	assert.Nil(t, err)
	assert.NotNil(t, md)

	assert.Equal(t, md.MarketID, "M")
	assert.Equal(t, len(md.GetBuy()), 0)
	assert.Equal(t, len(md.GetSell()), 0)
}

func TestParkedOrder2(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	// Create parked pegged order
	order1 := buildOrder("Pegged1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 0, 10, 10)
	order1.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1}
	order1.Status = types.Order_STATUS_PARKED
	event1 := events.NewOrderEvent(ctx, order1)
	mdb.Push(event1)

	// Create normal order
	order2 := buildOrder("Normal1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 1, 1)
	event2 := events.NewOrderEvent(ctx, order2)
	mdb.Push(event2)

	// Unpark pegged order
	order3 := buildOrder("Pegged1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 99, 10, 10)
	order3.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1}
	order3.Status = types.Order_STATUS_ACTIVE
	event3 := events.NewOrderEvent(ctx, order3)
	mdb.Push(event3)

	// Cancel normal order
	order4 := buildOrder("Normal1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 1, 1)
	order4.Status = types.Order_STATUS_CANCELLED
	event4 := events.NewOrderEvent(ctx, order4)
	mdb.Push(event4)

	// Park pegged order
	order5 := buildOrder("Pegged1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 99, 10, 10)
	order5.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1}
	order5.Status = types.Order_STATUS_PARKED
	event5 := events.NewOrderEvent(ctx, order5)
	mdb.Push(event5)

	// Create normal order
	order6 := buildOrder("Normal2", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 1, 1)
	event6 := events.NewOrderEvent(ctx, order6)
	mdb.Push(event6)

	// Unpark pegged order
	order7 := buildOrder("Pegged1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 99, 10, 10)
	order7.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1}
	order7.Status = types.Order_STATUS_ACTIVE
	event7 := events.NewOrderEvent(ctx, order7)
	mdb.Push(event7)

	// Fill normal order
	order8 := buildOrder("Normal2", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 100, 1, 0)
	order8.Status = types.Order_STATUS_FILLED
	event8 := events.NewOrderEvent(ctx, order8)
	mdb.Push(event8)

	// Create new matching order
	order9 := buildOrder("Normal3", types.Side_SIDE_SELL, types.Order_TYPE_LIMIT, 100, 1, 0)
	order9.Status = types.Order_STATUS_FILLED
	event9 := events.NewOrderEvent(ctx, order9)
	mdb.Push(event9)

	// Park pegged order
	order10 := buildOrder("Pegged1", types.Side_SIDE_BUY, types.Order_TYPE_LIMIT, 99, 10, 10)
	order10.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReference_PEGGED_REFERENCE_BEST_BID, Offset: -1}
	order10.Status = types.Order_STATUS_PARKED
	event10 := events.NewOrderEvent(ctx, order10)
	mdb.Push(event10)

	md, err := mdb.GetMarketDepth(ctx, "M", 0)
	assert.Nil(t, err)
	assert.NotNil(t, md)

	assert.Equal(t, md.MarketID, "M")
	assert.Equal(t, 0, len(md.GetBuy()))
	assert.Equal(t, 0, len(md.GetSell()))
}
