// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package subscribers_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/subscribers"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/assert"
)

func getTestMDB(t *testing.T, ctx context.Context, ack bool) *subscribers.MarketDepthBuilder {
	return subscribers.NewMarketDepthBuilder(ctx, nil, ack)
}

func buildOrder(id string, side types.Side, orderType types.OrderType, price uint64, size uint64, remaining uint64) *types.Order {
	order := &types.Order{
		ID:            id,
		Side:          side,
		Type:          orderType,
		Price:         num.NewUint(price),
		OriginalPrice: num.NewUint(price),
		Size:          size,
		Remaining:     remaining,
		TimeInForce:   types.OrderTimeInForceGTC,
		Status:        types.OrderStatusActive,
		MarketID:      "M",
	}
	return order
}

type OrderEventWithVegaTime struct {
	events.Order
	vegaTime time.Time
}

func (oe *OrderEventWithVegaTime) VegaTime() time.Time {
	return oe.vegaTime
}

func (oe *OrderEventWithVegaTime) GetOrder() *vega.Order {
	return oe.Order.Order()
}

func newOrderEvent(ctx context.Context, o *types.Order) *OrderEventWithVegaTime {
	oe := events.NewOrderEvent(ctx, o)
	return &OrderEventWithVegaTime{*oe, time.Now()}
}

func TestBuyPriceLevels(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order1 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 9, 9)
	event1 := newOrderEvent(ctx, order1)
	mdb.Push(event1)

	order2 := buildOrder("Order2", types.SideBuy, types.OrderTypeLimit, 102, 7, 7)
	event2 := newOrderEvent(ctx, order2)
	mdb.Push(event2)

	order3 := buildOrder("Order3", types.SideBuy, types.OrderTypeLimit, 101, 8, 8)
	event3 := newOrderEvent(ctx, order3)
	mdb.Push(event3)

	order4 := buildOrder("Order4", types.SideBuy, types.OrderTypeLimit, 99, 10, 10)
	event4 := newOrderEvent(ctx, order4)
	mdb.Push(event4)

	assert.Equal(t, 4, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(4), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(7), mdb.GetVolumeAtPrice("M", types.SideBuy, 102))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice("M", types.SideBuy, 102))

	assert.Equal(t, uint64(8), mdb.GetVolumeAtPrice("M", types.SideBuy, 101))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice("M", types.SideBuy, 101))

	assert.Equal(t, uint64(9), mdb.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice("M", types.SideBuy, 100))

	assert.Equal(t, uint64(10), mdb.GetVolumeAtPrice("M", types.SideBuy, 99))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice("M", types.SideBuy, 99))
}

func TestSellPriceLevels(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order1 := buildOrder("Order1", types.SideSell, types.OrderTypeLimit, 100, 9, 9)
	event1 := newOrderEvent(ctx, order1)
	mdb.Push(event1)

	order2 := buildOrder("Order2", types.SideSell, types.OrderTypeLimit, 102, 7, 7)
	event2 := newOrderEvent(ctx, order2)
	mdb.Push(event2)

	order3 := buildOrder("Order3", types.SideSell, types.OrderTypeLimit, 101, 8, 8)
	event3 := newOrderEvent(ctx, order3)
	mdb.Push(event3)

	order4 := buildOrder("Order4", types.SideSell, types.OrderTypeLimit, 99, 10, 10)
	event4 := newOrderEvent(ctx, order4)
	mdb.Push(event4)

	assert.Equal(t, 0, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 4, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(4), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(7), mdb.GetVolumeAtPrice("M", types.SideSell, 102))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice("M", types.SideSell, 102))

	assert.Equal(t, uint64(8), mdb.GetVolumeAtPrice("M", types.SideSell, 101))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice("M", types.SideSell, 101))

	assert.Equal(t, uint64(9), mdb.GetVolumeAtPrice("M", types.SideSell, 100))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice("M", types.SideSell, 100))

	assert.Equal(t, uint64(10), mdb.GetVolumeAtPrice("M", types.SideSell, 99))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice("M", types.SideSell, 99))
}

