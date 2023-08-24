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
	"github.com/stretchr/testify/assert"
)

func TestLiquidityProvisionDeepClone(t *testing.T) {
	ctx := context.Background()

	lp := &types.LiquidityProvision{
		ID:               "Id",
		Party:            "PartyId",
		CreatedAt:        10000,
		UpdatedAt:        20000,
		MarketID:         "MarketId",
		CommitmentAmount: num.NewUint(30000),
		Fee:              num.DecimalFromFloat(0.01),
		Version:          1,
		Status:           types.LiquidityProvisionStatusUndeployed,
		Reference:        "Reference",
	}

	// Create the event
	lpEvent := events.NewLiquidityProvisionEvent(ctx, lp)
	lp2 := lpEvent.LiquidityProvision()

	// Alter the original message
	lp.ID = "Changed"
	lp.Party = "Changed"
	lp.CreatedAt = 999
	lp.UpdatedAt = 999
	lp.MarketID = "Changed"
	lp.CommitmentAmount = num.NewUint(999)
	lp.Fee = num.DecimalFromFloat(99.9)
	lp.Version = 999
	lp.Status = types.LiquidityProvisionUnspecified
	lp.Reference = "Changed"

	// Check that values are different
	assert.NotEqual(t, lp.ID, lp2.Id)
	assert.NotEqual(t, lp.Party, lp2.PartyId)
	assert.NotEqual(t, lp.CreatedAt, lp2.CreatedAt)
	assert.NotEqual(t, lp.UpdatedAt, lp2.UpdatedAt)
	assert.NotEqual(t, lp.MarketID, lp2.MarketId)
	assert.NotEqual(t, lp.CommitmentAmount, lp2.CommitmentAmount)
	assert.NotEqual(t, lp.Fee, lp2.Fee)
	assert.NotEqual(t, lp.Version, lp2.Version)
	assert.NotEqual(t, lp.Status, lp2.Status)
	assert.NotEqual(t, lp.Reference, lp2.Reference)
}
