package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type Withdrawal struct {
	*Base
	w types.Withdrawal
}

func NewWithdrawalEvent(ctx context.Context, w types.Withdrawal) *Withdrawal {
	return &Withdrawal{
		Base: newBase(ctx, WithdrawalEvent),
		w:    w,
	}
}

func (w *Withdrawal) Withdrawal() types.Withdrawal {
	return w.w
}

func (w Withdrawal) IsParty(id string) bool {
	return (w.w.PartyID == id)
}

func (w Withdrawal) PartyID() string { return w.w.PartyID }

func (w Withdrawal) Proto() types.Withdrawal {
	return w.w
}

func (w Withdrawal) StreamMessage() *types.BusEvent {
	wit := w.w
	return &types.BusEvent{
		ID:    w.eventID(),
		Block: w.TraceID(),
		Type:  w.et.ToProto(),
		Event: &types.BusEvent_Withdrawal{
			Withdrawal: &wit,
		},
	}
}
