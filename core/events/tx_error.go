// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package events

import (
	"context"
	"fmt"

	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type TxErr struct {
	*Base
	evt *eventspb.TxErrorEvent
}

func NewTxErrEvent(ctx context.Context, err error, partyID string, tx interface{}, cmd string) *TxErr {
	evt := &TxErr{
		Base: newBase(ctx, TxErrEvent),
		evt: &eventspb.TxErrorEvent{
			PartyId: partyID,
			ErrMsg:  fmt.Sprintf("%v - %v", cmd, err.Error()),
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
	case *commandspb.LiquidityProvisionCancellation:
		evt.evt.Transaction = &eventspb.TxErrorEvent_LiquidityProvisionCancellation{
			LiquidityProvisionCancellation: tv,
		}
	case *commandspb.LiquidityProvisionAmendment:
		evt.evt.Transaction = &eventspb.TxErrorEvent_LiquidityProvisionAmendment{
			LiquidityProvisionAmendment: tv,
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
	case *commandspb.Transfer:
		evt.evt.Transaction = &eventspb.TxErrorEvent_Transfer{
			Transfer: tv,
		}
	case *commandspb.CancelTransfer:
		evt.evt.Transaction = &eventspb.TxErrorEvent_CancelTransfer{
			CancelTransfer: tv,
		}
	case *commandspb.AnnounceNode:
		evt.evt.Transaction = &eventspb.TxErrorEvent_AnnounceNode{
			AnnounceNode: tv,
		}
	case *commandspb.OracleDataSubmission:
		evt.evt.Transaction = &eventspb.TxErrorEvent_OracleDataSubmission{
			OracleDataSubmission: tv,
		}
	case *commandspb.ProtocolUpgradeProposal:
		evt.evt.Transaction = &eventspb.TxErrorEvent_ProtocolUpgradeProposal{
			ProtocolUpgradeProposal: tv,
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
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_TxErrEvent{
		TxErrEvent: t.evt,
	}

	return busEvent
}

func TxErrEventFromStream(ctx context.Context, be *eventspb.BusEvent) *TxErr {
	return &TxErr{
		Base: newBaseFromBusEvent(ctx, TxErrEvent, be),
		evt:  be.GetTxErrEvent(),
	}
}
