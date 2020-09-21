package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
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
	return (d.d.PartyID == id)
}

func (d Deposit) PartyID() string { return d.d.PartyID }

func (d Deposit) Proto() types.Deposit {
	return d.d
}

func (d Deposit) StreamMessage() *types.BusEvent {
	dep := d.d
	return &types.BusEvent{
		ID:   d.traceID,
		Type: d.et.ToProto(),
		Event: &types.BusEvent_Deposit{
			Deposit: &dep,
		},
	}
}
