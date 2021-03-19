package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
)

type Party struct {
	*Base
	p types.Party
}

func NewPartyEvent(ctx context.Context, p types.Party) *Party {
	cpy := p.DeepClone()
	return &Party{
		Base: newBase(ctx, PartyEvent),
		p:    *cpy,
	}
}

func (p Party) IsParty(id string) bool {
	return p.p.Id == id
}

func (p *Party) Party() types.Party {
	return p.p
}

func (p Party) Proto() types.Party {
	return p.p
}

func (p Party) StreamMessage() *eventspb.BusEvent {
	return &eventspb.BusEvent{
		Id:    p.eventID(),
		Block: p.TraceID(),
		Type:  p.et.ToProto(),
		Event: &eventspb.BusEvent_Party{
			Party: &p.p,
		},
	}
}
