package service_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/service"
	"code.vegaprotocol.io/data-node/subscribers/mocks"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func getTestMDS(t *testing.T, ctx context.Context, ack bool) *service.MarketDepth {
	return service.NewMarketDepth(nil, logging.NewTestLogger())
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

func TestBuyPriceLevels(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	order1 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 9, 9)
	mds.AddOrder(order1, vegaTime, 1)

	order2 := buildOrder("Order2", types.SideBuy, types.OrderTypeLimit, 102, 7, 7)
	mds.AddOrder(order2, vegaTime, 2)

	order3 := buildOrder("Order3", types.SideBuy, types.OrderTypeLimit, 101, 8, 8)
	mds.AddOrder(order3, vegaTime, 3)

	order4 := buildOrder("Order4", types.SideBuy, types.OrderTypeLimit, 99, 10, 10)
	mds.AddOrder(order4, vegaTime, 4)

	assert.Equal(t, 4, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(4), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(7), mds.GetVolumeAtPrice("M", types.SideBuy, 102))
	assert.Equal(t, uint64(1), mds.GetOrderCountAtPrice("M", types.SideBuy, 102))

	assert.Equal(t, uint64(8), mds.GetVolumeAtPrice("M", types.SideBuy, 101))
	assert.Equal(t, uint64(1), mds.GetOrderCountAtPrice("M", types.SideBuy, 101))

	assert.Equal(t, uint64(9), mds.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(1), mds.GetOrderCountAtPrice("M", types.SideBuy, 100))

	assert.Equal(t, uint64(10), mds.GetVolumeAtPrice("M", types.SideBuy, 99))
	assert.Equal(t, uint64(1), mds.GetOrderCountAtPrice("M", types.SideBuy, 99))
}

func TestSellPriceLevels(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	order1 := buildOrder("Order1", types.SideSell, types.OrderTypeLimit, 100, 9, 9)
	mds.AddOrder(order1, vegaTime, 1)

	order2 := buildOrder("Order2", types.SideSell, types.OrderTypeLimit, 102, 7, 7)
	mds.AddOrder(order2, vegaTime, 2)

	order3 := buildOrder("Order3", types.SideSell, types.OrderTypeLimit, 101, 8, 8)
	mds.AddOrder(order3, vegaTime, 3)

	order4 := buildOrder("Order4", types.SideSell, types.OrderTypeLimit, 99, 10, 10)
	mds.AddOrder(order4, vegaTime, 4)

	assert.Equal(t, 0, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 4, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(4), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(7), mds.GetVolumeAtPrice("M", types.SideSell, 102))
	assert.Equal(t, uint64(1), mds.GetOrderCountAtPrice("M", types.SideSell, 102))

	assert.Equal(t, uint64(8), mds.GetVolumeAtPrice("M", types.SideSell, 101))
	assert.Equal(t, uint64(1), mds.GetOrderCountAtPrice("M", types.SideSell, 101))

	assert.Equal(t, uint64(9), mds.GetVolumeAtPrice("M", types.SideSell, 100))
	assert.Equal(t, uint64(1), mds.GetOrderCountAtPrice("M", types.SideSell, 100))

	assert.Equal(t, uint64(10), mds.GetVolumeAtPrice("M", types.SideSell, 99))
	assert.Equal(t, uint64(1), mds.GetOrderCountAtPrice("M", types.SideSell, 99))
}

