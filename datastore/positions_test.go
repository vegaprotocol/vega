package datastore

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		assert.Nil(t, err)
	}
	return timestamp
}

func TestPositions(t *testing.T) {
		var memStore = NewMemStore([]string{testMarket}, []string{testParty, testPartyA, testPartyB})
		var newOrderStore = NewOrderStore(&memStore)
		var newTradeStore = NewTradeStore(&memStore)

		timestamp := populateStores(t, &newOrderStore, &newTradeStore)
		fmt.Printf("timestamp %d\n", timestamp)

		trades, _ := newTradeStore.GetByParty(testPartyA, GetParams{})
		fmt.Printf("stuff %d\n", len(trades))

		positions := newTradeStore.CalculateNetPositions(testPartyA)

		fmt.Printf("positions returned:\n")
		for key, val := range positions.positions {
			fmt.Printf("%+v %d\n", key, val)
		}
}