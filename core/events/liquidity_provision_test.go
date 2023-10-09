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
