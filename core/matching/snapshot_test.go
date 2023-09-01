// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package matching_test

import (
	"bytes"
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	market = "testing market"
	key    = market
	party  = "party"
)

var priceFactor = num.DecimalFromInt64(10)

type orderdata struct {
	id          string
	price       uint64
	size        uint64
	side        types.Side
	peggedOrder *types.PeggedOrder
}

func TestEmpty(t *testing.T) {
	ob := getTestOrderBook(t, market)

	payload, _, err := ob.ob.GetState(key)
	assert.NoError(t, err)
	assert.NotNil(t, payload)

	_, _, err = ob.ob.GetState(key)
	assert.NoError(t, err)
}

func TestBuyOrdersChangeState(t *testing.T) {
	ob := getTestOrderBook(t, market)

	orders := []orderdata{
		{id: vgcrypto.RandomHash(), price: 100, size: 10, side: types.SideBuy},
		{id: vgcrypto.RandomHash(), price: 101, size: 11, side: types.SideBuy},
		{id: vgcrypto.RandomHash(), price: 102, size: 12, side: types.SideBuy},
		{id: vgcrypto.RandomHash(), price: 103, size: 13, side: types.SideBuy},
	}

	addOrders(t, ob.ob, orders)

	s1, _, err := ob.ob.GetState(key)
	assert.NoError(t, err)
	s2, _, err := ob.ob.GetState(key)
	assert.NoError(t, err)
	// These should be the same
	assert.True(t, bytes.Equal(s1, s2))

	// Add one more order and check that the state value changes
	order := &types.Order{
		MarketID:    market,
		ID:          vgcrypto.RandomHash(),
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

	s3, _, err := ob.ob.GetState(key)
	assert.NoError(t, err)
	assert.False(t, bytes.Equal(s1, s3))
}

func TestSellOrdersChangeState(t *testing.T) {
	ob := getTestOrderBook(t, market)

	orders := []orderdata{
		{id: vgcrypto.RandomHash(), price: 100, size: 10, side: types.SideSell},
		{id: vgcrypto.RandomHash(), price: 101, size: 11, side: types.SideSell},
		{id: vgcrypto.RandomHash(), price: 102, size: 12, side: types.SideSell},
		{id: vgcrypto.RandomHash(), price: 103, size: 13, side: types.SideSell},
	}
	addOrders(t, ob.ob, orders)

	s1, _, err := ob.ob.GetState(key)
	assert.NoError(t, err)
	s2, _, err := ob.ob.GetState(key)
	assert.NoError(t, err)
	// These should be the same
	assert.True(t, bytes.Equal(s1, s2))

	// Add one more order and check that the state value changes
	order := &types.Order{
		MarketID:    market,
		ID:          vgcrypto.RandomHash(),
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

	s3, _, err := ob.ob.GetState(key)
	assert.NoError(t, err)
	assert.False(t, bytes.Equal(s1, s3))
}

func addOrders(t *testing.T, ob *matching.CachedOrderBook, orders []orderdata) {
	t.Helper()
	baseorder := &types.Order{
		MarketID:    market,
		Party:       party,
		Side:        types.SideSell,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
		Status:      types.OrderStatusActive,
	}

	createdAt := int64(1000)
	for _, i := range orders {
		order := baseorder.Clone()
		order.ID = i.id
		order.Price = num.NewUint(i.price)

		order.PeggedOrder = i.peggedOrder
		order.Size = i.size

		order.Remaining = i.size
		order.Side = i.side
		order.CreatedAt = createdAt
		createdAt++

		pf, _ := num.UintFromDecimal(priceFactor)
		order.OriginalPrice = order.Price.Clone()
		order.OriginalPrice.Div(order.Price, pf)

		orderConf, err := ob.SubmitOrder(order)
		assert.NotNil(t, orderConf)
		assert.NoError(t, err)
	}
}

func TestSaveAndLoadSnapshot(t *testing.T) {
	ob := getTestOrderBook(t, market)

	// Add some orders
	orders := []orderdata{
		{id: vgcrypto.RandomHash(), price: 99, size: 10, side: types.SideBuy},
		{id: vgcrypto.RandomHash(), price: 100, size: 11, side: types.SideBuy},
		{id: vgcrypto.RandomHash(), price: 102, size: 12, side: types.SideSell},
		{id: vgcrypto.RandomHash(), price: 103, size: 13, side: types.SideSell},
	}
	addOrders(t, ob.ob, orders)

	// now add some pegged orders
	details := &types.PeggedOrder{
		Offset:    num.NewUint(100),
		Reference: types.PeggedReferenceMid,
	}
	peggedOrders := []orderdata{
		{id: vgcrypto.RandomHash(), price: 95, size: 1, side: types.SideBuy, peggedOrder: details},
		{id: vgcrypto.RandomHash(), price: 105, size: 1, side: types.SideSell, peggedOrder: details},
		{id: vgcrypto.RandomHash(), price: 95, size: 1, side: types.SideBuy, peggedOrder: details},
		{id: vgcrypto.RandomHash(), price: 105, size: 1, side: types.SideSell, peggedOrder: details},
		{id: vgcrypto.RandomHash(), price: 95, size: 1, side: types.SideBuy, peggedOrder: details},
		{id: vgcrypto.RandomHash(), price: 105, size: 1, side: types.SideSell, peggedOrder: details},
	}

	addOrders(t, ob.ob, peggedOrders)

	// Create a snapshot
	payload, _, err := ob.ob.GetState(key)
	assert.NoError(t, err)

	before, _, err := ob.ob.GetState(key)
	assert.NoError(t, err)

	orders2 := []orderdata{
		{id: vgcrypto.RandomHash(), price: 95, size: 1, side: types.SideBuy},
		{id: vgcrypto.RandomHash(), price: 105, size: 1, side: types.SideSell},
	}
	addOrders(t, ob.ob, orders2)

	different, _, err := ob.ob.GetState(key)
	assert.NoError(t, err)

	// Load the snapshot back in
	ob2 := getTestOrderBook(t, market)
	snap := &snapshot.Payload{}
	err = proto.Unmarshal(payload, snap)
	assert.NoError(t, err)
	ob2.ob.LoadState(context.TODO(), types.PayloadFromProto(snap))

	// Get the state and check it's the same as before
	after, _, err := ob2.ob.GetState(key)
	assert.NoError(t, err)
	assert.True(t, bytes.Equal(before, after))
	assert.False(t, bytes.Equal(before, different))

	for _, order := range orders {
		o2, err := ob2.ob.OrderBook.GetOrderByID(order.id)
		require.NoError(t, err)

		// all original prices should be nil until we know the price factor
		assert.Nil(t, o2.OriginalPrice)
	}

	pf, _ := num.UintFromDecimal(priceFactor)
	ob2.ob.OrderBook.RestoreWithMarketPriceFactor(pf)

	// now the orders should be equal
	for _, order := range orders {
		o1, err := ob.ob.OrderBook.GetOrderByID(order.id)
		require.NoError(t, err)

		o2, err := ob2.ob.OrderBook.GetOrderByID(order.id)
		require.NoError(t, err)
		assert.Equal(t, o1, o2)
	}

	assert.Equal(t, ob.ob.GetActivePeggedOrderIDs(), ob2.ob.GetActivePeggedOrderIDs())
}

func TestStopSnapshotTaking(t *testing.T) {
	ob := getTestOrderBook(t, market)

	_, _, err := ob.ob.GetState(key)
	assert.NoError(t, err)
	_, _, err = ob.ob.GetState(key)
	assert.NoError(t, err)

	// signal to kill the engine's snapshots
	ob.ob.StopSnapshots()

	s, _, err := ob.ob.GetState(key)
	assert.NoError(t, err)
	assert.Nil(t, s)
	assert.True(t, ob.ob.Stopped())
}
