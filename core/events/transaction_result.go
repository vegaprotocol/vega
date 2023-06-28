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

type TransactionResult struct {
	*Base
	evt *eventspb.TransactionResult
}

func NewTransactionResultEventSuccess(
	ctx context.Context,
	hash, party string,
	tx interface{},
) *TransactionResult {
	evt := &TransactionResult{
		Base: newBase(ctx, TransactionResultEvent),
		evt: &eventspb.TransactionResult{
			PartyId: party,
			Hash:    hash,
			Status:  true,
		},
	}

	return evt.setTx(tx)
}

func NewTransactionResultEventFailure(
	ctx context.Context,
	hash, party string,
	err error,
	tx interface{},
) *TransactionResult {
	evt := &TransactionResult{
		Base: newBase(ctx, TransactionResultEvent),
		evt: &eventspb.TransactionResult{
			PartyId: party,
			Hash:    hash,
			Status:  false,
			Extra: &eventspb.TransactionResult_Failure{
				Failure: &eventspb.TransactionResult_FailureDetails{
					Error: err.Error(),
				},
			},
		},
	}

	return evt.setTx(tx)
}

func (t *TransactionResult) setTx(tx interface{}) *TransactionResult {
	switch tv := tx.(type) {
	case *commandspb.OrderSubmission:
		t.evt.Transaction = &eventspb.TransactionResult_OrderSubmission{
			OrderSubmission: tv,
		}
	case *commandspb.OrderCancellation:
		t.evt.Transaction = &eventspb.TransactionResult_OrderCancellation{
			OrderCancellation: tv,
		}
	case *commandspb.OrderAmendment:
		t.evt.Transaction = &eventspb.TransactionResult_OrderAmendment{
			OrderAmendment: tv,
		}
	case *commandspb.VoteSubmission:
		t.evt.Transaction = &eventspb.TransactionResult_VoteSubmission{
			VoteSubmission: tv,
		}
	case *commandspb.WithdrawSubmission:
		t.evt.Transaction = &eventspb.TransactionResult_WithdrawSubmission{
			WithdrawSubmission: tv,
		}
	case *commandspb.LiquidityProvisionSubmission:
		t.evt.Transaction = &eventspb.TransactionResult_LiquidityProvisionSubmission{
			LiquidityProvisionSubmission: tv,
		}
	case *commandspb.LiquidityProvisionCancellation:
		t.evt.Transaction = &eventspb.TransactionResult_LiquidityProvisionCancellation{
			LiquidityProvisionCancellation: tv,
		}
	case *commandspb.LiquidityProvisionAmendment:
		t.evt.Transaction = &eventspb.TransactionResult_LiquidityProvisionAmendment{
			LiquidityProvisionAmendment: tv,
		}
	case *commandspb.ProposalSubmission:
		t.evt.Transaction = &eventspb.TransactionResult_Proposal{
			Proposal: tv,
		}
	case *commandspb.DelegateSubmission:
		t.evt.Transaction = &eventspb.TransactionResult_DelegateSubmission{
			DelegateSubmission: tv,
		}
	case *commandspb.UndelegateSubmission:
		t.evt.Transaction = &eventspb.TransactionResult_UndelegateSubmission{
			UndelegateSubmission: tv,
		}
	case *commandspb.Transfer:
		t.evt.Transaction = &eventspb.TransactionResult_Transfer{
			Transfer: tv,
		}
	case *commandspb.CancelTransfer:
		t.evt.Transaction = &eventspb.TransactionResult_CancelTransfer{
			CancelTransfer: tv,
		}
	case *commandspb.AnnounceNode:
		t.evt.Transaction = &eventspb.TransactionResult_AnnounceNode{
			AnnounceNode: tv,
		}
	case *commandspb.OracleDataSubmission:
		t.evt.Transaction = &eventspb.TransactionResult_OracleDataSubmission{
			OracleDataSubmission: tv,
		}
	case *commandspb.ProtocolUpgradeProposal:
		t.evt.Transaction = &eventspb.TransactionResult_ProtocolUpgradeProposal{
			ProtocolUpgradeProposal: tv,
		}
	case *commandspb.IssueSignatures:
		t.evt.Transaction = &eventspb.TransactionResult_IssueSignatures{
			IssueSignatures: tv,
		}
	case *commandspb.BatchMarketInstructions:
		t.evt.Transaction = &eventspb.TransactionResult_BatchMarketInstructions{
			BatchMarketInstructions: tv,
		}
	case *commandspb.KeyRotateSubmission:
		t.evt.Transaction = &eventspb.TransactionResult_KeyRotateSubmission{
			KeyRotateSubmission: tv,
		}
	case *commandspb.EthereumKeyRotateSubmission:
		t.evt.Transaction = &eventspb.TransactionResult_EthereumKeyRotateSubmission{
			EthereumKeyRotateSubmission: tv,
		}
	case *commandspb.StopOrdersSubmission:
		t.evt.Transaction = &eventspb.TransactionResult_StopOrderSubmission{
			StopOrderSubmission: tv,
		}
	case *commandspb.StopOrdersCancellation:
		t.evt.Transaction = &eventspb.TransactionResult_StopOrderCancellation{
			StopOrderCancellation: tv,
		}
	default:
		panic(fmt.Sprintf("unsupported command: %v", tv))
	}

	return t
}

func (t TransactionResult) IsParty(id string) bool {
	return t.evt.PartyId == id
}

func (t TransactionResult) Proto() eventspb.TransactionResult {
	return *t.evt
}

func (t TransactionResult) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_TransactionResult{
		TransactionResult: t.evt,
	}

	return busEvent
}

func TransactionResultEventFromStream(ctx context.Context, be *eventspb.BusEvent) *TransactionResult {
	return &TransactionResult{
		Base: newBaseFromBusEvent(ctx, TransactionResultEvent, be),
		evt:  be.GetTransactionResult(),
	}
}
