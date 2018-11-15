package datastore

import (
	"testing"
	"vega/msg"
	"github.com/stretchr/testify/assert"
)

func TestOrderBookDepth_All(t *testing.T){

	var marketDepth MarketDepth

	ordersList := []*msg.Order{
		{Side: msg.Side_Buy,Price: 116, Remaining: 100},
		{Side: msg.Side_Buy,Price: 110, Remaining: 100},
		{Side: msg.Side_Buy,Price: 111, Remaining: 100},
		{Side: msg.Side_Buy,Price: 111, Remaining: 100},
		{Side: msg.Side_Buy,Price: 113, Remaining: 100},
		{Side: msg.Side_Buy,Price: 114, Remaining: 100},
		{Side: msg.Side_Buy,Price: 116, Remaining: 100},
	}

	for _, elem := range ordersList {
		marketDepth.updateWithRemaining(elem)
	}

	assert.Equal(t, marketDepth.Buy[0].Price, uint64(116))
	assert.Equal(t, marketDepth.Buy[0].Volume, uint64(200))
	assert.Equal(t, marketDepth.Buy[0].NumberOfOrders, uint64(2))
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


	marketDepth.updateWithRemainingDelta(&msg.Order{Side: msg.Side_Buy,Price: 111, Remaining: 50}, 50)
	marketDepth.updateWithRemainingDelta(&msg.Order{Side: msg.Side_Buy,Price: 114, Remaining: 80}, 20)
	marketDepth.removeWithRemaining(&msg.Order{Side: msg.Side_Buy,Price: 113, Remaining: 100})

	assert.Equal(t, marketDepth.Buy[0].Price, uint64(116))
	assert.Equal(t, marketDepth.Buy[0].Volume, uint64(200))
	assert.Equal(t, marketDepth.Buy[0].NumberOfOrders, uint64(2))
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

	ordersList = []*msg.Order{
		{Side: msg.Side_Sell,Price: 123, Remaining: 100},
		{Side: msg.Side_Sell,Price: 119, Remaining: 100},
		{Side: msg.Side_Sell,Price: 120, Remaining: 100},
		{Side: msg.Side_Sell,Price: 120, Remaining: 100},
		{Side: msg.Side_Sell,Price: 121, Remaining: 100},
		{Side: msg.Side_Sell,Price: 121, Remaining: 100},
		{Side: msg.Side_Sell,Price: 122, Remaining: 100},
		{Side: msg.Side_Sell,Price: 123, Remaining: 100},
	}

	for _, elem := range ordersList {
		marketDepth.updateWithRemaining(elem)
	}

	assert.Equal(t, marketDepth.Sell[0].Price, uint64(119))
	assert.Equal(t, marketDepth.Sell[0].Volume, uint64(100))
	assert.Equal(t, marketDepth.Sell[0].NumberOfOrders, uint64(1))
	assert.Equal(t, marketDepth.Sell[0].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Sell[1].Price, uint64(120))
	assert.Equal(t, marketDepth.Sell[1].Volume, uint64(200))
	assert.Equal(t, marketDepth.Sell[1].NumberOfOrders, uint64(2))
	assert.Equal(t, marketDepth.Sell[1].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Sell[2].Price, uint64(121))
	assert.Equal(t, marketDepth.Sell[2].Volume, uint64(200))
	assert.Equal(t, marketDepth.Sell[2].NumberOfOrders, uint64(2))
	assert.Equal(t, marketDepth.Sell[2].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Sell[3].Price, uint64(122))
	assert.Equal(t, marketDepth.Sell[3].Volume, uint64(100))
	assert.Equal(t, marketDepth.Sell[3].NumberOfOrders, uint64(1))
	assert.Equal(t, marketDepth.Sell[3].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Sell[4].Price, uint64(123))
	assert.Equal(t, marketDepth.Sell[4].Volume, uint64(200))
	assert.Equal(t, marketDepth.Sell[4].NumberOfOrders, uint64(2))
	assert.Equal(t, marketDepth.Sell[4].CumulativeVolume, uint64(0))

	marketDepth.updateWithRemainingDelta(&msg.Order{Side: msg.Side_Sell,Price: 119, Remaining: 100}, 50)
	marketDepth.updateWithRemainingDelta(&msg.Order{Side: msg.Side_Sell,Price: 120, Remaining: 100}, 20)
	marketDepth.removeWithRemaining(&msg.Order{Side: msg.Side_Sell,Price: 122, Remaining: 100})

	assert.Equal(t, marketDepth.Sell[0].Price, uint64(119))
	assert.Equal(t, marketDepth.Sell[0].Volume, uint64(50))
	assert.Equal(t, marketDepth.Sell[0].NumberOfOrders, uint64(1))
	assert.Equal(t, marketDepth.Sell[0].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Sell[1].Price, uint64(120))
	assert.Equal(t, marketDepth.Sell[1].Volume, uint64(180))
	assert.Equal(t, marketDepth.Sell[1].NumberOfOrders, uint64(2))
	assert.Equal(t, marketDepth.Sell[1].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Sell[2].Price, uint64(121))
	assert.Equal(t, marketDepth.Sell[2].Volume, uint64(200))
	assert.Equal(t, marketDepth.Sell[2].NumberOfOrders, uint64(2))
	assert.Equal(t, marketDepth.Sell[2].CumulativeVolume, uint64(0))

	assert.Equal(t, marketDepth.Sell[3].Price, uint64(123))
	assert.Equal(t, marketDepth.Sell[3].Volume, uint64(200))
	assert.Equal(t, marketDepth.Sell[3].NumberOfOrders, uint64(2))
	assert.Equal(t, marketDepth.Sell[3].CumulativeVolume, uint64(0))
}


