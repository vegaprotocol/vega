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

package commands

import (
	"errors"
	"strings"

	protoTypes "code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckBatchProposalSubmission(cmd *commandspb.BatchProposalSubmission) error {
	return checkBatchProposalSubmission(cmd).ErrorOrNil()
}

func checkBatchProposalSubmission(cmd *commandspb.BatchProposalSubmission) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("batch_proposal_submission", ErrIsRequired)
	}

	if len(cmd.Reference) > ReferenceMaxLen {
		errs.AddForProperty("batch_proposal_submission.reference", ErrReferenceTooLong)
	}

	if cmd.Rationale == nil {
		errs.AddForProperty("batch_proposal_submission.rationale", ErrIsRequired)
	} else {
		if cmd.Rationale != nil {
			if len(strings.Trim(cmd.Rationale.Description, " \n\r\t")) == 0 {
				errs.AddForProperty("batch_proposal_submission.rationale.description", ErrIsRequired)
			} else if len(cmd.Rationale.Description) > 20000 {
				errs.AddForProperty("batch_proposal_submission.rationale.description", ErrMustNotExceed20000Chars)
			}
			if len(strings.Trim(cmd.Rationale.Title, " \n\r\t")) == 0 {
				errs.AddForProperty("batch_proposal_submission.rationale.title", ErrIsRequired)
			} else if len(cmd.Rationale.Title) > 100 {
				errs.AddForProperty("batch_proposal_submission.rationale.title", ErrMustBeLessThan100Chars)
			}
		}
	}

	if cmd.Terms == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms", ErrIsRequired)
	}

	if len(cmd.Terms.Changes) == 0 {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes", ErrIsRequired)
	}

	if cmd.Terms.ClosingTimestamp <= 0 {
		errs.AddForProperty("batch_proposal_submission.terms.closing_timestamp", ErrMustBePositive)
	}

	for _, batchChange := range cmd.Terms.Changes {
		// check for enactment timestamp
		switch batchChange.Change.(type) {
		case *protoTypes.BatchProposalTermsChange_NewFreeform:
			if batchChange.EnactmentTimestamp != 0 {
				errs.AddForProperty("batch_proposal_submission.terms.enactment_timestamp", ErrIsNotSupported)
			}
		default:
			if batchChange.EnactmentTimestamp <= 0 {
				errs.AddForProperty("batch_proposal_submission.terms.enactment_timestamp", ErrMustBePositive)
			}

			if cmd.Terms.ClosingTimestamp > batchChange.EnactmentTimestamp {
				errs.AddForProperty("batch_proposal_submission.terms.closing_timestamp",
					errors.New("cannot be after enactment time"),
				)
			}
		}

		errs.Merge(checkBatchProposalChanges(batchChange))
	}

	return errs
}

func checkBatchProposalChanges(terms *protoTypes.BatchProposalTermsChange) Errors {
	errs := NewErrors()

	if terms.Change == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes", ErrIsRequired)
	}

	switch c := terms.Change.(type) {
	case *protoTypes.BatchProposalTermsChange_NewMarket:
		errs.Merge(checkNewMarketBatchChanges(c))
	case *protoTypes.BatchProposalTermsChange_UpdateMarket:
		errs.Merge(checkUpdateMarketBatchChanges(c))
	case *protoTypes.BatchProposalTermsChange_NewSpotMarket:
		errs.Merge(checkNewSpotMarketBatchChanges(c))
	case *protoTypes.BatchProposalTermsChange_UpdateSpotMarket:
		errs.Merge(checkUpdateSpotMarketBatchChanges(c))
	case *protoTypes.BatchProposalTermsChange_UpdateNetworkParameter:
		errs.Merge(checkNetworkParameterUpdateBatchChanges(c))
	case *protoTypes.BatchProposalTermsChange_UpdateAsset:
		errs.Merge(checkUpdateAssetBatchChanges(c))
	case *protoTypes.BatchProposalTermsChange_NewFreeform:
		errs.Merge(checkNewFreeformBatchChanges(c))
	case *protoTypes.BatchProposalTermsChange_NewTransfer:
		errs.Merge(checkNewTransferBatchChanges(c))
	case *protoTypes.BatchProposalTermsChange_CancelTransfer:
		errs.Merge(checkCancelTransferBatchChanges(c))
	case *protoTypes.BatchProposalTermsChange_UpdateMarketState:
		errs.Merge(checkMarketUpdateStateBatch(c))
	case *protoTypes.BatchProposalTermsChange_UpdateReferralProgram:
		errs.Merge(checkUpdateReferralProgramBatch(c, terms.EnactmentTimestamp))
	case *protoTypes.BatchProposalTermsChange_UpdateVolumeDiscountProgram:
		errs.Merge(checkVolumeDiscountProgramBatch(c, terms.EnactmentTimestamp))
	default:
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes", ErrIsNotValid)
	}

	return errs
}

func checkNewMarketBatchChanges(change *protoTypes.BatchProposalTermsChange_NewMarket) Errors {
	errs := NewErrors()

	if change.NewMarket == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.new_market", ErrIsRequired)
	}

	if change.NewMarket.Changes == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.new_market.changes", ErrIsRequired)
	}

	return checkNewMarketChangesConfiguration(change.NewMarket.Changes).AddPrefix("batch_proposal_submission.terms.changes.")
}

