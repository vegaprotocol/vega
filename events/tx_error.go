package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type TxErr struct {
	*Base
	evt *types.TxErrorEvent
}

func NewTxErrEvent(ctx context.Context, err error, partyID string, tx interface{}) *TxErr {
	evt := &TxErr{
		Base: newBase(ctx, TxErrEvent),
		evt: &types.TxErrorEvent{
			PartyID: partyID,
			ErrMsg:  err.Error(),
		},
	}
	switch tv := tx.(type) {
	case *types.Proposal:
		cpy := *tv
		evt.evt.Transaction = &types.TxErrorEvent_Proposal{
			Proposal: &cpy,
		}
	case types.Proposal:
		evt.evt.Transaction = &types.TxErrorEvent_Proposal{
			Proposal: &tv,
		}
	case *types.Vote:
		cpy := *tv
		evt.evt.Transaction = &types.TxErrorEvent_Vote{
			Vote: &cpy,
		}
	case types.Vote:
		evt.evt.Transaction = &types.TxErrorEvent_Vote{
			Vote: &tv,
		}
	case *types.OrderSubmission:
		cpy := *tv
		evt.evt.Transaction = &types.TxErrorEvent_OrderSubmission{
			OrderSubmission: &cpy,
		}
	case types.OrderSubmission:
		evt.evt.Transaction = &types.TxErrorEvent_OrderSubmission{
			OrderSubmission: &tv,
		}
	case *types.OrderCancellation:
		cpy := *tv
		evt.evt.Transaction = &types.TxErrorEvent_OrderCancellation{
			OrderCancellation: &cpy,
		}
	case types.OrderCancellation:
		evt.evt.Transaction = &types.TxErrorEvent_OrderCancellation{
			OrderCancellation: &tv,
		}
	case *types.OrderAmendment:
		cpy := *tv
		evt.evt.Transaction = &types.TxErrorEvent_OrderAmendment{
			OrderAmendment: &cpy,
		}
	case types.OrderAmendment:
		evt.evt.Transaction = &types.TxErrorEvent_OrderAmendment{
			OrderAmendment: &tv,
		}
	}
	return evt
}

func (t TxErr) IsParty(id string) bool {
	return (t.evt.PartyID == id)
}

func (t TxErr) Proto() types.TxErrorEvent {
	return *t.evt
}

func (t TxErr) StreamMessage() *types.BusEvent {
	return &types.BusEvent{
		ID:    t.eventID(),
		Block: t.TraceID(),
		Type:  t.et.ToProto(),
		Event: &types.BusEvent_TxErrEvent{
			TxErrEvent: t.evt,
		},
	}
}
