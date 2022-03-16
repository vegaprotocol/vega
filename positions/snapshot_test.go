package positions_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"testing"
	"time"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/positions"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"code.vegaprotocol.io/vega/libs/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fillTestPositions(e *positions.SnapshotEngine) {
	orders := []types.Order{
		{
			Party:     "test_party_1",
			Side:      types.SideBuy,
			Size:      uint64(100),
			Remaining: uint64(100),
			Price:     num.Zero(),
		},
		{
			Party:     "test_party_2",
			Side:      types.SideBuy,
			Size:      uint64(200),
			Remaining: uint64(200),
			Price:     num.Zero(),
		},
		{
			Party:     "test_party_3",
			Side:      types.SideBuy,
			Size:      uint64(300),
			Remaining: uint64(300),
			Price:     num.Zero(),
		},
		{
			Party:     "test_party_1",
			Side:      types.SideSell,
			Size:      uint64(1000),
			Remaining: uint64(1000),
			Price:     num.Zero(),
		},
	}

	for _, order := range orders {
		e.RegisterOrder(context.TODO(), &order)
	}

	trade := types.Trade{
		Type:      types.TradeTypeDefault,
		ID:        "trade_id",
		MarketID:  "market_id",
		Price:     num.NewUint(10000),
		Size:      uint64(15),
		Buyer:     "test_party_3",
		Seller:    "test_party_1",
		BuyOrder:  "buy_order_id",
		SellOrder: "sell_order_id",
		Timestamp: time.Now().Unix(),
	}
	e.Update(&trade)
}

func TestSnapshotSaveAndLoad(t *testing.T) {
	engine := getTestEngine(t)
	fillTestPositions(engine)

	keys := engine.Keys()
	require.Equal(t, 1, len(keys))
	require.Equal(t, "test_market", keys[0])

	h1, err := engine.GetHash(keys[0])
	require.Nil(t, err)
	h2, err := engine.GetHash(keys[0])
	require.Nil(t, err)

	// With no change the hashes are equal
	require.True(t, bytes.Equal(h1, h2))

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

	// Get hash again
	h3, err := snapEngine.GetHash(keys[0])
	require.Nil(t, err)
	require.True(t, bytes.Equal(h1, h3))
	require.Equal(t, len(engine.Positions()), len(snapEngine.Positions()))
	for _, p := range engine.Positions() {
		// find it in the other engine by partyID
		pos, found := snapEngine.GetPositionByPartyID(p.Party())
		require.True(t, found)
		require.Equal(t, p, pos)
	}
}

func TestSnapshotHashNoChanges(t *testing.T) {
	engine := getTestEngine(t)
	fillTestPositions(engine)

	keys := engine.Keys()
	h1, err := engine.GetHash(keys[0])
	require.Nil(t, err)
	h2, err := engine.GetHash(keys[0])
	require.Nil(t, err)

	// With no changes we expect the hashes are equal
	require.True(t, bytes.Equal(h1, h2))
}

func TestSnapshotHashRegisterOrder(t *testing.T) {
	engine := getTestEngine(t)
	fillTestPositions(engine)

	keys := engine.Keys()
	h1, err := engine.GetHash(keys[0])
	require.Nil(t, err)

	// Add and order and the hash should change
	newOrder := &types.Order{
		Party:     "test_party_1",
		Side:      types.SideBuy,
		Size:      uint64(150),
		Remaining: uint64(150),
		Price:     num.Zero(),
	}
	engine.RegisterOrder(context.TODO(), newOrder)
	h2, err := engine.GetHash(keys[0])
	require.Nil(t, err)
	require.False(t, bytes.Equal(h1, h2))
}

func TestSnapshotHashUnregisterOrder(t *testing.T) {
	engine := getTestEngine(t)
	fillTestPositions(engine)

	keys := engine.Keys()
	h1, err := engine.GetHash(keys[0])
	require.Nil(t, err)

	// Add and order and the hash should change
	newOrder := &types.Order{
		Party:     "test_party_1",
		Side:      types.SideBuy,
		Size:      uint64(10),
		Remaining: uint64(10),
		Price:     num.Zero(),
	}
	engine.RegisterOrder(context.TODO(), newOrder)
	h2, err := engine.GetHash(keys[0])
	require.Nil(t, err)
	require.False(t, bytes.Equal(h1, h2))
}

func TestSnapshotHashAmendOrder(t *testing.T) {
	engine := getTestEngine(t)
	fillTestPositions(engine)

	// Add and order and the hash should change
	newOrders := []*types.Order{
		{
			Party:     "test_party_1",
			Side:      types.SideBuy,
			Size:      uint64(100),
			Remaining: uint64(100),
			Price:     num.Zero(),
		},
		{
			Party:     "test_party_1",
			Side:      types.SideBuy,
			Size:      uint64(90),
			Remaining: uint64(90),
			Price:     num.Zero(),
		},
	}
	engine.RegisterOrder(context.TODO(), newOrders[0])
	keys := engine.Keys()
	h1, err := engine.GetHash(keys[0])
	require.Nil(t, err)

	// Amend it
	engine.AmendOrder(context.TODO(), newOrders[0], newOrders[1])
	h2, err := engine.GetHash(keys[0])
	require.Nil(t, err)
	require.False(t, bytes.Equal(h1, h2))

	// Then amend it back, hash should be the same as originally
	engine.AmendOrder(context.TODO(), newOrders[1], newOrders[0])
	h2, err = engine.GetHash(keys[0])
	require.Nil(t, err)
	require.True(t, bytes.Equal(h1, h2))
}

func TestSnapshotHashRemoveDistressed(t *testing.T) {
	engine := getTestEngine(t)
	fillTestPositions(engine)

	keys := engine.Keys()
	h1, err := engine.GetHash(keys[0])
	require.Nil(t, err)

	engine.RemoveDistressed(engine.Positions())
	h2, err := engine.GetHash(keys[0])
	require.Nil(t, err)
	require.False(t, bytes.Equal(h1, h2))
}

func TestSnapshotHashUpdateMarkPrice(t *testing.T) {
	engine := getTestEngine(t)
	fillTestPositions(engine)

	keys := engine.Keys()
	h1, err := engine.GetHash(keys[0])
	require.Nil(t, err)

	engine.UpdateMarkPrice(num.NewUint(12))
	h2, err := engine.GetHash(keys[0])
	require.Nil(t, err)
	require.False(t, bytes.Equal(h1, h2))
}

func TestSnapshotHashNoPositions(t *testing.T) {
	engine := getTestEngine(t)

	keys := engine.Keys()
	h1, err := engine.GetHash(keys[0])
	require.Nil(t, err)
	require.Equal(t, "278f2eff5adc1ea5b8365bd04c6e534ef64ca43df737c22ee61db46a8dac5870", hex.EncodeToString(h1))
}

func TestStopSnapshotTaking(t *testing.T) {
	engine := getTestEngine(t)
	keys := engine.Keys()

	// signal to kill the engine's snapshots
	engine.StopSnapshots()

	s, _, err := engine.GetState(keys[0])
	assert.NoError(t, err)
	assert.Nil(t, s)
	h, err := engine.GetHash(keys[0])
	assert.NoError(t, err)
	assert.Nil(t, h)
	assert.True(t, engine.Stopped())
}
