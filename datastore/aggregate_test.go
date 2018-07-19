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
				Party: testParty,
			},
		},
		&Trade{
			Trade: msg.Trade{
				Id:        tradeId,
				Price:     price,
				Market:    testMarket,
				Size:      size,
				Timestamp: timestamp,
				Buyer: testPartyA,
				Seller: testPartyB,
			},
			PassiveOrderId: orderId,
			AggressiveOrderId: orderId,
		},
	}
	return d
}

func populateStores(t *testing.T, orderStore OrderStore, tradeStore TradeStore) uint64 {
	price := uint64(100)
	timestamp := uint64(0)
	for i := 0; i < 100; i++ {
		if i%3 == 0{
			price--
		} else {
			price++
		}

		if i%5 == 0 {
			timestamp++
		}
		size := uint64(1000)

		// simulate timestamp gap
		if i == 10 {
			i = 15
			timestamp += 5
		}
		d := generateRandomOrderAndTrade(price, size, timestamp)

		err := orderStore.Post(*d.order)
		assert.Nil(t, err)
		err = tradeStore.Post(*d.trade)
		fmt.Printf("%+v\n", d.trade)
		assert.Nil(t, err)
	}
	return timestamp
}

func TestMemTradeStore_GetCandles(t *testing.T) {
	var memStore = NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
	var newOrderStore = NewOrderStore(&memStore)
	var newTradeStore = NewTradeStore(&memStore)

	timestamp := populateStores(t, newOrderStore, newTradeStore)

	candles, err := newTradeStore.GetCandles(testMarket, 0, timestamp, 3)
	fmt.Printf("candles returned:\n")
	for idx, c := range candles.Candles {
		fmt.Printf("%d %+v\n", idx, *c)
	}

	assert.Equal(t, uint64(10000), candles.Candles[0].Volume)
	assert.Equal(t, uint64(103), candles.Candles[0].High)
	assert.Equal(t, uint64(99), candles.Candles[0].Low)
	assert.Equal(t, uint64(99), candles.Candles[0].Open)
	assert.Equal(t, uint64(102), candles.Candles[0].Close)

	assert.Equal(t, uint64(0), candles.Candles[1].Volume)
	assert.Equal(t, uint64(102), candles.Candles[1].High)
	assert.Equal(t, uint64(102), candles.Candles[1].Low)
	assert.Equal(t, uint64(102), candles.Candles[1].Open)
	assert.Equal(t, uint64(102), candles.Candles[1].Close)

	assert.Equal(t, uint64(5000), candles.Candles[2].Volume)
	assert.Equal(t, uint64(105), candles.Candles[2].High)
	assert.Equal(t, uint64(103), candles.Candles[2].Low)
	assert.Equal(t, uint64(103), candles.Candles[2].Open)
	assert.Equal(t, uint64(105), candles.Candles[2].Close)

	assert.Equal(t, uint64(15000), candles.Candles[3].Volume)
	assert.Equal(t, uint64(110), candles.Candles[3].High)
	assert.Equal(t, uint64(105), candles.Candles[3].Low)
	assert.Equal(t, uint64(106), candles.Candles[3].Open)
	assert.Equal(t, uint64(110), candles.Candles[3].Close)

	fmt.Println()
	assert.Nil(t, err)
	assert.Equal(t, 9, len(candles.Candles))
}

