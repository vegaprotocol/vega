package events_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/proto"
	"github.com/stretchr/testify/assert"
)

func TestLiquidityProvisionDeepClone(t *testing.T) {
	ctx := context.Background()

	buyOrder := &proto.LiquidityOrderReference{
		OrderId: "OrderId1",
		LiquidityOrder: &proto.LiquidityOrder{
			Reference:  proto.PeggedReference_PEGGED_REFERENCE_MID,
			Proportion: 10,
			Offset:     -5,
		},
	}

	sellOrder := &proto.LiquidityOrderReference{
		OrderId: "OrderId1",
		LiquidityOrder: &proto.LiquidityOrder{
			Reference:  proto.PeggedReference_PEGGED_REFERENCE_MID,
			Proportion: 20,
			Offset:     5,
		},
	}

	lp := &proto.LiquidityProvision{
		Id:               "Id",
		PartyId:          "PartyId",
		CreatedAt:        10000,
		UpdatedAt:        20000,
		MarketId:         "MarketId",
		CommitmentAmount: 30000,
		Fee:              "0.01",
		Version:          "1",
		Status:           proto.LiquidityProvision_STATUS_UNDEPLOYED,
		Reference:        "Reference",
		Sells:            []*proto.LiquidityOrderReference{sellOrder},
		Buys:             []*proto.LiquidityOrderReference{buyOrder},
	}

	// Create the event
	lpEvent := events.NewLiquidityProvisionEvent(ctx, lp)
	lp2 := lpEvent.LiquidityProvision()

	// Alter the original message
	lp.Id = "Changed"
	lp.PartyId = "Changed"
	lp.CreatedAt = 999
	lp.UpdatedAt = 999
	lp.MarketId = "Changed"
	lp.CommitmentAmount = 999
	lp.Fee = "99.9"
	lp.Version = "999"
	lp.Status = proto.LiquidityProvision_STATUS_UNSPECIFIED
	lp.Reference = "Changed"
	sellOrder.OrderId = "Changed"
	sellOrder.LiquidityOrder.Offset = -999
	sellOrder.LiquidityOrder.Proportion = 999
	sellOrder.LiquidityOrder.Reference = proto.PeggedReference_PEGGED_REFERENCE_BEST_ASK
	buyOrder.OrderId = "Changed"
	buyOrder.LiquidityOrder.Offset = 999
	buyOrder.LiquidityOrder.Proportion = 999
	buyOrder.LiquidityOrder.Reference = proto.PeggedReference_PEGGED_REFERENCE_BEST_BID

	// Check that values are different
	assert.NotEqual(t, lp.Id, lp2.Id)
	assert.NotEqual(t, lp.PartyId, lp2.PartyId)
	assert.NotEqual(t, lp.CreatedAt, lp2.CreatedAt)
	assert.NotEqual(t, lp.UpdatedAt, lp2.UpdatedAt)
	assert.NotEqual(t, lp.MarketId, lp2.MarketId)
	assert.NotEqual(t, lp.CommitmentAmount, lp2.CommitmentAmount)
	assert.NotEqual(t, lp.Fee, lp2.Fee)
	assert.NotEqual(t, lp.Version, lp2.Version)
	assert.NotEqual(t, lp.Status, lp2.Status)
	assert.NotEqual(t, lp.Reference, lp2.Reference)
	assert.NotEqual(t, sellOrder.OrderId, lp2.Sells[0].OrderId)
	assert.NotEqual(t, sellOrder.LiquidityOrder.Offset, lp2.Sells[0].LiquidityOrder.Offset)
	assert.NotEqual(t, sellOrder.LiquidityOrder.Proportion, lp2.Sells[0].LiquidityOrder.Proportion)
	assert.NotEqual(t, sellOrder.LiquidityOrder.Reference, lp2.Sells[0].LiquidityOrder.Reference)
	assert.NotEqual(t, buyOrder.OrderId, lp2.Buys[0].OrderId)
	assert.NotEqual(t, buyOrder.LiquidityOrder.Offset, lp2.Buys[0].LiquidityOrder.Offset)
	assert.NotEqual(t, buyOrder.LiquidityOrder.Proportion, lp2.Buys[0].LiquidityOrder.Proportion)
	assert.NotEqual(t, buyOrder.LiquidityOrder.Reference, lp2.Buys[0].LiquidityOrder.Reference)
}
