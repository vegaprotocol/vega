package events

import (
	"context"

	proto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types"
)

type LiquidityProvision struct {
	*Base
	p *proto.LiquidityProvision
}

func NewLiquidityProvisionEvent(ctx context.Context, p *types.LiquidityProvision) *LiquidityProvision {
	order := &LiquidityProvision{
		Base: newBase(ctx, LiquidityProvisionEvent),
		p:    p.IntoProto(),
	}
	return order
}

func (p LiquidityProvision) IsParty(id string) bool {
	return p.p.PartyId == id
}

func (p LiquidityProvision) PartyID() string {
	return p.p.PartyId
}

func (p LiquidityProvision) MarketID() string {
	return p.p.MarketId
}

func (p LiquidityProvision) LiquidityProvision() *proto.LiquidityProvision {
	return p.p
}

func (p LiquidityProvision) Proto() *proto.LiquidityProvision {
	return p.p
}

func (p LiquidityProvision) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(p.Base)
	busEvent.Event = &eventspb.BusEvent_LiquidityProvision{
		LiquidityProvision: p.p,
	}

	return busEvent
}

func LiquidityProvisionEventFromStream(ctx context.Context, be *eventspb.BusEvent) *LiquidityProvision {
	order := &LiquidityProvision{
		Base: newBaseFromBusEvent(ctx, LiquidityProvisionEvent, be),
		p:    be.GetLiquidityProvision(),
	}
	return order
}
