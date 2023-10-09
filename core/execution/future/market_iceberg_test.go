// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package future_test

import (
	"context"
	"testing"
	"time"

	vegacontext "code.vegaprotocol.io/vega/libs/context"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarketSubmitCancelIceberg(t *testing.T) {
	party1 := "party1"
	now := time.Unix(100000, 0)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	tm := getTestMarket(t, now, nil, nil)
	defer tm.ctrl.Finish()

	addAccount(t, tm, party1)
	iceberg := &types.Order{
		Type:        types.OrderTypeLimit,
		TimeInForce: types.OrderTimeInForceGTC,
		Status:      types.OrderStatusActive,
		ID:          "someid",
		Side:        types.SideBuy,
		Party:       party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       num.NewUint(100),
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
		Version:     common.InitialOrderVersion,
		IcebergOrder: &types.IcebergOrder{
			PeakSize:           10,
			MinimumVisibleSize: 5,
		},
	}

	// submit order
	_, err := tm.market.SubmitOrder(context.Background(), iceberg)
	require.NoError(t, err)

	tm.now = tm.now.Add(time.Second)
	tm.market.OnTick(ctx, tm.now)
	require.Equal(t, types.MarketStateActive, tm.market.State()) // enter auction

	// check that its on the book and the volume is only the visible peak
	assert.Equal(t, int64(100), tm.market.GetVolumeOnBook())

	// and that the position represents the whole iceberg size
	tm.market.BlockEnd(vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash()))
	pos := requirePositionUpdate(t, tm.events)
	assert.Equal(t, int64(100), pos.PotentialBuys())

	// now cancel the order and check potential buy returns to 0
	tm.events = tm.events[:0]
	_, err = tm.market.CancelOrder(context.Background(), iceberg.Party, iceberg.ID, iceberg.ID)
	require.NoError(t, err)
	tm.market.BlockEnd(vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash()))
	pos = requirePositionUpdate(t, tm.events)
	assert.Equal(t, int64(0), pos.PotentialBuys())
}

func TestMarketAmendIceberg(t *testing.T) {
	party1 := "party1"
	now := time.Unix(100000, 0)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	tm := getTestMarket(t, now, nil, nil)
	defer tm.ctrl.Finish()

	addAccount(t, tm, party1)
	iceberg := &types.Order{
		Type:        types.OrderTypeLimit,
		TimeInForce: types.OrderTimeInForceGTC,
		Status:      types.OrderStatusActive,
		ID:          "someid",
		Side:        types.SideBuy,
		Party:       party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       num.NewUint(100),
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
		Version:     common.InitialOrderVersion,
		IcebergOrder: &types.IcebergOrder{
			PeakSize:           10,
			MinimumVisibleSize: 5,
		},
	}

	// submit order
	_, err := tm.market.SubmitOrder(context.Background(), iceberg)
	require.NoError(t, err)

	tm.now = tm.now.Add(time.Second)
	tm.market.OnTick(ctx, tm.now)
	require.Equal(t, types.MarketStateActive, tm.market.State()) // enter auction

	// now reduce the size of the iceberg so that only the reserved amount is reduced
	amendedOrder := &types.OrderAmendment{
		OrderID:     iceberg.ID,
		Price:       nil,
		SizeDelta:   -50,
		TimeInForce: types.OrderTimeInForceGTC,
	}

	tm.eventCount = 0
	tm.events = tm.events[:0]
	_, err = tm.market.AmendOrder(context.Background(), amendedOrder, party1, vgcrypto.RandomHash())
	require.NoError(t, err)
	amended := requireOrderEvent(t, tm.events)
	assert.Equal(t, uint64(50), amended.Size)
	assert.Equal(t, iceberg.Remaining, amended.Remaining)
	assert.Equal(t, uint64(40), amended.IcebergOrder.ReservedRemaining)

	// now increase the size delta and check that reserved remaining is increase, but remaining is the same
	amendedOrder.SizeDelta = 70
	tm.eventCount = 0
	tm.events = tm.events[:0]
	_, err = tm.market.AmendOrder(context.Background(), amendedOrder, party1, vgcrypto.RandomHash())
	require.NoError(t, err)
	amended = requireOrderEvent(t, tm.events)
	assert.Equal(t, uint64(120), amended.Size)
	assert.Equal(t, iceberg.Remaining, amended.Remaining)
	assert.Equal(t, uint64(110), amended.IcebergOrder.ReservedRemaining)

	// now reduce the size such that reserved is reduce to 0 and some remaining is removed too
	amendedOrder.SizeDelta = -115
	tm.eventCount = 0
	tm.events = tm.events[:0]
	_, err = tm.market.AmendOrder(context.Background(), amendedOrder, party1, vgcrypto.RandomHash())
	require.NoError(t, err)
	amended = requireOrderEvent(t, tm.events)
	assert.Equal(t, uint64(5), amended.Size)
	assert.Equal(t, uint64(5), amended.Remaining)
	assert.Equal(t, uint64(0), amended.IcebergOrder.ReservedRemaining)
}

func TestMarketAmendIcebergToNoReserve(t *testing.T) {
	party1 := "party1"
	now := time.Unix(100000, 0)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	tm := getTestMarket(t, now, nil, nil)
	defer tm.ctrl.Finish()

	addAccount(t, tm, party1)
	iceberg := &types.Order{
		Type:        types.OrderTypeLimit,
		TimeInForce: types.OrderTimeInForceGTC,
		Status:      types.OrderStatusActive,
		ID:          "someid",
		Side:        types.SideBuy,
		Party:       party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       num.NewUint(100),
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
		Version:     common.InitialOrderVersion,
		IcebergOrder: &types.IcebergOrder{
			PeakSize:           100,
			MinimumVisibleSize: 5,
		},
	}

	// submit order
	_, err := tm.market.SubmitOrder(context.Background(), iceberg)
	require.NoError(t, err)

	tm.now = tm.now.Add(time.Second)
	tm.market.OnTick(ctx, tm.now)
	require.Equal(t, types.MarketStateActive, tm.market.State()) // enter auction

	// now reduce the size of the iceberg so that only the reserved amount is reduced
	amendedOrder := &types.OrderAmendment{
		OrderID:     iceberg.ID,
		Price:       nil,
		SizeDelta:   -75,
		TimeInForce: types.OrderTimeInForceGTC,
	}

	tm.eventCount = 0
	tm.events = tm.events[:0]
	_, err = tm.market.AmendOrder(context.Background(), amendedOrder, party1, vgcrypto.RandomHash())
	require.NoError(t, err)
	amended := requireOrderEvent(t, tm.events)
	assert.Equal(t, uint64(25), amended.Size)
	assert.Equal(t, uint64(25), amended.Remaining)
	assert.Equal(t, uint64(0), amended.IcebergOrder.ReservedRemaining)
}

func requireOrderEvent(t *testing.T, evts []events.Event) *types.Order {
	t.Helper()
	for _, e := range evts {
		switch evt := e.(type) {
		case *events.Order:
			o, err := types.OrderFromProto(evt.Order())
			require.NoError(t, err)
			return o
		}
	}
	require.Fail(t, "did not find order event")
	return nil
}

func requirePositionUpdate(t *testing.T, evts []events.Event) *events.PositionState {
	t.Helper()
	for _, e := range evts {
		switch evt := e.(type) {
		case *events.PositionState:
			return evt
		}
	}
	require.Fail(t, "did not find position update event")
	return nil
}
