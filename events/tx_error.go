package events

import (
	"context"
	"fmt"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
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
	case *commandspb.OrderSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_OrderSubmission{
			OrderSubmission: tv,
		}
	case *commandspb.OrderCancellation:
		evt.evt.Transaction = &eventspb.TxErrorEvent_OrderCancellation{
			OrderCancellation: tv,
		}
	case *commandspb.OrderAmendment:
		evt.evt.Transaction = &eventspb.TxErrorEvent_OrderAmendment{
			OrderAmendment: tv,
		}
	case *commandspb.VoteSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_VoteSubmission{
			VoteSubmission: tv,
		}
	case *commandspb.WithdrawSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_WithdrawSubmission{
			WithdrawSubmission: tv,
		}
	case *commandspb.LiquidityProvisionSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_LiquidityProvisionSubmission{
			LiquidityProvisionSubmission: tv,
		}
	case *commandspb.ProposalSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_Proposal{
			Proposal: tv,
		}
	case *commandspb.DelegateSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_DelegateSubmission{
			DelegateSubmission: tv,
		}
	case *commandspb.UndelegateSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_UndelegateSubmission{
			UndelegateSubmission: tv,
		}
	case *commandspb.RestoreSnapshot:
		evt.evt.Transaction = &eventspb.TxErrorEvent_RestoreSnapshot{
			RestoreSnapshot: tv,
		}
	case *commandspb.Transfer:
		evt.evt.Transaction = &eventspb.TxErrorEvent_Transfer{
			Transfer: tv,
		}
	case *commandspb.CancelTransfer:
		evt.evt.Transaction = &eventspb.TxErrorEvent_CancelTransfer{
			CancelTransfer: tv,
		}
	case error: // unsupported command error
		evt.evt.ErrMsg = fmt.Sprintf("%v - %v", err, tv)
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
		Version: eventspb.Version,
		Id:      t.eventID(),
		Block:   t.TraceID(),
		ChainId: t.ChainID(),
		Type:    t.et.ToProto(),
		Event: &eventspb.BusEvent_TxErrEvent{
			TxErrEvent: t.evt,
		},
	}
}

func TxErrEventFromStream(ctx context.Context, be *eventspb.BusEvent) *TxErr {
	return &TxErr{
		Base: newBaseFromStream(ctx, TxErrEvent, be),
		evt:  be.GetTxErrEvent(),
	}
}