func TestAddOrderToEmptyBook(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	event := newOrderEvent(ctx, order)
	mdb.Push(event)

	assert.Equal(t, 1, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(1), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(10), mdb.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestCancelOrder(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	event := newOrderEvent(ctx, order)
	mdb.Push(event)

	cancelorder := *order
	cancelorder.Status = types.OrderStatusCancelled
	event2 := newOrderEvent(ctx, &cancelorder)
	mdb.Push(event2)

	assert.Equal(t, 0, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mdb.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mdb.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestStoppedOrder(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	event := newOrderEvent(ctx, order)
	mdb.Push(event)

	cancelorder := *order
	cancelorder.Status = types.OrderStatusStopped
	event2 := newOrderEvent(ctx, &cancelorder)
	mdb.Push(event2)

	assert.Equal(t, 0, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mdb.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mdb.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestExpiredOrder(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	event := newOrderEvent(ctx, order)
	mdb.Push(event)

	cancelorder := *order
	cancelorder.Status = types.OrderStatusExpired
	event2 := newOrderEvent(ctx, &cancelorder)
	mdb.Push(event2)

	assert.Equal(t, 0, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mdb.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mdb.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestAmendOrderPrice(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	event := newOrderEvent(ctx, order)
	mdb.Push(event)

	order2 := buildOrder("Order2", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	event2 := newOrderEvent(ctx, order2)
	mdb.Push(event2)

	// Amend the price to force a change in price level
	amendorder := *order
	amendorder.Price = num.NewUint(90)
	amendorder.OriginalPrice = num.NewUint(90)
	event3 := newOrderEvent(ctx, &amendorder)
	mdb.Push(event3)

	assert.Equal(t, 2, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(2), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(10), mdb.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(10), mdb.GetVolumeAtPrice("M", types.SideBuy, 90))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice("M", types.SideBuy, 90))
}

func TestAmendOrderVolumeUp(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	event := newOrderEvent(ctx, order)
	mdb.Push(event)

	amendorder := *order
	amendorder.Size = 20
	amendorder.Remaining = 20
	event2 := newOrderEvent(ctx, &amendorder)
	mdb.Push(event2)

	assert.Equal(t, 1, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(1), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(20), mdb.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestAmendOrderVolumeDown(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	event := newOrderEvent(ctx, order)
	mdb.Push(event)

	amendorder := *order
	amendorder.Size = 5
	amendorder.Remaining = 5
	event2 := newOrderEvent(ctx, &amendorder)
	mdb.Push(event2)

	assert.Equal(t, 1, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(1), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(5), mdb.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestAmendOrderVolumeDownToZero(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	event := newOrderEvent(ctx, order)
	mdb.Push(event)

	amendorder := *order
	amendorder.Size = 0
	amendorder.Remaining = 0
	event2 := newOrderEvent(ctx, &amendorder)
	mdb.Push(event2)

	assert.Equal(t, 0, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mdb.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mdb.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestPartialFill(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	event := newOrderEvent(ctx, order)
	mdb.Push(event)

	pforder := *order
	pforder.Remaining = 5
	event2 := newOrderEvent(ctx, &pforder)
	mdb.Push(event2)

	assert.Equal(t, 1, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(1), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(5), mdb.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestIOCPartialFill(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 5)
	order.Status = types.OrderStatusPartiallyFilled
	order.TimeInForce = types.OrderTimeInForceIOC
	event := newOrderEvent(ctx, order)
	mdb.Push(event)

	assert.Equal(t, 0, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mdb.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mdb.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestFullyFill(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	event := newOrderEvent(ctx, order)
	mdb.Push(event)

	fforder := *order
	fforder.Remaining = 0
	fforder.Status = types.OrderStatusFilled
	event2 := newOrderEvent(ctx, &fforder)
	mdb.Push(event2)

	assert.Equal(t, 0, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mdb.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mdb.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestMarketOrder(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	// market orders should not stay on the book
	marketorder := buildOrder("Order1", types.SideBuy, types.OrderTypeMarket, 100, 10, 10)
	event1 := newOrderEvent(ctx, marketorder)
	mdb.Push(event1)

	assert.Equal(t, 0, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mdb.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mdb.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestFOKOrder(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	// FOK orders do not stay on the book
	fokorder := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	fokorder.TimeInForce = types.OrderTimeInForceFOK
	event := newOrderEvent(ctx, fokorder)
	mdb.Push(event)

	assert.Equal(t, 0, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mdb.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mdb.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestIOCOrder(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	// IOC orders do not stay on the book
	iocorder := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	iocorder.TimeInForce = types.OrderTimeInForceIOC
	event := newOrderEvent(ctx, iocorder)
	mdb.Push(event)

	assert.Equal(t, 0, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mdb.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mdb.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestRejectedOrder(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	// Rejected orders should be ignored
	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	order.Status = types.OrderStatusRejected
	event := newOrderEvent(ctx, order)
	mdb.Push(event)

	assert.Equal(t, 0, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mdb.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mdb.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestInvalidOrder(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	// Invalid orders should be ignored
	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	order.Status = types.OrderStatusUnspecified
	event := newOrderEvent(ctx, order)
	mdb.Push(event)

	assert.Equal(t, 0, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mdb.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mdb.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestPartialMatchOrders(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order1 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	event1 := newOrderEvent(ctx, order1)
	mdb.Push(event1)
	order2 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 8)
	event2 := newOrderEvent(ctx, order2)
	mdb.Push(event2)

	order3 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 5)
	event3 := newOrderEvent(ctx, order3)
	mdb.Push(event3)
	order4 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 1)
	event4 := newOrderEvent(ctx, order4)
	mdb.Push(event4)

	assert.Equal(t, 1, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(1), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(1), mdb.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(1), mdb.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestFullyMatchOrders(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order1 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	event1 := newOrderEvent(ctx, order1)
	mdb.Push(event1)
	order2 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 8)
	event2 := newOrderEvent(ctx, order2)
	mdb.Push(event2)

	order3 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 5)
	event3 := newOrderEvent(ctx, order3)
	mdb.Push(event3)
	order4 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 0)
	order4.Status = types.OrderStatusFilled
	event4 := newOrderEvent(ctx, order4)
	mdb.Push(event4)

	assert.Equal(t, 0, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mdb.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mdb.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestRemovingPriceLevels(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order1 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 101, 10, 10)
	event1 := newOrderEvent(ctx, order1)
	mdb.Push(event1)
	order2 := buildOrder("Order2", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	event2 := newOrderEvent(ctx, order2)
	mdb.Push(event2)
	order3 := buildOrder("Order3", types.SideBuy, types.OrderTypeLimit, 102, 10, 10)
	event3 := newOrderEvent(ctx, order3)
	mdb.Push(event3)

	order4 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 101, 10, 0)
	order4.Status = types.OrderStatusFilled
	event4 := newOrderEvent(ctx, order4)
	mdb.Push(event4)

	assert.Equal(t, 2, mdb.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mdb.GetSellPriceLevels("M"))
	assert.Equal(t, int64(2), mdb.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mdb.GetVolumeAtPrice("M", types.SideBuy, 101))
	assert.Equal(t, uint64(0), mdb.GetOrderCountAtPrice("M", types.SideBuy, 101))
}

func TestMarketDepthFields(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	order1 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 101, 10, 10)
	event1 := newOrderEvent(ctx, order1)
	mdb.Push(event1)

	md, err := mdb.GetMarketDepth(ctx, "M", 0)
	assert.Nil(t, err)
	assert.NotNil(t, md)

	assert.Equal(t, "M", md.MarketId)
	assert.Equal(t, 1, len(md.GetBuy()))

	priceLevels := md.GetBuy()
	pl := priceLevels[0]
	assert.NotNil(t, pl)
	assert.Equal(t, uint64(1), pl.NumberOfOrders)
	assert.Equal(t, "101", pl.Price)
	assert.Equal(t, uint64(10), pl.Volume)
}

func TestParkingOrder(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	// Create a valid and live pegged order
	order1 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 101, 10, 10)
	order1.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceBestBid, Offset: num.NewUint(1)}
	event1 := newOrderEvent(ctx, order1)
	mdb.Push(event1)

	// Park it
	order2 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 0, 10, 10)
	order2.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceBestBid, Offset: num.NewUint(1)}
	order2.Status = types.OrderStatusParked
	event2 := newOrderEvent(ctx, order2)
	mdb.Push(event2)

	md, err := mdb.GetMarketDepth(ctx, "M", 0)
	assert.Nil(t, err)
	assert.NotNil(t, md)

	assert.Equal(t, "M", md.MarketId)
	assert.Equal(t, 0, len(md.GetBuy()))
	assert.Equal(t, 0, len(md.GetSell()))

	// Unpark it
	order3 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 101, 10, 10)
	order3.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceBestBid, Offset: num.NewUint(1)}
	order3.Status = types.OrderStatusActive
	event3 := newOrderEvent(ctx, order3)
	mdb.Push(event3)

	md2, err := mdb.GetMarketDepth(ctx, "M", 0)
	assert.Nil(t, err)
	assert.NotNil(t, md2)

	assert.Equal(t, "M", md2.MarketId)
	assert.Equal(t, 1, len(md2.GetBuy()))
	assert.Equal(t, 0, len(md2.GetSell()))
}

func TestParkedOrder(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	// Create a parked pegged order which should not go on the depth book
	order1 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 101, 10, 10)
	order1.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceBestBid, Offset: num.NewUint(1)}
	order1.Status = types.OrderStatusParked
	event1 := newOrderEvent(ctx, order1)
	mdb.Push(event1)

	md, err := mdb.GetMarketDepth(ctx, "M", 0)
	assert.Nil(t, err)
	assert.NotNil(t, md)

	assert.Equal(t, "M", md.MarketId)
	assert.Equal(t, 0, len(md.GetBuy()))
	assert.Equal(t, 0, len(md.GetSell()))
}

func TestParkedOrder2(t *testing.T) {
	ctx := context.Background()
	mdb := getTestMDB(t, ctx, true)

	// Create parked pegged order
	order1 := buildOrder("Pegged1", types.SideBuy, types.OrderTypeLimit, 0, 10, 10)
	order1.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceBestBid, Offset: num.NewUint(1)}
	order1.Status = types.OrderStatusParked
	event1 := newOrderEvent(ctx, order1)
	mdb.Push(event1)

	// Create normal order
	order2 := buildOrder("Normal1", types.SideBuy, types.OrderTypeLimit, 100, 1, 1)
	event2 := newOrderEvent(ctx, order2)
	mdb.Push(event2)

	// Unpark pegged order
	order3 := buildOrder("Pegged1", types.SideBuy, types.OrderTypeLimit, 99, 10, 10)
	order3.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceBestBid, Offset: num.NewUint(1)}
	order3.Status = types.OrderStatusActive
	event3 := newOrderEvent(ctx, order3)
	mdb.Push(event3)

	// Cancel normal order
	order4 := buildOrder("Normal1", types.SideBuy, types.OrderTypeLimit, 100, 1, 1)
	order4.Status = types.OrderStatusCancelled
	event4 := newOrderEvent(ctx, order4)
	mdb.Push(event4)

	// Park pegged order
	order5 := buildOrder("Pegged1", types.SideBuy, types.OrderTypeLimit, 99, 10, 10)
	order5.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceBestBid, Offset: num.NewUint(1)}
	order5.Status = types.OrderStatusParked
	event5 := newOrderEvent(ctx, order5)
	mdb.Push(event5)

	// Create normal order
	order6 := buildOrder("Normal2", types.SideBuy, types.OrderTypeLimit, 100, 1, 1)
	event6 := newOrderEvent(ctx, order6)
	mdb.Push(event6)

	// Unpark pegged order
	order7 := buildOrder("Pegged1", types.SideBuy, types.OrderTypeLimit, 99, 10, 10)
	order7.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceBestBid, Offset: num.NewUint(1)}
	order7.Status = types.OrderStatusActive
	event7 := newOrderEvent(ctx, order7)
	mdb.Push(event7)

	// Fill normal order
	order8 := buildOrder("Normal2", types.SideBuy, types.OrderTypeLimit, 100, 1, 0)
	order8.Status = types.OrderStatusFilled
	event8 := newOrderEvent(ctx, order8)
	mdb.Push(event8)

	// Create new matching order
	order9 := buildOrder("Normal3", types.SideSell, types.OrderTypeLimit, 100, 1, 0)
	order9.Status = types.OrderStatusFilled
	event9 := newOrderEvent(ctx, order9)
	mdb.Push(event9)

	// Park pegged order
	order10 := buildOrder("Pegged1", types.SideBuy, types.OrderTypeLimit, 99, 10, 10)
	order10.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceBestBid, Offset: num.NewUint(1)}
	order10.Status = types.OrderStatusParked
	event10 := newOrderEvent(ctx, order10)
	mdb.Push(event10)

	md, err := mdb.GetMarketDepth(ctx, "M", 0)
	assert.Nil(t, err)
	assert.NotNil(t, md)

	assert.Equal(t, "M", md.MarketId)
	assert.Equal(t, 0, len(md.GetBuy()))
	assert.Equal(t, 0, len(md.GetSell()))
}
