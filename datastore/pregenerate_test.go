package datastore

import (
	"testing"
	"vega/msg"
	"github.com/stretchr/testify/assert"
	"fmt"
)

func TestOrderBookDepth_Insert(t *testing.T){

	var marketDepth MarketDepth

	ordersList := []*Order{
		{msg.Order{Side: msg.Side_Buy,Price: 110, Remaining: 100}},
		{msg.Order{Side: msg.Side_Buy,Price: 111, Remaining: 100}},
		{msg.Order{Side: msg.Side_Buy,Price: 111, Remaining: 100}},
		{msg.Order{Side: msg.Side_Buy,Price: 113, Remaining: 100}},
		{msg.Order{Side: msg.Side_Buy,Price: 114, Remaining: 100}},
		{msg.Order{Side: msg.Side_Buy,Price: 116, Remaining: 100}},
	}

	for _, elem := range ordersList {
		marketDepth.updateWithRemaining(elem)
	}

	fmt.Printf("%+v", marketDepth)
	assert.Equal(t, marketDepth.Buy[0].Price, uint64(116))
	assert.Equal(t, marketDepth.Buy[0].Volume, uint64(100))
	assert.Equal(t, marketDepth.Buy[0].NumberOfOrders, uint64(1))
	assert.Equal(t, marketDepth.Buy[0].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Buy[1].Price, uint64(114))
	assert.Equal(t, marketDepth.Buy[1].Volume, uint64(100))
	assert.Equal(t, marketDepth.Buy[1].NumberOfOrders, uint64(1))
	assert.Equal(t, marketDepth.Buy[1].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Buy[2].Price, uint64(113))
	assert.Equal(t, marketDepth.Buy[2].Volume, uint64(100))
	assert.Equal(t, marketDepth.Buy[2].NumberOfOrders, uint64(1))
	assert.Equal(t, marketDepth.Buy[2].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Buy[3].Price, uint64(111))
	assert.Equal(t, marketDepth.Buy[3].Volume, uint64(200))
	assert.Equal(t, marketDepth.Buy[3].NumberOfOrders, uint64(2))
	assert.Equal(t, marketDepth.Buy[3].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Buy[4].Price, uint64(110))
	assert.Equal(t, marketDepth.Buy[4].Volume, uint64(100))
	assert.Equal(t, marketDepth.Buy[4].NumberOfOrders, uint64(1))
	assert.Equal(t, marketDepth.Buy[4].CumulativeVolume, uint64(0))


	ordersList = []*Order{
		{msg.Order{Side: msg.Side_Buy,Price: 111, Remaining: 50}},
		{msg.Order{Side: msg.Side_Buy,Price: 114, Remaining: 100}},
		{msg.Order{Side: msg.Side_Buy,Price: 113, Remaining: 100}},
	}

	marketDepth.updateWithRemainingDelta(&Order{msg.Order{Side: msg.Side_Buy,Price: 111, Remaining: 50}}, 50)
	marketDepth.updateWithRemainingDelta(&Order{msg.Order{Side: msg.Side_Buy,Price: 114, Remaining: 80}}, 20)
	marketDepth.removeWithRemaining(&Order{msg.Order{Side: msg.Side_Buy,Price: 113, Remaining: 100}})

	fmt.Printf("%+v", marketDepth)
	assert.Equal(t, marketDepth.Buy[0].Price, uint64(116))
	assert.Equal(t, marketDepth.Buy[0].Volume, uint64(100))
	assert.Equal(t, marketDepth.Buy[0].NumberOfOrders, uint64(1))
	assert.Equal(t, marketDepth.Buy[0].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Buy[1].Price, uint64(114))
	assert.Equal(t, marketDepth.Buy[1].Volume, uint64(80))
	assert.Equal(t, marketDepth.Buy[1].NumberOfOrders, uint64(1))
	assert.Equal(t, marketDepth.Buy[1].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Buy[2].Price, uint64(111))
	assert.Equal(t, marketDepth.Buy[2].Volume, uint64(150))
	assert.Equal(t, marketDepth.Buy[2].NumberOfOrders, uint64(2))
	assert.Equal(t, marketDepth.Buy[2].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Buy[3].Price, uint64(110))
	assert.Equal(t, marketDepth.Buy[3].Volume, uint64(100))
	assert.Equal(t, marketDepth.Buy[3].NumberOfOrders, uint64(1))
	assert.Equal(t, marketDepth.Buy[3].CumulativeVolume, uint64(0))



	// test sell side


}
