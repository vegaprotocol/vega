package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
)

type Deposit struct {
	*Base
	d types.Deposit
}

func NewDepositEvent(ctx context.Context, d types.Deposit) *Deposit {
	return &Deposit{
		Base: newBase(ctx, DepositEvent),
		d:    d,
	}
}

func (d *Deposit) Deposit() types.Deposit {
	return d.d
}

func (d Deposit) IsParty(id string) bool {
	return d.d.PartyId == id
}

func (d Deposit) PartyID() string { return d.d.PartyId }

func (d Deposit) Proto() types.Deposit {
	return d.d
}

func (d Deposit) StreamMessage() *eventspb.BusEvent {
	dep := d.d
	return &eventspb.BusEvent{
		Id:    d.eventID(),
		Block: d.TraceID(),
		Type:  d.et.ToProto(),
		Event: &eventspb.BusEvent_Deposit{
			Deposit: &dep,
		},
	}
}
