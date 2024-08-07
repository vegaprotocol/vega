// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package events

import (
	"context"
	"fmt"
	"sort"

	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type TransactionResult struct {
	*Base
	evt *eventspb.TransactionResult
}

func (tr *TransactionResult) PartyID() string {
	return tr.evt.PartyId
}

func (tr *TransactionResult) Status() bool {
	return tr.evt.Status
}

func (tr *TransactionResult) Hash() string {
	return tr.evt.Hash
}

func NewTransactionResultEventSuccess(
	ctx context.Context,
	hash, party string,
	tx interface{},
) *TransactionResult {
	evt := &TransactionResult{
		Base: newBase(ctx, TransactionResultEvent),
		evt: &eventspb.TransactionResult{
			PartyId:      party,
			Hash:         hash,
			Status:       true,
			StatusDetail: eventspb.TransactionResult_STATUS_SUCCESS,
		},
	}

	return evt.setTx(tx)
}

type RawErrors interface {
	GetRawErrors() map[string][]error
}

func makeFailureDetails(err error) *eventspb.TransactionResult_FailureDetails {
	if rawErr, isRawErr := err.(RawErrors); isRawErr {
		keyErrors := []*eventspb.TransactionResult_KeyErrors{}
		for k, v := range rawErr.GetRawErrors() {
			e := &eventspb.TransactionResult_KeyErrors{
				Key: k,
			}

			for _, ve := range v {
				e.Errors = append(e.Errors, ve.Error())
			}

			keyErrors = append(keyErrors, e)
		}

		sort.Slice(keyErrors, func(i, j int) bool {
			return keyErrors[i].Key < keyErrors[j].Key
		})

		return &eventspb.TransactionResult_FailureDetails{
			Errors: keyErrors,
		}
	}

	return &eventspb.TransactionResult_FailureDetails{
		Error: err.Error(),
	}
}

type PartialError interface {
	IsPartial() bool
}

func getErrorStatus(err error) eventspb.TransactionResult_Status {
	if partialErr, isPartialErr := err.(PartialError); isPartialErr {
		if partialErr.IsPartial() {
			return eventspb.TransactionResult_STATUS_PARTIAL_SUCCESS
		}
	}

	return eventspb.TransactionResult_STATUS_FAILURE
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
			PartyId:      party,
			Hash:         hash,
			Status:       false,
			StatusDetail: getErrorStatus(err),
			Extra: &eventspb.TransactionResult_Failure{
				Failure: makeFailureDetails(err),
			},
		},
	}

	return evt.setTx(tx)
}

func (t *TransactionResult) setTx(tx interface{}) *TransactionResult {
	switch tv := tx.(type) {
	case *commandspb.DelayedTransactionsWrapper:
		break
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
		t.evt.Transaction = &eventspb.TransactionResult_StopOrdersSubmission{
			StopOrdersSubmission: tv,
		}
	case *commandspb.StopOrdersCancellation:
		t.evt.Transaction = &eventspb.TransactionResult_StopOrdersCancellation{
			StopOrdersCancellation: tv,
		}
	case *commandspb.CreateReferralSet:
		t.evt.Transaction = &eventspb.TransactionResult_CreateReferralSet{
			CreateReferralSet: tv,
		}
	case *commandspb.UpdateReferralSet:
		t.evt.Transaction = &eventspb.TransactionResult_UpdateReferralSet{
			UpdateReferralSet: tv,
		}
	case *commandspb.ApplyReferralCode:
		t.evt.Transaction = &eventspb.TransactionResult_ApplyReferralCode{
			ApplyReferralCode: tv,
		}
	case *commandspb.UpdateMarginMode:
		t.evt.Transaction = &eventspb.TransactionResult_UpdateMarginMode{
			UpdateMarginMode: tv,
		}
	case *commandspb.JoinTeam:
		t.evt.Transaction = &eventspb.TransactionResult_JoinTeam{
			JoinTeam: tv,
		}
	case *commandspb.BatchProposalSubmission:
		t.evt.Transaction = &eventspb.TransactionResult_BatchProposal{
			BatchProposal: tv,
		}
	case *commandspb.UpdatePartyProfile:
		t.evt.Transaction = &eventspb.TransactionResult_UpdatePartyProfile{
			UpdatePartyProfile: tv,
		}
	case *commandspb.SubmitAMM:
		t.evt.Transaction = &eventspb.TransactionResult_SubmitAmm{
			SubmitAmm: tv,
		}
	case *commandspb.AmendAMM:
		t.evt.Transaction = &eventspb.TransactionResult_AmendAmm{
			AmendAmm: tv,
		}
	case *commandspb.CancelAMM:
		t.evt.Transaction = &eventspb.TransactionResult_CancelAmm{
			CancelAmm: tv,
		}
	default:
		panic(fmt.Sprintf("unsupported command %T", tv))
	}

	return t
}

func (t TransactionResult) IsParty(id string) bool {
	return t.evt.PartyId == id
}

func (t TransactionResult) Proto() eventspb.TransactionResult {
	return *t.evt
}

func (t TransactionResult) TransactionResult() TransactionResult {
	return t
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
