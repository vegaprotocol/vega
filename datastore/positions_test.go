package datastore

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"vega/proto"
)

type TestOrdersAndTrades struct {
	aggressiveOrder *Order
	passiveOrder    *Order
	trade           *Trade
}

func generateRandomOrdersAndTrade(price, size, timestamp uint64, passiveParty, aggressiveParty string) *TestOrdersAndTrades {
	passiveOrderId := fmt.Sprintf("%d", rand.Intn(1000000000000))
	aggressiveOrderId := fmt.Sprintf("%d", rand.Intn(1000000000000))
	tradeId := fmt.Sprintf("%d", rand.Intn(1000000000000))
	d := &TestOrdersAndTrades{
		&Order{
			Order: msg.Order{
				Id:     aggressiveOrderId,
				Market: testMarket,
				Party:  aggressiveParty,
			},
		},
		&Order{
			Order: msg.Order{
				Id:     passiveOrderId,
				Market: testMarket,
				Party:  passiveParty,
			},
		},
		&Trade{
			Trade: msg.Trade{
				Id:        tradeId,
				Price:     price,
				Market:    testMarket,
				Size:      size,
				Timestamp: timestamp,
				Buyer:     aggressiveParty,
				Seller:    passiveParty,
			},
			PassiveOrderId:    passiveOrderId,
			AggressiveOrderId: aggressiveOrderId,
		},
	}
	return d
}

