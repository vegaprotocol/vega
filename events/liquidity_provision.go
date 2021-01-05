package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

type LiquidityProvision struct {
	*Base
	p types.LiquidityProvision
}

func NewLiquidityProvisionEvent(ctx context.Context, p *types.LiquidityProvision) *LiquidityProvision {
	order := &LiquidityProvision{
		Base: newBase(ctx, LiquidityProvisionEvent),
		p:    *p,
	}
	return order
}

func (p LiquidityProvision) IsParty(id string) bool {
	return (p.p.PartyID == id)
}

func (p LiquidityProvision) PartyID() string {
	return p.p.PartyID
}

func (p LiquidityProvision) MarketID() string {
	return p.p.MarketID
}

func (p *LiquidityProvision) LiquidityProvision() *types.LiquidityProvision {
	return &p.p
}

func (p LiquidityProvision) Proto() types.LiquidityProvision {
	return p.p
}

func (p LiquidityProvision) StreamMessage() *types.BusEvent {
	return &types.BusEvent{
		ID:    p.eventID(),
		Block: p.TraceID(),
		Type:  p.et.ToProto(),
		Event: &types.BusEvent_LiquidityProvision{
			LiquidityProvision: &p.p,
		},
	}
}
