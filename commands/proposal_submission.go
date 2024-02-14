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
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/external/ethcall"
	ethcallcommon "code.vegaprotocol.io/vega/core/datasource/external/ethcall/common"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
	protoTypes "code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

const ReferenceMaxLen int = 100

var validTransfers = map[protoTypes.AccountType]map[protoTypes.AccountType]struct{}{
	protoTypes.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY: {
		protoTypes.AccountType_ACCOUNT_TYPE_GENERAL:                    {},
		protoTypes.AccountType_ACCOUNT_TYPE_GLOBAL_INSURANCE:           {},
		protoTypes.AccountType_ACCOUNT_TYPE_INSURANCE:                  {},
		protoTypes.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD:              {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES:     {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES:    {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES: {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS:    {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION:    {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_RELATIVE_RETURN:     {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY:   {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_VALIDATOR_RANKING:   {},
	},
	protoTypes.AccountType_ACCOUNT_TYPE_INSURANCE: {
		protoTypes.AccountType_ACCOUNT_TYPE_GENERAL:                    {},
		protoTypes.AccountType_ACCOUNT_TYPE_GLOBAL_INSURANCE:           {},
		protoTypes.AccountType_ACCOUNT_TYPE_INSURANCE:                  {},
		protoTypes.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY:           {},
		protoTypes.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD:              {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES:     {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES:    {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES: {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS:    {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION:    {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_RELATIVE_RETURN:     {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY:   {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_VALIDATOR_RANKING:   {},
	},
	protoTypes.AccountType_ACCOUNT_TYPE_GLOBAL_INSURANCE: {
		protoTypes.AccountType_ACCOUNT_TYPE_GENERAL:                    {},
		protoTypes.AccountType_ACCOUNT_TYPE_INSURANCE:                  {},
		protoTypes.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY:           {},
		protoTypes.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD:              {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES:     {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES:    {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES: {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS:    {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION:    {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_RELATIVE_RETURN:     {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY:   {},
		protoTypes.AccountType_ACCOUNT_TYPE_REWARD_VALIDATOR_RANKING:   {},
	},
}

func CheckProposalSubmission(cmd *commandspb.ProposalSubmission) error {
	return checkProposalSubmission(cmd).ErrorOrNil()
}

func checkProposalSubmission(cmd *commandspb.ProposalSubmission) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("proposal_submission", ErrIsRequired)
	}

	if len(cmd.Reference) > ReferenceMaxLen {
		errs.AddForProperty("proposal_submission.reference", ErrReferenceTooLong)
	}

	if cmd.Rationale == nil {
		errs.AddForProperty("proposal_submission.rationale", ErrIsRequired)
	} else {
		if cmd.Rationale != nil {
			if len(strings.Trim(cmd.Rationale.Description, " \n\r\t")) == 0 {
				errs.AddForProperty("proposal_submission.rationale.description", ErrIsRequired)
			} else if len(cmd.Rationale.Description) > 20000 {
				errs.AddForProperty("proposal_submission.rationale.description", ErrMustNotExceed20000Chars)
			}
			if len(strings.Trim(cmd.Rationale.Title, " \n\r\t")) == 0 {
				errs.AddForProperty("proposal_submission.rationale.title", ErrIsRequired)
			} else if len(cmd.Rationale.Title) > 100 {
				errs.AddForProperty("proposal_submission.rationale.title", ErrMustBeLessThan100Chars)
			}
		}
	}

	if cmd.Terms == nil {
		return errs.FinalAddForProperty("proposal_submission.terms", ErrIsRequired)
	}

	if cmd.Terms.ClosingTimestamp <= 0 {
		errs.AddForProperty("proposal_submission.terms.closing_timestamp", ErrMustBePositive)
	}

	if cmd.Terms.ValidationTimestamp < 0 {
		errs.AddForProperty("proposal_submission.terms.validation_timestamp", ErrMustBePositiveOrZero)
	}

	if cmd.Terms.ValidationTimestamp >= cmd.Terms.ClosingTimestamp {
		errs.AddForProperty("proposal_submission.terms.validation_timestamp",
			errors.New("cannot be after or equal to closing time"),
		)
	}

	// check for enactment timestamp
	switch cmd.Terms.Change.(type) {
	case *protoTypes.ProposalTerms_NewFreeform:
		if cmd.Terms.EnactmentTimestamp != 0 {
			errs.AddForProperty("proposal_submission.terms.enactment_timestamp", ErrIsNotSupported)
		}
	default:
		if cmd.Terms.EnactmentTimestamp <= 0 {
			errs.AddForProperty("proposal_submission.terms.enactment_timestamp", ErrMustBePositive)
		}

		if cmd.Terms.ClosingTimestamp > cmd.Terms.EnactmentTimestamp {
			errs.AddForProperty("proposal_submission.terms.closing_timestamp",
				errors.New("cannot be after enactment time"),
			)
		}
	}

	// check for validation timestamp
	switch cmd.Terms.Change.(type) {
	case *protoTypes.ProposalTerms_NewAsset:
		if cmd.Terms.ValidationTimestamp == 0 {
			errs.AddForProperty("proposal_submission.terms.validation_timestamp", ErrMustBePositive)
		}
		if cmd.Terms.ValidationTimestamp > cmd.Terms.ClosingTimestamp {
			errs.AddForProperty("proposal_submission.terms.validation_timestamp",
				errors.New("cannot be after closing time"),
			)
		}
	default:
		if cmd.Terms.ValidationTimestamp != 0 {
			errs.AddForProperty("proposal_submission.terms.validation_timestamp", ErrIsNotSupported)
		}
	}

	errs.Merge(checkProposalChanges(cmd.Terms))

	return errs
}

func checkProposalChanges(terms *protoTypes.ProposalTerms) Errors {
	errs := NewErrors()

	if terms.Change == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change", ErrIsRequired)
	}

	switch c := terms.Change.(type) {
	case *protoTypes.ProposalTerms_NewMarket:
		errs.Merge(checkNewMarketChanges(c))
	case *protoTypes.ProposalTerms_UpdateMarket:
		errs.Merge(checkUpdateMarketChanges(c))
	case *protoTypes.ProposalTerms_NewSpotMarket:
		errs.Merge(checkNewSpotMarketChanges(c))
	case *protoTypes.ProposalTerms_UpdateSpotMarket:
		errs.Merge(checkUpdateSpotMarketChanges(c))
	case *protoTypes.ProposalTerms_UpdateNetworkParameter:
		errs.Merge(checkNetworkParameterUpdateChanges(c))
	case *protoTypes.ProposalTerms_NewAsset:
		errs.Merge(checkNewAssetChanges(c))
	case *protoTypes.ProposalTerms_UpdateAsset:
		errs.Merge(checkUpdateAssetChanges(c))
	case *protoTypes.ProposalTerms_NewFreeform:
		errs.Merge(CheckNewFreeformChanges(c))
	case *protoTypes.ProposalTerms_NewTransfer:
		errs.Merge(checkNewTransferChanges(c))
	case *protoTypes.ProposalTerms_CancelTransfer:
		errs.Merge(checkCancelTransferChanges(c))
	case *protoTypes.ProposalTerms_UpdateMarketState:
		errs.Merge(checkMarketUpdateState(c))
	case *protoTypes.ProposalTerms_UpdateReferralProgram:
		errs.Merge(checkUpdateReferralProgram(terms, c))
	case *protoTypes.ProposalTerms_UpdateVolumeDiscountProgram:
		errs.Merge(checkVolumeDiscountProgram(terms, c))
	default:
		return errs.FinalAddForProperty("proposal_submission.terms.change", ErrIsNotValid)
	}

	return errs
}

func checkNetworkParameterUpdateChanges(change *protoTypes.ProposalTerms_UpdateNetworkParameter) Errors {
	errs := NewErrors()

	if change.UpdateNetworkParameter == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_network_parameter", ErrIsRequired)
	}

	if change.UpdateNetworkParameter.Changes == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_network_parameter.changes", ErrIsRequired)
	}

	return checkNetworkParameterUpdate(change.UpdateNetworkParameter.Changes).AddPrefix("proposal_submission.terms.change.")
}

func checkNetworkParameterUpdate(parameter *vegapb.NetworkParameter) Errors {
	errs := NewErrors()

	if len(parameter.Key) == 0 {
		errs.AddForProperty("update_network_parameter.changes.key", ErrIsRequired)
	}

	if len(parameter.Value) == 0 {
		errs.AddForProperty("update_network_parameter.changes.value", ErrIsRequired)
	}
	return errs
}

func checkNewAssetChanges(change *protoTypes.ProposalTerms_NewAsset) Errors {
	errs := NewErrors()

	if change.NewAsset == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_asset", ErrIsRequired)
	}

	if change.NewAsset.Changes == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_asset.changes", ErrIsRequired)
	}

	if len(change.NewAsset.Changes.Name) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.name", ErrIsRequired)
	}
	if len(change.NewAsset.Changes.Symbol) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.symbol", ErrIsRequired)
	}

	if len(change.NewAsset.Changes.Quantum) <= 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.quantum", ErrIsRequired)
	} else if quantum, err := num.DecimalFromString(change.NewAsset.Changes.Quantum); err != nil {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.quantum", ErrIsNotValidNumber)
	} else if quantum.LessThanOrEqual(num.DecimalZero()) {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.quantum", ErrMustBePositive)
	}

	if change.NewAsset.Changes.Source == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_asset.changes.source", ErrIsRequired)
	}

	switch s := change.NewAsset.Changes.Source.(type) {
	case *protoTypes.AssetDetails_BuiltinAsset:
		errs.Merge(checkBuiltinAssetSource(s))
	case *protoTypes.AssetDetails_Erc20:
		errs.Merge(checkERC20AssetSource(s))
	default:
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_asset.changes.source", ErrIsNotValid)
	}

	return errs
}

func CheckNewFreeformChanges(change *protoTypes.ProposalTerms_NewFreeform) Errors {
	errs := NewErrors()

	if change.NewFreeform == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_freeform", ErrIsRequired)
	}
	return errs
}

func checkCancelTransferChanges(change *protoTypes.ProposalTerms_CancelTransfer) Errors {
	errs := NewErrors()
	if change.CancelTransfer == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.cancel_transfer", ErrIsRequired)
	}

	if change.CancelTransfer.Changes == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.cancel_transfer.changes", ErrIsRequired)
	}

	changes := change.CancelTransfer.Changes
	if len(changes.TransferId) == 0 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.cancel_transfer.changes.transferId", ErrIsRequired)
	}
	return errs
}

func checkUpdateReferralProgram(terms *vegapb.ProposalTerms, change *vegapb.ProposalTerms_UpdateReferralProgram) Errors {
	errs := NewErrors()
	if change.UpdateReferralProgram == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_referral_program", ErrIsRequired)
	}
	if change.UpdateReferralProgram.Changes == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_referral_program.changes", ErrIsRequired)
	}

	return checkReferralProgramChanges(change.UpdateReferralProgram.Changes, terms.EnactmentTimestamp).
		AddPrefix("proposal_submission.terms.change.")
}

func checkReferralProgramChanges(changes *vegapb.ReferralProgramChanges, enactmentTimestamp int64) Errors {
	errs := NewErrors()

	if changes.EndOfProgramTimestamp == 0 {
		errs.AddForProperty("update_referral_program.changes.end_of_program_timestamp", ErrIsRequired)
	} else if changes.EndOfProgramTimestamp < 0 {
		errs.AddForProperty("update_referral_program.changes.end_of_program_timestamp", ErrMustBePositive)
	} else if changes.EndOfProgramTimestamp < enactmentTimestamp {
		errs.AddForProperty("update_referral_program.changes.end_of_program_timestamp", ErrMustBeGreaterThanEnactmentTimestamp)
	}
	if changes.WindowLength == 0 {
		errs.AddForProperty("update_referral_program.changes.window_length", ErrIsRequired)
	} else if changes.WindowLength > 100 {
		errs.AddForProperty("update_referral_program.changes.window_length", ErrMustBeAtMost100)
	}

	tiers := map[string]struct{}{}
	for i, tier := range changes.BenefitTiers {
		errs.Merge(checkBenefitTier(i, tier))
		k := tier.MinimumEpochs + "_" + tier.MinimumRunningNotionalTakerVolume
		if _, ok := tiers[k]; ok {
			errs.AddForProperty(fmt.Sprintf("update_referral_program.changes.benefit_tiers.%d", i), fmt.Errorf("duplicate benefit tier"))
		}
		tiers[k] = struct{}{}
	}

	tiers = map[string]struct{}{}
	for i, tier := range changes.StakingTiers {
		errs.Merge(checkStakingTier(i, tier))
		k := tier.MinimumStakedTokens
		if _, ok := tiers[k]; ok {
			errs.AddForProperty(fmt.Sprintf("update_referral_program.changes.staking_tiers.%d", i), fmt.Errorf("duplicate staking tier"))
		}
		tiers[k] = struct{}{}
	}
	return errs
}

func checkVolumeDiscountProgram(terms *vegapb.ProposalTerms, change *vegapb.ProposalTerms_UpdateVolumeDiscountProgram) Errors {
	errs := NewErrors()
	if change.UpdateVolumeDiscountProgram == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_volume_discount_program", ErrIsRequired)
	}
	if change.UpdateVolumeDiscountProgram.Changes == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_volume_discount_program.changes", ErrIsRequired)
	}

	return checkVolumeDiscountProgramChanges(change.UpdateVolumeDiscountProgram.Changes, terms.EnactmentTimestamp).
		AddPrefix("proposal_submission.terms.change.")
}

func checkVolumeDiscountProgramChanges(changes *vegapb.VolumeDiscountProgramChanges, enactmentTimestamp int64) Errors {
	errs := NewErrors()

	if changes.EndOfProgramTimestamp == 0 {
		errs.AddForProperty("update_volume_discount_program.changes.end_of_program_timestamp", ErrIsRequired)
	} else if changes.EndOfProgramTimestamp < 0 {
		errs.AddForProperty("update_volume_discount_program.changes.end_of_program_timestamp", ErrMustBePositive)
	} else if changes.EndOfProgramTimestamp < enactmentTimestamp {
		errs.AddForProperty("update_volume_discount_program.changes.end_of_program_timestamp", ErrMustBeGreaterThanEnactmentTimestamp)
	}
	if changes.WindowLength == 0 {
		errs.AddForProperty("update_volume_discount_program.changes.window_length", ErrIsRequired)
	} else if changes.WindowLength > 100 {
		errs.AddForProperty("update_volume_discount_program.changes.window_length", ErrMustBeAtMost100)
	}
	for i, tier := range changes.BenefitTiers {
		errs.Merge(checkVolumeBenefitTier(i, tier))
	}

	return errs
}

func checkVolumeBenefitTier(index int, tier *vegapb.VolumeBenefitTier) Errors {
	errs := NewErrors()
	propertyPath := fmt.Sprintf("update_volume_discount_program.changes.benefit_tiers.%d", index)
	if len(tier.MinimumRunningNotionalTakerVolume) == 0 {
		errs.AddForProperty(propertyPath+".minimum_running_notional_taker_volume", ErrIsRequired)
	} else {
		mrtv, overflow := num.UintFromString(tier.MinimumRunningNotionalTakerVolume, 10)
		if overflow {
			errs.AddForProperty(propertyPath+".minimum_running_notional_taker_volume", ErrIsNotValidNumber)
		} else if mrtv.IsNegative() || mrtv.IsZero() {
			errs.AddForProperty(propertyPath+".minimum_running_notional_taker_volume", ErrMustBePositive)
		}
	}
	if len(tier.VolumeDiscountFactor) == 0 {
		errs.AddForProperty(propertyPath+".volume_discount_factor", ErrIsRequired)
	} else {
		rdf, err := num.DecimalFromString(tier.VolumeDiscountFactor)
		if err != nil {
			errs.AddForProperty(propertyPath+".volume_discount_factor", ErrIsNotValidNumber)
		} else if rdf.IsNegative() {
			errs.AddForProperty(propertyPath+".volume_discount_factor", ErrMustBePositiveOrZero)
		}
	}
	return errs
}

func checkBenefitTier(index int, tier *vegapb.BenefitTier) Errors {
	errs := NewErrors()

	propertyPath := fmt.Sprintf("update_referral_program.changes.benefit_tiers.%d", index)

	if len(tier.MinimumRunningNotionalTakerVolume) == 0 {
		errs.AddForProperty(propertyPath+".minimum_running_notional_taker_volume", ErrIsRequired)
	} else {
		mrtv, overflow := num.UintFromString(tier.MinimumRunningNotionalTakerVolume, 10)
		if overflow {
			errs.AddForProperty(propertyPath+".minimum_running_notional_taker_volume", ErrIsNotValidNumber)
		} else if mrtv.IsNegative() || mrtv.IsZero() {
			errs.AddForProperty(propertyPath+".minimum_running_notional_taker_volume", ErrMustBePositive)
		}
	}

	if len(tier.MinimumEpochs) == 0 {
		errs.AddForProperty(propertyPath+".minimum_epochs", ErrIsRequired)
	} else {
		me, overflow := num.UintFromString(tier.MinimumEpochs, 10)
		if overflow {
			errs.AddForProperty(propertyPath+".minimum_epochs", ErrIsNotValidNumber)
		} else if me.IsNegative() || me.IsZero() {
			errs.AddForProperty(propertyPath+".minimum_epochs", ErrMustBePositive)
		}
	}

	if len(tier.ReferralRewardFactor) == 0 {
		errs.AddForProperty(propertyPath+".referral_reward_factor", ErrIsRequired)
	} else {
		rrf, err := num.DecimalFromString(tier.ReferralRewardFactor)
		if err != nil {
			errs.AddForProperty(propertyPath+".referral_reward_factor", ErrIsNotValidNumber)
		} else if rrf.IsNegative() {
			errs.AddForProperty(propertyPath+".referral_reward_factor", ErrMustBePositiveOrZero)
		}
	}

	if len(tier.ReferralDiscountFactor) == 0 {
		errs.AddForProperty(propertyPath+".referral_discount_factor", ErrIsRequired)
	} else {
		rdf, err := num.DecimalFromString(tier.ReferralDiscountFactor)
		if err != nil {
			errs.AddForProperty(propertyPath+".referral_discount_factor", ErrIsNotValidNumber)
		} else if rdf.IsNegative() {
			errs.AddForProperty(propertyPath+".referral_discount_factor", ErrMustBePositiveOrZero)
		}
	}

	return errs
}

func checkStakingTier(index int, tier *vegapb.StakingTier) Errors {
	errs := NewErrors()

	propertyPath := fmt.Sprintf("update_referral_program.changes.staking_tiers.%d", index)

	if len(tier.MinimumStakedTokens) == 0 {
		errs.AddForProperty(propertyPath+".minimum_staked_tokens", ErrIsRequired)
	} else {
		stakedTokens, overflow := num.UintFromString(tier.MinimumStakedTokens, 10)
		if overflow {
			errs.AddForProperty(propertyPath+".minimum_staked_tokens", ErrIsNotValidNumber)
		} else if stakedTokens.IsNegative() || stakedTokens.IsZero() {
			errs.AddForProperty(propertyPath+".minimum_staked_tokens", ErrMustBePositive)
		}
	}

	if len(tier.ReferralRewardMultiplier) == 0 {
		errs.AddForProperty(propertyPath+".referral_reward_multiplier", ErrIsRequired)
	} else {
		rrm, err := num.DecimalFromString(tier.ReferralRewardMultiplier)
		if err != nil {
			errs.AddForProperty(propertyPath+".referral_reward_multiplier", ErrIsNotValidNumber)
		} else if !rrm.GreaterThanOrEqual(num.DecimalOne()) {
			errs.AddForProperty(propertyPath+".referral_reward_multiplier", ErrMustBeGTE1)
		}
	}

	return errs
}

func checkMarketUpdateState(change *protoTypes.ProposalTerms_UpdateMarketState) Errors {
	errs := NewErrors()
	if change.UpdateMarketState == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market_state", ErrIsRequired)
	}
	if change.UpdateMarketState.Changes == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market_state.changes", ErrIsRequired)
	}
	return checkMarketUpdateConfiguration(change.UpdateMarketState.Changes).AddPrefix("proposal_submission.terms.change.")
}

func checkMarketUpdateConfiguration(changes *vegapb.UpdateMarketStateConfiguration) Errors {
	errs := NewErrors()

	if len(changes.MarketId) == 0 {
		return errs.FinalAddForProperty("update_market_state.changes.marketId", ErrIsRequired)
	}
	if changes.UpdateType == 0 {
		return errs.FinalAddForProperty("update_market_state.changes.updateType", ErrIsRequired)
	}
	// if the update type is not terminate, price must be empty
	if changes.UpdateType != vega.MarketStateUpdateType_MARKET_STATE_UPDATE_TYPE_TERMINATE && changes.Price != nil {
		return errs.FinalAddForProperty("update_market_state.changes.price", ErrMustBeEmpty)
	}

	// if termination and price is provided it must be a valid uint
	if changes.UpdateType == vega.MarketStateUpdateType_MARKET_STATE_UPDATE_TYPE_TERMINATE && changes.Price != nil && len(*changes.Price) > 0 {
		n, overflow := num.UintFromString(*changes.Price, 10)
		if overflow || n.IsNegative() {
			return errs.FinalAddForProperty("update_market_state.changes.price", ErrIsNotValid)
		}
	}
	return errs
}

func checkNewTransferChanges(change *protoTypes.ProposalTerms_NewTransfer) Errors {
	errs := NewErrors()
	if change.NewTransfer == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer", ErrIsRequired)
	}

	if change.NewTransfer.Changes == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer.changes", ErrIsRequired)
	}

	return checkNewTransferConfiguration(change.NewTransfer.Changes).AddPrefix("proposal_submission.terms.change.")
}

