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

package future_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	vegacontext "code.vegaprotocol.io/vega/libs/context"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"

	"code.vegaprotocol.io/vega/core/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGTTAmendToGTCAmendInPlace_OrderGetExpired(t *testing.T) {
	now := time.Unix(5, 0)
	tm := getTestMarket(t, now, nil, nil)
	ctx := context.Background()
	ctx = vegacontext.WithTraceID(ctx, vgcrypto.RandomHash())

	addAccount(t, tm, "aaa")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTT, "Order01", types.SideBuy, "aaa", 1, 10)
	o1.ExpiresAt = now.Add(5 * time.Second).UnixNano()
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NoError(t, err)
	require.NotNil(t, o1conf)

	// now we edit the order t make it GTC so it should not expire
	amendment := &types.OrderAmendment{
		OrderID:     o1.ID,
		TimeInForce: types.OrderTimeInForceGTC,
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment, "aaa", vgcrypto.RandomHash())
	require.NotNil(t, amendConf)
	require.NoError(t, err)
	assert.Equal(t, types.OrderStatusActive, amendConf.Order.Status)

	// now expire, and nothing should be returned
	ctx = vegacontext.WithTraceID(context.Background(), vgcrypto.RandomHash())

	tm.events = nil
	tm.market.OnTick(ctx, now.Add(10*time.Second))

	t.Run("no orders expired", func(t *testing.T) {
		// First collect all the orders events
		orders := []*types.Order{}
		for _, e := range tm.events {
			switch evt := e.(type) {
			case *events.Order:
				if evt.Order().Status == types.OrderStatusExpired {
					orders = append(orders, mustOrderFromProto(evt.Order()))
				}
			}
		}
		require.Equal(t, 0, len(orders))
	})
}
