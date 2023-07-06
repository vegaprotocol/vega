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

	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderBookAmends_FlipToGTT(t *testing.T) {
	now := time.Unix(5, 0)
	tm := getTestMarket(t, now, nil, nil)
	ctx := context.Background()
	defer tm.ctrl.Finish()

	addAccount(t, tm, "aaa")
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	o1 := getMarketOrder(tm, now, types.OrderTypeLimit, types.OrderTimeInForceGTC, "Order01", types.SideBuy, "aaa", 2, 100)
	o1conf, err := tm.market.SubmitOrder(ctx, o1)
	require.NoError(t, err)
	require.NotNil(t, o1conf)
	require.Equal(t, 0, tm.market.GetPeggedExpiryOrderCount())

	// now we edit the order t make it GTC so it should not expire
	v10 := now.Add(10 * time.Second).UnixNano()
	amendment := &types.OrderAmendment{
		OrderID:     o1.ID,
		TimeInForce: types.OrderTimeInForceGTT,
		ExpiresAt:   &v10,
	}

	amendConf, err := tm.market.AmendOrder(ctx, amendment, "aaa", vgcrypto.RandomHash())
	require.NotNil(t, amendConf)
	require.NoError(t, err)
	assert.Equal(t, types.OrderStatusActive, amendConf.Order.Status)
	require.Equal(t, 1, tm.market.GetPeggedExpiryOrderCount())

	// now we edit the order t make it GTC so it should not expire
	v := now.Add(20 * time.Second).UnixNano()
	amendment2 := &types.OrderAmendment{
		OrderID:     o1.ID,
		TimeInForce: types.OrderTimeInForceGTT,
		ExpiresAt:   &v,
	}

	amendConf2, err := tm.market.AmendOrder(ctx, amendment2, "aaa", vgcrypto.RandomHash())
	require.NotNil(t, amendConf2)
	require.NoError(t, err)
	assert.Equal(t, types.OrderStatusActive, amendConf2.Order.Status)
	require.Equal(t, 1, tm.market.GetPeggedExpiryOrderCount())

	// now we edit the order t make it GTC so it should not expire
	amendment3 := &types.OrderAmendment{
		OrderID:     o1.ID,
		TimeInForce: types.OrderTimeInForceGTC,
	}

	amendConf3, err := tm.market.AmendOrder(ctx, amendment3, "aaa", vgcrypto.RandomHash())
	require.NotNil(t, amendConf3)
	require.NoError(t, err)
	assert.Equal(t, types.OrderStatusActive, amendConf3.Order.Status)
	require.Equal(t, 0, tm.market.GetPeggedExpiryOrderCount())
}