func checkNewTransferConfiguration(changes *vegapb.NewTransferConfiguration) Errors {
	errs := NewErrors()

	if changes.SourceType == protoTypes.AccountType_ACCOUNT_TYPE_UNSPECIFIED {
		return errs.FinalAddForProperty("new_transfer.changes.source_type", ErrIsRequired)
	}
	validDest, ok := validTransfers[changes.SourceType]
	// source account type may be one of the following:
	if !ok {
		return errs.FinalAddForProperty("new_transfer.changes.source_type", ErrIsNotValid)
	}
	if changes.DestinationType == protoTypes.AccountType_ACCOUNT_TYPE_UNSPECIFIED {
		return errs.FinalAddForProperty("new_transfer.changes.destination_type", ErrIsRequired)
	}

	if _, ok := validDest[changes.DestinationType]; !ok {
		return errs.FinalAddForProperty("new_transfer.changes.destination_type", ErrIsNotValid)
	}
	dest := changes.DestinationType

	// party accounts: check pubkey
	if dest == protoTypes.AccountType_ACCOUNT_TYPE_GENERAL && !IsVegaPublicKey(changes.Destination) {
		errs.AddForProperty("new_transfer.changes.destination", ErrShouldBeAValidVegaPublicKey)
	}

	// insurance account type requires a source, other sources are global
	if changes.SourceType == protoTypes.AccountType_ACCOUNT_TYPE_INSURANCE {
		if len(changes.Source) == 0 {
			return errs.FinalAddForProperty("new_transfer.changes.source", ErrIsNotValid)
		}
		// destination == source
		if dest == changes.SourceType && changes.Source == changes.Destination {
			return errs.FinalAddForProperty("new_transfer.changes.destination", ErrIsNotValid)
		}
	} else if len(changes.Source) > 0 {
		return errs.FinalAddForProperty("new_transfer.changes.source", ErrIsNotValid)
	}

	// global destination accounts == no source
	if (dest == protoTypes.AccountType_ACCOUNT_TYPE_GENERAL ||
		dest == protoTypes.AccountType_ACCOUNT_TYPE_INSURANCE) &&
		len(changes.Destination) == 0 {
		return errs.FinalAddForProperty("new_transfer.changes.destination", ErrIsNotValid)
	}

	if changes.TransferType == protoTypes.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_UNSPECIFIED {
		return errs.FinalAddForProperty("new_transfer.changes.transfer_type", ErrIsRequired)
	}

	if len(changes.Amount) == 0 {
		return errs.FinalAddForProperty("new_transfer.changes.amount", ErrIsRequired)
	}

	n, overflow := num.UintFromString(changes.Amount, 10)
	if overflow || n.IsNegative() {
		return errs.FinalAddForProperty("new_transfer.changes.amount", ErrIsNotValid)
	}

	if len(changes.Asset) == 0 {
		return errs.FinalAddForProperty("new_transfer.changes.asset", ErrIsRequired)
	}

	if len(changes.FractionOfBalance) == 0 {
		return errs.FinalAddForProperty("new_transfer.changes.fraction_of_balance", ErrIsRequired)
	}

	fraction, err := num.DecimalFromString(changes.FractionOfBalance)
	if err != nil {
		return errs.FinalAddForProperty("new_transfer.changes.fraction_of_balance", ErrIsNotValid)
	}
	if !fraction.IsPositive() {
		return errs.FinalAddForProperty("new_transfer.changes.fraction_of_balance", ErrMustBePositive)
	}

	if fraction.GreaterThan(num.DecimalOne()) {
		return errs.FinalAddForProperty("new_transfer.changes.fraction_of_balance", ErrMustBeLTE1)
	}

	if oneoff := changes.GetOneOff(); oneoff != nil {
		if changes.DestinationType == vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES ||
			changes.DestinationType == vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES ||
			changes.DestinationType == vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES ||
			changes.DestinationType == vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS ||
			changes.DestinationType == vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION ||
			changes.DestinationType == vega.AccountType_ACCOUNT_TYPE_REWARD_RELATIVE_RETURN ||
			changes.DestinationType == vega.AccountType_ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY ||
			changes.DestinationType == vega.AccountType_ACCOUNT_TYPE_REWARD_VALIDATOR_RANKING {
			errs.AddForProperty("new_transfer.changes.destination_type", ErrIsNotValid)
		}
		if oneoff.DeliverOn < 0 {
			return errs.FinalAddForProperty("new_transfer.changes.oneoff.deliveron", ErrMustBePositiveOrZero)
		}
	}

	if recurring := changes.GetRecurring(); recurring != nil {
		if recurring.EndEpoch != nil && *recurring.EndEpoch < recurring.StartEpoch {
			return errs.FinalAddForProperty("new_transfer.changes.recurring.end_epoch", ErrIsNotValid)
		}

		if recurring.DispatchStrategy != nil {
			if len(changes.Destination) > 0 {
				errs.AddForProperty("new_transfer.changes.destination", ErrIsNotValid)
			}

			validateDispatchStrategy(changes.DestinationType, recurring.DispatchStrategy, errs, "new_transfer.changes.recurring.dispatch_strategy", "new_transfer.changes.destination_type")
		}
	}

	if changes.GetRecurring() == nil && changes.GetOneOff() == nil {
		return errs.FinalAddForProperty("new_transfer.changes.kind", ErrIsRequired)
	}

	return errs
}

