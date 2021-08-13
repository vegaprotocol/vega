package events

import (
	"context"

	proto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

type Party struct {
	*Base
	p proto.Party
}

func NewPartyEvent(ctx context.Context, p proto.Party) *Party {
	cpy := p.DeepClone()
	return &Party{
		Base: newBase(ctx, PartyEvent),
		p:    *cpy,
	}
}

func (p Party) IsParty(id string) bool {
	return p.p.Id == id
}

func (p *Party) Party() proto.Party {
	return p.p
}

func (p Party) Proto() proto.Party {
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

func PartyEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Party {
	return &Party{
		Base: newBaseFromStream(ctx, PartyEvent, be),
		p:    *be.GetParty(),
	}
}
