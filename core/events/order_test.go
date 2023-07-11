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

package events_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	proto "code.vegaprotocol.io/vega/protos/vega"
	"github.com/stretchr/testify/assert"
)

func TestOrderDeepClone(t *testing.T) {
	ctx := context.Background()

	o := &types.Order{
		ID:          "Id",
		MarketID:    "MarketId",
		Party:       "PartyId",
		Side:        proto.Side_SIDE_BUY,
		Price:       num.NewUint(1000),
		Size:        2000,
		Remaining:   3000,
		TimeInForce: proto.Order_TIME_IN_FORCE_GFN,
		Type:        proto.Order_TYPE_LIMIT,
		CreatedAt:   4000,
		Status:      proto.Order_STATUS_ACTIVE,
		ExpiresAt:   5000,
		Reference:   "Reference",
		Reason:      proto.ErrEditNotAllowed,
		UpdatedAt:   6000,
		Version:     7000,
		BatchID:     8000,
		PeggedOrder: &types.PeggedOrder{
			Reference: proto.PeggedReference_PEGGED_REFERENCE_MID,
			Offset:    num.NewUint(9000),
		},
	}

	oEvent := events.NewOrderEvent(ctx, o)
	o2 := oEvent.Order()

	// Change the original values
	o.ID = "Changed"
	o.MarketID = "Changed"
	o.Party = "Changed"
	o.Side = proto.Side_SIDE_UNSPECIFIED
	o.Price = num.NewUint(999)
	o.Size = 999
	o.Remaining = 999
	o.TimeInForce = proto.Order_TIME_IN_FORCE_UNSPECIFIED
	o.Type = proto.Order_TYPE_UNSPECIFIED
	o.CreatedAt = 999
	o.Status = proto.Order_STATUS_UNSPECIFIED
	o.ExpiresAt = 999
	o.Reference = "Changed"
	o.Reason = proto.ErrInvalidMarketID
	o.UpdatedAt = 999
	o.Version = 999
	o.BatchID = 999
	o.PeggedOrder.Reference = proto.PeggedReference_PEGGED_REFERENCE_UNSPECIFIED
	o.PeggedOrder.Offset = num.NewUint(999)

	// Check things have changed
	assert.NotEqual(t, o.ID, o2.Id)
	assert.NotEqual(t, o.MarketID, o2.MarketId)
	assert.NotEqual(t, o.Party, o2.PartyId)
	assert.NotEqual(t, o.Side, o2.Side)
	assert.NotEqual(t, o.Price, o2.Price)
	assert.NotEqual(t, o.Size, o2.Size)
	assert.NotEqual(t, o.Remaining, o2.Remaining)
	assert.NotEqual(t, o.TimeInForce, o2.TimeInForce)
	assert.NotEqual(t, o.Type, o2.Type)
	assert.NotEqual(t, o.CreatedAt, o2.CreatedAt)
	assert.NotEqual(t, o.Status, o2.Status)
	assert.NotEqual(t, o.ExpiresAt, o2.ExpiresAt)
	assert.NotEqual(t, o.Reference, o2.Reference)
	assert.NotEqual(t, o.Reason, o2.Reason)
	assert.NotEqual(t, o.UpdatedAt, o2.UpdatedAt)
	assert.NotEqual(t, o.Version, o2.Version)
	assert.NotEqual(t, o.BatchID, o2.BatchId)
	assert.NotEqual(t, o.PeggedOrder.Reference, o2.PeggedOrder.Reference)
	assert.NotEqual(t, o.PeggedOrder.Offset, o2.PeggedOrder.Offset)
}