func checkBuiltinAssetSource(s *protoTypes.AssetDetails_BuiltinAsset) Errors {
	errs := NewErrors()

	if s.BuiltinAsset == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_asset.changes.source.builtin_asset", ErrIsRequired)
	}

	asset := s.BuiltinAsset

	if len(asset.MaxFaucetAmountMint) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.builtin_asset.max_faucet_amount_mint", ErrIsRequired)
	} else {
		if maxFaucetAmount, ok := big.NewInt(0).SetString(asset.MaxFaucetAmountMint, 10); !ok {
			return errs.FinalAddForProperty("proposal_submission.terms.change.new_asset.changes.source.builtin_asset.max_faucet_amount_mint", ErrIsNotValidNumber)
		} else if maxFaucetAmount.Cmp(big.NewInt(0)) <= 0 {
			errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.builtin_asset.max_faucet_amount_mint", ErrMustBePositive)
		}
	}

	return errs
}

func checkERC20AssetSource(s *protoTypes.AssetDetails_Erc20) Errors {
	errs := NewErrors()

	if s.Erc20 == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_asset.changes.source.erc20", ErrIsRequired)
	}

	asset := s.Erc20

	if len(asset.ContractAddress) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.erc20.contract_address", ErrIsRequired)
	}
	if len(asset.LifetimeLimit) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.erc20.lifetime_limit", ErrIsRequired)
	} else {
		if lifetimeLimit, ok := big.NewInt(0).SetString(asset.LifetimeLimit, 10); !ok {
			errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.erc20.lifetime_limit", ErrIsNotValidNumber)
		} else {
			if lifetimeLimit.Cmp(big.NewInt(0)) <= 0 {
				errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.erc20.lifetime_limit", ErrMustBePositive)
			}
		}
	}
	if len(asset.WithdrawThreshold) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.erc20.withdraw_threshold", ErrIsRequired)
	} else {
		if withdrawThreshold, ok := big.NewInt(0).SetString(asset.WithdrawThreshold, 10); !ok {
			errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.erc20.withdraw_threshold", ErrIsNotValidNumber)
		} else {
			if withdrawThreshold.Cmp(big.NewInt(0)) <= 0 {
				errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.erc20.withdraw_threshold", ErrMustBePositive)
			}
		}
	}

	return errs
}

func checkUpdateAssetChanges(change *protoTypes.ProposalTerms_UpdateAsset) Errors {
	errs := NewErrors()

	if change.UpdateAsset == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_asset", ErrIsRequired)
	}

	return checkUpdateAsset(change.UpdateAsset).AddPrefix("proposal_submission.terms.change.")
}

func checkUpdateAsset(updateAsset *vegapb.UpdateAsset) Errors {
	errs := NewErrors()

	if len(updateAsset.AssetId) == 0 {
		errs.AddForProperty("update_asset.asset_id", ErrIsRequired)
	} else if !IsVegaID(updateAsset.AssetId) {
		errs.AddForProperty("update_asset.asset_id", ErrShouldBeAValidVegaID)
	}

	if updateAsset.Changes == nil {
		return errs.FinalAddForProperty("update_asset.changes", ErrIsRequired)
	}

	if len(updateAsset.Changes.Quantum) <= 0 {
		errs.AddForProperty("update_asset.changes.quantum", ErrIsRequired)
	} else if quantum, err := num.DecimalFromString(updateAsset.Changes.Quantum); err != nil {
		errs.AddForProperty("update_asset.changes.quantum", ErrIsNotValidNumber)
	} else if quantum.LessThanOrEqual(num.DecimalZero()) {
		errs.AddForProperty("update_asset.changes.quantum", ErrMustBePositive)
	}

	if updateAsset.Changes.Source == nil {
		return errs.FinalAddForProperty("update_asset.changes.source", ErrIsRequired)
	}

	switch s := updateAsset.Changes.Source.(type) {
	case *protoTypes.AssetDetailsUpdate_Erc20:
		errs.Merge(checkERC20UpdateAssetSource(s))
	default:
		return errs.FinalAddForProperty("update_asset.changes.source", ErrIsNotValid)
	}

	return errs
}

func checkERC20UpdateAssetSource(s *protoTypes.AssetDetailsUpdate_Erc20) Errors {
	errs := NewErrors()

	if s.Erc20 == nil {
		return errs.FinalAddForProperty("update_asset.changes.source.erc20", ErrIsRequired)
	}

	asset := s.Erc20

	if len(asset.LifetimeLimit) == 0 {
		errs.AddForProperty("update_asset.changes.source.erc20.lifetime_limit", ErrIsRequired)
	} else {
		if lifetimeLimit, ok := big.NewInt(0).SetString(asset.LifetimeLimit, 10); !ok {
			errs.AddForProperty("update_asset.changes.source.erc20.lifetime_limit", ErrIsNotValidNumber)
		} else {
			if lifetimeLimit.Cmp(big.NewInt(0)) <= 0 {
				errs.AddForProperty("update_asset.changes.source.erc20.lifetime_limit", ErrMustBePositive)
			}
		}
	}

	if len(asset.WithdrawThreshold) == 0 {
		errs.AddForProperty("update_asset.changes.source.erc20.withdraw_threshold", ErrIsRequired)
	} else {
		if withdrawThreshold, ok := big.NewInt(0).SetString(asset.WithdrawThreshold, 10); !ok {
			errs.AddForProperty("update_asset.changes.source.erc20.withdraw_threshold", ErrIsNotValidNumber)
		} else {
			if withdrawThreshold.Cmp(big.NewInt(0)) <= 0 {
				errs.AddForProperty("update_asset.changes.source.erc20.withdraw_threshold", ErrMustBePositive)
			}
		}
	}

	return errs
}

func checkNewSpotMarketChanges(change *protoTypes.ProposalTerms_NewSpotMarket) Errors {
	errs := NewErrors()

	if change.NewSpotMarket == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_spot_market", ErrIsRequired)
	}

	if change.NewSpotMarket.Changes == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_spot_market.changes", ErrIsRequired)
	}

	errs.Merge(checkNewSpotMarketConfiguration(change.NewSpotMarket.Changes).AddPrefix("proposal_submission.terms.change."))
	return errs
}

func checkNewSpotMarketConfiguration(changes *vegapb.NewSpotMarketConfiguration) Errors {
	errs := NewErrors()

	isCorrectProduct := false

	if changes.Instrument == nil {
		return errs.FinalAddForProperty("new_spot_market.changes.instrument", ErrIsRequired)
	}

	if changes.Instrument.Product == nil {
		return errs.FinalAddForProperty("new_spot_market.changes.instrument.product", ErrIsRequired)
	}

	switch changes.Instrument.Product.(type) {
	case *protoTypes.InstrumentConfiguration_Spot:
		isCorrectProduct = true
	default:
		isCorrectProduct = false
	}

	if !isCorrectProduct {
		return errs.FinalAddForProperty("new_spot_market.changes.instrument.product", ErrIsMismatching)
	}
	if changes.DecimalPlaces >= 150 {
		errs.AddForProperty("new_spot_market.changes.decimal_places", ErrMustBeLessThan150)
	}

	if changes.PositionDecimalPlaces >= 7 || changes.PositionDecimalPlaces <= -7 {
		errs.AddForProperty("new_spot_market.changes.position_decimal_places", ErrMustBeWithinRange7)
	}
	errs.Merge(checkPriceMonitoring(changes.PriceMonitoringParameters, "new_spot_market.changes"))
	errs.Merge(checkTargetStakeParams(changes.TargetStakeParameters, "new_spot_market.changes"))
	errs.Merge(checkNewInstrument(changes.Instrument, "new_spot_market.changes.instrument"))
	errs.Merge(checkNewSpotRiskParameters(changes))
	errs.Merge(checkSLAParams(changes.SlaParams, "new_spot_market.changes.sla_params"))

	return errs
}

func checkNewMarketChanges(change *protoTypes.ProposalTerms_NewMarket) Errors {
	errs := NewErrors()

	if change.NewMarket == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market", ErrIsRequired)
	}

	if change.NewMarket.Changes == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes", ErrIsRequired)
	}

	return checkNewMarketChangesConfiguration(change.NewMarket.Changes).AddPrefix("proposal_submission.terms.change.")
}

func checkNewMarketChangesConfiguration(changes *vegapb.NewMarketConfiguration) Errors {
	errs := NewErrors()

	if changes.DecimalPlaces >= 150 {
		errs.AddForProperty("new_market.changes.decimal_places", ErrMustBeLessThan150)
	}

	if changes.PositionDecimalPlaces >= 7 || changes.PositionDecimalPlaces <= -7 {
		errs.AddForProperty("new_market.changes.position_decimal_places", ErrMustBeWithinRange7)
	}

	if len(changes.LinearSlippageFactor) > 0 {
		linearSlippage, err := num.DecimalFromString(changes.LinearSlippageFactor)
		if err != nil {
			errs.AddForProperty("new_market.changes.linear_slippage_factor", ErrIsNotValidNumber)
		} else if linearSlippage.IsNegative() {
			errs.AddForProperty("new_market.changes.linear_slippage_factor", ErrMustBePositiveOrZero)
		} else if linearSlippage.GreaterThan(num.DecimalFromInt64(1000000)) {
			errs.AddForProperty("new_market.changes.linear_slippage_factor", ErrMustBeAtMost1M)
		}
	}

	if successor := changes.Successor; successor != nil {
		if len(successor.InsurancePoolFraction) == 0 {
			errs.AddForProperty("new_market.changes.successor.insurance_pool_fraction", ErrIsRequired)
		} else {
			if ipf, err := num.DecimalFromString(successor.InsurancePoolFraction); err != nil {
				errs.AddForProperty("new_market.changes.successor.insurance_pool_fraction", ErrIsNotValidNumber)
			} else if ipf.IsNegative() || ipf.GreaterThan(num.DecimalFromInt64(1)) {
				errs.AddForProperty("new_market.changes.successor.insurance_pool_fraction", ErrMustBeWithinRange01)
			}
		}
	}

	errs.Merge(checkLiquidationStrategy(changes.LiquidationStrategy, "new_market.changes"))
	errs.Merge(checkPriceMonitoring(changes.PriceMonitoringParameters, "new_market.changes"))
	errs.Merge(checkLiquidityMonitoring(changes.LiquidityMonitoringParameters, "new_market.changes"))
	errs.Merge(checkNewInstrument(changes.Instrument, "new_market.changes.instrument"))
	errs.Merge(checkNewRiskParameters(changes))
	errs.Merge(checkSLAParams(changes.LiquiditySlaParameters, "new_market.changes.sla_params"))
	errs.Merge(checkLiquidityFeeSettings(changes.LiquidityFeeSettings, "new_market.changes.liquidity_fee_settings"))
	errs.Merge(checkCompositePriceConfiguration(changes.MarkPriceConfiguration, "new_market.changes.mark_price_configuration"))
	return errs
}

func checkUpdateMarketChanges(change *protoTypes.ProposalTerms_UpdateMarket) Errors {
	errs := NewErrors()

	if change.UpdateMarket == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market", ErrIsRequired)
	}

	return checkUpdateMarket(change.UpdateMarket).AddPrefix("proposal_submission.terms.change.")
}

func checkUpdateMarket(updateMarket *vegapb.UpdateMarket) Errors {
	errs := NewErrors()

	if len(updateMarket.MarketId) == 0 {
		errs.AddForProperty("update_market.market_id", ErrIsRequired)
	} else if !IsVegaID(updateMarket.MarketId) {
		errs.AddForProperty("update_market.market_id", ErrShouldBeAValidVegaID)
	}

	if updateMarket.Changes == nil {
		return errs.FinalAddForProperty("update_market.changes", ErrIsRequired)
	}

	changes := updateMarket.Changes

	if len(changes.LinearSlippageFactor) > 0 {
		linearSlippage, err := num.DecimalFromString(changes.LinearSlippageFactor)
		if err != nil {
			errs.AddForProperty("update_market.changes.linear_slippage_factor", ErrIsNotValidNumber)
		} else if linearSlippage.IsNegative() {
			errs.AddForProperty("update_market.changes.linear_slippage_factor", ErrMustBePositiveOrZero)
		} else if linearSlippage.GreaterThan(num.DecimalFromInt64(1000000)) {
			errs.AddForProperty("update_market.changes.linear_slippage_factor", ErrMustBeAtMost1M)
		}
	}

	errs.Merge(checkLiquidationStrategy(changes.LiquidationStrategy, "update_market.changes"))
	errs.Merge(checkPriceMonitoring(changes.PriceMonitoringParameters, "update_market.changes"))
	errs.Merge(checkLiquidityMonitoring(changes.LiquidityMonitoringParameters, "update_market.changes"))
	errs.Merge(checkUpdateInstrument(changes.Instrument))
	errs.Merge(checkUpdateRiskParameters(changes))
	errs.Merge(checkSLAParams(changes.LiquiditySlaParameters, "update_market.changes.sla_params"))
	errs.Merge(checkLiquidityFeeSettings(changes.LiquidityFeeSettings, "update_market.changes.liquidity_fee_settings"))
	errs.Merge(checkCompositePriceConfiguration(changes.MarkPriceConfiguration, "update_market.changes.mark_price_configuration"))
	return errs
}

