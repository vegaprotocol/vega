package datastore

import (
	"fmt"
	"math/rand"
	"testing"
	"vega/msg"

	"github.com/stretchr/testify/assert"
)
//
//type TestOrderAndTrades struct {
//	order *msg.Order
//	trade *msg.Trade
//}
//
//func generateRandomOrderAndTrade(price, size, timestamp uint64) *TestOrderAndTrades {
//	orderId := fmt.Sprintf("%d", rand.Intn(1000000000000))
//	tradeId := fmt.Sprintf("%d", rand.Intn(1000000000000))
//	d := &TestOrderAndTrades{
//		order: &msg.Order{
//				Id:     orderId,
//				Market: testMarket,
//				Party: testPartyA,
//		},
//		trade: &msg.Trade{
//				Id:        tradeId,
//				Price:     price,
//				Market:    testMarket,
//				Size:      size,
//				Timestamp: timestamp,
//				Buyer: testPartyA,
//				Seller: testPartyB,
//		},
//	}
//	return d
//}
//
//func populateStores(t *testing.T, orderStore OrderStore, tradeStore TradeStore) uint64 {
//	price := uint64(100)
//	timestamp := uint64(0)
//	for i := 0; i < 100; i++ {
//		if i%3 == 0{
//			price--
//		} else {
//			price++
//		}
//
//		if i%5 == 0 {
//			timestamp++
//		}
//		size := uint64(1000)
//
//		// simulate timestamp gap
//		if i == 10 {
//			i = 15
//			timestamp += 5
//		}
//		d := generateRandomOrderAndTrade(price, size, timestamp)
//
//		err := orderStore.Post(d.order)
//		assert.Nil(t, err)
//		err = tradeStore.Post(*d.trade)
//		fmt.Printf("%+v\n", d.trade)
//		assert.Nil(t, err)
//	}
//	return timestamp
//}
//
//func populateStoresWithEmptyStartingTrading(t *testing.T, orderStore OrderStore, tradeStore TradeStore) uint64 {
//	price := uint64(100)
//	timestamp := uint64(0)
//	for i := 0; i < 100; i++ {
//		if i%3 == 0{
//			price--
//		} else {
//			price++
//		}
//
//		if i%5 == 0 {
//			timestamp++
//		}
//		size := uint64(1000)
//
//		// simulate timestamp gap
//		if i == 10 {
//			i = 50
//			timestamp += 40
//		}
//		d := generateRandomOrderAndTrade(price, size, timestamp)
//
//		err := orderStore.Post(d.order)
//		assert.Nil(t, err)
//		err = tradeStore.Post(*d.trade)
//		fmt.Printf("%+v\n", d.trade)
//		assert.Nil(t, err)
//	}
//	return timestamp
//}
//
//func populateStoresWithEmptyMidAndEndingTrading(t *testing.T, orderStore OrderStore, tradeStore TradeStore) uint64 {
//	price := uint64(100)
//	timestamp := uint64(0)
//	for i := 0; i < 100; i++ {
//		if i%3 == 0{
//			price--
//		} else {
//			price++
//		}
//
//		if i%5 == 0 {
//			timestamp++
//		}
//		size := uint64(1000)
//
//		// simulate timestamp gap
//		if i == 50 {
//			i = 60
//			timestamp += 10
//		}
//
//		if i == 80 {
//			i = 100
//			timestamp += 20
//		}
//
//		d := generateRandomOrderAndTrade(price, size, timestamp)
//
//		err := orderStore.Post(d.order)
//		assert.Nil(t, err)
//		err = tradeStore.Post(*d.trade)
//		fmt.Printf("%+v\n", d.trade)
//		assert.Nil(t, err)
//	}
//	return timestamp
//}
//
//func TestMemTradeStore_GetCandles(t *testing.T) {
//	var memStore = NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
//	var newOrderStore = NewOrderStore("../tmp/orderstore")
//	defer newOrderStore.Close()
//	var newTradeStore = NewTradeStore(&memStore)
//
//	timestamp := populateStores(t, newOrderStore, newTradeStore)
//
//	candles, err := newTradeStore.GetCandles(testMarket, 0, timestamp, 3)
//	fmt.Printf("candles returned:\n")
//	for idx, c := range candles.Candles {
//		fmt.Printf("%d %+v\n", idx, *c)
//	}
//
//	assert.Equal(t, uint64(10000), candles.Candles[0].Volume)
//	assert.Equal(t, uint64(103), candles.Candles[0].High)
//	assert.Equal(t, uint64(99), candles.Candles[0].Low)
//	assert.Equal(t, uint64(99), candles.Candles[0].Open)
//	assert.Equal(t, uint64(102), candles.Candles[0].Close)
//
//	assert.Equal(t, uint64(0), candles.Candles[1].Volume)
//	assert.Equal(t, uint64(102), candles.Candles[1].High)
//	assert.Equal(t, uint64(102), candles.Candles[1].Low)
//	assert.Equal(t, uint64(102), candles.Candles[1].Open)
//	assert.Equal(t, uint64(102), candles.Candles[1].Close)
//
//	assert.Equal(t, uint64(5000), candles.Candles[2].Volume)
//	assert.Equal(t, uint64(105), candles.Candles[2].High)
//	assert.Equal(t, uint64(103), candles.Candles[2].Low)
//	assert.Equal(t, uint64(103), candles.Candles[2].Open)
//	assert.Equal(t, uint64(105), candles.Candles[2].Close)
//
//	assert.Equal(t, uint64(15000), candles.Candles[3].Volume)
//	assert.Equal(t, uint64(110), candles.Candles[3].High)
//	assert.Equal(t, uint64(105), candles.Candles[3].Low)
//	assert.Equal(t, uint64(106), candles.Candles[3].Open)
//	assert.Equal(t, uint64(110), candles.Candles[3].Close)
//
//	fmt.Println()
//	assert.Nil(t, err)
//	assert.Equal(t, 9, len(candles.Candles))
//}
//
//func TestMemTradeStore_GetCandles2(t *testing.T) {
//	var memStore = NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
//	var newOrderStore = NewOrderStore("./orderStore")
//	defer newOrderStore.Close()
//
//	var newTradeStore = NewTradeStore(&memStore)
//
//	timestamp := populateStores(t, newOrderStore, newTradeStore)
//
//	candles, err := newTradeStore.GetCandles(testMarket, 5, timestamp, 3)
//	fmt.Printf("candles returned:\n")
//	for idx, c := range candles.Candles {
//		fmt.Printf("%d %+v\n", idx, *c)
//	}
//	fmt.Println()
//	assert.Nil(t, err)
//	assert.Equal(t, 7, len(candles.Candles))
//
//	assert.Equal(t, uint64(0), candles.Candles[0].Volume)
//	assert.Equal(t, uint64(102), candles.Candles[0].High)
//	assert.Equal(t, uint64(102), candles.Candles[0].Low)
//	assert.Equal(t, uint64(102), candles.Candles[0].Open)
//	assert.Equal(t, uint64(102), candles.Candles[0].Close)
//
//	assert.Equal(t, uint64(15000), candles.Candles[1].Volume)
//	assert.Equal(t, uint64(109), candles.Candles[1].High)
//	assert.Equal(t, uint64(103), candles.Candles[1].Low)
//	assert.Equal(t, uint64(103), candles.Candles[1].Open)
//	assert.Equal(t, uint64(109), candles.Candles[1].Close)
//
//	assert.Equal(t, uint64(15000), candles.Candles[2].Volume)
//	assert.Equal(t, uint64(114), candles.Candles[2].High)
//	assert.Equal(t, uint64(108), candles.Candles[2].Low)
//	assert.Equal(t, uint64(108), candles.Candles[2].Open)
//	assert.Equal(t, uint64(114), candles.Candles[2].Close)
//}
//
//func TestMemTradeStore_GetCandles3(t *testing.T) {
//	var memStore = NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
//	var newOrderStore = NewOrderStore("./orderStore")
//	defer newOrderStore.Close()
//	var newTradeStore = NewTradeStore(&memStore)
//
//	timestamp := populateStores(t, newOrderStore, newTradeStore)
//	candles, err := newTradeStore.GetCandles(testMarket, 5, timestamp, 2)
//	fmt.Printf("candles returned:\n")
//	for idx, c := range candles.Candles {
//		fmt.Printf("%d %+v\n", idx, *c)
//	}
//	fmt.Println()
//	assert.Nil(t, err)
//	assert.Equal(t, 10, len(candles.Candles))
//
//	assert.Equal(t, uint64(0), candles.Candles[0].Volume)
//	assert.Equal(t, uint64(102), candles.Candles[0].High)
//	assert.Equal(t, uint64(102), candles.Candles[0].Low)
//	assert.Equal(t, uint64(102), candles.Candles[0].Open)
//	assert.Equal(t, uint64(102), candles.Candles[0].Close)
//
//	assert.Equal(t, uint64(5000), candles.Candles[1].Volume)
//	assert.Equal(t, uint64(105), candles.Candles[1].High)
//	assert.Equal(t, uint64(103), candles.Candles[1].Low)
//	assert.Equal(t, uint64(103), candles.Candles[1].Open)
//	assert.Equal(t, uint64(105), candles.Candles[1].Close)
//
//	assert.Equal(t, uint64(10000), candles.Candles[2].Volume)
//	assert.Equal(t, uint64(109), candles.Candles[2].High)
//	assert.Equal(t, uint64(105), candles.Candles[2].Low)
//	assert.Equal(t, uint64(106), candles.Candles[2].Open)
//	assert.Equal(t, uint64(109), candles.Candles[2].Close)
//}
//
//func TestMemTradeStore_GetCandles4(t *testing.T) {
//	var memStore= NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
//	var newOrderStore = NewOrderStore("./orderStore")
//	defer newOrderStore.Close()
//	var newTradeStore= NewTradeStore(&memStore)
//
//	timestamp := populateStores(t, newOrderStore, newTradeStore)
//	candles, err := newTradeStore.GetCandles(testMarket, 10, timestamp, 2)
//	fmt.Printf("candles returned:\n")
//	for idx, c := range candles.Candles {
//		fmt.Printf("%d %+v\n", idx, *c)
//	}
//	fmt.Println()
//	assert.Nil(t, err)
//	assert.Equal(t, 8, len(candles.Candles))
//
//	assert.Equal(t, uint64(10000), candles.Candles[0].Volume)
//	assert.Equal(t, uint64(110), candles.Candles[0].High)
//	assert.Equal(t, uint64(107), candles.Candles[0].Low)
//	assert.Equal(t, uint64(107), candles.Candles[0].Open)
//	assert.Equal(t, uint64(110), candles.Candles[0].Close)
//
//
//	assert.Equal(t, uint64(5000), candles.Candles[7].Volume)
//	assert.Equal(t, uint64(132), candles.Candles[7].High)
//	assert.Equal(t, uint64(130), candles.Candles[7].Low)
//	assert.Equal(t, uint64(131), candles.Candles[7].Open)
//	assert.Equal(t, uint64(131), candles.Candles[7].Close)
//}
//
//func TestMemTradeStore_GetCandle(t *testing.T) {
//	var memStore= NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
//	var newOrderStore = NewOrderStore("./orderStore")
//	defer newOrderStore.Close()
//	var newTradeStore= NewTradeStore(&memStore)
//
//	timestamp := populateStores(t, newOrderStore, newTradeStore)
//
//	candle, err := newTradeStore.GetCandle(testMarket,timestamp, timestamp)
//	fmt.Printf("candle returned: %+v\n", candle)
//	assert.Nil(t, err)
//
//	assert.Equal(t, uint64(5000), candle.Volume)
//	assert.Equal(t, uint64(132), candle.High)
//	assert.Equal(t, uint64(130), candle.Low)
//	assert.Equal(t, uint64(131), candle.Open)
//	assert.Equal(t, uint64(131), candle.Close)
//
//	candle, err = newTradeStore.GetCandle(testMarket,timestamp+10, timestamp+10)
//	fmt.Printf("candle returned: %+v\n", candle)
//	assert.Nil(t, err)
//
//	assert.Equal(t, uint64(0), candle.Volume)
//	assert.Equal(t, uint64(131), candle.High)
//	assert.Equal(t, uint64(131), candle.Low)
//	assert.Equal(t, uint64(131), candle.Open)
//	assert.Equal(t, uint64(131), candle.Close)
//}
//
//func TestMemTradeStore_GetCandles5NonTradingSinceCandles(t *testing.T) {
//	var memStore= NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
//	var newOrderStore = NewOrderStore("./orderStore")
//	defer newOrderStore.Close()
//	var newTradeStore= NewTradeStore(&memStore)
//
//	timestamp := populateStoresWithEmptyStartingTrading(t, newOrderStore, newTradeStore)
//	candles, err := newTradeStore.GetCandles(testMarket, 10, timestamp, 2)
//	fmt.Printf("candles returned:\n")
//	for idx, c := range candles.Candles {
//		fmt.Printf("%d %+v\n", idx, *c)
//	}
//	fmt.Println()
//	assert.Nil(t, err)
//	assert.Equal(t, 22, len(candles.Candles))
//
//	assert.Equal(t, uint64(0), candles.Candles[0].Volume)
//	assert.Equal(t, uint64(102), candles.Candles[0].High)
//	assert.Equal(t, uint64(102), candles.Candles[0].Low)
//	assert.Equal(t, uint64(102), candles.Candles[0].Open)
//	assert.Equal(t, uint64(102), candles.Candles[0].Close)
//
//	assert.Equal(t, uint64(0), candles.Candles[15].Volume)
//	assert.Equal(t, uint64(102), candles.Candles[15].High)
//	assert.Equal(t, uint64(102), candles.Candles[15].Low)
//	assert.Equal(t, uint64(102), candles.Candles[15].Open)
//	assert.Equal(t, uint64(102), candles.Candles[15].Close)
//
//	assert.Equal(t, uint64(5000), candles.Candles[16].Volume)
//	assert.Equal(t, uint64(104), candles.Candles[16].High)
//	assert.Equal(t, uint64(102), candles.Candles[16].Low)
//	assert.Equal(t, uint64(103), candles.Candles[16].Open)
//	assert.Equal(t, uint64(103), candles.Candles[16].Close)
//}
//
//func TestMemTradeStore_GetCandles6NonTradingSinceCandles(t *testing.T) {
//	var memStore= NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
//	var newOrderStore = NewOrderStore("./orderStore")
//	defer newOrderStore.Close()
//	var newTradeStore= NewTradeStore(&memStore)
//
//	timestamp := populateStoresWithEmptyStartingTrading(t, newOrderStore, newTradeStore)
//	candles, err := newTradeStore.GetCandles(testMarket, 11, timestamp, 2)
//	fmt.Printf("candles returned:\n")
//	for idx, c := range candles.Candles {
//		fmt.Printf("%d %+v\n", idx, *c)
//	}
//	fmt.Println()
//	assert.Nil(t, err)
//	assert.Equal(t, 21, len(candles.Candles))
//
//	assert.Equal(t, uint64(0), candles.Candles[0].Volume)
//	assert.Equal(t, uint64(102), candles.Candles[0].High)
//	assert.Equal(t, uint64(102), candles.Candles[0].Low)
//	assert.Equal(t, uint64(102), candles.Candles[0].Open)
//	assert.Equal(t, uint64(102), candles.Candles[0].Close)
//
//	assert.Equal(t, uint64(0), candles.Candles[15].Volume)
//	assert.Equal(t, uint64(102), candles.Candles[15].High)
//	assert.Equal(t, uint64(102), candles.Candles[15].Low)
//	assert.Equal(t, uint64(102), candles.Candles[15].Open)
//	assert.Equal(t, uint64(102), candles.Candles[15].Close)
//
//	assert.Equal(t, uint64(10000), candles.Candles[16].Volume)
//	assert.Equal(t, uint64(106), candles.Candles[16].High)
//	assert.Equal(t, uint64(102), candles.Candles[16].Low)
//	assert.Equal(t, uint64(103), candles.Candles[16].Open)
//	assert.Equal(t, uint64(106), candles.Candles[16].Close)
//}
//
//func TestMemTradeStore_GetCandles7NonTradingSinceCandles(t *testing.T) {
//	var memStore= NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
//	var newOrderStore = NewOrderStore("./orderStore")
//	defer newOrderStore.Close()
//	var newTradeStore= NewTradeStore(&memStore)
//
//	timestamp := populateStoresWithEmptyStartingTrading(t, newOrderStore, newTradeStore)
//	candles, err := newTradeStore.GetCandles(testMarket, 12, timestamp, 2)
//	fmt.Printf("candles returned:\n")
//	for idx, c := range candles.Candles {
//		fmt.Printf("%d %+v\n", idx, *c)
//	}
//	fmt.Println()
//	assert.Nil(t, err)
//	assert.Equal(t, 21, len(candles.Candles))
//
//	assert.Equal(t, uint64(0), candles.Candles[0].Volume)
//	assert.Equal(t, uint64(102), candles.Candles[0].High)
//	assert.Equal(t, uint64(102), candles.Candles[0].Low)
//	assert.Equal(t, uint64(102), candles.Candles[0].Open)
//	assert.Equal(t, uint64(102), candles.Candles[0].Close)
//
//	assert.Equal(t, uint64(0), candles.Candles[14].Volume)
//	assert.Equal(t, uint64(102), candles.Candles[14].High)
//	assert.Equal(t, uint64(102), candles.Candles[14].Low)
//	assert.Equal(t, uint64(102), candles.Candles[14].Open)
//	assert.Equal(t, uint64(102), candles.Candles[14].Close)
//
//	assert.Equal(t, uint64(5000), candles.Candles[15].Volume)
//	assert.Equal(t, uint64(104), candles.Candles[15].High)
//	assert.Equal(t, uint64(102), candles.Candles[15].Low)
//	assert.Equal(t, uint64(103), candles.Candles[15].Open)
//	assert.Equal(t, uint64(103), candles.Candles[15].Close)
//}
//
//func TestMemTradeStore_GetCandles8NonTradingSinceCandles(t *testing.T) {
//	var memStore= NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
//	var newOrderStore = NewOrderStore("./orderStore")
//	defer newOrderStore.Close()
//	var newTradeStore= NewTradeStore(&memStore)
//
//	timestamp := populateStoresWithEmptyMidAndEndingTrading(t, newOrderStore, newTradeStore)
//	candles, err := newTradeStore.GetCandles(testMarket, 12, timestamp, 2)
//	fmt.Printf("candles returned:\n")
//	for idx, c := range candles.Candles {
//		fmt.Printf("%d %+v\n", idx, *c)
//	}
//	fmt.Println()
//	assert.Nil(t, err)
//	assert.Equal(t, 17, len(candles.Candles))
//
//	assert.Equal(t, uint64(0), candles.Candles[0].Volume)
//	assert.Equal(t, uint64(116), candles.Candles[0].High)
//	assert.Equal(t, uint64(116), candles.Candles[0].Low)
//	assert.Equal(t, uint64(116), candles.Candles[0].Open)
//	assert.Equal(t, uint64(116), candles.Candles[0].Close)
//
//	assert.Equal(t, uint64(0), candles.Candles[3].Volume)
//	assert.Equal(t, uint64(116), candles.Candles[3].High)
//	assert.Equal(t, uint64(116), candles.Candles[3].Low)
//	assert.Equal(t, uint64(116), candles.Candles[3].Open)
//	assert.Equal(t, uint64(116), candles.Candles[3].Close)
//
//	assert.Equal(t, uint64(5000), candles.Candles[4].Volume)
//	assert.Equal(t, uint64(119), candles.Candles[4].High)
//	assert.Equal(t, uint64(117), candles.Candles[4].Low)
//	assert.Equal(t, uint64(117), candles.Candles[4].Open)
//	assert.Equal(t, uint64(119), candles.Candles[4].Close)
//
//	assert.Equal(t, uint64(5000), candles.Candles[6].Volume)
//	assert.Equal(t, uint64(124), candles.Candles[6].High)
//	assert.Equal(t, uint64(122), candles.Candles[6].Low)
//	assert.Equal(t, uint64(122), candles.Candles[6].Open)
//	assert.Equal(t, uint64(124), candles.Candles[6].Close)
//
//	assert.Equal(t, uint64(0), candles.Candles[7].Volume)
//	assert.Equal(t, uint64(124), candles.Candles[7].High)
//	assert.Equal(t, uint64(124), candles.Candles[7].Low)
//	assert.Equal(t, uint64(124), candles.Candles[7].Open)
//	assert.Equal(t, uint64(124), candles.Candles[7].Close)
//
//	assert.Equal(t, uint64(0), candles.Candles[15].Volume)
//	assert.Equal(t, uint64(124), candles.Candles[15].High)
//	assert.Equal(t, uint64(124), candles.Candles[15].Low)
//	assert.Equal(t, uint64(124), candles.Candles[15].Open)
//	assert.Equal(t, uint64(124), candles.Candles[15].Close)
//
//	assert.Equal(t, uint64(1000), candles.Candles[16].Volume)
//	assert.Equal(t, uint64(125), candles.Candles[16].High)
//	assert.Equal(t, uint64(125), candles.Candles[16].Low)
//	assert.Equal(t, uint64(125), candles.Candles[16].Open)
//	assert.Equal(t, uint64(125), candles.Candles[16].Close)
//}