func populateStores(t *testing.T, orderStore OrderStore, tradeStore TradeStore) uint64 {
	price := uint64(100)
	timestamp := uint64(0)
	var passiveParty, aggressiveParty string
	for i := 0; i < 100; i++ {
		if i%3 == 0 {
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

		if i%2 == 0 {
			passiveParty = testPartyA
			aggressiveParty = testPartyB
		} else {
			passiveParty = testPartyB
			aggressiveParty = testPartyA
		}

		d := generateRandomOrdersAndTrade(price, size, timestamp, passiveParty, aggressiveParty)

		err := orderStore.Post(*d.passiveOrder)
		assert.Nil(t, err)
		err = orderStore.Post(*d.aggressiveOrder)
		assert.Nil(t, err)
		err = tradeStore.Post(*d.trade)
		assert.Nil(t, err)
	}
	return timestamp
}

//func TestPositions(t *testing.T) {
//	var memStore = NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
//	var newOrderStore = NewOrderStore(&memStore)
//	var newTradeStore = NewTradeStore(&memStore)
//
//	timestamp := populateStores(t, newOrderStore, newTradeStore)
//
//	positions := newTradeStore.GetPositionsByParty(testPartyA)
//
//	fmt.Printf("positions returned for partyA:\n")
//	for key, val := range positions {
//		fmt.Printf("%+v %v\n", key, val)
//		assert.Equal(t, key, testMarket)
//		assert.Equal(t, val.Market, testMarket)
//
//		assert.Equal(t, int64(118000), val.RealisedVolume)
//		assert.Equal(t, int64(1000), val.RealisedPNL)
//		assert.Equal(t, int64(0), val.UnrealisedVolume)
//		assert.Equal(t, int64(0), val.UnrealisedPNL)
//	}
//
//	positions = newTradeStore.GetPositionsByParty(testPartyB)
//
//	fmt.Printf("positions returned for partyB:\n")
//	for key, val := range positions {
//		fmt.Printf("%+v %v\n", key, val)
//		assert.Equal(t, key, testMarket)
//		assert.Equal(t, val.Market, testMarket)
//
//		assert.Equal(t, int64(-1000), val.RealisedVolume)
//		assert.Equal(t, int64(-1000), val.RealisedPNL)
//		assert.Equal(t, int64(1000), val.UnrealisedVolume)
//		assert.Equal(t, int64(118000), val.UnrealisedPNL)
//	}
//
//	// close position
//	passiveOrderId := fmt.Sprintf("%d", rand.Intn(1000000000000))
//	aggressiveOrderId := fmt.Sprintf("%d", rand.Intn(1000000000000))
//	tradeId := fmt.Sprintf("%d", rand.Intn(1000000000000))
//	d := &TestOrdersAndTrades{
//		&Order{
//			Order: msg.Order{
//				Id:     aggressiveOrderId,
//				Market: testMarket,
//				Party:  testPartyA,
//			},
//		},
//		&Order{
//			Order: msg.Order{
//				Id:     passiveOrderId,
//				Market: testMarket,
//				Party:  testPartyB,
//			},
//		},
//		&Trade{
//			Trade: msg.Trade{
//				Id:        tradeId,
//				Price:     118,
//				Market:    testMarket,
//				Size:      1000,
//				Timestamp: timestamp,
//				Buyer:     testPartyB,
//				Seller:    testPartyA,
//			},
//			PassiveOrderId:    passiveOrderId,
//			AggressiveOrderId: aggressiveOrderId,
//		},
//	}
//	err := newOrderStore.Post(*d.passiveOrder)
//	assert.Nil(t, err)
//	err = newOrderStore.Post(*d.aggressiveOrder)
//	assert.Nil(t, err)
//	err = newTradeStore.Post(*d.trade)
//	assert.Nil(t, err)
//
//	positions = newTradeStore.GetPositionsByParty(testPartyA)
//
//	fmt.Printf("positions returned:\n")
//	for key, val := range positions {
//		fmt.Printf("%+v %v\n", key, val)
//		assert.Equal(t, key, testMarket)
//		assert.Equal(t, val.Market, testMarket)
//
//		assert.Equal(t, int64(1000), val.RealisedVolume)
//		assert.Equal(t, int64(1000), val.RealisedPNL)
//		assert.Equal(t, int64(0), val.UnrealisedVolume)
//		assert.Equal(t, int64(0), val.UnrealisedPNL)
//	}
//}

func TestPositions1(t *testing.T) {
	var memStore = NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
	var newOrderStore = NewOrderStore(&memStore)
	var newTradeStore = NewTradeStore(&memStore)

	passiveOrderId := fmt.Sprintf("%d", rand.Intn(1000000000000))
	aggressiveOrderId := fmt.Sprintf("%d", rand.Intn(1000000000000))
	tradeId := fmt.Sprintf("%d", rand.Intn(1000000000000))

	aggressiveOrder := &Order{
		Order: msg.Order{
			Id:     aggressiveOrderId,
			Market: testMarket,
			Party:  testPartyA,
			Side:   msg.Side_Buy,
		},
	}
	passiveOrder := &Order{
		Order: msg.Order{
			Id:     passiveOrderId,
			Market: testMarket,
			Party:  testPartyB,
			Side:   msg.Side_Sell,
		},
	}

	trade := &Trade{
		Trade: msg.Trade{
			Id:        tradeId,
			Price:     100,
			Market:    testMarket,
			Size:      500,
			Timestamp: 0,
			Buyer:     testPartyA,
			Seller:    testPartyB,
			Aggressor: msg.Side_Buy,
		},
		PassiveOrderId:    passiveOrderId,
		AggressiveOrderId: aggressiveOrderId,
	}

	err := newOrderStore.Post(*passiveOrder)
	assert.Nil(t, err)
	err = newOrderStore.Post(*aggressiveOrder)
	assert.Nil(t, err)
	err = newTradeStore.Post(*trade)
	assert.Nil(t, err)

	passiveOrderId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	aggressiveOrderId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	tradeId = fmt.Sprintf("%d", rand.Intn(1000000000000))

	aggressiveOrder = &Order{
		Order: msg.Order{
			Id:     aggressiveOrderId,
			Market: testMarket,
			Party:  testPartyA,
			Side:   msg.Side_Buy,
		},
	}
	passiveOrder = &Order{
		Order: msg.Order{
			Id:     passiveOrderId,
			Market: testMarket,
			Party:  testPartyB,
			Side:   msg.Side_Sell,
		},
	}

	trade = &Trade{
		Trade: msg.Trade{
			Id:        tradeId,
			Price:     100,
			Market:    testMarket,
			Size:      500,
			Timestamp: 0,
			Buyer:     testPartyA,
			Seller:    testPartyB,
			Aggressor: msg.Side_Buy,
		},
		PassiveOrderId:    passiveOrderId,
		AggressiveOrderId: aggressiveOrderId,
	}

	err = newOrderStore.Post(*passiveOrder)
	assert.Nil(t, err)
	err = newOrderStore.Post(*aggressiveOrder)
	assert.Nil(t, err)
	err = newTradeStore.Post(*trade)
	assert.Nil(t, err)


	// two trades of 500 contracts done at the same price of 100

	positions := newTradeStore.GetPositionsByParty(testPartyA)

	fmt.Printf("positions returned:\n")
	for key, val := range positions {
		fmt.Printf("%+v\n", val)
		assert.Equal(t, testMarket, key)
		assert.Equal(t, testMarket, val.Market)

		assert.Equal(t, int64(0), val.RealisedVolume)
		assert.Equal(t, int64(0), val.RealisedPNL)
		assert.Equal(t, int64(1000), val.UnrealisedVolume)
		assert.Equal(t, int64(0), val.UnrealisedPNL)
	}

	positions = newTradeStore.GetPositionsByParty(testPartyB)

	fmt.Printf("positions returned:\n")
	for key, val := range positions {
		fmt.Printf("%+v\n", val)
		assert.Equal(t, testMarket, key)
		assert.Equal(t, testMarket, val.Market)

		assert.Equal(t, int64(0), val.RealisedVolume)
		assert.Equal(t, int64(0), val.RealisedPNL)
		assert.Equal(t, int64(-1000), val.UnrealisedVolume)
		assert.Equal(t, int64(0), val.UnrealisedPNL)
	}

	// market moves by 10 up what is the PNL?
	passiveOrderId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	aggressiveOrderId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	tradeId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	aggressiveOrder = &Order{
		Order: msg.Order{
			Id:     aggressiveOrderId,
			Market: testMarket,
			Party:  "partyC",
			Side:   msg.Side_Buy,
		},
	}
	passiveOrder = &Order{
		Order: msg.Order{
			Id:     passiveOrderId,
			Market: testMarket,
			Party:  "partyD",
			Side:   msg.Side_Sell,
		},
	}

	trade = &Trade{
		Trade: msg.Trade{
			Id:        tradeId,
			Price:     110,
			Market:    testMarket,
			Size:      1,
			Timestamp: 0,
			Buyer:     "partyC",
			Seller:    "partyD",
			Aggressor: msg.Side_Buy,
		},
		PassiveOrderId:    passiveOrderId,
		AggressiveOrderId: aggressiveOrderId,
	}

	err = newOrderStore.Post(*passiveOrder)
	assert.Nil(t, err)
	err = newOrderStore.Post(*aggressiveOrder)
	assert.Nil(t, err)
	err = newTradeStore.Post(*trade)
	assert.Nil(t, err)

	// current mark price for testMarket should be 110, the PNL for partyA and partyB should change
	markPrice, err := newTradeStore.GetMarkPrice(testMarket)
	assert.Equal(t, uint64(110), markPrice)
	assert.Nil(t, err)

	positions = newTradeStore.GetPositionsByParty(testPartyA)

	fmt.Printf("positions returned:\n")
	for key, val := range positions {
		fmt.Printf("%+v\n", val)
		assert.Equal(t, testMarket, key)
		assert.Equal(t, testMarket, val.Market)

		assert.Equal(t, int64(0), val.RealisedVolume)
		assert.Equal(t, int64(0), val.RealisedPNL)
		assert.Equal(t, int64(1000), val.UnrealisedVolume)
		assert.Equal(t, int64(10*1000), val.UnrealisedPNL)
	}

	positions = newTradeStore.GetPositionsByParty(testPartyB)

	fmt.Printf("positions returned:\n")
	for key, val := range positions {
		fmt.Printf("%+v\n", val)
		assert.Equal(t, testMarket, key)
		assert.Equal(t, testMarket, val.Market)

		assert.Equal(t, int64(0), val.RealisedVolume)
		assert.Equal(t, int64(0), val.RealisedPNL)
		assert.Equal(t, int64(-1000), val.UnrealisedVolume)
		assert.Equal(t, int64(-10*1000), val.UnrealisedPNL)
	}


	// close 90% of position at 110
	passiveOrderId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	aggressiveOrderId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	tradeId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	aggressiveOrder = &Order{
		Order: msg.Order{
			Id:     aggressiveOrderId,
			Market: testMarket,
			Party:  testPartyA,
			Side:   msg.Side_Sell,
		},
	}
	passiveOrder = &Order{
		Order: msg.Order{
			Id:     passiveOrderId,
			Market: testMarket,
			Party:  testPartyB,
			Side:   msg.Side_Buy,
		},
	}

	trade = &Trade{
		Trade: msg.Trade{
			Id:        tradeId,
			Price:     110,
			Market:    testMarket,
			Size:      900,
			Timestamp: 0,
			Buyer:     testPartyB,
			Seller:    testPartyA,
			Aggressor: msg.Side_Sell,
		},
		PassiveOrderId:    passiveOrderId,
		AggressiveOrderId: aggressiveOrderId,
	}

	err = newOrderStore.Post(*passiveOrder)
	assert.Nil(t, err)
	err = newOrderStore.Post(*aggressiveOrder)
	assert.Nil(t, err)
	err = newTradeStore.Post(*trade)
	assert.Nil(t, err)

	positions = newTradeStore.GetPositionsByParty(testPartyA)
	fmt.Printf("positions returned:\n")
	for key, val := range positions {
		fmt.Printf("%+v\n", val)
		assert.Equal(t, testMarket, key)
		assert.Equal(t, testMarket, val.Market)

		assert.Equal(t, int64(900), val.RealisedVolume)
		assert.Equal(t, int64(9000), val.RealisedPNL)
		assert.Equal(t, int64(100), val.UnrealisedVolume)
		assert.Equal(t, int64(10*100), val.UnrealisedPNL)
	}

	positions = newTradeStore.GetPositionsByParty(testPartyB)

	fmt.Printf("positions returned:\n")
	for key, val := range positions {
		fmt.Printf("%+v\n", val)
		assert.Equal(t, testMarket, key)
		assert.Equal(t, testMarket, val.Market)

		assert.Equal(t, int64(900), val.RealisedVolume)
		assert.Equal(t, int64(-9000), val.RealisedPNL)
		assert.Equal(t, int64(-100), val.UnrealisedVolume)
		assert.Equal(t, int64(-10*100), val.UnrealisedPNL)
	}

	// close remaining 10% of position at 110
	passiveOrderId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	aggressiveOrderId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	tradeId = fmt.Sprintf("%d", rand.Intn(1000000000000))
	aggressiveOrder = &Order{
		Order: msg.Order{
			Id:     aggressiveOrderId,
			Market: testMarket,
			Party:  testPartyA,
			Side:   msg.Side_Sell,
		},
	}
	passiveOrder = &Order{
		Order: msg.Order{
			Id:     passiveOrderId,
			Market: testMarket,
			Party:  testPartyB,
			Side:   msg.Side_Buy,
		},
	}

	trade = &Trade{
		Trade: msg.Trade{
			Id:        tradeId,
			Price:     110,
			Market:    testMarket,
			Size:      100,
			Timestamp: 0,
			Buyer:     testPartyB,
			Seller:    testPartyA,
			Aggressor: msg.Side_Sell,
		},
		PassiveOrderId:    passiveOrderId,
		AggressiveOrderId: aggressiveOrderId,
	}

	err = newOrderStore.Post(*passiveOrder)
	assert.Nil(t, err)
	err = newOrderStore.Post(*aggressiveOrder)
	assert.Nil(t, err)
	err = newTradeStore.Post(*trade)
	assert.Nil(t, err)

	positions = newTradeStore.GetPositionsByParty(testPartyA)
	fmt.Printf("positions returned:\n")
	for key, val := range positions {
		fmt.Printf("%+v\n", val)
		assert.Equal(t, testMarket, key)
		assert.Equal(t, testMarket, val.Market)

		assert.Equal(t, int64(1000), val.RealisedVolume)
		assert.Equal(t, int64(10000), val.RealisedPNL)
		assert.Equal(t, int64(0), val.UnrealisedVolume)
		assert.Equal(t, int64(0), val.UnrealisedPNL)
	}

	positions = newTradeStore.GetPositionsByParty(testPartyB)

	fmt.Printf("positions returned:\n")
	for key, val := range positions {
		fmt.Printf("%+v\n", val)
		assert.Equal(t, testMarket, key)
		assert.Equal(t, testMarket, val.Market)

		assert.Equal(t, int64(1000), val.RealisedVolume)
		assert.Equal(t, int64(-10000), val.RealisedPNL)
		assert.Equal(t, int64(0), val.UnrealisedVolume)
		assert.Equal(t, int64(0), val.UnrealisedPNL)
	}

}