func checkUpdateSpotMarketChanges(change *protoTypes.ProposalTerms_UpdateSpotMarket) Errors {
	errs := NewErrors()

	if change.UpdateSpotMarket == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_spot_market", ErrIsRequired)
	}
	return checkUpdateSpotMarket(change.UpdateSpotMarket).AddPrefix("proposal_submission.terms.change.")
}

func checkUpdateSpotMarket(updateSpotMarket *vegapb.UpdateSpotMarket) Errors {
	errs := NewErrors()

	if len(updateSpotMarket.MarketId) == 0 {
		errs.AddForProperty("update_spot_market.market_id", ErrIsRequired)
	} else if !IsVegaID(updateSpotMarket.MarketId) {
		errs.AddForProperty("update_spot_market.market_id", ErrShouldBeAValidVegaID)
	}

	if updateSpotMarket.Changes == nil {
		return errs.FinalAddForProperty("update_spot_market.changes", ErrIsRequired)
	}

	changes := updateSpotMarket.Changes
	errs.Merge(checkPriceMonitoring(changes.PriceMonitoringParameters, "update_spot_market.changes"))
	errs.Merge(checkTargetStakeParams(changes.TargetStakeParameters, "update_spot_market.changes"))
	errs.Merge(checkUpdateSpotRiskParameters(changes))
	errs.Merge(checkSLAParams(changes.SlaParams, "update_spot_market.changes.sla_params"))
	return errs
}

func checkPriceMonitoring(parameters *protoTypes.PriceMonitoringParameters, parentProperty string) Errors {
	errs := NewErrors()

	if parameters == nil || len(parameters.Triggers) == 0 {
		return errs
	}

	if len(parameters.Triggers) > 5 {
		errs.AddForProperty(fmt.Sprintf("%s.price_monitoring_parameters.triggers", parentProperty), errors.New("maximum 5 triggers allowed"))
	}

	for i, trigger := range parameters.Triggers {
		if trigger.Horizon <= 0 {
			errs.AddForProperty(fmt.Sprintf("%s.price_monitoring_parameters.triggers.%d.horizon", parentProperty, i), ErrMustBePositive)
		}
		if trigger.AuctionExtension <= 0 {
			errs.AddForProperty(fmt.Sprintf("%s.price_monitoring_parameters.triggers.%d.auction_extension", parentProperty, i), ErrMustBePositive)
		}

		probability, err := strconv.ParseFloat(trigger.Probability, 64)
		if err != nil {
			errs.AddForProperty(fmt.Sprintf("%s.price_monitoring_parameters.triggers.%d.probability", parentProperty, i),
				errors.New("must be numeric and be between 0 (exclusive) and 1 (exclusive)"),
			)
		}

		if probability <= 0.9 || probability >= 1 {
			errs.AddForProperty(fmt.Sprintf("%s.price_monitoring_parameters.triggers.%d.probability", parentProperty, i),
				errors.New("should be between 0.9 (exclusive) and 1 (exclusive)"),
			)
		}
	}

	return errs
}

func checkLiquidationStrategy(params *protoTypes.LiquidationStrategy, parent string) Errors {
	errs := NewErrors()
	if params == nil {
		// @TODO these will be required, in that case the check for nil should be removed
		// or return an error.
		return errs
	}
	dispFrac, err := num.DecimalFromString(params.DisposalFraction)
	if err != nil || dispFrac.IsNegative() || dispFrac.IsZero() || dispFrac.GreaterThan(num.DecimalOne()) {
		errs.AddForProperty(fmt.Sprintf("%s.liquidation_strategy.disposal_fraction", parent), ErrMustBeBetween01)
	}
	maxFrac, err := num.DecimalFromString(params.MaxFractionConsumed)
	if err != nil || maxFrac.IsNegative() || maxFrac.IsZero() || maxFrac.GreaterThan(num.DecimalOne()) {
		errs.AddForProperty(fmt.Sprintf("%s.liquidation_strategy.max_fraction_consumed", parent), ErrMustBeBetween01)
	}
	if params.DisposalTimeStep < 1 {
		errs.AddForProperty(fmt.Sprintf("%s.liquidation_strategy.disposal_time_step", parent), ErrMustBePositive)
	} else if params.DisposalTimeStep > 3600 {
		errs.AddForProperty(fmt.Sprintf("%s.liquidation_strategy.disposal_time_step", parent), ErrMustBeAtMost3600)
	}
	return errs
}

func checkLiquidityMonitoring(parameters *protoTypes.LiquidityMonitoringParameters, parentProperty string) Errors {
	errs := NewErrors()

	if parameters == nil {
		return errs.FinalAddForProperty(fmt.Sprintf("%s.liquidity_monitoring_parameters", parentProperty), ErrIsRequired)
	}

	if parameters.TargetStakeParameters == nil {
		return errs.FinalAddForProperty(fmt.Sprintf("%s.liquidity_monitoring_parameters.target_stake_parameters", parentProperty), ErrIsRequired)
	}

	if parameters.TargetStakeParameters.TimeWindow <= 0 {
		errs.AddForProperty(fmt.Sprintf("%s.liquidity_monitoring_parameters.target_stake_parameters.time_window", parentProperty), ErrMustBePositive)
	}
	if parameters.TargetStakeParameters.ScalingFactor <= 0 {
		errs.AddForProperty(fmt.Sprintf("%s.liquidity_monitoring_parameters.target_stake_parameters.scaling_factor", parentProperty), ErrMustBePositive)
	}
	return errs
}

func checkTargetStakeParams(targetStakeParameters *protoTypes.TargetStakeParameters, parentProperty string) Errors {
	errs := NewErrors()
	if targetStakeParameters == nil {
		return errs.FinalAddForProperty(fmt.Sprintf("%s.target_stake_parameters", parentProperty), ErrIsRequired)
	}

	if targetStakeParameters.TimeWindow <= 0 {
		errs.AddForProperty(fmt.Sprintf("%s.target_stake_parameters.time_window", parentProperty), ErrMustBePositive)
	}
	if targetStakeParameters.ScalingFactor <= 0 {
		errs.AddForProperty(fmt.Sprintf("%s.target_stake_parameters.scaling_factor", parentProperty), ErrMustBePositive)
	}
	return errs
}

func checkNewInstrument(instrument *protoTypes.InstrumentConfiguration, parent string) Errors {
	errs := NewErrors()

	if instrument == nil {
		return errs.FinalAddForProperty(parent, ErrIsRequired)
	}

	if len(instrument.Name) == 0 {
		errs.AddForProperty(fmt.Sprintf("%s.name", parent), ErrIsRequired)
	}
	if len(instrument.Code) == 0 {
		errs.AddForProperty(fmt.Sprintf("%s.code", parent), ErrIsRequired)
	}

	if instrument.Product == nil {
		return errs.FinalAddForProperty(fmt.Sprintf("%s.product", parent), ErrIsRequired)
	}

	switch product := instrument.Product.(type) {
	case *protoTypes.InstrumentConfiguration_Future:
		errs.Merge(checkNewFuture(product.Future))
	case *protoTypes.InstrumentConfiguration_Perpetual:
		errs.Merge(checkNewPerps(product.Perpetual, fmt.Sprintf("%s.product", parent)))
	case *protoTypes.InstrumentConfiguration_Spot:
		errs.Merge(checkNewSpot(product.Spot))
	default:
		return errs.FinalAddForProperty(fmt.Sprintf("%s.product", parent), ErrIsNotValid)
	}

	return errs
}

func checkUpdateInstrument(instrument *protoTypes.UpdateInstrumentConfiguration) Errors {
	errs := NewErrors()

	if instrument == nil {
		return errs.FinalAddForProperty("update_market.changes.instrument", ErrIsRequired)
	}

	if len(instrument.Code) == 0 {
		errs.AddForProperty("update_market.changes.instrument.code", ErrIsRequired)
	}

	if len(instrument.Name) == 0 {
		errs.AddForProperty("update_market.changes.instrument.name", ErrIsRequired)
	}

	if instrument.Product == nil {
		return errs.FinalAddForProperty("update_market.changes.instrument.product", ErrIsRequired)
	}

	switch product := instrument.Product.(type) {
	case *protoTypes.UpdateInstrumentConfiguration_Future:
		errs.Merge(checkUpdateFuture(product.Future))
	case *protoTypes.UpdateInstrumentConfiguration_Perpetual:
		errs.Merge(checkUpdatePerps(product.Perpetual, "update_market.changes.instrument.product"))
	default:
		return errs.FinalAddForProperty("update_market.changes.instrument.product", ErrIsNotValid)
	}

	return errs
}

func checkNewFuture(future *protoTypes.FutureProduct) Errors {
	errs := NewErrors()

	if future == nil {
		return errs.FinalAddForProperty("new_market.changes.instrument.product.future", ErrIsRequired)
	}

	if len(future.SettlementAsset) == 0 {
		errs.AddForProperty("new_market.changes.instrument.product.future.settlement_asset", ErrIsRequired)
	}
	if len(future.QuoteName) == 0 {
		errs.AddForProperty("new_market.changes.instrument.product.future.quote_name", ErrIsRequired)
	}

	errs.Merge(checkDataSourceSpec(future.DataSourceSpecForSettlementData, "data_source_spec_for_settlement_data", "new_market.changes.instrument.product.future", true))
	errs.Merge(checkDataSourceSpec(future.DataSourceSpecForTradingTermination, "data_source_spec_for_trading_termination", "new_market.changes.instrument.product.future", false))
	errs.Merge(checkNewOracleBinding(future))

	return errs
}

func checkNewPerps(perps *protoTypes.PerpetualProduct, parentProperty string) Errors {
	errs := NewErrors()

	if perps == nil {
		return errs.FinalAddForProperty(fmt.Sprintf("%s.perps", parentProperty), ErrIsRequired)
	}

	if len(perps.SettlementAsset) == 0 {
		errs.AddForProperty(fmt.Sprintf("%s.perps.settlement_asset", parentProperty), ErrIsRequired)
	}
	if len(perps.QuoteName) == 0 {
		errs.AddForProperty(fmt.Sprintf("%s.perps.quote_name", parentProperty), ErrIsRequired)
	}

	if len(perps.MarginFundingFactor) <= 0 {
		errs.AddForProperty(fmt.Sprintf("%s.perps.margin_funding_factor", parentProperty), ErrIsRequired)
	} else {
		mff, err := num.DecimalFromString(perps.MarginFundingFactor)
		if err != nil {
			errs.AddForProperty(fmt.Sprintf("%s.perps.margin_funding_factor", parentProperty), ErrIsNotValidNumber)
		} else if mff.IsNegative() || mff.GreaterThan(num.DecimalOne()) {
			errs.AddForProperty(fmt.Sprintf("%s.perps.margin_funding_factor", parentProperty), ErrMustBeWithinRange01)
		}
	}

	if len(perps.InterestRate) <= 0 {
		errs.AddForProperty(fmt.Sprintf("%s.perps.interest_rate", parentProperty), ErrIsRequired)
	} else {
		mff, err := num.DecimalFromString(perps.InterestRate)
		if err != nil {
			errs.AddForProperty(fmt.Sprintf("%s.perps.interest_rate", parentProperty), ErrIsNotValidNumber)
		} else if mff.LessThan(num.MustDecimalFromString("-1")) || mff.GreaterThan(num.DecimalOne()) {
			errs.AddForProperty(fmt.Sprintf("%s.perps.interest_rate", parentProperty), ErrMustBeWithinRange11)
		}
	}

	var (
		okClampLowerBound, okClampUpperBound bool
		clampLowerBound, clampUpperBound     num.Decimal
		err                                  error
	)

	if len(perps.ClampLowerBound) <= 0 {
		errs.AddForProperty(fmt.Sprintf("%s.perps.clamp_lower_bound", parentProperty), ErrIsRequired)
	} else {
		clampLowerBound, err = num.DecimalFromString(perps.ClampLowerBound)
		if err != nil {
			errs.AddForProperty(fmt.Sprintf("%s.perps.clamp_lower_bound", parentProperty), ErrIsNotValidNumber)
		} else if clampLowerBound.LessThan(num.MustDecimalFromString("-1")) || clampLowerBound.GreaterThan(num.DecimalOne()) {
			errs.AddForProperty(fmt.Sprintf("%s.perps.clamp_lower_bound", parentProperty), ErrMustBeWithinRange11)
		} else {
			okClampLowerBound = true
		}
	}

	if len(perps.ClampUpperBound) <= 0 {
		errs.AddForProperty(fmt.Sprintf("%s.perps.clamp_upper_bound", parentProperty), ErrIsRequired)
	} else {
		clampUpperBound, err = num.DecimalFromString(perps.ClampUpperBound)
		if err != nil {
			errs.AddForProperty(fmt.Sprintf("%s.perps.clamp_upper_bound", parentProperty), ErrIsNotValidNumber)
		} else if clampUpperBound.LessThan(num.MustDecimalFromString("-1")) || clampUpperBound.GreaterThan(num.DecimalOne()) {
			errs.AddForProperty(fmt.Sprintf("%s.perps.clamp_upper_bound", parentProperty), ErrMustBeWithinRange11)
		} else {
			okClampUpperBound = true
		}
	}

	if okClampLowerBound && okClampUpperBound && clampUpperBound.LessThan(clampLowerBound) {
		errs.AddForProperty(fmt.Sprintf("%s.perps.clamp_upper_bound", parentProperty), ErrMustBeGTEClampLowerBound)
	}

	if perps.FundingRateScalingFactor != nil {
		sf, err := num.DecimalFromString(*perps.FundingRateScalingFactor)
		if err != nil {
			errs.AddForProperty(fmt.Sprintf("%s.perps.funding_rate_scaling_factor", parentProperty), ErrIsNotValidNumber)
		}
		if !sf.IsPositive() {
			errs.AddForProperty(fmt.Sprintf("%s.perps.funding_rate_scaling_factor", parentProperty), ErrMustBePositive)
		}
	}

	var lowerBound, upperBound num.Decimal
	if perps.FundingRateLowerBound != nil {
		if lowerBound, err = num.DecimalFromString(*perps.FundingRateLowerBound); err != nil {
			errs.AddForProperty(fmt.Sprintf("%s.perps.funding_rate_lower_bound", parentProperty), ErrIsNotValidNumber)
		}
	}

	if perps.FundingRateUpperBound != nil {
		if upperBound, err = num.DecimalFromString(*perps.FundingRateUpperBound); err != nil {
			errs.AddForProperty(fmt.Sprintf("%s.perps.funding_rate_upper_bound", parentProperty), ErrIsNotValidNumber)
		}
	}

	if perps.FundingRateLowerBound != nil && perps.FundingRateUpperBound != nil {
		if lowerBound.GreaterThan(upperBound) {
			errs.AddForProperty(fmt.Sprintf("%s.perps.funding_rate_lower_bound", parentProperty), ErrIsNotValid)
		}
	}

	errs.Merge(checkDataSourceSpec(perps.DataSourceSpecForSettlementData, "data_source_spec_for_settlement_data", fmt.Sprintf("%s.perps", parentProperty), true))
	errs.Merge(checkDataSourceSpec(perps.DataSourceSpecForSettlementSchedule, "data_source_spec_for_settlement_schedule", fmt.Sprintf("%s.perps", parentProperty), true))
	errs.Merge(checkNewPerpsOracleBinding(perps))

	if perps.InternalCompositePriceConfiguration != nil {
		errs.Merge(checkCompositePriceConfiguration(perps.InternalCompositePriceConfiguration, fmt.Sprintf("%s.perps.internal_composite_price_configuration", parentProperty)))
	}

	return errs
}

