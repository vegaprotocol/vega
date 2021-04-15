package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type LiquidityProvision struct {
	*Base
	p types.LiquidityProvision
}

func NewLiquidityProvisionEvent(ctx context.Context, p *types.LiquidityProvision) *LiquidityProvision {
	// Manually copy the pointer objects across
	buys := p.Buys
	p.Buys = make([]*types.LiquidityOrderReference, len(buys))
	sells := p.Sells
	p.Sells = make([]*types.LiquidityOrderReference, len(sells))

	for i, lor := range buys {
		tempBuy := *lor
		tempLO := *lor.LiquidityOrder
		tempBuy.LiquidityOrder = &tempLO
		p.Buys[i] = &tempBuy
	}

	for i, lor := range sells {
		tempSell := *lor
		tempLO := *lor.LiquidityOrder
		tempSell.LiquidityOrder = &tempLO
		p.Sells[i] = &tempSell
	}

	order := &LiquidityProvision{
		Base: newBase(ctx, LiquidityProvisionEvent),
		p:    *p,
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

func (p LiquidityProvision) StreamMessage() *types.BusEvent {
	return &types.BusEvent{
		Id:    p.eventID(),
		Block: p.TraceID(),
		Type:  p.et.ToProto(),
		Event: &types.BusEvent_LiquidityProvision{
			LiquidityProvision: &p.p,
		},
	}
}