func TestOrderBookDepthBuySide(t *testing.T) {
	// test orderbook depth

	// Scenario:

	// POST few orders to datastore
	// call getMarketDepth and see if order book depth is OK

	// create impacted orders and call PUT on them
	// call getMarketDepth and see if order book depth is OK

	// call DELETE on orders
	// call getMarketDepth and see if order book depth is OK

	//var memStore = NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
	var newOrderStore = NewOrderStore("../tmp/orderstore")
	defer newOrderStore.Close()

	orders := []*msg.Order{
		&msg.Order{
			Id:     fmt.Sprintf("%d", rand.Intn(1000000000000)),
			Side: msg.Side_Buy,
			Market: testMarket,
			Party: testPartyA,
			Price: 111,
			Remaining: 100,
		},
		&msg.Order{
			Id:     fmt.Sprintf("%d", rand.Intn(1000000000000)),
			Side: msg.Side_Buy,
			Market: testMarket,
			Party: testPartyA,
			Price: 112,
			Remaining: 100,
		},
		&msg.Order{
			Id:     fmt.Sprintf("%d", rand.Intn(1000000000000)),
			Side: msg.Side_Buy,
			Market: testMarket,
			Party: testPartyA,
			Price: 112,
			Remaining: 100,
		},
		&msg.Order{
			Id:     fmt.Sprintf("%d", rand.Intn(1000000000000)),
			Side: msg.Side_Buy,
			Market: testMarket,
			Party: testPartyA,
			Price: 113,
			Remaining: 100,
		},
	}

	for idx, _ := range orders {
		newOrderStore.Post(orders[idx])
	}

	newOrderStore.Commit()

	marketDepth, _ := newOrderStore.GetMarketDepth(testMarket)

	assert.Equal(t, uint64(113), marketDepth.Buy[0].Price)
	assert.Equal(t, uint64(100), marketDepth.Buy[0].Volume)
	assert.Equal(t, uint64(1), marketDepth.Buy[0].NumberOfOrders)
	assert.Equal(t, uint64(100), marketDepth.Buy[0].CumulativeVolume)

	assert.Equal(t, uint64(112), marketDepth.Buy[1].Price)
	assert.Equal(t, uint64(200), marketDepth.Buy[1].Volume)
	assert.Equal(t, uint64(2), marketDepth.Buy[1].NumberOfOrders)
	assert.Equal(t, uint64(300), marketDepth.Buy[1].CumulativeVolume)

	assert.Equal(t, uint64(111), marketDepth.Buy[2].Price)
	assert.Equal(t, uint64(100), marketDepth.Buy[2].Volume)
	assert.Equal(t, uint64(1), marketDepth.Buy[2].NumberOfOrders)
	assert.Equal(t, uint64(400), marketDepth.Buy[2].CumulativeVolume)

	ordersUpdate := []*msg.Order{
			&msg.Order{
				Id:     orders[0].Id,
				Side: msg.Side_Buy,
				Market: testMarket,
				Party: testPartyA,
				Price: 111,
				Remaining: 50,
			},
			&msg.Order{
				Id:     orders[2].Id,
				Side: msg.Side_Buy,
				Market: testMarket,
				Party: testPartyA,
				Price: 112,
				Remaining: 50,
			},
			&msg.Order{
				Id:    orders[3].Id,
				Side: msg.Side_Buy,
				Market: testMarket,
				Party: testPartyA,
				Price: 113,
				Remaining: 80,
				Status: msg.Order_Expired,
			},
	}

	for idx, _ := range ordersUpdate {
		newOrderStore.Put(ordersUpdate[idx])
	}

	marketDepth, _ = newOrderStore.GetMarketDepth(testMarket)

	// 113 is removed

	assert.Equal(t, uint64(112), marketDepth.Buy[0].Price)
	assert.Equal(t, uint64(150), marketDepth.Buy[0].Volume)
	assert.Equal(t, uint64(2), marketDepth.Buy[0].NumberOfOrders)
	assert.Equal(t, uint64(150), marketDepth.Buy[0].CumulativeVolume)

	assert.Equal(t, uint64(111), marketDepth.Buy[1].Price)
	assert.Equal(t, uint64(50), marketDepth.Buy[1].Volume)
	assert.Equal(t, uint64(1), marketDepth.Buy[1].NumberOfOrders)
	assert.Equal(t, uint64(200), marketDepth.Buy[1].CumulativeVolume)

	ordersRemove := []*msg.Order{
			&msg.Order{
				Id:     orders[0].Id,
				Side: msg.Side_Buy,
				Market: testMarket,
				Party: testPartyA,
				Price: 111,
				Remaining: 0,
			},
			&msg.Order{
				Id:     orders[1].Id,
				Side: msg.Side_Buy,
				Market: testMarket,
				Party: testPartyA,
				Price: 112,
				Remaining: 100,
			},
	}

	for idx, _ := range ordersRemove {
		newOrderStore.Delete(ordersRemove[idx])
	}

	marketDepth, _ = newOrderStore.GetMarketDepth(testMarket)

	assert.Equal(t, uint64(112), marketDepth.Buy[0].Price)
	assert.Equal(t, uint64(50), marketDepth.Buy[0].Volume)
	assert.Equal(t, uint64(1), marketDepth.Buy[0].NumberOfOrders)
	assert.Equal(t, uint64(50), marketDepth.Buy[0].CumulativeVolume)

	assert.Equal(t, 1, len(marketDepth.Buy))
}