func checkNewSpot(spot *protoTypes.SpotProduct) Errors {
	errs := NewErrors()

	if spot == nil {
		return errs.FinalAddForProperty("new_spot_market.changes.instrument.product.spot", ErrIsRequired)
	}

	if len(spot.BaseAsset) == 0 {
		errs.AddForProperty("new_spot_market.changes.instrument.product.spot.base_asset", ErrIsRequired)
	}
	if len(spot.QuoteAsset) == 0 {
		errs.AddForProperty("new_spot_market.changes.instrument.product.spot.quote_asset", ErrIsRequired)
	}
	if spot.BaseAsset == spot.QuoteAsset {
		errs.AddForProperty("new_spot_market.changes.instrument.product.spot.quote_asset", ErrIsNotValid)
	}
	if len(spot.Name) == 0 {
		errs.AddForProperty("new_spot_market.changes.instrument.product.spot.name", ErrIsRequired)
	}
	return errs
}

func checkUpdateFuture(future *protoTypes.UpdateFutureProduct) Errors {
	errs := NewErrors()

	if future == nil {
		return errs.FinalAddForProperty("update_market.changes.instrument.product.future", ErrIsRequired)
	}

	if len(future.QuoteName) == 0 {
		errs.AddForProperty("update_market.changes.instrument.product.future.quote_name", ErrIsRequired)
	}

	errs.Merge(checkDataSourceSpec(future.DataSourceSpecForSettlementData, "data_source_spec_for_settlement_data", "update_market.changes.instrument.product.future", true))
	errs.Merge(checkDataSourceSpec(future.DataSourceSpecForTradingTermination, "data_source_spec_for_trading_termination", "update_market.changes.instrument.product.future", false))
	errs.Merge(checkUpdateOracleBinding(future))

	return errs
}

func checkUpdatePerps(perps *protoTypes.UpdatePerpetualProduct, parentProperty string) Errors {
	errs := NewErrors()

	if perps == nil {
		return errs.FinalAddForProperty(fmt.Sprintf("%s.perps", parentProperty), ErrIsRequired)
	}

	if len(perps.QuoteName) == 0 {
		errs.AddForProperty(fmt.Sprintf("%s.perps.quote_name", parentProperty), ErrIsRequired)
	}

	if len(perps.MarginFundingFactor) <= 0 {
		errs.AddForProperty(fmt.Sprintf("%s.perps.margin_funding_factor", parentProperty), ErrIsRequired)
	} else {
		mff, err := num.DecimalFromString(perps.MarginFundingFactor)
		if err != nil {
			errs.AddForProperty(fmt.Sprintf("%s.perps.margin_funding_factor", parentProperty), ErrIsNotValidNumber)
		} else if mff.IsNegative() || mff.GreaterThan(num.DecimalOne()) {
			errs.AddForProperty(fmt.Sprintf("%s.perps.margin_funding_factor", parentProperty), ErrMustBeWithinRange01)
		}
	}

	if len(perps.InterestRate) <= 0 {
		errs.AddForProperty(fmt.Sprintf("%s.perps.interest_rate", parentProperty), ErrIsRequired)
	} else {
		mff, err := num.DecimalFromString(perps.InterestRate)
		if err != nil {
			errs.AddForProperty(fmt.Sprintf("%s.perps.interest_rate", parentProperty), ErrIsNotValidNumber)
		} else if mff.LessThan(num.MustDecimalFromString("-1")) || mff.GreaterThan(num.DecimalOne()) {
			errs.AddForProperty(fmt.Sprintf("%s.perps.interest_rate", parentProperty), ErrMustBeWithinRange11)
		}
	}

	var (
		okClampLowerBound, okClampUpperBound bool
		clampLowerBound, clampUpperBound     num.Decimal
		err                                  error
	)

	if len(perps.ClampLowerBound) <= 0 {
		errs.AddForProperty(fmt.Sprintf("%s.perps.clamp_lower_bound", parentProperty), ErrIsRequired)
	} else {
		clampLowerBound, err = num.DecimalFromString(perps.ClampLowerBound)
		if err != nil {
			errs.AddForProperty(fmt.Sprintf("%s.perps.clamp_lower_bound", parentProperty), ErrIsNotValidNumber)
		} else if clampLowerBound.LessThan(num.MustDecimalFromString("-1")) || clampLowerBound.GreaterThan(num.DecimalOne()) {
			errs.AddForProperty(fmt.Sprintf("%s.perps.clamp_lower_bound", parentProperty), ErrMustBeWithinRange11)
		} else {
			okClampLowerBound = true
		}
	}

	if len(perps.ClampUpperBound) <= 0 {
		errs.AddForProperty(fmt.Sprintf("%s.perps.clamp_upper_bound", parentProperty), ErrIsRequired)
	} else {
		clampUpperBound, err = num.DecimalFromString(perps.ClampUpperBound)
		if err != nil {
			errs.AddForProperty(fmt.Sprintf("%s.perps.clamp_upper_bound", parentProperty), ErrIsNotValidNumber)
		} else if clampUpperBound.LessThan(num.MustDecimalFromString("-1")) || clampUpperBound.GreaterThan(num.DecimalOne()) {
			errs.AddForProperty(fmt.Sprintf("%s.perps.clamp_upper_bound", parentProperty), ErrMustBeWithinRange11)
		} else {
			okClampUpperBound = true
		}
	}

	if okClampLowerBound && okClampUpperBound && clampUpperBound.LessThan(clampLowerBound) {
		errs.AddForProperty(fmt.Sprintf("%s.perps.clamp_upper_bound", parentProperty), ErrMustBeGTEClampLowerBound)
	}

	if perps.FundingRateScalingFactor != nil {
		sf, err := num.DecimalFromString(*perps.FundingRateScalingFactor)
		if err != nil {
			errs.AddForProperty(fmt.Sprintf("%s.perps.funding_rate_scaling_factor", parentProperty), ErrIsNotValidNumber)
		}
		if !sf.IsPositive() {
			errs.AddForProperty(fmt.Sprintf("%s.perps.funding_rate_scaling_factor", parentProperty), ErrMustBePositive)
		}
	}

	var lowerBound, upperBound num.Decimal
	if perps.FundingRateLowerBound != nil {
		if lowerBound, err = num.DecimalFromString(*perps.FundingRateLowerBound); err != nil {
			errs.AddForProperty(fmt.Sprintf("%s.perps.funding_rate_lower_bound", parentProperty), ErrIsNotValidNumber)
		}
	}

	if perps.FundingRateUpperBound != nil {
		if upperBound, err = num.DecimalFromString(*perps.FundingRateUpperBound); err != nil {
			errs.AddForProperty(fmt.Sprintf("%s.perps.funding_rate_upper_bound", parentProperty), ErrIsNotValidNumber)
		}
	}

	if perps.FundingRateLowerBound != nil && perps.FundingRateUpperBound != nil {
		if lowerBound.GreaterThan(upperBound) {
			errs.AddForProperty(fmt.Sprintf("%s.perps.funding_rate_lower_bound", parentProperty), ErrIsNotValid)
		}
	}

	errs.Merge(checkDataSourceSpec(perps.DataSourceSpecForSettlementData, "data_source_spec_for_settlement_data", "proposal_submission.terms.change.update_market.changes.instrument.product.future", true))
	errs.Merge(checkDataSourceSpec(perps.DataSourceSpecForSettlementSchedule, "data_source_spec_for_settlement_schedule", "proposal_submission.terms.change.new_market.changes.instrument.product.perps", true))
	errs.Merge(checkUpdatePerpsOracleBinding(perps))

	if perps.InternalCompositePriceConfiguration != nil {
		errs.Merge(checkCompositePriceConfiguration(perps.InternalCompositePriceConfiguration, fmt.Sprintf("%s.perps.internal_composite_price_configuration", parentProperty)))
	}

	return errs
}

