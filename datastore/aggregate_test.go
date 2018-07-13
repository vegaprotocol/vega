package datastore

import (
	"fmt"
	"math/rand"
	"testing"
	"vega/services/msg"

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
		if rand.Intn(3) == 1 {
			price--
		} else {
			price++
		}

		if rand.Intn(5) == 1 {
			timestamp++
		}
		size := uint64(rand.Intn(400) + 800)

		// simulate timestamp gap
		if i == 10 {
			i = 15
			timestamp += 5
		}
		d := generateRandomOrderAndTrade(price, size, timestamp)

		err := newOrderStore.Post(*d.order)
		assert.Nil(t, err)
		err = newTradeStore.Post(*d.trade)
		assert.Nil(t, err)
	}

	candles, err := newTradeStore.GetCandles(testMarket, 0, timestamp, 3)
	fmt.Printf("candles returned:\n")
	for idx, c := range candles.Candles {
		fmt.Printf("%d %+v\n", idx, *c)
	}
	assertCandleIsEmpty(t, candles.Candles[2])
	fmt.Println()
	assert.Nil(t, err)
	assert.Equal(t, 10, len(candles.Candles))

	candles, err = newTradeStore.GetCandles(testMarket, 5, timestamp, 3)
	fmt.Printf("candles returned:\n")
	for idx, c := range candles.Candles {
		fmt.Printf("%d %+v\n", idx, *c)
	}
	fmt.Println()
	assert.Nil(t, err)
	assert.Equal(t, 8, len(candles.Candles))
	assertCandleIsEmpty(t, candles.Candles[0])

	candles, err = newTradeStore.GetCandles(testMarket, 5, timestamp, 2)
	fmt.Printf("candles returned:\n")
	for idx, c := range candles.Candles {
		fmt.Printf("%d %+v\n", idx, *c)
	}
	fmt.Println()
	assert.Nil(t, err)
	assert.Equal(t, 12, len(candles.Candles))
	assertCandleIsEmpty(t, candles.Candles[0])
	assertCandleIsEmpty(t, candles.Candles[1])

	candles, err = newTradeStore.GetCandles(testMarket, 10, timestamp, 2)
	fmt.Printf("candles returned:\n")
	for idx, c := range candles.Candles {
		fmt.Printf("%d %+v\n", idx, *c)
	}
	fmt.Println()
	assert.Nil(t, err)
	assert.Equal(t, 9, len(candles.Candles))

}

func assertCandleIsEmpty(t assert.TestingT, candle *msg.Candle) {
	assert.Equal(t, uint64(0), candle.Volume)
	assert.Equal(t, uint64(0), candle.High)
	assert.Equal(t, uint64(0), candle.Low)
	assert.Equal(t, uint64(0), candle.Open)
	assert.Equal(t, uint64(0), candle.Close)
}