func checkUpdateMarketBatchChanges(change *protoTypes.BatchProposalTermsChange_UpdateMarket) Errors {
	errs := NewErrors()

	if change.UpdateMarket == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.update_market", ErrIsRequired)
	}

	return checkUpdateMarket(change.UpdateMarket).AddPrefix("batch_proposal_submission.terms.changes.")
}

func checkNewSpotMarketBatchChanges(change *protoTypes.BatchProposalTermsChange_NewSpotMarket) Errors {
	errs := NewErrors()

	if change.NewSpotMarket == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.new_spot_market", ErrIsRequired)
	}

	if change.NewSpotMarket.Changes == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.new_spot_market.changes", ErrIsRequired)
	}

	return checkNewSpotMarketConfiguration(change.NewSpotMarket.Changes).AddPrefix("batch_proposal_submission.terms.changes.")
}

func checkUpdateSpotMarketBatchChanges(change *protoTypes.BatchProposalTermsChange_UpdateSpotMarket) Errors {
	errs := NewErrors()

	if change.UpdateSpotMarket == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.update_spot_market", ErrIsRequired)
	}

	return checkUpdateSpotMarket(change.UpdateSpotMarket).AddPrefix("batch_proposal_submission.terms.changes.")
}

func checkNetworkParameterUpdateBatchChanges(change *protoTypes.BatchProposalTermsChange_UpdateNetworkParameter) Errors {
	errs := NewErrors()

	if change.UpdateNetworkParameter == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.update_network_parameter", ErrIsRequired)
	}

	if change.UpdateNetworkParameter.Changes == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.update_network_parameter.changes", ErrIsRequired)
	}

	return checkNetworkParameterUpdate(change.UpdateNetworkParameter.Changes).AddPrefix("batch_proposal_submission.terms.changes.")
}

func checkUpdateAssetBatchChanges(change *protoTypes.BatchProposalTermsChange_UpdateAsset) Errors {
	errs := NewErrors()

	if change.UpdateAsset == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.update_asset", ErrIsRequired)
	}

	return checkUpdateAsset(change.UpdateAsset).AddPrefix("batch_proposal_submission.terms.changes.")
}

func checkNewFreeformBatchChanges(change *protoTypes.BatchProposalTermsChange_NewFreeform) Errors {
	errs := NewErrors()

	if change.NewFreeform == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.new_freeform", ErrIsRequired)
	}
	return errs
}

func checkNewTransferBatchChanges(change *protoTypes.BatchProposalTermsChange_NewTransfer) Errors {
	errs := NewErrors()
	if change.NewTransfer == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.new_transfer", ErrIsRequired)
	}

	if change.NewTransfer.Changes == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.new_transfer.changes", ErrIsRequired)
	}

	return checkNewTransferConfiguration(change.NewTransfer.Changes).AddPrefix("batch_proposal_submission.terms.changes")
}

func checkCancelTransferBatchChanges(change *protoTypes.BatchProposalTermsChange_CancelTransfer) Errors {
	errs := NewErrors()
	if change.CancelTransfer == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.cancel_transfer", ErrIsRequired)
	}

	if change.CancelTransfer.Changes == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.cancel_transfer.changes", ErrIsRequired)
	}

	changes := change.CancelTransfer.Changes
	if len(changes.TransferId) == 0 {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.cancel_transfer.changes.transferId", ErrIsRequired)
	}
	return errs
}

func checkMarketUpdateStateBatch(change *protoTypes.BatchProposalTermsChange_UpdateMarketState) Errors {
	errs := NewErrors()
	if change.UpdateMarketState == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.update_market_state", ErrIsRequired)
	}
	if change.UpdateMarketState.Changes == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.update_market_state.changes", ErrIsRequired)
	}
	return checkMarketUpdateConfiguration(change.UpdateMarketState.Changes).AddPrefix("batch_proposal_submission.terms.changes.")
}

func checkUpdateReferralProgramBatch(change *vegapb.BatchProposalTermsChange_UpdateReferralProgram, enactmentTimestamp int64) Errors {
	errs := NewErrors()
	if change.UpdateReferralProgram == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.update_referral_program", ErrIsRequired)
	}
	if change.UpdateReferralProgram.Changes == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.update_referral_program.changes", ErrIsRequired)
	}

	return checkReferralProgramChanges(change.UpdateReferralProgram.Changes, enactmentTimestamp).
		AddPrefix("batch_proposal_submission.terms.changes.")
}

func checkVolumeDiscountProgramBatch(change *vegapb.BatchProposalTermsChange_UpdateVolumeDiscountProgram, enactmentTimestamp int64) Errors {
	errs := NewErrors()
	if change.UpdateVolumeDiscountProgram == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.update_volume_discount_program", ErrIsRequired)
	}
	if change.UpdateVolumeDiscountProgram.Changes == nil {
		return errs.FinalAddForProperty("batch_proposal_submission.terms.changes.update_volume_discount_program.changes", ErrIsRequired)
	}

	return checkVolumeDiscountProgramChanges(change.UpdateVolumeDiscountProgram.Changes, enactmentTimestamp).
		AddPrefix("batch_proposal_submission.terms.changes.")
}