func TestMemTradeStore_GetCandles2(t *testing.T) {
	var memStore = NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
	var newOrderStore = NewOrderStore(&memStore)
	var newTradeStore = NewTradeStore(&memStore)

	timestamp := populateStores(t, newOrderStore, newTradeStore)

	candles, err := newTradeStore.GetCandles(testMarket, 5, timestamp, 3)
	fmt.Printf("candles returned:\n")
	for idx, c := range candles.Candles {
		fmt.Printf("%d %+v\n", idx, *c)
	}
	fmt.Println()
	assert.Nil(t, err)
	assert.Equal(t, 7, len(candles.Candles))

	assert.Equal(t, uint64(0), candles.Candles[0].Volume)
	assert.Equal(t, uint64(102), candles.Candles[0].High)
	assert.Equal(t, uint64(102), candles.Candles[0].Low)
	assert.Equal(t, uint64(102), candles.Candles[0].Open)
	assert.Equal(t, uint64(102), candles.Candles[0].Close)

	assert.Equal(t, uint64(15000), candles.Candles[1].Volume)
	assert.Equal(t, uint64(109), candles.Candles[1].High)
	assert.Equal(t, uint64(103), candles.Candles[1].Low)
	assert.Equal(t, uint64(103), candles.Candles[1].Open)
	assert.Equal(t, uint64(109), candles.Candles[1].Close)

	assert.Equal(t, uint64(15000), candles.Candles[2].Volume)
	assert.Equal(t, uint64(114), candles.Candles[2].High)
	assert.Equal(t, uint64(108), candles.Candles[2].Low)
	assert.Equal(t, uint64(108), candles.Candles[2].Open)
	assert.Equal(t, uint64(114), candles.Candles[2].Close)
}

func TestMemTradeStore_GetCandles3(t *testing.T) {
	var memStore = NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
	var newOrderStore = NewOrderStore(&memStore)
	var newTradeStore = NewTradeStore(&memStore)

	timestamp := populateStores(t, newOrderStore, newTradeStore)
	candles, err := newTradeStore.GetCandles(testMarket, 5, timestamp, 2)
	fmt.Printf("candles returned:\n")
	for idx, c := range candles.Candles {
		fmt.Printf("%d %+v\n", idx, *c)
	}
	fmt.Println()
	assert.Nil(t, err)
	assert.Equal(t, 10, len(candles.Candles))

	assert.Equal(t, uint64(0), candles.Candles[0].Volume)
	assert.Equal(t, uint64(102), candles.Candles[0].High)
	assert.Equal(t, uint64(102), candles.Candles[0].Low)
	assert.Equal(t, uint64(102), candles.Candles[0].Open)
	assert.Equal(t, uint64(102), candles.Candles[0].Close)

	assert.Equal(t, uint64(5000), candles.Candles[1].Volume)
	assert.Equal(t, uint64(105), candles.Candles[1].High)
	assert.Equal(t, uint64(103), candles.Candles[1].Low)
	assert.Equal(t, uint64(103), candles.Candles[1].Open)
	assert.Equal(t, uint64(105), candles.Candles[1].Close)

	assert.Equal(t, uint64(10000), candles.Candles[2].Volume)
	assert.Equal(t, uint64(109), candles.Candles[2].High)
	assert.Equal(t, uint64(105), candles.Candles[2].Low)
	assert.Equal(t, uint64(106), candles.Candles[2].Open)
	assert.Equal(t, uint64(109), candles.Candles[2].Close)
}

func TestMemTradeStore_GetCandles4(t *testing.T) {
	var memStore= NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
	var newOrderStore= NewOrderStore(&memStore)
	var newTradeStore= NewTradeStore(&memStore)

	timestamp := populateStores(t, newOrderStore, newTradeStore)
	candles, err := newTradeStore.GetCandles(testMarket, 10, timestamp, 2)
	fmt.Printf("candles returned:\n")
	for idx, c := range candles.Candles {
		fmt.Printf("%d %+v\n", idx, *c)
	}
	fmt.Println()
	assert.Nil(t, err)
	assert.Equal(t, 8, len(candles.Candles))

	assert.Equal(t, uint64(10000), candles.Candles[0].Volume)
	assert.Equal(t, uint64(110), candles.Candles[0].High)
	assert.Equal(t, uint64(107), candles.Candles[0].Low)
	assert.Equal(t, uint64(107), candles.Candles[0].Open)
	assert.Equal(t, uint64(110), candles.Candles[0].Close)


	assert.Equal(t, uint64(5000), candles.Candles[7].Volume)
	assert.Equal(t, uint64(132), candles.Candles[7].High)
	assert.Equal(t, uint64(130), candles.Candles[7].Low)
	assert.Equal(t, uint64(131), candles.Candles[7].Open)
	assert.Equal(t, uint64(131), candles.Candles[7].Close)

}