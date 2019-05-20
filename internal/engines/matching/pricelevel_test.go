package matching

import (
	"testing"

	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func TestGetPriceLevel(t *testing.T) {
	side := &OrderBookSide{}
	assert.Equal(t, 0, len(side.levels))
	side.getPriceLevel(100, types.Side_Sell)
	assert.Equal(t, 1, len(side.levels))

	side.getPriceLevel(110, types.Side_Sell)
	assert.Equal(t, 2, len(side.levels))

	side.getPriceLevel(100, types.Side_Sell)
	assert.Equal(t, 2, len(side.levels))
}

func TestAddAndRemoveOrdersToPriceLevel(t *testing.T) {
	side := &OrderBookSide{}
	l := side.getPriceLevel(100, types.Side_Sell)
	order := &types.Order{
		MarketID:  "testOrderBook",
		PartyID:   "A",
		Side:      types.Side_Sell,
		Price:     101,
		Size:      100,
		Remaining: 100,
		Type:      types.Order_GTC,
		CreatedAt: 0,
	}

	// add orders
	assert.Equal(t, 0, len(l.orders))
	l.addOrder(order)
	assert.Equal(t, 1, len(l.orders))
	l.addOrder(order)
	assert.Equal(t, 2, len(l.orders))

	// remove orders
	l.removeOrder(1)
	assert.Equal(t, 1, len(l.orders))
	l.removeOrder(0)
	assert.Equal(t, 0, len(l.orders))
}

func TestUncross(t *testing.T) {
	logger := logging.NewTestLogger()
	defer logger.Sync()

	side := &OrderBookSide{}
	l := side.getPriceLevel(100, types.Side_Sell)
	passiveOrder := &types.Order{
		MarketID:  "testOrderBook",
		PartyID:   "A",
		Side:      types.Side_Sell,
		Price:     101,
		Size:      100,
		Remaining: 100,
		Type:      types.Order_GTC,
		CreatedAt: 0,
	}
	l.addOrder(passiveOrder)

	aggresiveOrder := &types.Order{
		MarketID:  "testOrderBook",
		PartyID:   "B",
		Side:      types.Side_Buy,
		Price:     101,
		Size:      100,
		Remaining: 100,
		Type:      types.Order_GTC,
		CreatedAt: 0,
	}
	filled, trades, impactedOrders := l.uncross(aggresiveOrder)
	assert.Equal(t, true, filled)
	assert.Equal(t, 1, len(trades))
	assert.Equal(t, 1, len(impactedOrders))
}

func benchmarkGetPriceLevel(priceLevelCnt int, b *testing.B) {
	priceToFind1 := uint64(float64(priceLevelCnt) * 0.75)
	priceToFind2 := uint64(float64(priceLevelCnt) * 0.50)
	priceToFind3 := uint64(float64(priceLevelCnt) * 0.25)
	bookside := OrderBookSide{levels: []*PriceLevel{}}
	for i := 0; i < priceLevelCnt; i += 1 {
		bookside.getPriceLevel(uint64(i), types.Side_Buy)
		bookside.getPriceLevel(uint64(i), types.Side_Sell)
	}

	for n := 0; n < b.N; n++ {
		bookside.getPriceLevel(priceToFind1, types.Side_Buy)
		bookside.getPriceLevel(priceToFind2, types.Side_Buy)
		bookside.getPriceLevel(priceToFind3, types.Side_Buy)

		bookside.getPriceLevel(priceToFind1, types.Side_Sell)
		bookside.getPriceLevel(priceToFind2, types.Side_Sell)
		bookside.getPriceLevel(priceToFind3, types.Side_Sell)
	}
}

func BenchmarkGetPriceLevel100(b *testing.B)   { benchmarkGetPriceLevel(100, b) }
func BenchmarkGetPriceLevel1000(b *testing.B)  { benchmarkGetPriceLevel(1000, b) }
func BenchmarkGetPriceLevel2500(b *testing.B)  { benchmarkGetPriceLevel(2500, b) }
func BenchmarkGetPriceLevel5000(b *testing.B)  { benchmarkGetPriceLevel(5000, b) }
func BenchmarkGetPriceLevel10000(b *testing.B) { benchmarkGetPriceLevel(10000, b) }
