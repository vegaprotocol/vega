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

package positions_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/positions"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fillTestPositions(e *positions.SnapshotEngine) {
	orders := []*types.Order{
		{
			Party:     "test_party_1",
			Side:      types.SideBuy,
			Size:      uint64(100),
			Remaining: uint64(100),
			Price:     num.UintZero(),
		},
		{
			Party:     "test_party_2",
			Side:      types.SideBuy,
			Size:      uint64(200),
			Remaining: uint64(200),
			Price:     num.UintZero(),
		},
	}

	matchingPrice := num.NewUint(10000)
	tradeSize := uint64(15)
	passiveOrder := &types.Order{
		ID:        "buy_order_id",
		Party:     "test_party_3",
		Side:      types.SideBuy,
		Size:      tradeSize,
		Remaining: tradeSize,
		Price:     matchingPrice,
	}

	aggresiveOrder := &types.Order{
		ID:        "sell_order_id",
		Party:     "test_party_1",
		Side:      types.SideSell,
		Size:      tradeSize,
		Remaining: tradeSize,
		Price:     matchingPrice,
	}

	orders = append(orders, passiveOrder, aggresiveOrder)

	for _, order := range orders {
		e.RegisterOrder(context.TODO(), order)
	}

	trade := types.Trade{
		Type:      types.TradeTypeDefault,
		ID:        "trade_id",
		MarketID:  "market_id",
		Price:     matchingPrice,
		Size:      tradeSize,
		Buyer:     passiveOrder.Party,
		Seller:    aggresiveOrder.Party,
		BuyOrder:  passiveOrder.ID,
		SellOrder: aggresiveOrder.ID,
		Timestamp: time.Now().Unix(),
	}
	e.Update(context.Background(), &trade, passiveOrder, aggresiveOrder)
}

func TestSnapshotSaveAndLoad(t *testing.T) {
	engine := getTestEngine(t)
	fillTestPositions(engine)

	keys := engine.Keys()
	require.Equal(t, 1, len(keys))
	require.Equal(t, "test_market", keys[0])

	s1, _, err := engine.GetState(keys[0])
	require.Nil(t, err)
	s2, _, err := engine.GetState(keys[0])
	require.Nil(t, err)

	// With no change the states are equal
	require.True(t, bytes.Equal(s1, s2))

	data, _, err := engine.GetState(keys[0])
	require.Nil(t, err)

	snap := &snapshot.Payload{}
	err = proto.Unmarshal(data, snap)
	require.Nil(t, err)

	snapEngine := getTestEngine(t)
	_, err = snapEngine.LoadState(
		context.TODO(),
		types.PayloadFromProto(snap),
	)
	require.Nil(t, err)

	// Get state again
	s3, _, err := snapEngine.GetState(keys[0])
	require.Nil(t, err)
	require.True(t, bytes.Equal(s1, s3))
	require.Equal(t, len(engine.Positions()), len(snapEngine.Positions()))
	for _, p := range engine.Positions() {
		// find it in the other engine by partyID
		pos, found := snapEngine.GetPositionByPartyID(p.Party())
		require.True(t, found)
		require.Equal(t, p, pos)
	}
}

func TestSnapshotStateNoChanges(t *testing.T) {
	engine := getTestEngine(t)
	fillTestPositions(engine)

	keys := engine.Keys()
	s1, _, err := engine.GetState(keys[0])
	require.Nil(t, err)
	s2, _, err := engine.GetState(keys[0])
	require.Nil(t, err)

	// With no changes we expect the states are equal
	require.True(t, bytes.Equal(s1, s2))
}

func TestSnapshotStateRegisterOrder(t *testing.T) {
	engine := getTestEngine(t)
	fillTestPositions(engine)

	keys := engine.Keys()
	s1, _, err := engine.GetState(keys[0])
	require.Nil(t, err)

	// Add and order and the state should change
	newOrder := &types.Order{
		Party:     "test_party_1",
		Side:      types.SideBuy,
		Size:      uint64(150),
		Remaining: uint64(150),
		Price:     num.UintZero(),
	}
	engine.RegisterOrder(context.TODO(), newOrder)
	s2, _, err := engine.GetState(keys[0])
	require.Nil(t, err)
	require.False(t, bytes.Equal(s1, s2))
}

func TestSnapshotStateUnregisterOrder(t *testing.T) {
	engine := getTestEngine(t)
	fillTestPositions(engine)

	keys := engine.Keys()
	s1, _, err := engine.GetState(keys[0])
	require.Nil(t, err)

	// Add and order and the state should change
	newOrder := &types.Order{
		Party:     "test_party_1",
		Side:      types.SideBuy,
		Size:      uint64(10),
		Remaining: uint64(10),
		Price:     num.UintZero(),
	}
	engine.RegisterOrder(context.TODO(), newOrder)
	s2, _, err := engine.GetState(keys[0])
	require.Nil(t, err)
	require.False(t, bytes.Equal(s1, s2))
}

