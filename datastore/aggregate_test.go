package datastore

import (
	"fmt"
	"math/rand"
	"testing"
	"vega/proto"

	"github.com/stretchr/testify/assert"
)

type TestOrderAndTrades struct {
	order *Order
	trade *Trade
}

func generateRandomOrderAndTrade(price, size, timestamp uint64) *TestOrderAndTrades {
	orderId := fmt.Sprintf("%d", rand.Intn(1000000000000))
	tradeId := fmt.Sprintf("%d", rand.Intn(1000000000000))
	d := &TestOrderAndTrades{
		&Order{
			Order: msg.Order{
				Id:     orderId,
				Market: testMarket,
			},
		},
		&Trade{
			Trade: msg.Trade{
				Id:        tradeId,
				Price:     price,
				Market:    testMarket,
				Size:      size,
				Timestamp: timestamp,
			},
			OrderId: orderId,
		},
	}
	return d
}

func TestMemTradeStore_GetCandles(t *testing.T) {
	var memStore = NewMemStore([]string{testMarket})
	var newOrderStore = NewOrderStore(&memStore)
	var newTradeStore = NewTradeStore(&memStore)

	price := uint64(100)
	timestamp := uint64(0)
	for i := 0; i < 100; i++ {
		if rand.Intn(3) == 1{
			price--
		} else {
			price++
		}

		if rand.Intn(5) == 1 {
			timestamp++
		}
		size := uint64(rand.Intn(400) + 800)

		d := generateRandomOrderAndTrade(price, size, timestamp)

		err := newOrderStore.Post(*d.order)
		assert.Nil(t, err)
		err = newTradeStore.Post(*d.trade)
		assert.Nil(t, err)
	}

	candles, err := newTradeStore.GetCandles(testMarket, 0, 3)
	fmt.Printf("candles returned:\n")
	for idx, c := range candles.Candles {
		fmt.Printf("%d %+v\n", idx, *c)
	}
	fmt.Println()
	assert.Nil(t, err)
	assert.Equal(t, 8, len(candles.Candles))


	candles, err = newTradeStore.GetCandles(testMarket, 5, 3)
	fmt.Printf("candles returned:\n")
	for idx, c := range candles.Candles {
		fmt.Printf("%d %+v\n", idx, *c)
	}
	fmt.Println()
	assert.Nil(t, err)
	assert.Equal(t, 7, len(candles.Candles))

	candles, err = newTradeStore.GetCandles(testMarket, 5, 2)
	fmt.Printf("candles returned:\n")
	for idx, c := range candles.Candles {
		fmt.Printf("%d %+v\n", idx, *c)
	}
	fmt.Println()
	assert.Nil(t, err)
	assert.Equal(t, 10, len(candles.Candles))

	candles, err = newTradeStore.GetCandles(testMarket, 10, 2)
	fmt.Printf("candles returned:\n")
	for idx, c := range candles.Candles {
		fmt.Printf("%d %+v\n", idx, *c)
	}
	fmt.Println()
	assert.Nil(t, err)
	assert.Equal(t, 7, len(candles.Candles))

}