func checkDataSourceSpec(spec *vegapb.DataSourceDefinition, name string, parentProperty string, tryToSettle bool,
) Errors {
	errs := NewErrors()
	if spec == nil {
		return errs.FinalAddForProperty(fmt.Sprintf("%s.%s", parentProperty, name), ErrIsRequired)
	}

	if spec.SourceType == nil {
		return errs.FinalAddForProperty(fmt.Sprintf("%s.%s", parentProperty, name+".source_type"), ErrIsRequired)
	}

	switch tp := spec.SourceType.(type) {
	case *vegapb.DataSourceDefinition_Internal:
		switch tp.Internal.SourceType.(type) {
		case *vegapb.DataSourceDefinitionInternal_Time:
			if tryToSettle {
				return errs.FinalAddForProperty(fmt.Sprintf("%s.%s", parentProperty, name), ErrIsNotValid)
			}

			t := tp.Internal.GetTime()
			if t != nil {
				if len(t.Conditions) == 0 {
					errs.AddForProperty(fmt.Sprintf("%s.%s.internal.time.conditions", parentProperty, name), ErrIsRequired)
				}

				if len(t.Conditions) != 0 {
					for j, condition := range t.Conditions {
						if len(condition.Value) == 0 {
							errs.AddForProperty(fmt.Sprintf("%s.%s.internal.time.conditions.%d.value", parentProperty, name, j), ErrIsRequired)
						}
						if condition.Operator == datapb.Condition_OPERATOR_UNSPECIFIED {
							errs.AddForProperty(fmt.Sprintf("%s.%s.internal.time.conditions.%d.operator", parentProperty, name, j), ErrIsRequired)
						}

						if _, ok := datapb.Condition_Operator_name[int32(condition.Operator)]; !ok {
							errs.AddForProperty(fmt.Sprintf("%s.%s.internal.time.conditions.%d.operator", parentProperty, name, j), ErrIsNotValid)
						}
					}
				}
			} else {
				return errs.FinalAddForProperty(fmt.Sprintf("%s.%s.internal.time", parentProperty, name), ErrIsRequired)
			}

		case *vegapb.DataSourceDefinitionInternal_TimeTrigger:
			spl := strings.Split(parentProperty, ".")
			if spl[len(spl)-1] == "future" {
				errs.AddForProperty(fmt.Sprintf("%s.%s.internal.timetrigger", parentProperty, name), ErrIsNotValid)
			}

			t := tp.Internal.GetTimeTrigger()
			if len(t.Triggers) != 1 {
				errs.AddForProperty(fmt.Sprintf("%s.%s.internal.timetrigger", parentProperty, name), ErrOneTimeTriggerAllowedMax)
			} else {
				for i, v := range t.Triggers {
					if v.Initial != nil && *v.Initial <= 0 {
						errs.AddForProperty(fmt.Sprintf("%s.%s.internal.timetrigger.triggers.%d.initial", parentProperty, name, i), ErrIsNotValid)
					}
					if v.Every <= 0 {
						errs.AddForProperty(fmt.Sprintf("%s.%s.internal.timetrigger.triggers.%d.every", parentProperty, name, i), ErrIsNotValid)
					}
				}
			}
		}
	case *vegapb.DataSourceDefinition_External:
		if tp.External == nil {
			errs.AddForProperty(fmt.Sprintf("%s.%s.external", parentProperty, name), ErrIsRequired)
			return errs
		}
		switch tp.External.SourceType.(type) {
		case *vegapb.DataSourceDefinitionExternal_Oracle:
			// If data source type is external - check if the signers are present first.
			o := tp.External.GetOracle()
			if o != nil {
				signers := o.Signers
				if len(signers) == 0 {
					errs.AddForProperty(fmt.Sprintf("%s.%s.external.oracle.signers", parentProperty, name), ErrIsRequired)
				}

				for i, key := range signers {
					signer := dstypes.SignerFromProto(key)
					if signer.IsEmpty() {
						errs.AddForProperty(fmt.Sprintf("%s.%s.external.oracle.signers.%d", parentProperty, name, i), ErrIsNotValid)
					} else if pubkey := signer.GetSignerPubKey(); pubkey != nil && !crypto.IsValidVegaPubKey(pubkey.Key) {
						errs.AddForProperty(fmt.Sprintf("%s.%s.external.oracle.signers.%d", parentProperty, name, i), ErrIsNotValidVegaPubkey)
					} else if address := signer.GetSignerETHAddress(); address != nil && !crypto.EthereumIsValidAddress(address.Address) {
						errs.AddForProperty(fmt.Sprintf("%s.%s.external.oracle.signers.%d", parentProperty, name, i), ErrIsNotValidEthereumAddress)
					}
				}

				filters := o.Filters
				errs.Merge(checkDataSourceSpecFilters(filters, fmt.Sprintf("%s.external.oracle", name), parentProperty))
			} else {
				errs.AddForProperty(fmt.Sprintf("%s.%s.external.oracle", parentProperty, name), ErrIsRequired)
			}
		case *vegapb.DataSourceDefinitionExternal_EthOracle:
			ethOracle := tp.External.GetEthOracle()

			if ethOracle != nil {
				if !crypto.EthereumIsValidAddress(ethOracle.Address) {
					errs.AddForProperty(fmt.Sprintf("%s.%s.external.ethoracle.address", parentProperty, name), ErrIsNotValidEthereumAddress)
				}

				spec, err := ethcallcommon.SpecFromProto(ethOracle)
				if err != nil {
					switch {
					case errors.Is(err, ethcallcommon.ErrCallSpecIsNil):
						errs.AddForProperty(fmt.Sprintf("%s.%s.external.ethoracle", parentProperty, name), ErrEmptyEthereumCallSpec)
					case errors.Is(err, ethcallcommon.ErrInvalidCallTrigger):
						errs.AddForProperty(fmt.Sprintf("%s.%s.external.ethoracle.trigger", parentProperty, name), ErrInvalidEthereumCallTrigger)
					case errors.Is(err, ethcallcommon.ErrInvalidCallArgs):
						errs.AddForProperty(fmt.Sprintf("%s.%s.external.ethoracle.args", parentProperty, name), ErrInvalidEthereumCallArgs)
					default:
						errs.AddForProperty(fmt.Sprintf("%s.%s.external.ethoracle", parentProperty, name), ErrInvalidEthereumCallSpec)
					}
				}

				if _, err := ethcall.NewCall(spec); err != nil {
					switch {
					case errors.Is(err, ethcallcommon.ErrInvalidEthereumAbi):
						errs.AddForProperty(fmt.Sprintf("%s.%s.external.ethoracle.abi", parentProperty, name), ErrInvalidEthereumAbi)
					case errors.Is(err, ethcallcommon.ErrInvalidCallArgs):
						errs.AddForProperty(fmt.Sprintf("%s.%s.external.ethoracle.callargs", parentProperty, name), ErrInvalidEthereumCallArgs)
					case errors.Is(err, ethcallcommon.ErrInvalidFilters):
						errs.AddForProperty(fmt.Sprintf("%s.%s.external.ethoracle.filters", parentProperty, name), ErrInvalidEthereumFilters)
					default:
						errs.AddForProperty(fmt.Sprintf("%s.%s.external.ethoracle", parentProperty, name), ErrInvalidEthereumCallSpec)
					}
				}

				filters := ethOracle.Filters
				errs.Merge(checkDataSourceSpecFilters(filters, fmt.Sprintf("%s.external.ethoracle", name), parentProperty))

				if len(ethOracle.Abi) == 0 {
					errs.AddForProperty(fmt.Sprintf("%s.%s.external.ethoracle.abi", parentProperty, name), ErrIsRequired)
				}

				if len(strings.TrimSpace(ethOracle.Method)) == 0 {
					errs.AddForProperty(fmt.Sprintf("%s.%s.external.ethoracle.method", parentProperty, name), ErrIsRequired)
				}

				if len(ethOracle.Normalisers) == 0 {
					errs.AddForProperty(fmt.Sprintf("%s.%s.external.ethoracle.normalisers", parentProperty, name), ErrIsRequired)
				}

				if ethOracle.Trigger == nil {
					errs.AddForProperty(fmt.Sprintf("%s.%s.external.ethoracle.trigger", parentProperty, name), ErrIsRequired)
				}
			} else {
				errs.AddForProperty(fmt.Sprintf("%s.%s.external.oracle", parentProperty, name), ErrIsRequired)
			}
		}
	}
	return errs
}

func checkDataSourceSpecFilters(filters []*datapb.Filter, name string, parentProperty string) Errors {
	errs := NewErrors()

	if len(filters) == 0 {
		return errs.FinalAddForProperty(fmt.Sprintf("%s.%s.filters", parentProperty, name), ErrIsRequired)
	}

	for i, filter := range filters {
		if filter.Key == nil {
			errs.AddForProperty(fmt.Sprintf("%s.%s.filters.%d.key", parentProperty, name, i), ErrIsNotValid)
		} else {
			if len(filter.Key.Name) == 0 {
				errs.AddForProperty(fmt.Sprintf("%s.%s.filters.%d.key.name", parentProperty, name, i), ErrIsRequired)
			}
			if filter.Key.Type == datapb.PropertyKey_TYPE_UNSPECIFIED {
				errs.AddForProperty(fmt.Sprintf("%s.%s.filters.%d.key.type", parentProperty, name, i), ErrIsRequired)
			}
			if _, ok := datapb.PropertyKey_Type_name[int32(filter.Key.Type)]; !ok {
				errs.AddForProperty(fmt.Sprintf("%s.%s.filters.%d.key.type", parentProperty, name, i), ErrIsNotValid)
			}
		}

		if len(filter.Conditions) != 0 {
			for j, condition := range filter.Conditions {
				if len(condition.Value) == 0 {
					errs.AddForProperty(fmt.Sprintf("%s.%s.filters.%d.conditions.%d.value", parentProperty, name, i, j), ErrIsRequired)
				}
				if condition.Operator == datapb.Condition_OPERATOR_UNSPECIFIED {
					errs.AddForProperty(fmt.Sprintf("%s.%s.filters.%d.conditions.%d.operator", parentProperty, name, i, j), ErrIsRequired)
				}
				if _, ok := datapb.Condition_Operator_name[int32(condition.Operator)]; !ok {
					errs.AddForProperty(fmt.Sprintf("%s.%s.filters.%d.conditions.%d.operator", parentProperty, name, i, j), ErrIsNotValid)
				}
			}
		}
	}

	return errs
}

func isBindingMatchingSpec(spec *vegapb.DataSourceDefinition, bindingProperty string) bool {
	if spec == nil {
		return false
	}

	switch specType := spec.SourceType.(type) {
	case *vegapb.DataSourceDefinition_External:
		switch specType.External.SourceType.(type) {
		case *vegapb.DataSourceDefinitionExternal_Oracle:
			return isBindingMatchingSpecFilters(spec, bindingProperty)
		case *vegapb.DataSourceDefinitionExternal_EthOracle:
			ethOracle := specType.External.GetEthOracle()

			isNormaliser := false

			for _, v := range ethOracle.Normalisers {
				if v.Name == bindingProperty {
					isNormaliser = true
					break
				}
			}

			return isNormaliser || isBindingMatchingSpecFilters(spec, bindingProperty)
		}

	case *vegapb.DataSourceDefinition_Internal:
		return isBindingMatchingSpecFilters(spec, bindingProperty)
	}

	return isBindingMatchingSpecFilters(spec, bindingProperty)
}

// This is the legacy oracles way of checking that the spec has a property matching the binding property by iterating
// over the filters, but is it not possible to not have filters, or a filter that does not match the oracle property so
// this would break?
func isBindingMatchingSpecFilters(spec *vegapb.DataSourceDefinition, bindingProperty string) bool {
	bindingPropertyFound := false
	filters := []*datapb.Filter{}
	if spec != nil {
		filters = spec.GetFilters()
	}
	if spec != nil && filters != nil {
		for _, filter := range filters {
			if filter.Key != nil && filter.Key.Name == bindingProperty {
				bindingPropertyFound = true
			}
		}
	}
	return bindingPropertyFound
}

func checkCompositePriceBinding(binding *vegapb.SpecBindingForCompositePrice, definition *vegapb.DataSourceDefinition, property string) Errors {
	errs := NewErrors()

	if binding == nil {
		errs.AddForProperty(property, ErrIsRequired)
		return errs
	}

	if len(binding.PriceSourceProperty) == 0 {
		errs.AddForProperty(property, ErrIsRequired)
	} else if !isBindingMatchingSpec(definition, binding.PriceSourceProperty) {
		errs.AddForProperty(fmt.Sprintf("%s.price_source_property", property), ErrIsMismatching)
	}
	return errs
}

func checkNewOracleBinding(future *protoTypes.FutureProduct) Errors {
	errs := NewErrors()
	if future.DataSourceSpecBinding != nil {
		if len(future.DataSourceSpecBinding.SettlementDataProperty) == 0 {
			errs.AddForProperty("new_market.changes.instrument.product.future.data_source_spec_binding.settlement_data_property", ErrIsRequired)
		} else {
			if !isBindingMatchingSpec(future.DataSourceSpecForSettlementData, future.DataSourceSpecBinding.SettlementDataProperty) {
				errs.AddForProperty("new_market.changes.instrument.product.future.data_source_spec_binding.settlement_data_property", ErrIsMismatching)
			}
		}

		if len(future.DataSourceSpecBinding.TradingTerminationProperty) == 0 {
			errs.AddForProperty("new_market.changes.instrument.product.future.data_source_spec_binding.trading_termination_property", ErrIsRequired)
		} else {
			if future.DataSourceSpecForTradingTermination == nil || future.DataSourceSpecForTradingTermination.GetExternal() != nil && !isBindingMatchingSpec(future.DataSourceSpecForTradingTermination, future.DataSourceSpecBinding.TradingTerminationProperty) {
				errs.AddForProperty("new_market.changes.instrument.product.future.data_source_spec_binding.trading_termination_property", ErrIsMismatching)
			}
		}
	} else {
		errs.AddForProperty("new_market.changes.instrument.product.future.data_source_spec_binding", ErrIsRequired)
	}

	return errs
}

func checkNewPerpsOracleBinding(perps *protoTypes.PerpetualProduct) Errors {
	errs := NewErrors()

	if perps.DataSourceSpecBinding != nil {
		if len(perps.DataSourceSpecBinding.SettlementDataProperty) == 0 {
			errs.AddForProperty("new_market.changes.instrument.product.perps.data_source_spec_binding.settlement_data_property", ErrIsRequired)
		} else {
			if !isBindingMatchingSpec(perps.DataSourceSpecForSettlementData, perps.DataSourceSpecBinding.SettlementDataProperty) {
				errs.AddForProperty("new_market.changes.instrument.product.perps.data_source_spec_binding.settlement_data_property", ErrIsMismatching)
			}
		}
	} else {
		errs.AddForProperty("new_market.changes.instrument.product.perps.data_source_spec_binding", ErrIsRequired)
	}

	return errs
}

func checkUpdateOracleBinding(future *protoTypes.UpdateFutureProduct) Errors {
	errs := NewErrors()
	if future.DataSourceSpecBinding != nil {
		if len(future.DataSourceSpecBinding.SettlementDataProperty) == 0 {
			errs.AddForProperty("update_market.changes.instrument.product.future.data_source_spec_binding.settlement_data_property", ErrIsRequired)
		} else {
			if !isBindingMatchingSpec(future.DataSourceSpecForSettlementData, future.DataSourceSpecBinding.SettlementDataProperty) {
				errs.AddForProperty("update_market.changes.instrument.product.future.data_source_spec_binding.settlement_data_property", ErrIsMismatching)
			}
		}

		if len(future.DataSourceSpecBinding.TradingTerminationProperty) == 0 {
			errs.AddForProperty("update_market.changes.instrument.product.future.data_source_spec_binding.trading_termination_property", ErrIsRequired)
		} else {
			if !isBindingMatchingSpec(future.DataSourceSpecForTradingTermination, future.DataSourceSpecBinding.TradingTerminationProperty) {
				errs.AddForProperty("update_market.changes.instrument.product.future.data_source_spec_binding.trading_termination_property", ErrIsMismatching)
			}
		}
	} else {
		errs.AddForProperty("update_market.changes.instrument.product.future.data_source_spec_binding", ErrIsRequired)
	}

	return errs
}