func TestSnapshotStateAmendOrder(t *testing.T) {
	engine := getTestEngine(t)
	fillTestPositions(engine)

	// Add and order and the state should change
	newOrders := []*types.Order{
		{
			Party:     "test_party_1",
			Side:      types.SideBuy,
			Size:      uint64(100),
			Remaining: uint64(100),
			Price:     num.UintZero(),
		},
		{
			Party:     "test_party_1",
			Side:      types.SideBuy,
			Size:      uint64(90),
			Remaining: uint64(90),
			Price:     num.UintZero(),
		},
	}
	engine.RegisterOrder(context.TODO(), newOrders[0])
	keys := engine.Keys()
	s1, _, err := engine.GetState(keys[0])
	require.Nil(t, err)

	// Amend it
	engine.AmendOrder(context.TODO(), newOrders[0], newOrders[1])
	s2, _, err := engine.GetState(keys[0])
	require.Nil(t, err)
	require.False(t, bytes.Equal(s1, s2))

	// Then amend it back, state should be the same as originally
	engine.AmendOrder(context.TODO(), newOrders[1], newOrders[0])
	s2, _, err = engine.GetState(keys[0])
	require.Nil(t, err)
	require.True(t, bytes.Equal(s1, s2))
}

func TestSnapshotStateRemoveDistressed(t *testing.T) {
	engine := getTestEngine(t)
	fillTestPositions(engine)

	keys := engine.Keys()
	s1, _, err := engine.GetState(keys[0])
	require.Nil(t, err)

	engine.RemoveDistressed(engine.Positions())
	s2, _, err := engine.GetState(keys[0])
	require.Nil(t, err)
	require.False(t, bytes.Equal(s1, s2))
}

func TestSnapshotStaeUpdateMarkPrice(t *testing.T) {
	engine := getTestEngine(t)
	fillTestPositions(engine)

	keys := engine.Keys()
	s1, _, err := engine.GetState(keys[0])
	require.Nil(t, err)

	engine.UpdateMarkPrice(num.NewUint(12))
	s2, _, err := engine.GetState(keys[0])
	require.Nil(t, err)
	require.False(t, bytes.Equal(s1, s2))
}

func TestSnapshotHashNoPositions(t *testing.T) {
	engine := getTestEngine(t)

	keys := engine.Keys()
	s1, _, err := engine.GetState(keys[0])
	require.Nil(t, err)
	require.Equal(t, "278f2eff5adc1ea5b8365bd04c6e534ef64ca43df737c22ee61db46a8dac5870", hex.EncodeToString(crypto.Hash(s1)))
}

func TestStopSnapshotTaking(t *testing.T) {
	engine := getTestEngine(t)
	keys := engine.Keys()

	// signal to kill the engine's snapshots
	engine.StopSnapshots()

	s, _, err := engine.GetState(keys[0])
	assert.NoError(t, err)
	assert.Nil(t, s)
	assert.True(t, engine.Stopped())
}

func TestSnapshotClosedPositionStillSerializeStats(t *testing.T) {
	engine := getTestEngine(t)
	// fillTestPositions(engine)

	keys := engine.Keys()
	s1, _, err := engine.GetState(keys[0])
	require.Nil(t, err)

	// first load two orders

	buyOrder := &types.Order{
		Party:     "test_party_1",
		Side:      types.SideBuy,
		Size:      uint64(150),
		Remaining: uint64(150),
		Price:     num.UintZero(),
	}
	engine.RegisterOrder(context.TODO(), buyOrder)

	sellOrder := &types.Order{
		Party:     "test_party_2",
		Side:      types.SideSell,
		Size:      uint64(150),
		Remaining: uint64(150),
		Price:     num.UintZero(),
	}
	engine.RegisterOrder(context.TODO(), sellOrder)

	// then get them to update the positions

	trade := &types.Trade{
		Size:   150,
		Buyer:  "test_party_1",
		Seller: "test_party_2",
	}

	engine.Update(context.TODO(), trade, buyOrder, sellOrder)

	// now we close the positions
	// just swap the parties on the orders
	// and trades
	buyOrder.Party, sellOrder.Party = sellOrder.Party, buyOrder.Party
	trade.Buyer, trade.Seller = trade.Seller, trade.Buyer

	engine.RegisterOrder(context.TODO(), buyOrder)
	engine.RegisterOrder(context.TODO(), sellOrder)
	engine.Update(context.TODO(), trade, buyOrder, sellOrder)

	// should get 2 closed positions
	assert.Len(t, engine.GetClosedPositions(), 2)

	s2, _, err := engine.GetState(keys[0])
	require.Nil(t, err)
	require.False(t, bytes.Equal(s1, s2))

	payload := &snapshot.Payload{}
	assert.NoError(t, proto.Unmarshal(s2, payload))

	marketPositions := payload.Data.(*snapshot.Payload_MarketPositions)
	assert.NotNil(t, marketPositions)
	// assert the records are saved
	assert.Len(t, marketPositions.MarketPositions.PartiesRecords, 2)
	// while yet, there's no positions anymores.
	assert.Len(t, marketPositions.MarketPositions.Positions, 0)
}
