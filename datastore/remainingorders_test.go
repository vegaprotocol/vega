package datastore

import (
	"testing"
	"vega/proto"

	"github.com/stretchr/testify/assert"
)

func TestBuySideRemainingOrders_Insert(t *testing.T){

	var ro BuySideRemainingOrders

	ordersList := []*Order{
		{msg.Order{Price: 110, Remaining: 100}},
		{msg.Order{Price: 111, Remaining: 100}},
		{msg.Order{Price: 113, Remaining: 100}},
		{msg.Order{Price: 114, Remaining: 100}},
		{msg.Order{Price: 116, Remaining: 100}},
	}

	for _, elem := range ordersList {
		ro.insert(elem)
	}

	assert.Equal(t, ro.orders[0].price, uint64(116))
	assert.Equal(t, ro.orders[1].price, uint64(114))
	assert.Equal(t, ro.orders[2].price, uint64(113))
	assert.Equal(t, ro.orders[3].price, uint64(111))
	assert.Equal(t, ro.orders[4].price, uint64(110))

	ordersList = []*Order{
		{msg.Order{Price: 112, Remaining: 100}},
		{msg.Order{Price: 115, Remaining: 100}},
	}

	for _, elem := range ordersList {
		ro.insert(elem)
	}

	assert.Equal(t, ro.orders[0].price, uint64(116))
	assert.Equal(t, ro.orders[1].price, uint64(115))
	assert.Equal(t, ro.orders[2].price, uint64(114))
	assert.Equal(t, ro.orders[3].price, uint64(113))
	assert.Equal(t, ro.orders[4].price, uint64(112))
	assert.Equal(t, ro.orders[5].price, uint64(111))
	assert.Equal(t, ro.orders[6].price, uint64(110))
}

func TestBuySideRemainingOrders_Remove(t *testing.T) {

	var ro BuySideRemainingOrders

	ordersList := []*Order{
		{msg.Order{Price: 110, Remaining: 100}},
		{msg.Order{Price: 111, Remaining: 100}},
		{msg.Order{Price: 112, Remaining: 100}},
		{msg.Order{Price: 113, Remaining: 100}},
		{msg.Order{Price: 114, Remaining: 100}},
	}

	for _, elem := range ordersList {
		ro.insert(elem)
	}

	ordersList = []*Order{
		{msg.Order{Price: 112, Remaining: 100}},
		{msg.Order{Price: 113, Remaining: 100}},
	}

	for _, elem := range ordersList {
		ro.remove(elem)
	}

	assert.Equal(t, ro.orders[0].price, uint64(114))
	assert.Equal(t, ro.orders[1].price, uint64(111))
	assert.Equal(t, ro.orders[2].price, uint64(110))

	ro.remove(&Order{msg.Order{Price: 112, Remaining: 100}})

}

func TestBuySideRemainingOrders_Update(t *testing.T) {

	var ro BuySideRemainingOrders

	ordersList := []*Order{
		{msg.Order{Price: 110, Remaining: 100}},
		{msg.Order{Price: 111, Remaining: 100}},
		{msg.Order{Price: 112, Remaining: 100}},
		{msg.Order{Price: 113, Remaining: 100}},
		{msg.Order{Price: 114, Remaining: 100}},
	}

	for _, elem := range ordersList {
		ro.insert(elem)
	}

	ordersList = []*Order{
		{msg.Order{Price: 112, Remaining: 200}},
		{msg.Order{Price: 113, Remaining: 200}},
	}

	for _, elem := range ordersList {
		ro.update(elem)
	}

	assert.Equal(t, ro.orders[0].price, uint64(114))
	assert.Equal(t, ro.orders[1].price, uint64(113))
	assert.Equal(t, ro.orders[2].price, uint64(112))
	assert.Equal(t, ro.orders[3].price, uint64(111))
	assert.Equal(t, ro.orders[4].price, uint64(110))

	assert.Equal(t, ro.orders[0].remaining, uint64(100))
	assert.Equal(t, ro.orders[1].remaining, uint64(200))
	assert.Equal(t, ro.orders[2].remaining, uint64(200))
	assert.Equal(t, ro.orders[3].remaining, uint64(100))
	assert.Equal(t, ro.orders[4].remaining, uint64(100))

	ro.remove(&Order{msg.Order{Price: 120, Remaining: 100}})
}



