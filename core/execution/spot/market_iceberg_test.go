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

package spot_test

import (
	"context"
	"testing"
	"time"

	vegacontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/require"
)

func TestMarketSubmitCancelIceberg(t *testing.T) {
	party1 := "party1"
	now := time.Unix(100000, 0)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	tm := newTestMarket(t, defaultPriceMonitorSettings, &types.AuctionDuration{Duration: 1}, now)
	defer tm.ctrl.Finish()
	tm.market.StartOpeningAuction(ctx)

	addAccountWithAmount(tm, party1, 10000000, tm.quoteAsset)
	iceberg := &types.Order{
		Type:        types.OrderTypeLimit,
		TimeInForce: types.OrderTimeInForceGTC,
		Status:      types.OrderStatusActive,
		ID:          "someid",
		Side:        types.SideBuy,
		Party:       party1,
		MarketID:    tm.market.GetID(),
		Size:        100,
		Price:       num.NewUint(10),
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
	oid := crypto.RandomHash()
	_, err := tm.market.SubmitOrder(context.Background(), iceberg.IntoSubmission(), party1, oid)
	require.NoError(t, err)

	tm.now = tm.now.Add(time.Second)
	tm.market.OnTick(ctx, tm.now)
	require.Equal(t, types.MarketStatePending, tm.market.GetMarketState())
	haBalance1, err := tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "1000", haBalance1.Balance.String())

	// now cancel the order and check potential buy returns to 0
	tm.events = tm.events[:0]
	_, err = tm.market.CancelOrder(context.Background(), iceberg.Party, oid, oid)
	require.NoError(t, err)
	tm.market.BlockEnd(vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash()))

	haBalance2, err := tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "0", haBalance2.Balance.String())
}

func TestMarketAmendIceberg(t *testing.T) {
	party1 := "party1"
	now := time.Unix(100000, 0)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	tm := newTestMarket(t, defaultPriceMonitorSettings, &types.AuctionDuration{Duration: 1}, now)
	defer tm.ctrl.Finish()
	tm.market.StartOpeningAuction(ctx)

	addAccountWithAmount(tm, party1, 10000000, tm.quoteAsset)
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
	oid := crypto.RandomHash()
	_, err := tm.market.SubmitOrder(context.Background(), iceberg.IntoSubmission(), party1, oid)
	require.NoError(t, err)

	tm.now = tm.now.Add(time.Second)
	tm.market.OnTick(ctx, tm.now)
	require.Equal(t, types.MarketStatePending, tm.market.GetMarketState())

	haBalance1, err := tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "10000", haBalance1.Balance.String())

	gaBalance1, err := tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "9990000", gaBalance1.Balance.String())

	// now reduce the size of the iceberg so that only the reserved amount is reduced
	amendedOrder := &types.OrderAmendment{
		OrderID:     oid,
		Price:       nil,
		SizeDelta:   -50,
		TimeInForce: types.OrderTimeInForceGTC,
	}

	tm.eventCount = 0
	tm.events = tm.events[:0]
	_, err = tm.market.AmendOrder(context.Background(), amendedOrder, party1, vgcrypto.RandomHash())
	require.NoError(t, err)

	haBalance2, err := tm.collateralEngine.GetPartyHoldingAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "5000", haBalance2.Balance.String())

	gaBalance2, err := tm.collateralEngine.GetPartyGeneralAccount("party1", tm.quoteAsset)
	require.NoError(t, err)
	require.Equal(t, "9995000", gaBalance2.Balance.String())

	amended := requireOrderEvent(t, tm.events)
	require.Equal(t, uint64(50), amended.Size)
	require.Equal(t, uint64(10), amended.Remaining)
	require.Equal(t, uint64(40), amended.IcebergOrder.ReservedRemaining)

	// now increase the size delta and check that reserved remaining is increased, but remaining is the same
	amendedOrder.SizeDelta = 70
	tm.eventCount = 0
	tm.events = tm.events[:0]
	_, err = tm.market.AmendOrder(context.Background(), amendedOrder, party1, vgcrypto.RandomHash())
	require.NoError(t, err)
	amended = requireOrderEvent(t, tm.events)
	require.Equal(t, uint64(120), amended.Size)
	require.Equal(t, uint64(10), amended.Remaining)
	require.Equal(t, uint64(110), amended.IcebergOrder.ReservedRemaining)

	// now reduce the size such that reserved is reduce to 0 and some remaining is removed too
	amendedOrder.SizeDelta = -115
	tm.eventCount = 0
	tm.events = tm.events[:0]
	_, err = tm.market.AmendOrder(context.Background(), amendedOrder, party1, vgcrypto.RandomHash())
	require.NoError(t, err)
	amended = requireOrderEvent(t, tm.events)
	require.Equal(t, uint64(5), amended.Size)
	require.Equal(t, uint64(5), amended.Remaining)
	require.Equal(t, uint64(0), amended.IcebergOrder.ReservedRemaining)
}

func TestMarketAmendIcebergToNoReserve(t *testing.T) {
	party1 := "party1"
	now := time.Unix(100000, 0)
	ctx := vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())
	tm := newTestMarket(t, defaultPriceMonitorSettings, &types.AuctionDuration{Duration: 1}, now)
	defer tm.ctrl.Finish()
	tm.market.StartOpeningAuction(ctx)

	addAccountWithAmount(tm, party1, 10000000, tm.quoteAsset)

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
	oid := crypto.RandomHash()
	_, err := tm.market.SubmitOrder(context.Background(), iceberg.IntoSubmission(), "party1", oid)
	require.NoError(t, err)

	tm.now = tm.now.Add(time.Second)
	tm.market.OnTick(ctx, tm.now)
	require.Equal(t, types.MarketStatePending, tm.market.GetMarketState())

	// now reduce the size of the iceberg so that only the reserved amount is reduced
	amendedOrder := &types.OrderAmendment{
		OrderID:     oid,
		Price:       nil,
		SizeDelta:   -75,
		TimeInForce: types.OrderTimeInForceGTC,
	}

	tm.eventCount = 0
	tm.events = tm.events[:0]
	_, err = tm.market.AmendOrder(context.Background(), amendedOrder, party1, vgcrypto.RandomHash())
	require.NoError(t, err)
	amended := requireOrderEvent(t, tm.events)
	require.Equal(t, uint64(25), amended.Size)
	require.Equal(t, uint64(25), amended.Remaining)
	require.Equal(t, uint64(0), amended.IcebergOrder.ReservedRemaining)
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