func TestOrderBookDepthSellSide(t *testing.T) {
	// test orderbook depth

	// Scenario:

	// POST few orders to datastore
	// call getMarketDepth and see if order book depth is OK

	// create impacted orders and call PUT on them
	// call getMarketDepth and see if order book depth is OK

	// call DELETE on orders
	// call getMarketDepth and see if order book depth is OK

	//var memStore = NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
	var newOrderStore = NewOrderStore("../tmp/orderStore")
	defer newOrderStore.Close()

	orders := []*msg.Order{
		&msg.Order{
			Id:     fmt.Sprintf("%d", rand.Intn(1000000000000)),
			Side: msg.Side_Sell,
			Market: testMarket,
			Party: testPartyA,
			Price: 111,
			Remaining: 100,
		},
		&msg.Order{
			Id:     fmt.Sprintf("%d", rand.Intn(1000000000000)),
			Side: msg.Side_Sell,
			Market: testMarket,
			Party: testPartyA,
			Price: 112,
			Remaining: 100,
		},
		&msg.Order{
			Id:     fmt.Sprintf("%d", rand.Intn(1000000000000)),
			Side: msg.Side_Sell,
			Market: testMarket,
			Party: testPartyA,
			Price: 112,
			Remaining: 100,
		},
		&msg.Order{
			Id:     fmt.Sprintf("%d", rand.Intn(1000000000000)),
			Side: msg.Side_Sell,
			Market: testMarket,
			Party: testPartyA,
			Price: 113,
			Remaining: 100,
		},
	}

	for idx, _ := range orders {
		newOrderStore.Post(orders[idx])
	}

	newOrderStore.Commit()

	marketDepth, _ := newOrderStore.GetMarketDepth(testMarket)

	assert.Equal(t, uint64(111), marketDepth.Sell[0].Price)
	assert.Equal(t, uint64(100), marketDepth.Sell[0].Volume)
	assert.Equal(t, uint64(1), marketDepth.Sell[0].NumberOfOrders)
	assert.Equal(t, uint64(100), marketDepth.Sell[0].CumulativeVolume)

	assert.Equal(t, uint64(112), marketDepth.Sell[1].Price)
	assert.Equal(t, uint64(200), marketDepth.Sell[1].Volume)
	assert.Equal(t, uint64(2), marketDepth.Sell[1].NumberOfOrders)
	assert.Equal(t, uint64(300), marketDepth.Sell[1].CumulativeVolume)

	assert.Equal(t, uint64(113), marketDepth.Sell[2].Price)
	assert.Equal(t, uint64(100), marketDepth.Sell[2].Volume)
	assert.Equal(t, uint64(1), marketDepth.Sell[2].NumberOfOrders)
	assert.Equal(t, uint64(400), marketDepth.Sell[2].CumulativeVolume)

	ordersUpdate := []*msg.Order{
		&msg.Order{
			Id:     orders[0].Id,
			Side: msg.Side_Sell,
			Market: testMarket,
			Party: testPartyA,
			Price: 111,
			Remaining: 50,
		},
		&msg.Order{
			Id:     orders[2].Id,
			Side: msg.Side_Sell,
			Market: testMarket,
			Party: testPartyA,
			Price: 112,
			Remaining: 50,
		},
		&msg.Order{
			Id:    orders[3].Id,
			Side: msg.Side_Sell,
			Market: testMarket,
			Party: testPartyA,
			Price: 113,
			Remaining: 80,
			Status: msg.Order_Expired,
		},
	}

	for idx, _ := range ordersUpdate {
		newOrderStore.Put(ordersUpdate[idx])
	}

	marketDepth, _ = newOrderStore.GetMarketDepth(testMarket)

	assert.Equal(t, uint64(111), marketDepth.Sell[0].Price)
	assert.Equal(t, uint64(50), marketDepth.Sell[0].Volume)
	assert.Equal(t, uint64(1), marketDepth.Sell[0].NumberOfOrders)
	assert.Equal(t, uint64(50), marketDepth.Sell[0].CumulativeVolume)

	assert.Equal(t, uint64(112), marketDepth.Sell[1].Price)
	assert.Equal(t, uint64(150), marketDepth.Sell[1].Volume)
	assert.Equal(t, uint64(2), marketDepth.Sell[1].NumberOfOrders)
	assert.Equal(t, uint64(200), marketDepth.Sell[1].CumulativeVolume)

	// 113 is removed
	assert.Equal(t, 2, len(marketDepth.Sell))

	ordersRemove := []*msg.Order{
		&msg.Order{
			Id:     orders[0].Id,
			Side: msg.Side_Sell,
			Market: testMarket,
			Party: testPartyA,
			Price: 111,
			Remaining: 0,
		},
		&msg.Order{
			Id:     orders[1].Id,
			Side: msg.Side_Sell,
			Market: testMarket,
			Party: testPartyA,
			Price: 112,
			Remaining: 100,
		},
	}

	for idx, _ := range ordersRemove {
		newOrderStore.Delete(ordersRemove[idx])
	}

	marketDepth, _ = newOrderStore.GetMarketDepth(testMarket)

	assert.Equal(t, uint64(112), marketDepth.Sell[0].Price)
	assert.Equal(t, uint64(50), marketDepth.Sell[0].Volume)
	assert.Equal(t, uint64(1), marketDepth.Sell[0].NumberOfOrders)
	assert.Equal(t, uint64(50), marketDepth.Sell[0].CumulativeVolume)

	assert.Equal(t, 1, len(marketDepth.Sell))
}