func TestSellSideRemainingOrders_Insert(t *testing.T){

	var ro SellSideRemainingOrders

	ordersList := []*Order{
		{msg.Order{Price: 110, Remaining: 100}},
		{msg.Order{Price: 111, Remaining: 100}},
		{msg.Order{Price: 113, Remaining: 100}},
		{msg.Order{Price: 114, Remaining: 100}},
		{msg.Order{Price: 116, Remaining: 100}},
	}

	for _, elem := range ordersList {
		ro.insert(elem)
	}

	assert.Equal(t, ro.orders[0].price, uint64(110))
	assert.Equal(t, ro.orders[1].price, uint64(111))
	assert.Equal(t, ro.orders[2].price, uint64(113))
	assert.Equal(t, ro.orders[3].price, uint64(114))
	assert.Equal(t, ro.orders[4].price, uint64(116))

	ordersList = []*Order{
		{msg.Order{Price: 112, Remaining: 100}},
		{msg.Order{Price: 115, Remaining: 100}},
	}

	for _, elem := range ordersList {
		ro.insert(elem)
	}

	assert.Equal(t, ro.orders[0].price, uint64(110))
	assert.Equal(t, ro.orders[1].price, uint64(111))
	assert.Equal(t, ro.orders[2].price, uint64(112))
	assert.Equal(t, ro.orders[3].price, uint64(113))
	assert.Equal(t, ro.orders[4].price, uint64(114))
	assert.Equal(t, ro.orders[5].price, uint64(115))
	assert.Equal(t, ro.orders[6].price, uint64(116))
}

func TestSellSideRemainingOrders_Remove(t *testing.T) {

	var ro SellSideRemainingOrders

	ordersList := []*Order{
		{msg.Order{Price: 110, Remaining: 100}},
		{msg.Order{Price: 111, Remaining: 100}},
		{msg.Order{Price: 112, Remaining: 100}},
		{msg.Order{Price: 113, Remaining: 100}},
		{msg.Order{Price: 114, Remaining: 100}},
	}

	for _, elem := range ordersList {
		ro.insert(elem)
	}

	ordersList = []*Order{
		{msg.Order{Price: 112, Remaining: 100}},
		{msg.Order{Price: 113, Remaining: 100}},
	}

	for _, elem := range ordersList {
		ro.remove(elem)
	}

	assert.Equal(t, ro.orders[0].price, uint64(110))
	assert.Equal(t, ro.orders[1].price, uint64(111))
	assert.Equal(t, ro.orders[2].price, uint64(114))

	ro.remove(&Order{msg.Order{Price: 112, Remaining: 100}})

}

func TestSellSideRemainingOrders_Update(t *testing.T) {

	var ro SellSideRemainingOrders

	ordersList := []*Order{
		{msg.Order{Price: 110, Remaining: 100}},
		{msg.Order{Price: 111, Remaining: 100}},
		{msg.Order{Price: 112, Remaining: 100}},
		{msg.Order{Price: 113, Remaining: 100}},
		{msg.Order{Price: 114, Remaining: 100}},
	}

	for _, elem := range ordersList {
		ro.insert(elem)
	}

	ordersList = []*Order{
		{msg.Order{Price: 112, Remaining: 200}},
		{msg.Order{Price: 113, Remaining: 200}},
	}

	for _, elem := range ordersList {
		ro.update(elem)
	}

	assert.Equal(t, ro.orders[0].price, uint64(110))
	assert.Equal(t, ro.orders[1].price, uint64(111))
	assert.Equal(t, ro.orders[2].price, uint64(112))
	assert.Equal(t, ro.orders[3].price, uint64(113))
	assert.Equal(t, ro.orders[4].price, uint64(114))

	assert.Equal(t, ro.orders[0].remaining, uint64(100))
	assert.Equal(t, ro.orders[1].remaining, uint64(100))
	assert.Equal(t, ro.orders[2].remaining, uint64(200))
	assert.Equal(t, ro.orders[3].remaining, uint64(200))
	assert.Equal(t, ro.orders[4].remaining, uint64(100))

	ro.remove(&Order{msg.Order{Price: 120, Remaining: 100}})
}