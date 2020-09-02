package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type Party struct {
	*Base
	p types.Party
}

func NewPartyEvent(ctx context.Context, p types.Party) *Party {
	return &Party{
		Base: newBase(ctx, PartyEvent),
		p:    p,
	}
}

func (p *Party) Party() types.Party {
	return p.p
}

func (p Party) Proto() types.Party {
	return p.p
}

func (p Party) StreamMessage() *types.BusEvent {
	return &types.BusEvent{
		ID:   p.traceID,
		Type: p.et.ToProto(),
		Event: &types.BusEvent_Party{
			Party: &p.p,
		},
	}
}