func checkUpdatePerpsOracleBinding(perps *protoTypes.UpdatePerpetualProduct) Errors {
	errs := NewErrors()
	if perps.DataSourceSpecBinding != nil {
		if len(perps.DataSourceSpecBinding.SettlementDataProperty) == 0 {
			errs.AddForProperty("update_market.changes.instrument.product.perps.data_source_spec_binding.settlement_data_property", ErrIsRequired)
		} else {
			if !isBindingMatchingSpec(perps.DataSourceSpecForSettlementData, perps.DataSourceSpecBinding.SettlementDataProperty) {
				errs.AddForProperty("update_market.changes.instrument.product.perps.data_source_spec_binding.settlement_data_property", ErrIsMismatching)
			}
		}
	} else {
		errs.AddForProperty("update_market.changes.instrument.product.perps.data_source_spec_binding", ErrIsRequired)
	}

	return errs
}

func checkNewRiskParameters(config *protoTypes.NewMarketConfiguration) Errors {
	errs := NewErrors()

	if config.RiskParameters == nil {
		return errs.FinalAddForProperty("new_market.changes.risk_parameters", ErrIsRequired)
	}

	switch parameters := config.RiskParameters.(type) {
	case *protoTypes.NewMarketConfiguration_Simple:
		errs.Merge(checkNewSimpleParameters(parameters))
	case *protoTypes.NewMarketConfiguration_LogNormal:
		errs.Merge(checkNewLogNormalRiskParameters(parameters))
	default:
		errs.AddForProperty("new_market.changes.risk_parameters", ErrIsNotValid)
	}

	return errs
}

func checkSLAParams(config *protoTypes.LiquiditySLAParameters, parent string) Errors {
	errs := NewErrors()
	if config == nil {
		return errs.FinalAddForProperty(fmt.Sprintf("%s.sla_params", parent), ErrIsRequired)
	}

	lppr, err := num.DecimalFromString(config.PriceRange)
	if err != nil {
		errs.AddForProperty(fmt.Sprintf("%s.price_range", parent), ErrIsNotValidNumber)
	} else if lppr.IsZero() || lppr.LessThan(num.DecimalZero()) || lppr.GreaterThan(num.DecimalFromFloat(20)) {
		errs.AddForProperty(fmt.Sprintf("%s.price_range", parent), ErrMustBeWithinRangeGT0LT20)
	}

	commitmentMinTimeFraction, err := num.DecimalFromString(config.CommitmentMinTimeFraction)
	if err != nil {
		errs.AddForProperty(fmt.Sprintf("%s.commitment_min_time_fraction", parent), ErrIsNotValidNumber)
	} else if commitmentMinTimeFraction.LessThan(num.DecimalZero()) || commitmentMinTimeFraction.GreaterThan(num.DecimalOne()) {
		errs.AddForProperty(fmt.Sprintf("%s.commitment_min_time_fraction", parent), ErrMustBeWithinRange01)
	}

	slaCompetitionFactor, err := num.DecimalFromString(config.SlaCompetitionFactor)
	if err != nil {
		errs.AddForProperty(fmt.Sprintf("%s.sla_competition_factor", parent), ErrIsNotValidNumber)
	} else if slaCompetitionFactor.LessThan(num.DecimalZero()) || slaCompetitionFactor.GreaterThan(num.DecimalOne()) {
		errs.AddForProperty(fmt.Sprintf("%s.sla_competition_factor", parent), ErrMustBeWithinRange01)
	}

	if config.PerformanceHysteresisEpochs > 366 {
		errs.AddForProperty(fmt.Sprintf("%s.performance_hysteresis_epochs", parent), ErrMustBeLessThen366)
	}

	return errs
}

func checkLiquidityFeeSettings(config *protoTypes.LiquidityFeeSettings, parent string) Errors {
	errs := NewErrors()
	if config == nil {
		return nil // no error, we'll default to margin-cost method
	}

	// check for valid enum range
	if config.Method == protoTypes.LiquidityFeeSettings_METHOD_UNSPECIFIED {
		errs.AddForProperty(fmt.Sprintf("%s.method", parent), ErrIsRequired)
	}
	if _, ok := protoTypes.LiquidityFeeSettings_Method_name[int32(config.Method)]; !ok {
		errs.AddForProperty(fmt.Sprintf("%s.method", parent), ErrIsNotValid)
	}

	if config.FeeConstant == nil && config.Method == protoTypes.LiquidityFeeSettings_METHOD_CONSTANT {
		errs.AddForProperty(fmt.Sprintf("%s.fee_constant", parent), ErrIsRequired)
	}

	if config.FeeConstant != nil {
		if config.Method != protoTypes.LiquidityFeeSettings_METHOD_CONSTANT {
			errs.AddForProperty(fmt.Sprintf("%s.method", parent), ErrIsNotValid)
		}

		fee, err := num.DecimalFromString(*config.FeeConstant)
		switch {
		case err != nil:
			errs.AddForProperty(fmt.Sprintf("%s.fee_constant", parent), ErrIsNotValidNumber)
		case fee.IsNegative():
			errs.AddForProperty(fmt.Sprintf("%s.fee_constant", parent), ErrMustBePositiveOrZero)
		case fee.GreaterThan(num.DecimalOne()):
			errs.AddForProperty(fmt.Sprintf("%s.fee_constant", parent), ErrMustBeWithinRange01)
		}
	}

	return errs
}

func checkCompositePriceConfiguration(config *protoTypes.CompositePriceConfiguration, parent string) Errors {
	errs := NewErrors()
	if config == nil {
		errs.AddForProperty(parent, ErrIsRequired)
		return errs
	}
	if config.CompositePriceType == protoTypes.CompositePriceType_COMPOSITE_PRICE_TYPE_UNSPECIFIED {
		errs.AddForProperty(fmt.Sprintf("%s.composite_price_type", parent), ErrIsRequired)
	}

	if _, ok := protoTypes.CompositePriceType_name[int32(config.CompositePriceType)]; !ok {
		errs.AddForProperty(fmt.Sprintf("%s.composite_price_type", parent), ErrIsNotValid)
	}

	if config.CompositePriceType != protoTypes.CompositePriceType_COMPOSITE_PRICE_TYPE_LAST_TRADE {
		if config.DecayPower > 3 || config.DecayPower < 1 {
			errs.AddForProperty(fmt.Sprintf("%s.decay_power", parent), fmt.Errorf("must be in {1, 2, 3}"))
		}
		if len(config.DecayWeight) == 0 {
			errs.AddForProperty(fmt.Sprintf("%s.decay_weight", parent), ErrIsRequired)
		} else {
			dw, err := num.DecimalFromString(config.DecayWeight)
			if err != nil {
				errs.AddForProperty(fmt.Sprintf("%s.decay_weight", parent), ErrIsNotValidNumber)
			} else if dw.LessThan(num.DecimalZero()) || dw.GreaterThan(num.DecimalOne()) {
				errs.AddForProperty(fmt.Sprintf("%s.decay_weight", parent), ErrMustBeWithinRange01)
			}
		}
		if len(config.CashAmount) == 0 {
			errs.AddForProperty(fmt.Sprintf("%s.cash_amount", parent), ErrIsRequired)
		} else if n, overflow := num.UintFromString(config.CashAmount, 10); overflow || n.IsNegative() {
			errs.AddForProperty(fmt.Sprintf("%s.cash_amount", parent), ErrIsNotValidNumber)
		}
	} else {
		if config.DecayPower != 0 {
			errs.AddForProperty(fmt.Sprintf("%s.decay_power", parent), fmt.Errorf("must not be defined for price type last trade"))
		}
		if len(config.DecayWeight) > 0 {
			errs.AddForProperty(fmt.Sprintf("%s.decay_weight", parent), fmt.Errorf("must not be defined for price type last trade"))
		}
		if len(config.CashAmount) > 0 {
			errs.AddForProperty(fmt.Sprintf("%s.cash_amount", parent), fmt.Errorf("must not be defined for price type last trade"))
		}
		if len(config.SourceStalenessTolerance) > 0 {
			errs.AddForProperty(fmt.Sprintf("%s.source_staleness_tolerance", parent), fmt.Errorf("must not be defined for price type last trade"))
		}
		if len(config.SourceWeights) > 0 {
			errs.AddForProperty(fmt.Sprintf("%s.source_weights", parent), fmt.Errorf("must not be defined for price type last trade"))
		}
		if len(config.DataSourcesSpec) > 0 {
			errs.AddForProperty(fmt.Sprintf("%s.data_sources_spec", parent), fmt.Errorf("must not be defined for price type last trade"))
		}
		if len(config.DataSourcesSpec) > 0 {
			errs.AddForProperty(fmt.Sprintf("%s.data_sources_spec_binding", parent), fmt.Errorf("must not be defined for price type last trade"))
		}
	}

	if config.CompositePriceType != protoTypes.CompositePriceType_COMPOSITE_PRICE_TYPE_WEIGHTED && len(config.SourceWeights) > 0 {
		errs.AddForProperty(fmt.Sprintf("%s.source_weights", parent), fmt.Errorf("must be empty if composite price type is not weighted"))
	}

	if config.CompositePriceType == protoTypes.CompositePriceType_COMPOSITE_PRICE_TYPE_WEIGHTED && len(config.SourceWeights) != 3+len(config.DataSourcesSpec) {
		errs.AddForProperty(fmt.Sprintf("%s.source_weights", parent), fmt.Errorf("must be defined for all price sources"))
	}

	if config.CompositePriceType == protoTypes.CompositePriceType_COMPOSITE_PRICE_TYPE_WEIGHTED && len(config.SourceWeights) != len(config.SourceStalenessTolerance) {
		errs.AddForProperty(fmt.Sprintf("%s.source_staleness_tolerance", parent), fmt.Errorf("must have the same length as source_weights"))
	}

	weightSum := num.DecimalZero()
	for i, v := range config.SourceWeights {
		if d, err := num.DecimalFromString(v); err != nil {
			errs.AddForProperty(fmt.Sprintf("%s.source_weights.%d", parent, i), ErrIsNotValidNumber)
		} else if d.LessThan(num.DecimalZero()) {
			errs.AddForProperty(fmt.Sprintf("%s.source_weights.%d", parent, i), ErrMustBePositiveOrZero)
		} else {
			weightSum = weightSum.Add(d)
		}
	}
	if config.CompositePriceType == protoTypes.CompositePriceType_COMPOSITE_PRICE_TYPE_WEIGHTED && weightSum.IsZero() {
		errs.AddForProperty(fmt.Sprintf("%s.source_weights", parent), fmt.Errorf("must have at least one none zero weight"))
	}

	for i, v := range config.SourceStalenessTolerance {
		if _, err := time.ParseDuration(v); err != nil {
			errs.AddForProperty(fmt.Sprintf("%s.source_staleness_tolerance.%d", parent, i), fmt.Errorf("must be a valid duration"))
		}
	}
	if len(config.DataSourcesSpec) > 0 && len(config.DataSourcesSpec) != len(config.DataSourcesSpecBinding) {
		errs.AddForProperty(fmt.Sprintf("%s.data_sources_spec", parent), fmt.Errorf("must be have the same number of elements as the corresponding bindings"))
	}
	if len(config.DataSourcesSpec) > 5 {
		errs.AddForProperty(fmt.Sprintf("%s.data_sources_spec", parent), fmt.Errorf("too many data source specs - must be less than or equal to 5"))
	}
	if config.CompositePriceType != protoTypes.CompositePriceType_COMPOSITE_PRICE_TYPE_LAST_TRADE && len(config.SourceStalenessTolerance) != 3+len(config.DataSourcesSpec) {
		errs.AddForProperty(fmt.Sprintf("%s.source_staleness_tolerance", parent), fmt.Errorf("must included staleness information for all price sources"))
	}

	if config.CompositePriceType == protoTypes.CompositePriceType_COMPOSITE_PRICE_TYPE_LAST_TRADE && len(config.DataSourcesSpec) > 0 {
		errs.AddForProperty(fmt.Sprintf("%s.data_sources_spec", parent), fmt.Errorf("are not supported for last trade composite price type"))
	}
	if len(config.DataSourcesSpec) != len(config.DataSourcesSpecBinding) {
		errs.AddForProperty(fmt.Sprintf("%s.data_sources_spec_binding", parent), fmt.Errorf("must be defined for all oracles"))
	} else if len(config.DataSourcesSpec) > 0 {
		for i, dsd := range config.DataSourcesSpec {
			errs.Merge(checkDataSourceSpec(dsd, fmt.Sprintf("data_sources_spec.%d", i), parent, true))
			errs.Merge(checkCompositePriceBinding(config.DataSourcesSpecBinding[i], dsd, fmt.Sprintf("%s.data_sources_spec_binding.%d", parent, i)))
		}
	}

	return errs
}

func checkNewSpotRiskParameters(config *protoTypes.NewSpotMarketConfiguration) Errors {
	errs := NewErrors()

	if config.RiskParameters == nil {
		return errs.FinalAddForProperty("new_spot_market.changes.risk_parameters", ErrIsRequired)
	}

	switch parameters := config.RiskParameters.(type) {
	case *protoTypes.NewSpotMarketConfiguration_Simple:
		errs.Merge(checkNewSpotSimpleParameters(parameters))
	case *protoTypes.NewSpotMarketConfiguration_LogNormal:
		errs.Merge(checkNewSpotLogNormalRiskParameters(parameters))
	default:
		errs.AddForProperty("new_spot_market.changes.risk_parameters", ErrIsNotValid)
	}

	return errs
}

