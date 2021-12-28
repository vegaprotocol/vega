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
	return &eventspb.BusEvent{
		Version: eventspb.Version,
		Id:      p.eventID(),
		Block:   p.TraceID(),
		ChainId: p.ChainID(),
		Type:    p.et.ToProto(),
		Event: &eventspb.BusEvent_LiquidityProvision{
			LiquidityProvision: p.p,
		},
	}
}

func LiquidityProvisionEventFromStream(ctx context.Context, be *eventspb.BusEvent) *LiquidityProvision {
	order := &LiquidityProvision{
		Base: newBaseFromStream(ctx, LiquidityProvisionEvent, be),
		p:    be.GetLiquidityProvision(),
	}
	return order
}