func TestAddOrderToEmptyBook(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	mds.AddOrder(order, vegaTime, 1)

	assert.Equal(t, 1, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(1), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(10), mds.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(1), mds.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestCancelOrder(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	mds.AddOrder(order, vegaTime, 1)

	cancelorder := *order
	cancelorder.Status = types.OrderStatusCancelled
	mds.AddOrder(&cancelorder, vegaTime, 2)

	assert.Equal(t, 0, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mds.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mds.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestStoppedOrder(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	mds.AddOrder(order, vegaTime, 1)

	cancelorder := *order
	cancelorder.Status = types.OrderStatusStopped
	mds.AddOrder(&cancelorder, vegaTime, 2)

	assert.Equal(t, 0, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mds.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mds.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestExpiredOrder(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	mds.AddOrder(order, vegaTime, 1)

	cancelorder := *order
	cancelorder.Status = types.OrderStatusExpired
	mds.AddOrder(&cancelorder, vegaTime, 2)

	assert.Equal(t, 0, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mds.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mds.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestAmendOrderPrice(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	order2 := buildOrder("Order2", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)

	mds.AddOrder(order, vegaTime, 1)
	mds.AddOrder(order2, vegaTime, 2)

	// Amend the price to force a change in price level
	amendorder := *order
	amendorder.Price = num.NewUint(90)
	amendorder.OriginalPrice = num.NewUint(90)
	mds.AddOrder(&amendorder, vegaTime, 3)

	assert.Equal(t, 2, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(2), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(10), mds.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(10), mds.GetVolumeAtPrice("M", types.SideBuy, 90))
	assert.Equal(t, uint64(1), mds.GetOrderCountAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(1), mds.GetOrderCountAtPrice("M", types.SideBuy, 90))
}

func TestAmendOrderVolumeUp(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	mds.AddOrder(order, vegaTime, 1)

	amendorder := *order
	amendorder.Size = 20
	amendorder.Remaining = 20
	mds.AddOrder(&amendorder, vegaTime, 2)

	assert.Equal(t, 1, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(1), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(20), mds.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(1), mds.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestAmendOrderVolumeDown(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	mds.AddOrder(order, vegaTime, 1)

	amendorder := *order
	amendorder.Size = 5
	amendorder.Remaining = 5
	mds.AddOrder(&amendorder, vegaTime, 2)

	assert.Equal(t, 1, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(1), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(5), mds.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(1), mds.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestAmendOrderVolumeDownToZero(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	mds.AddOrder(order, vegaTime, 1)

	amendorder := *order
	amendorder.Size = 0
	amendorder.Remaining = 0
	mds.AddOrder(&amendorder, vegaTime, 2)

	assert.Equal(t, 0, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mds.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mds.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestPartialFill(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	mds.AddOrder(order, vegaTime, 1)

	pforder := *order
	pforder.Remaining = 5
	mds.AddOrder(&pforder, vegaTime, 2)

	assert.Equal(t, 1, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(1), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(5), mds.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(1), mds.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestIOCPartialFill(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 5)
	order.Status = types.OrderStatusPartiallyFilled
	order.TimeInForce = types.OrderTimeInForceIOC
	mds.AddOrder(order, vegaTime, 1)

	assert.Equal(t, 0, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mds.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mds.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestFullyFill(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	mds.AddOrder(order, vegaTime, 1)

	fforder := *order
	fforder.Remaining = 0
	fforder.Status = types.OrderStatusFilled
	mds.AddOrder(&fforder, vegaTime, 2)

	assert.Equal(t, 0, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mds.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mds.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestMarketOrder(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	// market orders should not stay on the book
	marketorder := buildOrder("Order1", types.SideBuy, types.OrderTypeMarket, 100, 10, 10)
	mds.AddOrder(marketorder, vegaTime, 1)

	assert.Equal(t, 0, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mds.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mds.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestFOKOrder(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	// FOK orders do not stay on the book
	fokorder := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	fokorder.TimeInForce = types.OrderTimeInForceFOK
	mds.AddOrder(fokorder, vegaTime, 1)

	assert.Equal(t, 0, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mds.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mds.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestIOCOrder(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	// IOC orders do not stay on the book
	iocorder := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	iocorder.TimeInForce = types.OrderTimeInForceIOC
	mds.AddOrder(iocorder, vegaTime, 1)

	assert.Equal(t, 0, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mds.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mds.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestRejectedOrder(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	// Rejected orders should be ignored
	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	order.Status = types.OrderStatusRejected
	mds.AddOrder(order, vegaTime, 1)

	assert.Equal(t, 0, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mds.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mds.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestInvalidOrder(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	// Invalid orders should be ignored
	order := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	order.Status = types.OrderStatusUnspecified
	mds.AddOrder(order, vegaTime, 1)

	assert.Equal(t, 0, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mds.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mds.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestPartialMatchOrders(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	order1 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	order2 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 8)
	order3 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 5)
	order4 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 1)

	mds.AddOrder(order1, vegaTime, 1)
	mds.AddOrder(order2, vegaTime, 2)
	mds.AddOrder(order3, vegaTime, 3)
	mds.AddOrder(order4, vegaTime, 4)

	assert.Equal(t, 1, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(1), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(1), mds.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(1), mds.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestFullyMatchOrders(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	order1 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	order2 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 8)
	order3 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 5)
	order4 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 100, 10, 0)
	order4.Status = types.OrderStatusFilled

	mds.AddOrder(order1, vegaTime, 1)
	mds.AddOrder(order2, vegaTime, 2)
	mds.AddOrder(order3, vegaTime, 3)
	mds.AddOrder(order4, vegaTime, 4)

	assert.Equal(t, 0, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(0), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mds.GetVolumeAtPrice("M", types.SideBuy, 100))
	assert.Equal(t, uint64(0), mds.GetOrderCountAtPrice("M", types.SideBuy, 100))
}

func TestRemovingPriceLevels(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	order1 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 101, 10, 10)
	order2 := buildOrder("Order2", types.SideBuy, types.OrderTypeLimit, 100, 10, 10)
	order3 := buildOrder("Order3", types.SideBuy, types.OrderTypeLimit, 102, 10, 10)
	order4 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 101, 10, 0)
	order4.Status = types.OrderStatusFilled

	mds.AddOrder(order1, vegaTime, 1)
	mds.AddOrder(order2, vegaTime, 2)
	mds.AddOrder(order3, vegaTime, 3)
	mds.AddOrder(order4, vegaTime, 4)

	assert.Equal(t, 2, mds.GetBuyPriceLevels("M"))
	assert.Equal(t, 0, mds.GetSellPriceLevels("M"))
	assert.Equal(t, int64(2), mds.GetOrderCount("M"))

	assert.Equal(t, uint64(0), mds.GetVolumeAtPrice("M", types.SideBuy, 101))
	assert.Equal(t, uint64(0), mds.GetOrderCountAtPrice("M", types.SideBuy, 101))
}

func TestMarketDepthFields(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	order1 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 101, 10, 10)
	mds.AddOrder(order1, vegaTime, 1)

	md := mds.GetMarketDepth("M", 0)
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
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	// Create a valid and live pegged order
	order1 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 101, 10, 10)
	order1.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceBestBid, Offset: num.NewUint(1)}
	mds.AddOrder(order1, vegaTime, 1)

	// Park it
	order2 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 0, 10, 10)
	order2.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceBestBid, Offset: num.NewUint(1)}
	order2.Status = types.OrderStatusParked
	mds.AddOrder(order2, vegaTime, 2)

	md := mds.GetMarketDepth("M", 0)
	assert.NotNil(t, md)

	assert.Equal(t, "M", md.MarketId)
	assert.Equal(t, 0, len(md.GetBuy()))
	assert.Equal(t, 0, len(md.GetSell()))

	// Unpark it
	order3 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 101, 10, 10)
	order3.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceBestBid, Offset: num.NewUint(1)}
	order3.Status = types.OrderStatusActive
	mds.AddOrder(order3, vegaTime, 3)

	md2 := mds.GetMarketDepth("M", 0)
	assert.NotNil(t, md2)

	assert.Equal(t, "M", md2.MarketId)
	assert.Equal(t, 1, len(md2.GetBuy()))
	assert.Equal(t, 0, len(md2.GetSell()))
}

func TestParkedOrder(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	// Create a parked pegged order which should not go on the depth book
	order1 := buildOrder("Order1", types.SideBuy, types.OrderTypeLimit, 101, 10, 10)
	order1.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceBestBid, Offset: num.NewUint(1)}
	order1.Status = types.OrderStatusParked
	mds.AddOrder(order1, vegaTime, 1)

	md := mds.GetMarketDepth("M", 0)
	assert.NotNil(t, md)

	assert.Equal(t, "M", md.MarketId)
	assert.Equal(t, 0, len(md.GetBuy()))
	assert.Equal(t, 0, len(md.GetSell()))
}

func TestParkedOrder2(t *testing.T) {
	ctx := context.Background()
	mds := getTestMDS(t, ctx, true)
	vegaTime := time.Now()

	// Create parked pegged order
	order1 := buildOrder("Pegged1", types.SideBuy, types.OrderTypeLimit, 0, 10, 10)
	order1.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceBestBid, Offset: num.NewUint(1)}
	order1.Status = types.OrderStatusParked
	mds.AddOrder(order1, vegaTime, 1)

	// Create normal order
	order2 := buildOrder("Normal1", types.SideBuy, types.OrderTypeLimit, 100, 1, 1)
	mds.AddOrder(order2, vegaTime, 2)

	// Unpark pegged order
	order3 := buildOrder("Pegged1", types.SideBuy, types.OrderTypeLimit, 99, 10, 10)
	order3.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceBestBid, Offset: num.NewUint(1)}
	order3.Status = types.OrderStatusActive
	mds.AddOrder(order3, vegaTime, 3)

	// Cancel normal order
	order4 := buildOrder("Normal1", types.SideBuy, types.OrderTypeLimit, 100, 1, 1)
	order4.Status = types.OrderStatusCancelled
	mds.AddOrder(order4, vegaTime, 4)

	// Park pegged order
	order5 := buildOrder("Pegged1", types.SideBuy, types.OrderTypeLimit, 99, 10, 10)
	order5.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceBestBid, Offset: num.NewUint(1)}
	order5.Status = types.OrderStatusParked
	mds.AddOrder(order5, vegaTime, 5)

	// Create normal order
	order6 := buildOrder("Normal2", types.SideBuy, types.OrderTypeLimit, 100, 1, 1)
	mds.AddOrder(order6, vegaTime, 6)

	// Unpark pegged order
	order7 := buildOrder("Pegged1", types.SideBuy, types.OrderTypeLimit, 99, 10, 10)
	order7.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceBestBid, Offset: num.NewUint(1)}
	order7.Status = types.OrderStatusActive
	mds.AddOrder(order7, vegaTime, 7)

	// Fill normal order
	order8 := buildOrder("Normal2", types.SideBuy, types.OrderTypeLimit, 100, 1, 0)
	order8.Status = types.OrderStatusFilled
	mds.AddOrder(order8, vegaTime, 8)

	// Create new matching order
	order9 := buildOrder("Normal3", types.SideSell, types.OrderTypeLimit, 100, 1, 0)
	order9.Status = types.OrderStatusFilled
	mds.AddOrder(order9, vegaTime, 9)

	// Park pegged order
	order10 := buildOrder("Pegged1", types.SideBuy, types.OrderTypeLimit, 99, 10, 10)
	order10.PeggedOrder = &types.PeggedOrder{Reference: types.PeggedReferenceBestBid, Offset: num.NewUint(1)}
	order10.Status = types.OrderStatusParked
	mds.AddOrder(order10, vegaTime, 10)

	md := mds.GetMarketDepth("M", 0)
	assert.NotNil(t, md)

	assert.Equal(t, "M", md.MarketId)
	assert.Equal(t, 0, len(md.GetBuy()))
	assert.Equal(t, 0, len(md.GetSell()))
}

func TestInitFromSqlStore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("Init from SQL Store when SQL Store is in use", func(t *testing.T) {
		store := mocks.NewMockSqlOrderStore(ctrl)
		store.EXPECT().GetLiveOrders(gomock.Any()).Return([]entities.Order{
			{
				ID:              entities.NewOrderID("22EEA97BF1D9067D7533D0E671FC97C22146CE6785B4B142EBDF53FF0ED73E25"),
				MarketID:        entities.NewMarketID("2EBD1AF4C84D5E004FD2797FF268258BFA21A37A6D0BCE289FB21151ACEF0F36"),
				PartyID:         entities.NewPartyID("FB0C9F50787E5E090591E6600DBBEB5A4771D5A0C9B1AE09BC673AB9F471D210"),
				Side:            2,
				Price:           1200,
				Size:            5,
				Remaining:       5,
				TimeInForce:     1,
				Type:            1,
				Status:          1,
				Reference:       "",
				Reason:          0,
				Version:         1,
				PeggedOffset:    0,
				BatchID:         0,
				PeggedReference: 0,
				LpID:            nil,
				CreatedAt:       time.Time{},
				UpdatedAt:       time.Time{},
				ExpiresAt:       time.Time{},
				VegaTime:        time.Date(2022, 3, 8, 14, 14, 45, 762739000, time.UTC),
				SeqNum:          32,
			},
			{
				ID:              entities.NewOrderID("0E6BFB468B1D57B6463B3A2D133DEA107A56B34CC641235469E834145DE55803"),
				MarketID:        entities.NewMarketID("52D3FCF2EFC15518EDFA25154E909348A2D7F45903C72CD88CB32EFD747CA001"),
				PartyID:         entities.NewPartyID("29FE22227631DE06D9FBBCF2450DEA492E685E5953AEF60A76A95D0DA156806D"),
				Side:            1,
				Price:           22,
				Size:            26,
				Remaining:       26,
				TimeInForce:     1,
				Type:            1,
				Status:          1,
				Reference:       "",
				Reason:          0,
				Version:         2,
				PeggedOffset:    0,
				BatchID:         1,
				PeggedReference: 0,
				LpID:            nil,
				CreatedAt:       time.Time{},
				UpdatedAt:       time.Time{},
				ExpiresAt:       time.Time{},
				VegaTime:        time.Date(2022, 3, 8, 14, 11, 39, 901022000, time.UTC),
				SeqNum:          32,
			},
			{
				ID:              entities.NewOrderID("D8DA96D3B61F1E745061F85D46CE4440E188F846BBD76F7475C7D8AF0E9AB971"),
				MarketID:        entities.NewMarketID("2EBD1AF4C84D5E004FD2797FF268258BFA21A37A6D0BCE289FB21151ACEF0F36"),
				PartyID:         entities.NewPartyID("5F9A129B40E17BA0A17272697E3D521356AFC20BB56BF68C9242097AAFF879BF"),
				Side:            1,
				Price:           900,
				Size:            5,
				Remaining:       5,
				TimeInForce:     1,
				Type:            1,
				Status:          1,
				Reference:       "",
				Reason:          0,
				Version:         1,
				PeggedOffset:    0,
				BatchID:         0,
				PeggedReference: 0,
				LpID:            nil,
				CreatedAt:       time.Time{},
				UpdatedAt:       time.Time{},
				ExpiresAt:       time.Time{},
				VegaTime:        time.Date(2022, 3, 8, 14, 14, 45, 762739000, time.UTC),
				SeqNum:          39,
			},
			{
				ID:              entities.NewOrderID("9CABDED74F357688E96AAD50353122F23C441CF6134BA1B31E4B75D5D5EB7B36"),
				MarketID:        entities.NewMarketID("2EBD1AF4C84D5E004FD2797FF268258BFA21A37A6D0BCE289FB21151ACEF0F36"),
				PartyID:         entities.NewPartyID("5F9A129B40E17BA0A17272697E3D521356AFC20BB56BF68C9242097AAFF879BF"),
				Side:            1,
				Price:           100,
				Size:            1,
				Remaining:       1,
				TimeInForce:     1,
				Type:            1,
				Status:          1,
				Reference:       "",
				Reason:          0,
				Version:         1,
				PeggedOffset:    0,
				BatchID:         0,
				PeggedReference: 0,
				LpID:            nil,
				CreatedAt:       time.Time{},
				UpdatedAt:       time.Time{},
				ExpiresAt:       time.Time{},
				VegaTime:        time.Date(2022, 3, 8, 14, 14, 45, 762739000, time.UTC),
				SeqNum:          43,
			},
			{
				ID:              entities.NewOrderID("4300A037014C7ACFFC1C371697BD7A0ECAE4A54FCC4BFCB8A43E6EF4140A4F64"),
				MarketID:        entities.NewMarketID("2EBD1AF4C84D5E004FD2797FF268258BFA21A37A6D0BCE289FB21151ACEF0F36"),
				PartyID:         entities.NewPartyID("FB0C9F50787E5E090591E6600DBBEB5A4771D5A0C9B1AE09BC673AB9F471D210"),
				Side:            2,
				Price:           100000,
				Size:            1,
				Remaining:       1,
				TimeInForce:     1,
				Type:            1,
				Status:          1,
				Reference:       "",
				Reason:          0,
				Version:         2,
				PeggedOffset:    0,
				BatchID:         0,
				PeggedReference: 0,
				LpID:            nil,
				CreatedAt:       time.Time{},
				UpdatedAt:       time.Time{},
				ExpiresAt:       time.Time{},
				VegaTime:        time.Date(2022, 3, 8, 14, 14, 45, 762739000, time.UTC),
				SeqNum:          53,
			},
			{
				ID:              entities.NewOrderID("F8062CA2F4EE26C6208881CFC9844F12BEE6AA0A087D155BE695AFF6FF00AB00"),
				MarketID:        entities.NewMarketID("2EBD1AF4C84D5E004FD2797FF268258BFA21A37A6D0BCE289FB21151ACEF0F36"),
				PartyID:         entities.NewPartyID("076E3373D4F4197731A3161D2F50CE286B93278BF2B650705691514DD49EFDA1"),
				Side:            2,
				Price:           1201,
				Size:            1301,
				Remaining:       1301,
				TimeInForce:     1,
				Type:            1,
				Status:          1,
				Reference:       "",
				Reason:          0,
				Version:         1,
				PeggedOffset:    0,
				BatchID:         1,
				PeggedReference: 0,
				LpID:            nil,
				CreatedAt:       time.Time{},
				UpdatedAt:       time.Time{},
				ExpiresAt:       time.Time{},
				VegaTime:        time.Date(2022, 3, 8, 14, 14, 58, 985875000, time.UTC),
				SeqNum:          61,
			},
			{
				ID:              entities.NewOrderID("15E8D38DD216C5EE969EC7B7A2EB031E56474A9552CC10E00036A7DC1C0546B5"),
				MarketID:        entities.NewMarketID("2EBD1AF4C84D5E004FD2797FF268258BFA21A37A6D0BCE289FB21151ACEF0F36"),
				PartyID:         entities.NewPartyID("076E3373D4F4197731A3161D2F50CE286B93278BF2B650705691514DD49EFDA1"),
				Side:            1,
				Price:           899,
				Size:            1738,
				Remaining:       1738,
				TimeInForce:     1,
				Type:            1,
				Status:          1,
				Reference:       "",
				Reason:          0,
				Version:         1,
				PeggedOffset:    0,
				BatchID:         1,
				PeggedReference: 0,
				LpID:            nil,
				CreatedAt:       time.Time{},
				UpdatedAt:       time.Time{},
				ExpiresAt:       time.Time{},
				VegaTime:        time.Date(2022, 3, 8, 14, 14, 58, 985875000, time.UTC),
				SeqNum:          66,
			},
		}, nil).Times(1)
		svc := service.NewMarketDepth(store, logging.NewTestLogger())
		svc.Initialise(ctx)
	})
}
