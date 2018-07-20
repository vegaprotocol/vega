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

func TestPositions(t *testing.T) {
	var memStore = NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
	var newOrderStore = NewOrderStore(&memStore)
	var newTradeStore = NewTradeStore(&memStore)

	timestamp := populateStores(t, newOrderStore, newTradeStore)
	fmt.Printf("timestamp %d\n", timestamp)

	trades, _ := newTradeStore.GetByParty(testPartyA, GetParams{})
	fmt.Printf("stuff %d\n", len(trades))

	positions := newTradeStore.GetNetPositionsByParty(testPartyA)

	fmt.Printf("positions returned:\n")
	for key, val := range positions {
		fmt.Printf("%+v %d\n", key, val)
	}

	// close position

	passiveOrderId := fmt.Sprintf("%d", rand.Intn(1000000000000))
	aggressiveOrderId := fmt.Sprintf("%d", rand.Intn(1000000000000))
	tradeId := fmt.Sprintf("%d", rand.Intn(1000000000000))
	d := &TestOrdersAndTrades{
		&Order{
			Order: msg.Order{
				Id:     aggressiveOrderId,
				Market: testMarket,
				Party:  testPartyA,
			},
		},
		&Order{
			Order: msg.Order{
				Id:     passiveOrderId,
				Market: testMarket,
				Party:  testPartyB,
			},
		},
		&Trade{
			Trade: msg.Trade{
				Id:        tradeId,
				Price:     118,
				Market:    testMarket,
				Size:      1000,
				Timestamp: timestamp,
				Buyer:     testPartyB,
				Seller:    testPartyA,
			},
			PassiveOrderId:    passiveOrderId,
			AggressiveOrderId: aggressiveOrderId,
		},
	}
	err := newOrderStore.Post(*d.passiveOrder)
	assert.Nil(t, err)
	err = newOrderStore.Post(*d.aggressiveOrder)
	assert.Nil(t, err)
	err = newTradeStore.Post(*d.trade)
	assert.Nil(t, err)

	positions = newTradeStore.GetNetPositionsByParty(testPartyA)

	fmt.Printf("positions returned:\n")
	for key, val := range positions {
		fmt.Printf("%+v %d\n", key, val)
	}
}
