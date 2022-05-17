package liquidity_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	snapshotpb "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/idgeneration"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"code.vegaprotocol.io/vega/libs/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var initialTime = time.Date(2020, 10, 20, 1, 1, 1, 0, time.UTC)

func TestSnapshotRoundTrip(t *testing.T) {
	var (
		party1 = "p1"
		party2 = "p2"
		party3 = "p3"
		market = "market-id"
		ctx    = context.Background()
		e1     = newTestEngine(t, initialTime)
		e2     = newTestEngine(t, initialTime)
		e3     = newTestEngine(t, initialTime)
	)

	e1.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	e2.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	e2.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	e3.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	e3.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()

	lp1 := &types.LiquidityProvisionSubmission{
		MarketID:         market,
		Fee:              num.MustDecimalFromString("0.01"),
		CommitmentAmount: num.NewUint(1000),
		Buys: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReferenceMid,
				Offset:     num.NewUint(1),
				Proportion: 1,
			},
		},
		Sells: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReferenceMid,
				Offset:     num.NewUint(1),
				Proportion: 1,
			},
		},
	}
	lp2 := &types.LiquidityProvisionSubmission{
		MarketID:         market,
		Fee:              num.MustDecimalFromString("0.42"),
		CommitmentAmount: num.NewUint(4242),
		Buys: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReferenceMid,
				Offset:     num.NewUint(10),
				Proportion: 42,
			},
		},
		Sells: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReferenceMid,
				Offset:     num.NewUint(42),
				Proportion: 58,
			},
		},
	}

	idgen1 := idgeneration.New("f663375fd6843a0807d17b10ad8425a6ba45c8c2dd6339f400c5b2426f900c13")
	require.NoError(t,
		e1.engine.SubmitLiquidityProvision(ctx, lp1, party1, idgen1),
	)
	idgen2 := idgeneration.New("0454d8b74441ca3bac8f9b141408502d9b1f297e8ef1054d45775566677a8072")
	require.NoError(t,
		e1.engine.SubmitLiquidityProvision(ctx, lp2, party2, idgen2),
	)

	keys := e1.engine.Keys()
	kToH := map[string][]byte{}
	kToS := map[string][]byte{}

	expectedHashes := map[string]string{
		"parameters:market-id":             "d663375fd6843a0807d17b10ad8425a6ba45c8c2dd6339f400c5b2426f900c13",
		"partiesLiquidityOrders:market-id": "0254d8b74441ca3bac8f9b141408502d9b1f297e8ef1054d45775566677a8072",
		"partiesOrders:market-id":          "f9cb31b1c4c8df91f6a348d43978c302c8887336107c265259bc74fdddf00e19",
		"pendingProvisions:market-id":      "6cc4d407a2ea45e37e27993eb6f94134b3f906d080777d94bf99551aa82dc461",
		"provisions:market-id":             "7c76902e145d0eaf0abb83382575c027097abdb418364c351e2ad085e1c69c3e",
		"liquiditySupplied:market-id":      "3276bba2a77778ba710ec29e3a6e59212452dbda69eaac8f9160930d1270da1d",
	}

	for _, key := range keys {
		s, _, err := e1.engine.GetState(key)
		assert.NoError(t, err)
		kToS[key] = s
		h := crypto.Hash(s)
		assert.NoError(t, err)
		kToH[key] = h

		// compare hashes to the expected ones
		assert.Equalf(t, expectedHashes[key], hex.EncodeToString(h), "hashes for key %q does not match", key)
	}

	// now we reload the keys / state
	for _, s := range kToS {
		pl := snapshotpb.Payload{}
		assert.NoError(t, proto.Unmarshal(s, &pl))
		_, err := e2.engine.LoadState(ctx, types.PayloadFromProto(&pl))
		assert.NoError(t, err)
	}

	// now ensure both are producing same hashes
	for k, e1h := range kToH {
		e2s, _, err := e2.engine.GetState(k)
		e2h := crypto.Hash(e2s)
		assert.NoError(t, err)
		assert.True(t, bytes.Equal(e1h, e2h))
	}

	// now we update the state of e2 to see if hashes changes

	expectedHashes2 := map[string]string{
		"parameters:market-id":             "b5eec91c297baf1f06830350dbcb37d79937561ae605d2304eb12680e443775c",
		"partiesLiquidityOrders:market-id": "c29d0b4d9265cf7951cc396ce8d6350ab7b2e978782423c4090842cbb2619f76",
		"partiesOrders:market-id":          "f9cb31b1c4c8df91f6a348d43978c302c8887336107c265259bc74fdddf00e19",
		"pendingProvisions:market-id":      "627ef55af7f36bea0d09b0081b85d66531a01df060d8e9447e17049a4e152b12",
		"provisions:market-id":             "89335d14e98ca80b144cb6502e9b508d97d63027ba0c7733d6024030cdf102ed",
		"liquiditySupplied:market-id":      "3276bba2a77778ba710ec29e3a6e59212452dbda69eaac8f9160930d1270da1d",
	}

	lp3 := &types.LiquidityProvisionSubmission{
		MarketID:         market,
		Fee:              num.MustDecimalFromString("0.2"),
		CommitmentAmount: num.NewUint(5000),
		Buys: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReferenceMid,
				Offset:     num.NewUint(10),
				Proportion: 42,
			},
		},
		Sells: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReferenceMid,
				Offset:     num.NewUint(42),
				Proportion: 58,
			},
		},
	}

	idgen3 := idgeneration.New("59a8634e030ecf0548d3a77c74a9a251e6e2c90c65af32136e97dcb889e92774")
	require.NoError(t,
		e2.engine.SubmitLiquidityProvision(ctx, lp3, party3, idgen3),
	)

	require.NoError(t,
		e2.engine.OnMarketLiquidityProvisionShapesMaxSizeUpdate(4200),
	)

	repriceFN := func(
		order *types.PeggedOrder, side types.Side,
	) (*num.Uint, *types.PeggedOrder, error) {
		return num.NewUint(100), order, nil
	}

	e2.priceMonitor.EXPECT().GetValidPriceRange().
		Return(num.NewWrappedDecimal(num.Zero(), num.DecimalZero()), num.NewWrappedDecimal(num.NewUint(90), num.DecimalFromInt64(110))).
		AnyTimes()

	_, _, err := e2.engine.Update(ctx, num.DecimalFromFloat(99), num.DecimalFromFloat(101),
		repriceFN, []*types.Order{
			{
				ID:        "order-id-1",
				Party:     party1,
				MarketID:  market,
				Side:      types.SideBuy,
				Price:     num.NewUint(90),
				Size:      10,
				Remaining: 10,
			},
		},
	)

	require.NoError(t, err)

	for _, key := range keys {
		s, _, err := e2.engine.GetState(key)
		assert.NoError(t, err)
		h := crypto.Hash(s)

		// compare hashes to the expected ones
		assert.Equalf(t, expectedHashes2[key], hex.EncodeToString(h), "hashes for key %q does not match", key)

		pl := snapshotpb.Payload{}
		assert.NoError(t, proto.Unmarshal(s, &pl))
		_, err = e3.engine.LoadState(ctx, types.PayloadFromProto(&pl))
		assert.NoError(t, err)
	}

	for _, key := range keys {
		s, _, err := e3.engine.GetState(key)
		assert.NoError(t, err)
		h := crypto.Hash(s)
		// compare hashes to the expected ones
		assert.Equalf(t, expectedHashes2[key], hex.EncodeToString(h), "hashes for key %q does not match", key)
	}

	// now reconcile with orderbook
	orders := e3.engine.GetLiquidityOrders(party1)
	require.Len(t, orders, 2)

	// create copies of the orders
	idToOrder := map[string]*types.Order{
		orders[0].ID: orders[0].Clone(),
		orders[1].ID: orders[1].Clone(),
	}

	e3.orderbook.EXPECT().GetOrderByID(gomock.Any()).AnyTimes().DoAndReturn(
		func(orderID string) (*types.Order, error) {
			if o, ok := idToOrder[orderID]; ok {
				return o, nil
			}
			return nil, errors.New("not gound")
		},
	)
	e3.engine.ReconcileWithOrderBook(e3.orderbook)

	// change a value and check the orders in the engine also change i.e they both point to the same order
	now := time.Now().UnixNano()
	for _, o := range idToOrder {
		o.UpdatedAt = now
	}

	orders = e3.engine.GetLiquidityOrders(party1)
	require.Len(t, orders, 2)
	for _, o := range orders {
		assert.Equal(t, now, o.UpdatedAt)
	}
}

