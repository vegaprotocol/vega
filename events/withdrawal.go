package events

import (
	"context"

	proto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types"
)

type Withdrawal struct {
	*Base
	w proto.Withdrawal
}

func NewWithdrawalEvent(ctx context.Context, w types.Withdrawal) *Withdrawal {
	return &Withdrawal{
		Base: newBase(ctx, WithdrawalEvent),
		w:    *w.IntoProto(),
	}
}

func (w *Withdrawal) Withdrawal() proto.Withdrawal {
	return w.w
}

func (w Withdrawal) IsParty(id string) bool {
	return w.w.PartyId == id
}

func (w Withdrawal) PartyID() string { return w.w.PartyId }

func (w Withdrawal) Proto() proto.Withdrawal {
	return w.w
}

func (w Withdrawal) StreamMessage() *eventspb.BusEvent {
	wit := w.w
	return &eventspb.BusEvent{
		Version: eventspb.Version,
		Id:      w.eventID(),
		Block:   w.TraceID(),
		Type:    w.et.ToProto(),
		Event: &eventspb.BusEvent_Withdrawal{
			Withdrawal: &wit,
		},
	}
}

func WithdrawalEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Withdrawal {
	return &Withdrawal{
		Base: newBaseFromStream(ctx, WithdrawalEvent, be),
		w:    *be.GetWithdrawal(),
	}
}
