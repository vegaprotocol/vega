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
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestVersioning(t *testing.T) {
	party1 := "party1"
	now := time.Unix(10, 0)
	tm := getTestMarket(t, now, nil, nil)
	price := uint64(100)
	size := uint64(100)

	addAccount(t, tm, party1)
	tm.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	orderBuy := &types.Order{
		Status:      types.OrderStatusActive,
		Type:        types.OrderTypeLimit,
		TimeInForce: types.OrderTimeInForceGTC,
		ID:          "someid",
		Side:        types.SideBuy,
		Party:       party1,
		MarketID:    tm.market.GetID(),
		Size:        size,
		Price:       num.NewUint(price),
		Remaining:   100,
		CreatedAt:   now.UnixNano(),
		Reference:   "party1-buy-order",
	}
	// Create an order and check version is set to 1
	confirmation, err := tm.market.SubmitOrder(context.TODO(), orderBuy)
	assert.NotNil(t, confirmation)
	assert.NoError(t, err)
	assert.EqualValues(t, confirmation.Order.Version, uint64(1))

	orderID := confirmation.Order.ID

	// Amend price up, check version moves to 2
	amend := &types.OrderAmendment{
		OrderID:  orderID,
		MarketID: tm.market.GetID(),
		Price:    num.NewUint(price + 1),
	}

	amendment, err := tm.market.AmendOrder(context.TODO(), amend, party1, vgcrypto.RandomHash())
	assert.NotNil(t, amendment)
	assert.NoError(t, err)

	// Amend price down, check version moves to 3
	amend.Price = num.NewUint(price - 1)
	amendment, err = tm.market.AmendOrder(context.TODO(), amend, party1, vgcrypto.RandomHash())
	assert.NotNil(t, amendment)
	assert.NoError(t, err)

	// Amend quantity up, check version moves to 4
	amend.Price = nil
	amend.SizeDelta = 1
	amendment, err = tm.market.AmendOrder(context.TODO(), amend, party1, vgcrypto.RandomHash())
	assert.NotNil(t, amendment)
	assert.NoError(t, err)

	// Amend quantity down, check version moves to 5
	amend.SizeDelta = -2
	amendment, err = tm.market.AmendOrder(context.TODO(), amend, party1, vgcrypto.RandomHash())
	assert.NotNil(t, amendment)
	assert.NoError(t, err)

	// Flip to GTT, check version moves to 6
	amend.TimeInForce = types.OrderTimeInForceGTT
	exp := now.UnixNano() + 100000000000
	amend.ExpiresAt = &exp
	amend.SizeDelta = 0
	amendment, err = tm.market.AmendOrder(context.TODO(), amend, party1, vgcrypto.RandomHash())
	assert.NotNil(t, amendment)
	assert.NoError(t, err)

	// Update expiry time, check version moves to 7
	exp = now.UnixNano() + 100000000000
	amend.ExpiresAt = &exp
	amendment, err = tm.market.AmendOrder(context.TODO(), amend, party1, vgcrypto.RandomHash())
	assert.NotNil(t, amendment)
	assert.NoError(t, err)

	// Flip back GTC, check version moves to 8
	amend.TimeInForce = types.OrderTimeInForceGTC
	amend.ExpiresAt = nil
	amendment, err = tm.market.AmendOrder(context.TODO(), amend, party1, vgcrypto.RandomHash())
	assert.NotNil(t, amendment)
	assert.NoError(t, err)
}
