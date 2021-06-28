package events

import (
	"context"

	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
	"code.vegaprotocol.io/vega/types"
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
		ptv, _ := tv.IntoProto()
		evt.evt.Transaction = &eventspb.TxErrorEvent_Proposal{
			Proposal: ptv,
		}
	case types.ProposalSubmission:
		ptv, _ := (&tv).IntoProto()
		evt.evt.Transaction = &eventspb.TxErrorEvent_Proposal{
			Proposal: ptv,
		}
	case *types.VoteSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_VoteSubmission{
			VoteSubmission: tv.IntoProto(),
		}
	case types.VoteSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_VoteSubmission{
			VoteSubmission: tv.IntoProto(),
		}
	case *commandspb.OrderSubmission:
		cpy := *tv
		evt.evt.Transaction = &eventspb.TxErrorEvent_OrderSubmission{
			OrderSubmission: &cpy,
		}
	case commandspb.OrderSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_OrderSubmission{
			OrderSubmission: &tv,
		}
	case *commandspb.OrderCancellation:
		cpy := *tv
		evt.evt.Transaction = &eventspb.TxErrorEvent_OrderCancellation{
			OrderCancellation: &cpy,
		}
	case commandspb.OrderCancellation:
		evt.evt.Transaction = &eventspb.TxErrorEvent_OrderCancellation{
			OrderCancellation: &tv,
		}
	case *commandspb.OrderAmendment:
		cpy := *tv
		evt.evt.Transaction = &eventspb.TxErrorEvent_OrderAmendment{
			OrderAmendment: &cpy,
		}
	case commandspb.OrderAmendment:
		evt.evt.Transaction = &eventspb.TxErrorEvent_OrderAmendment{
			OrderAmendment: &tv,
		}
	case *commandspb.LiquidityProvisionSubmission:
		cpy := *tv
		evt.evt.Transaction = &eventspb.TxErrorEvent_LiquidityProvisionSubmission{
			LiquidityProvisionSubmission: &cpy,
		}
	case commandspb.LiquidityProvisionSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_LiquidityProvisionSubmission{
			LiquidityProvisionSubmission: &tv,
		}
	case *commandspb.WithdrawSubmission:
		cpy := *tv
		evt.evt.Transaction = &eventspb.TxErrorEvent_WithdrawSubmission{
			WithdrawSubmission: &cpy,
		}
	case commandspb.WithdrawSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_WithdrawSubmission{
			WithdrawSubmission: &tv,
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