func TestStopSnapshotTaking(t *testing.T) {
	te := newTestEngine(t, initialTime)
	keys := te.engine.Keys()

	// signal to kill the engine's snapshots
	te.engine.StopSnapshots()

	s, _, err := te.engine.GetState(keys[0])
	assert.NoError(t, err)
	assert.Nil(t, s)
	assert.True(t, te.engine.Stopped())
}

func TestSnapshotChangeOnUpdate(t *testing.T) {
	var (
		party1 = "p1"
		market = "market-id"
		ctx    = context.Background()
		e1     = newTestEngine(t, initialTime)
	)

	e1.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	lp1 := &types.LiquidityProvisionSubmission{
		MarketID:         market,
		Fee:              num.MustDecimalFromString("0.01"),
		CommitmentAmount: num.NewUint(1000),
		Buys: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReferenceMid,
				Offset:     num.NewUint(1),
				Proportion: 1,
			},
		},
		Sells: []*types.LiquidityOrder{
			{
				Reference:  types.PeggedReferenceMid,
				Offset:     num.NewUint(1),
				Proportion: 1,
			},
		},
	}
	idgen1 := idgeneration.New("f663375fd6843a0807d17b10ad8425a6ba45c8c2dd6339f400c5b2426f900c13")
	require.NoError(t,
		e1.engine.SubmitLiquidityProvision(ctx, lp1, party1, idgen1),
	)

	key := "provisions:market-id"
	s1, _, err := e1.engine.GetState(key)
	assert.NoError(t, err)
	assert.NotEmpty(t, s1)

	repriceFN := func(
		order *types.PeggedOrder, side types.Side,
	) (*num.Uint, *types.PeggedOrder, error) {
		return num.NewUint(100), order, nil
	}

	e1.priceMonitor.EXPECT().GetValidPriceRange().
		Return(num.NewWrappedDecimal(num.Zero(), num.DecimalZero()), num.NewWrappedDecimal(num.NewUint(90), num.DecimalFromInt64(110))).
		AnyTimes()

	_, _, err = e1.engine.Update(ctx, num.DecimalFromFloat(99), num.DecimalFromFloat(101),
		repriceFN, []*types.Order{
			{
				ID:        "order-id-1",
				Party:     party1,
				MarketID:  market,
				Side:      types.SideBuy,
				Price:     num.NewUint(90),
				Size:      10,
				Remaining: 10,
			},
		},
	)
	require.NoError(t, err)

	// Get new hash, it should have changed
	s2, _, err := e1.engine.GetState(key)
	require.NoError(t, err)
	require.NotEmpty(t, s1)
	require.False(t, bytes.Equal(s1, s2))
}
