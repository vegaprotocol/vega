package events_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/assert"
)

func TestLiquidityProvisionDeepClone(t *testing.T) {
	ctx := context.Background()

	buyOrder := &types.LiquidityOrderReference{
		OrderID: "OrderId1",
		LiquidityOrder: &types.LiquidityOrder{
			Reference:  types.PeggedReferenceMid,
			Proportion: 10,
			Offset:     -5,
		},
	}

	sellOrder := &types.LiquidityOrderReference{
		OrderID: "OrderId1",
		LiquidityOrder: &types.LiquidityOrder{
			Reference:  types.PeggedReferenceMid,
			Proportion: 20,
			Offset:     5,
		},
	}

	lp := &types.LiquidityProvision{
		ID:               "Id",
		Party:            "PartyId",
		CreatedAt:        10000,
		UpdatedAt:        20000,
		MarketID:         "MarketId",
		CommitmentAmount: num.NewUint(30000),
		Fee:              num.DecimalFromFloat(0.01),
		Version:          "1",
		Status:           types.LiquidityProvisionStatusUndeployed,
		Reference:        "Reference",
		Sells:            []*types.LiquidityOrderReference{sellOrder},
		Buys:             []*types.LiquidityOrderReference{buyOrder},
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
	lp.Version = "999"
	lp.Status = types.LiquidityProvisionUnspecified
	lp.Reference = "Changed"
	sellOrder.OrderID = "Changed"
	sellOrder.LiquidityOrder.Offset = -999
	sellOrder.LiquidityOrder.Proportion = 999
	sellOrder.LiquidityOrder.Reference = types.PeggedReferenceBestAsk
	buyOrder.OrderID = "Changed"
	buyOrder.LiquidityOrder.Offset = 999
	buyOrder.LiquidityOrder.Proportion = 999
	buyOrder.LiquidityOrder.Reference = types.PeggedReferenceBestBid

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
	assert.NotEqual(t, sellOrder.OrderID, lp2.Sells[0].OrderId)
	assert.NotEqual(t, sellOrder.LiquidityOrder.Offset, lp2.Sells[0].LiquidityOrder.Offset)
	assert.NotEqual(t, sellOrder.LiquidityOrder.Proportion, lp2.Sells[0].LiquidityOrder.Proportion)
	assert.NotEqual(t, sellOrder.LiquidityOrder.Reference, lp2.Sells[0].LiquidityOrder.Reference)
	assert.NotEqual(t, buyOrder.OrderID, lp2.Buys[0].OrderId)
	assert.NotEqual(t, buyOrder.LiquidityOrder.Offset, lp2.Buys[0].LiquidityOrder.Offset)
	assert.NotEqual(t, buyOrder.LiquidityOrder.Proportion, lp2.Buys[0].LiquidityOrder.Proportion)
	assert.NotEqual(t, buyOrder.LiquidityOrder.Reference, lp2.Buys[0].LiquidityOrder.Reference)
}
