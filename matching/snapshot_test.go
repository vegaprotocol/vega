package matching_test

import (
	"context"
	"testing"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/matching"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/stretchr/testify/assert"
)

const (
	market = "testing market"
	key    = market
	party  = "party"
)

type orderdata struct {
	id    string
	price uint64
	size  uint64
	side  types.Side
}

func TestEmpty(t *testing.T) {
	ob := getTestOrderBook(t, market)

	payload, _, err := ob.ob.GetState(key)
	assert.NoError(t, err)
	assert.NotNil(t, payload)

	hash, err := ob.ob.GetHash(key)
	assert.NoError(t, err)
	assert.Equal(t, 32, len(hash))
}

func TestBuyOrdersChangeHash(t *testing.T) {
	ob := getTestOrderBook(t, market)

	orders := []orderdata{
		{id: "id01", price: 100, size: 10, side: types.SideBuy},
		{id: "id02", price: 101, size: 11, side: types.SideBuy},
		{id: "id03", price: 102, size: 12, side: types.SideBuy},
		{id: "id04", price: 103, size: 13, side: types.SideBuy},
	}

	addOrders(t, ob.ob, orders)

	hash1, err := ob.ob.GetHash(key)
	assert.NoError(t, err)
	hash2, err := ob.ob.GetHash(key)
	assert.NoError(t, err)
	// These should be the same
	assert.Equal(t, hash1, hash2)

	// Add one more order and check that the hash value changes
	order := &types.Order{
		MarketID:    market,
		ID:          "id05",
		Price:       num.NewUint(104),
		Size:        14,
		Remaining:   14,
		Party:       party,
		Side:        types.SideBuy,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
		CreatedAt:   1010,
	}
	orderConf, err := ob.ob.SubmitOrder(order)
	assert.NotNil(t, orderConf)
	assert.NoError(t, err)

	hash3, err := ob.ob.GetHash(key)
	assert.NoError(t, err)
	assert.NotEqual(t, hash1, hash3)
}

func TestSellOrdersChangeHash(t *testing.T) {
	ob := getTestOrderBook(t, market)

	orders := []orderdata{
		{id: "id01", price: 100, size: 10, side: types.SideSell},
		{id: "id02", price: 101, size: 11, side: types.SideSell},
		{id: "id03", price: 102, size: 12, side: types.SideSell},
		{id: "id04", price: 103, size: 13, side: types.SideSell},
	}
	addOrders(t, ob.ob, orders)

	hash1, err := ob.ob.GetHash(key)
	assert.NoError(t, err)
	hash2, err := ob.ob.GetHash(key)
	assert.NoError(t, err)
	// These should be the same
	assert.Equal(t, hash1, hash2)

	// Add one more order and check that the hash value changes
	order := &types.Order{
		MarketID:    market,
		ID:          "id05",
		Price:       num.NewUint(104),
		Size:        14,
		Remaining:   14,
		Party:       party,
		Side:        types.SideSell,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
		CreatedAt:   1010,
	}
	orderConf, err := ob.ob.SubmitOrder(order)
	assert.NotNil(t, orderConf)
	assert.NoError(t, err)

	hash3, err := ob.ob.GetHash(key)
	assert.NoError(t, err)
	assert.NotEqual(t, hash1, hash3)
}

func addOrders(t *testing.T, ob *matching.CachedOrderBook, orders []orderdata) {
	t.Helper()
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
		order.Side = i.side
		order.CreatedAt = createdAt
		createdAt++

		orderConf, err := ob.SubmitOrder(order)
		assert.NotNil(t, orderConf)
		assert.NoError(t, err)
	}
}

func TestSaveAndLoadSnapshot(t *testing.T) {
	ob := getTestOrderBook(t, market)

	// Add some orders
	orders := []orderdata{
		{id: "id01", price: 99, size: 10, side: types.SideBuy},
		{id: "id02", price: 100, size: 11, side: types.SideBuy},
		{id: "id03", price: 102, size: 12, side: types.SideSell},
		{id: "id04", price: 103, size: 13, side: types.SideSell},
	}
	addOrders(t, ob.ob, orders)

	// Create a snapshot and hash
	payload, _, err := ob.ob.GetState(key)
	assert.NoError(t, err)

	beforeHash, err := ob.ob.GetHash(key)
	assert.NoError(t, err)

	orders2 := []orderdata{
		{id: "id10", price: 95, size: 1, side: types.SideBuy},
		{id: "id11", price: 105, size: 1, side: types.SideSell},
	}
	addOrders(t, ob.ob, orders2)
	differentHash, err := ob.ob.GetHash(key)
	assert.NoError(t, err)

	// Load the snapshot back in
	ob2 := getTestOrderBook(t, market)
	snap := &snapshot.Payload{}
	err = proto.Unmarshal(payload, snap)
	assert.NoError(t, err)
	ob2.ob.LoadState(context.TODO(), types.PayloadFromProto(snap))

	// Get the hash and check it's the same as before
	afterHash, err := ob2.ob.GetHash(key)
	assert.NoError(t, err)
	assert.Equal(t, beforeHash, afterHash)
	assert.NotEqual(t, beforeHash, differentHash)
}

func TestStopSnapshotTaking(t *testing.T) {
	ob := getTestOrderBook(t, market)

	_, _, err := ob.ob.GetState(key)
	assert.NoError(t, err)
	_, err = ob.ob.GetHash(key)
	assert.NoError(t, err)

	// signal to kill the engine's snapshots
	ob.ob.StopSnapshots()

	s, _, err := ob.ob.GetState(key)
	assert.NoError(t, err)
	assert.Nil(t, s)
	h, err := ob.ob.GetHash(key)
	assert.NoError(t, err)
	assert.Nil(t, h)
	assert.True(t, ob.ob.Stopped())
}
