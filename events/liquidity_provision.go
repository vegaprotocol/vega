package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
)

type LiquidityProvision struct {
	*Base
	p types.LiquidityProvision
}

func NewLiquidityProvisionEvent(ctx context.Context, p *types.LiquidityProvision) *LiquidityProvision {
	cpy := p.DeepClone()

	order := &LiquidityProvision{
		Base: newBase(ctx, LiquidityProvisionEvent),
		p:    *cpy,
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

func (p LiquidityProvision) LiquidityProvision() types.LiquidityProvision {
	return p.p
}

func (p LiquidityProvision) Proto() types.LiquidityProvision {
	return p.p
}

func (p LiquidityProvision) StreamMessage() *eventspb.BusEvent {
	return &eventspb.BusEvent{
		Id:    p.eventID(),
		Block: p.TraceID(),
		Type:  p.et.ToProto(),
		Event: &eventspb.BusEvent_LiquidityProvision{
			LiquidityProvision: &p.p,
		},
	}
}
