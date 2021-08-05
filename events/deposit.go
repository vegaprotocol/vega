package events

import (
	"context"

	"code.vegaprotocol.io/data-node/types"
	proto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

type Deposit struct {
	*Base
	d proto.Deposit
}

func NewDepositEvent(ctx context.Context, d types.Deposit) *Deposit {
	return &Deposit{
		Base: newBase(ctx, DepositEvent),
		d:    *d.IntoProto(),
	}
}

func (d *Deposit) Deposit() proto.Deposit {
	return d.d
}

func (d Deposit) IsParty(id string) bool {
	return d.d.PartyId == id
}

func (d Deposit) PartyID() string { return d.d.PartyId }

func (d Deposit) Proto() proto.Deposit {
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

func DepositEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Deposit {
	return &Deposit{
		Base: newBaseFromStream(ctx, DepositEvent, be),
		d:    *be.GetDeposit(),
	}
}
