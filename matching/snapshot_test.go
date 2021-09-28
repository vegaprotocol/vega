package matching_test

import (
	"testing"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/assert"
)

const (
	key    = "matching engine"
	market = "testing market"
	party  = "party"
)

func TestEmpty(t *testing.T) {
	ob := getTestOrderBook(t, market)

	bytes, err := ob.ob.Snapshot()
	assert.NoError(t, err)
	hash1 := crypto.Hash(bytes[key])
	hash2, err := ob.ob.GetHash(key)
	assert.Equal(t, hash1, hash2)
}

func TestBuyOrdersChangeHash(t *testing.T) {
	ob := getTestOrderBook(t, market)

	type orderdata struct {
		id    string
		price uint64
		size  uint64
	}

	orders := []orderdata{
		{id: "id01", price: 100, size: 10},
		{id: "id02", price: 101, size: 11},
		{id: "id03", price: 102, size: 12},
		{id: "id04", price: 103, size: 13},
	}

	baseorder := &types.Order{
		MarketID:    market,
		Party:       party,
		Side:        types.SideBuy,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}

	createdAt := int64(1000)
	for _, i := range orders {
		order := baseorder.Clone()
		order.ID = i.id
		order.Price = num.NewUint(i.price)
		order.Size = i.size
		order.Remaining = i.size
		order.CreatedAt = createdAt
		createdAt++

		orderConf, err := ob.ob.SubmitOrder(order)
		assert.NotNil(t, orderConf)
		assert.NoError(t, err)
	}

	hash1, err := ob.ob.GetHash(key)
	hash2, err := ob.ob.GetHash(key)
	// These should be the same
	assert.Equal(t, hash1, hash2)

	// Add one more order and check that the hash value changes
	baseorder.ID = "id05"
	baseorder.Price = num.NewUint(104)
	baseorder.Size = 14
	baseorder.Remaining = 14
	baseorder.CreatedAt = createdAt

	orderConf, err := ob.ob.SubmitOrder(baseorder)
	assert.NotNil(t, orderConf)
	assert.NoError(t, err)

	hash3, err := ob.ob.GetHash(key)
	assert.NotEqual(t, hash1, hash3)
}

func TestSellOrdersChangeHash(t *testing.T) {
	ob := getTestOrderBook(t, market)

	type orderdata struct {
		id    string
		price uint64
		size  uint64
	}

	orders := []orderdata{
		{id: "id01", price: 100, size: 10},
		{id: "id02", price: 101, size: 11},
		{id: "id03", price: 102, size: 12},
		{id: "id04", price: 103, size: 13},
	}

	baseorder := &types.Order{
		MarketID:    market,
		Party:       party,
		Side:        types.SideSell,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
	}

	createdAt := int64(1000)
	for _, i := range orders {
		order := baseorder.Clone()
		order.ID = i.id
		order.Price = num.NewUint(i.price)
		order.Size = i.size
		order.Remaining = i.size
		order.CreatedAt = createdAt
		createdAt++

		orderConf, err := ob.ob.SubmitOrder(order)
		assert.NotNil(t, orderConf)
		assert.NoError(t, err)
	}

	hash1, err := ob.ob.GetHash(key)
	hash2, err := ob.ob.GetHash(key)
	// These should be the same
	assert.Equal(t, hash1, hash2)

	// Add one more order and check that the hash value changes
	baseorder.ID = "id05"
	baseorder.Price = num.NewUint(104)
	baseorder.Size = 14
	baseorder.Remaining = 14
	baseorder.CreatedAt = createdAt

	orderConf, err := ob.ob.SubmitOrder(baseorder)
	assert.NotNil(t, orderConf)
	assert.NoError(t, err)

	hash3, err := ob.ob.GetHash(key)
	assert.NotEqual(t, hash1, hash3)
}
