package events

import (
	"context"

	eventspb "code.vegaprotocol.io/data-node/proto/vega/events/v1"
	"code.vegaprotocol.io/data-node/types"
)

type TxErr struct {
	*Base
	evt *eventspb.TxErrorEvent
}

func NewTxErrEvent(ctx context.Context, err error, partyID string, tx interface{}) *TxErr {
	evt := &TxErr{
		Base: newBase(ctx, TxErrEvent),
		evt: &eventspb.TxErrorEvent{
			PartyId: partyID,
			ErrMsg:  err.Error(),
		},
	}
	switch tv := tx.(type) {
	case *types.ProposalSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_Proposal{
			Proposal: tv.IntoProto(),
		}
	case types.ProposalSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_Proposal{
			Proposal: tv.IntoProto(),
		}
	case *types.VoteSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_VoteSubmission{
			VoteSubmission: tv.IntoProto(),
		}
	case types.VoteSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_VoteSubmission{
			VoteSubmission: tv.IntoProto(),
		}
	case *types.OrderSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_OrderSubmission{
			OrderSubmission: tv.IntoProto(),
		}
	case types.OrderSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_OrderSubmission{
			OrderSubmission: tv.IntoProto(),
		}
	case *types.OrderCancellation:
		evt.evt.Transaction = &eventspb.TxErrorEvent_OrderCancellation{
			OrderCancellation: tv.IntoProto(),
		}
	case types.OrderCancellation:
		evt.evt.Transaction = &eventspb.TxErrorEvent_OrderCancellation{
			OrderCancellation: tv.IntoProto(),
		}
	case *types.OrderAmendment:
		evt.evt.Transaction = &eventspb.TxErrorEvent_OrderAmendment{
			OrderAmendment: tv.IntoProto(),
		}
	case types.OrderAmendment:
		evt.evt.Transaction = &eventspb.TxErrorEvent_OrderAmendment{
			OrderAmendment: tv.IntoProto(),
		}
	case *types.LiquidityProvisionSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_LiquidityProvisionSubmission{
			LiquidityProvisionSubmission: tv.IntoProto(),
		}
	case types.LiquidityProvisionSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_LiquidityProvisionSubmission{
			LiquidityProvisionSubmission: tv.IntoProto(),
		}
	case *types.WithdrawSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_WithdrawSubmission{
			WithdrawSubmission: tv.IntoProto(),
		}
	case types.WithdrawSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_WithdrawSubmission{
			WithdrawSubmission: tv.IntoProto(),
		}
	}
	return evt
}

func (t TxErr) IsParty(id string) bool {
	return t.evt.PartyId == id
}

func (t TxErr) Proto() eventspb.TxErrorEvent {
	return *t.evt
}

func (t TxErr) StreamMessage() *eventspb.BusEvent {
	return &eventspb.BusEvent{
		Id:    t.eventID(),
		Block: t.TraceID(),
		Type:  t.et.ToProto(),
		Event: &eventspb.BusEvent_TxErrEvent{
			TxErrEvent: t.evt,
		},
	}
}