func checkUpdateRiskParameters(config *protoTypes.UpdateMarketConfiguration) Errors {
	errs := NewErrors()

	if config.RiskParameters == nil {
		return errs.FinalAddForProperty("update_market.changes.risk_parameters", ErrIsRequired)
	}

	switch parameters := config.RiskParameters.(type) {
	case *protoTypes.UpdateMarketConfiguration_Simple:
		errs.Merge(checkUpdateSimpleParameters(parameters))
	case *protoTypes.UpdateMarketConfiguration_LogNormal:
		errs.Merge(checkUpdateLogNormalRiskParameters(parameters))
	default:
		errs.AddForProperty("update_market.changes.risk_parameters", ErrIsNotValid)
	}

	return errs
}

func checkUpdateSpotRiskParameters(config *protoTypes.UpdateSpotMarketConfiguration) Errors {
	errs := NewErrors()

	if config.RiskParameters == nil {
		return errs.FinalAddForProperty("update_spot_market.changes.risk_parameters", ErrIsRequired)
	}

	switch parameters := config.RiskParameters.(type) {
	case *protoTypes.UpdateSpotMarketConfiguration_Simple:
		errs.Merge(checkUpdateSpotSimpleParameters(parameters))
	case *protoTypes.UpdateSpotMarketConfiguration_LogNormal:
		errs.Merge(checkUpdateSpotLogNormalRiskParameters(parameters))
	default:
		errs.AddForProperty("update_spot_market.changes.risk_parameters", ErrIsNotValid)
	}

	return errs
}

func checkNewSimpleParameters(params *protoTypes.NewMarketConfiguration_Simple) Errors {
	errs := NewErrors()

	if params.Simple == nil {
		return errs.FinalAddForProperty("new_market.changes.risk_parameters.simple", ErrIsRequired)
	}

	if params.Simple.MinMoveDown > 0 {
		errs.AddForProperty("new_market.changes.risk_parameters.simple.min_move_down", ErrMustBeNegativeOrZero)
	}

	if params.Simple.MaxMoveUp < 0 {
		errs.AddForProperty("new_market.changes.risk_parameters.simple.max_move_up", ErrMustBePositiveOrZero)
	}

	if params.Simple.ProbabilityOfTrading < 0 || params.Simple.ProbabilityOfTrading > 1 {
		errs.AddForProperty("new_market.changes.risk_parameters.simple.probability_of_trading",
			fmt.Errorf("should be between 0 (inclusive) and 1 (inclusive)"),
		)
	}

	return errs
}

func checkNewSpotSimpleParameters(params *protoTypes.NewSpotMarketConfiguration_Simple) Errors {
	errs := NewErrors()

	if params.Simple == nil {
		return errs.FinalAddForProperty("new_spot_market.changes.risk_parameters.simple", ErrIsRequired)
	}

	if params.Simple.MinMoveDown > 0 {
		errs.AddForProperty("new_spot_market.changes.risk_parameters.simple.min_move_down", ErrMustBeNegativeOrZero)
	}

	if params.Simple.MaxMoveUp < 0 {
		errs.AddForProperty("new_spot_market.changes.risk_parameters.simple.max_move_up", ErrMustBePositiveOrZero)
	}

	if params.Simple.ProbabilityOfTrading < 0 || params.Simple.ProbabilityOfTrading > 1 {
		errs.AddForProperty("new_spot_market.changes.risk_parameters.simple.probability_of_trading",
			fmt.Errorf("should be between 0 (inclusive) and 1 (inclusive)"),
		)
	}

	return errs
}

func checkUpdateSimpleParameters(params *protoTypes.UpdateMarketConfiguration_Simple) Errors {
	errs := NewErrors()

	if params.Simple == nil {
		return errs.FinalAddForProperty("update_market.changes.risk_parameters.simple", ErrIsRequired)
	}

	if params.Simple.MinMoveDown > 0 {
		errs.AddForProperty("update_market.changes.risk_parameters.simple.min_move_down", ErrMustBeNegativeOrZero)
	}

	if params.Simple.MaxMoveUp < 0 {
		errs.AddForProperty("update_market.changes.risk_parameters.simple.max_move_up", ErrMustBePositiveOrZero)
	}

	if params.Simple.ProbabilityOfTrading < 0 || params.Simple.ProbabilityOfTrading > 1 {
		errs.AddForProperty("update_market.changes.risk_parameters.simple.probability_of_trading",
			fmt.Errorf("should be between 0 (inclusive) and 1 (inclusive)"),
		)
	}

	return errs
}

func checkUpdateSpotSimpleParameters(params *protoTypes.UpdateSpotMarketConfiguration_Simple) Errors {
	errs := NewErrors()

	if params.Simple == nil {
		return errs.FinalAddForProperty("update_spot_market.changes.risk_parameters.simple", ErrIsRequired)
	}

	if params.Simple.MinMoveDown > 0 {
		errs.AddForProperty("update_spot_market.changes.risk_parameters.simple.min_move_down", ErrMustBeNegativeOrZero)
	}

	if params.Simple.MaxMoveUp < 0 {
		errs.AddForProperty("update_spot_market.changes.risk_parameters.simple.max_move_up", ErrMustBePositiveOrZero)
	}

	if params.Simple.ProbabilityOfTrading < 0 || params.Simple.ProbabilityOfTrading > 1 {
		errs.AddForProperty("update_spot_market.changes.risk_parameters.simple.probability_of_trading",
			fmt.Errorf("should be between 0 (inclusive) and 1 (inclusive)"),
		)
	}

	return errs
}

func checkNewLogNormalRiskParameters(params *protoTypes.NewMarketConfiguration_LogNormal) Errors {
	errs := NewErrors()

	if params.LogNormal == nil {
		return errs.FinalAddForProperty("new_market.changes.risk_parameters.log_normal", ErrIsRequired)
	}

	if params.LogNormal.Params == nil {
		return errs.FinalAddForProperty("new_market.changes.risk_parameters.log_normal.params", ErrIsRequired)
	}

	if params.LogNormal.RiskAversionParameter < 1e-8 || params.LogNormal.RiskAversionParameter > 0.1 {
		return errs.FinalAddForProperty("new_market.changes.risk_parameters.log_normal.risk_aversion_parameter", errors.New("must be between [1e-8, 0.1]"))
	}

	if params.LogNormal.Tau < 1e-8 || params.LogNormal.Tau > 1 {
		return errs.FinalAddForProperty("new_market.changes.risk_parameters.log_normal.tau", errors.New("must be between [1e-8, 1]"))
	}

	if math.IsNaN(params.LogNormal.Params.Mu) {
		return errs.FinalAddForProperty("new_market.changes.risk_parameters.log_normal.params.mu", ErrIsNotValidNumber)
	}

	if params.LogNormal.Params.Mu < -1e-6 || params.LogNormal.Params.Mu > 1e-6 {
		return errs.FinalAddForProperty("new_market.changes.risk_parameters.log_normal.params.mu", errors.New("must be between [-1e-6,1e-6]"))
	}

	if math.IsNaN(params.LogNormal.Params.Sigma) {
		return errs.FinalAddForProperty("new_market.changes.risk_parameters.log_normal.params.sigma", ErrIsNotValidNumber)
	}

	if params.LogNormal.Params.Sigma < 1e-3 || params.LogNormal.Params.Sigma > 50 {
		return errs.FinalAddForProperty("new_market.changes.risk_parameters.log_normal.params.sigma", errors.New("must be between [1e-3,50]"))
	}

	if math.IsNaN(params.LogNormal.Params.R) {
		return errs.FinalAddForProperty("new_market.changes.risk_parameters.log_normal.params.r", ErrIsNotValidNumber)
	}

	if params.LogNormal.Params.R < -1 || params.LogNormal.Params.R > 1 {
		return errs.FinalAddForProperty("new_market.changes.risk_parameters.log_normal.params.r", errors.New("must be between [-1,1]"))
	}

	return errs
}

func checkUpdateLogNormalRiskParameters(params *protoTypes.UpdateMarketConfiguration_LogNormal) Errors {
	errs := NewErrors()

	if params.LogNormal == nil {
		return errs.FinalAddForProperty("update_market.changes.risk_parameters.log_normal", ErrIsRequired)
	}

	if params.LogNormal.Params == nil {
		return errs.FinalAddForProperty("update_market.changes.risk_parameters.log_normal.params", ErrIsRequired)
	}

	if params.LogNormal.RiskAversionParameter <= 0 {
		return errs.FinalAddForProperty("update_market.changes.risk_parameters.log_normal.risk_aversion_parameter", ErrMustBePositive)
	}

	if params.LogNormal.Tau <= 0 {
		return errs.FinalAddForProperty("update_market.changes.risk_parameters.log_normal.tau", ErrMustBePositive)
	}

	if math.IsNaN(params.LogNormal.Params.Mu) {
		return errs.FinalAddForProperty("update_market.changes.risk_parameters.log_normal.params.mu", ErrIsNotValidNumber)
	}

	if math.IsNaN(params.LogNormal.Params.Sigma) {
		return errs.FinalAddForProperty("update_market.changes.risk_parameters.log_normal.params.sigma", ErrIsNotValidNumber)
	}

	if params.LogNormal.Params.Sigma <= 0 {
		return errs.FinalAddForProperty("update_market.changes.risk_parameters.log_normal.params.sigma", ErrMustBePositive)
	}

	if math.IsNaN(params.LogNormal.Params.R) {
		return errs.FinalAddForProperty("update_market.changes.risk_parameters.log_normal.params.r", ErrIsNotValidNumber)
	}

	return errs
}

func checkNewSpotLogNormalRiskParameters(params *protoTypes.NewSpotMarketConfiguration_LogNormal) Errors {
	errs := NewErrors()

	if params.LogNormal == nil {
		return errs.FinalAddForProperty("new_spot_market.changes.risk_parameters.log_normal", ErrIsRequired)
	}

	if params.LogNormal.Params == nil {
		return errs.FinalAddForProperty("new_spot_market.changes.risk_parameters.log_normal.params", ErrIsRequired)
	}

	if params.LogNormal.RiskAversionParameter < 1e-8 || params.LogNormal.RiskAversionParameter > 0.1 {
		return errs.FinalAddForProperty("new_spot_market.changes.risk_parameters.log_normal.risk_aversion_parameter", errors.New("must be between [1e-8, 0.1]"))
	}

	if params.LogNormal.Tau < 1e-8 || params.LogNormal.Tau > 1 {
		return errs.FinalAddForProperty("new_spot_market.changes.risk_parameters.log_normal.tau", errors.New("must be between [1e-8, 1]"))
	}

	if math.IsNaN(params.LogNormal.Params.Mu) {
		return errs.FinalAddForProperty("new_spot_market.changes.risk_parameters.log_normal.params.mu", ErrIsNotValidNumber)
	}

	if params.LogNormal.Params.Mu < -1e-6 || params.LogNormal.Params.Mu > 1e-6 {
		return errs.FinalAddForProperty("new_spot_market.changes.risk_parameters.log_normal.params.mu", errors.New("must be between [-1e-6,1e-6]"))
	}

	if math.IsNaN(params.LogNormal.Params.Sigma) {
		return errs.FinalAddForProperty("new_spot_market.changes.risk_parameters.log_normal.params.sigma", ErrIsNotValidNumber)
	}

	if params.LogNormal.Params.Sigma < 1e-3 || params.LogNormal.Params.Sigma > 50 {
		return errs.FinalAddForProperty("new_spot_market.changes.risk_parameters.log_normal.params.sigma", errors.New("must be between [1e-3,50]"))
	}

	if math.IsNaN(params.LogNormal.Params.R) {
		return errs.FinalAddForProperty("new_spot_market.changes.risk_parameters.log_normal.params.r", ErrIsNotValidNumber)
	}

	if params.LogNormal.Params.R < -1 || params.LogNormal.Params.R > 1 {
		return errs.FinalAddForProperty("new_spot_market.changes.risk_parameters.log_normal.params.r", errors.New("must be between [-1,1]"))
	}

	return errs
}

func checkUpdateSpotLogNormalRiskParameters(params *protoTypes.UpdateSpotMarketConfiguration_LogNormal) Errors {
	errs := NewErrors()

	if params.LogNormal == nil {
		return errs.FinalAddForProperty("update_spot_market.changes.risk_parameters.log_normal", ErrIsRequired)
	}

	if params.LogNormal.Params == nil {
		return errs.FinalAddForProperty("update_spot_market.changes.risk_parameters.log_normal.params", ErrIsRequired)
	}

	if params.LogNormal.RiskAversionParameter <= 0 {
		return errs.FinalAddForProperty("update_spot_market.changes.risk_parameters.log_normal.risk_aversion_parameter", ErrMustBePositive)
	}

	if params.LogNormal.Tau <= 0 {
		return errs.FinalAddForProperty("update_spot_market.changes.risk_parameters.log_normal.tau", ErrMustBePositive)
	}

	if math.IsNaN(params.LogNormal.Params.Mu) {
		return errs.FinalAddForProperty("update_spot_market.changes.risk_parameters.log_normal.params.mu", ErrIsNotValidNumber)
	}

	if math.IsNaN(params.LogNormal.Params.Sigma) {
		return errs.FinalAddForProperty("update_spot_market.changes.risk_parameters.log_normal.params.sigma", ErrIsNotValidNumber)
	}

	if params.LogNormal.Params.Sigma <= 0 {
		return errs.FinalAddForProperty("update_spot_market.changes.risk_parameters.log_normal.params.sigma", ErrMustBePositive)
	}

	if math.IsNaN(params.LogNormal.Params.R) {
		return errs.FinalAddForProperty("update_spot_market.changes.risk_parameters.log_normal.params.r", ErrIsNotValidNumber)
	}

	return errs
}
