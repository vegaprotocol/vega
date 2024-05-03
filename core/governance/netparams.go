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

package governance

import (
	"errors"

	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
)

var (
	ErrEmptyNetParamKey   = errors.New("empty network parameter key")
	ErrEmptyNetParamValue = errors.New("empty network parameter value")
)

func (e *Engine) getNewSpotMarketProposalParameters() *types.ProposalParameters {
	return e.getProposalParametersFromNetParams(
		netparams.GovernanceProposalMarketMinClose,
		netparams.GovernanceProposalMarketMaxClose,
		netparams.GovernanceProposalMarketMinEnact,
		netparams.GovernanceProposalMarketMaxEnact,
		netparams.GovernanceProposalMarketRequiredParticipation,
		netparams.GovernanceProposalMarketRequiredMajority,
		netparams.GovernanceProposalMarketMinProposerBalance,
		netparams.GovernanceProposalMarketMinVoterBalance,
		"0",
		"0",
		"0",
	)
}

func (e *Engine) getNewMarketProposalParameters() *types.ProposalParameters {
	return e.getProposalParametersFromNetParams(
		netparams.GovernanceProposalMarketMinClose,
		netparams.GovernanceProposalMarketMaxClose,
		netparams.GovernanceProposalMarketMinEnact,
		netparams.GovernanceProposalMarketMaxEnact,
		netparams.GovernanceProposalMarketRequiredParticipation,
		netparams.GovernanceProposalMarketRequiredMajority,
		netparams.GovernanceProposalMarketMinProposerBalance,
		netparams.GovernanceProposalMarketMinVoterBalance,
		"0",
		"0",
		"0",
	)
}

func (e *Engine) getUpdateMarketProposalParameters() *types.ProposalParameters {
	return e.getProposalParametersFromNetParams(
		netparams.GovernanceProposalUpdateMarketMinClose,
		netparams.GovernanceProposalUpdateMarketMaxClose,
		netparams.GovernanceProposalUpdateMarketMinEnact,
		netparams.GovernanceProposalUpdateMarketMaxEnact,
		netparams.GovernanceProposalUpdateMarketRequiredParticipation,
		netparams.GovernanceProposalUpdateMarketRequiredMajority,
		netparams.GovernanceProposalUpdateMarketMinProposerBalance,
		netparams.GovernanceProposalUpdateMarketMinVoterBalance,
		netparams.GovernanceProposalUpdateMarketRequiredParticipationLP,
		netparams.GovernanceProposalUpdateMarketRequiredMajorityLP,
		netparams.GovernanceProposalUpdateMarketMinProposerEquityLikeShare,
	)
}

func (e *Engine) getUpdateSpotMarketProposalParameters() *types.ProposalParameters {
	return e.getProposalParametersFromNetParams(
		netparams.GovernanceProposalUpdateMarketMinClose,
		netparams.GovernanceProposalUpdateMarketMaxClose,
		netparams.GovernanceProposalUpdateMarketMinEnact,
		netparams.GovernanceProposalUpdateMarketMaxEnact,
		netparams.GovernanceProposalUpdateMarketRequiredParticipation,
		netparams.GovernanceProposalUpdateMarketRequiredMajority,
		netparams.GovernanceProposalUpdateMarketMinProposerBalance,
		netparams.GovernanceProposalUpdateMarketMinVoterBalance,
		netparams.GovernanceProposalUpdateMarketRequiredParticipationLP,
		netparams.GovernanceProposalUpdateMarketRequiredMajorityLP,
		netparams.GovernanceProposalUpdateMarketMinProposerEquityLikeShare,
	)
}

// getUpdatetMarketStateProposalParameters is reusing the net params defined for market update!
func (e *Engine) getUpdateMarketStateProposalParameters() *types.ProposalParameters {
	return e.getProposalParametersFromNetParams(netparams.GovernanceProposalUpdateMarketMinClose,
		netparams.GovernanceProposalUpdateMarketMaxClose,
		netparams.GovernanceProposalUpdateMarketMinEnact,
		netparams.GovernanceProposalUpdateMarketMaxEnact,
		netparams.GovernanceProposalUpdateMarketRequiredParticipation,
		netparams.GovernanceProposalUpdateMarketRequiredMajority,
		netparams.GovernanceProposalUpdateMarketMinProposerBalance,
		netparams.GovernanceProposalUpdateMarketMinVoterBalance,
		netparams.GovernanceProposalUpdateMarketRequiredParticipationLP,
		netparams.GovernanceProposalUpdateMarketRequiredMajorityLP,
		netparams.GovernanceProposalUpdateMarketMinProposerEquityLikeShare,
	)
}

func (e *Engine) getReferralProgramNetworkParameters() *types.ProposalParameters {
	return e.getProposalParametersFromNetParams(
		netparams.GovernanceProposalReferralProgramMinClose,
		netparams.GovernanceProposalReferralProgramMaxClose,
		netparams.GovernanceProposalReferralProgramMinEnact,
		netparams.GovernanceProposalReferralProgramMaxEnact,
		netparams.GovernanceProposalReferralProgramRequiredParticipation,
		netparams.GovernanceProposalReferralProgramRequiredMajority,
		netparams.GovernanceProposalReferralProgramMinProposerBalance,
		netparams.GovernanceProposalReferralProgramMinVoterBalance,
		"0",
		"0",
		"0",
	)
}

func (e *Engine) getVolumeDiscountProgramNetworkParameters() *types.ProposalParameters {
	return e.getProposalParametersFromNetParams(
		netparams.GovernanceProposalVolumeDiscountProgramMinClose,
		netparams.GovernanceProposalVolumeDiscountProgramMaxClose,
		netparams.GovernanceProposalVolumeDiscountProgramMinEnact,
		netparams.GovernanceProposalVolumeDiscountProgramMaxEnact,
		netparams.GovernanceProposalVolumeDiscountProgramRequiredParticipation,
		netparams.GovernanceProposalVolumeDiscountProgramRequiredMajority,
		netparams.GovernanceProposalVolumeDiscountProgramMinProposerBalance,
		netparams.GovernanceProposalVolumeDiscountProgramMinVoterBalance,
		"0",
		"0",
		"0",
	)
}

func (e *Engine) getVolumeRebateProgramNetworkParameters() *types.ProposalParameters {
	return e.getProposalParametersFromNetParams(
		netparams.GovernanceProposalVolumeRebateProgramMinClose,
		netparams.GovernanceProposalVolumeRebateProgramMaxClose,
		netparams.GovernanceProposalVolumeRebateProgramMinEnact,
		netparams.GovernanceProposalVolumeRebateProgramMaxEnact,
		netparams.GovernanceProposalVolumeRebateProgramRequiredParticipation,
		netparams.GovernanceProposalVolumeRebateProgramRequiredMajority,
		netparams.GovernanceProposalVolumeRebateProgramMinProposerBalance,
		netparams.GovernanceProposalVolumeRebateProgramMinVoterBalance,
		"0",
		"0",
		"0",
	)
}

func (e *Engine) getUpdateMarketCommunityTagsParameters() *types.ProposalParameters {
	return e.getProposalParametersFromNetParams(
		netparams.GovernanceProposalUpdateCommunityTagsMinClose,
		netparams.GovernanceProposalUpdateCommunityTagsMaxClose,
		netparams.GovernanceProposalUpdateCommunityTagsMinEnact,
		netparams.GovernanceProposalUpdateCommunityTagsMaxEnact,
		netparams.GovernanceProposalUpdateCommunityTagsRequiredParticipation,
		netparams.GovernanceProposalUpdateCommunityTagsRequiredMajority,
		netparams.GovernanceProposalUpdateCommunityTagsMinProposerBalance,
		netparams.GovernanceProposalUpdateCommunityTagsMinVoterBalance,
		"0",
		"0",
		"0",
	)
}

func (e *Engine) getNewAssetProposalParameters() *types.ProposalParameters {
	return e.getProposalParametersFromNetParams(
		netparams.GovernanceProposalAssetMinClose,
		netparams.GovernanceProposalAssetMaxClose,
		netparams.GovernanceProposalAssetMinEnact,
		netparams.GovernanceProposalAssetMaxEnact,
		netparams.GovernanceProposalAssetRequiredParticipation,
		netparams.GovernanceProposalAssetRequiredMajority,
		netparams.GovernanceProposalAssetMinProposerBalance,
		netparams.GovernanceProposalAssetMinVoterBalance,
		"0",
		"0",
		"0",
	)
}

func (e *Engine) getUpdateAssetProposalParameters() *types.ProposalParameters {
	return e.getProposalParametersFromNetParams(
		netparams.GovernanceProposalUpdateAssetMinClose,
		netparams.GovernanceProposalUpdateAssetMaxClose,
		netparams.GovernanceProposalUpdateAssetMinEnact,
		netparams.GovernanceProposalUpdateAssetMaxEnact,
		netparams.GovernanceProposalUpdateAssetRequiredParticipation,
		netparams.GovernanceProposalUpdateAssetRequiredMajority,
		netparams.GovernanceProposalUpdateAssetMinProposerBalance,
		netparams.GovernanceProposalUpdateAssetMinVoterBalance,
		"0",
		"0",
		"0",
	)
}

func (e *Engine) getUpdateNetworkParameterProposalParameters() *types.ProposalParameters {
	return e.getProposalParametersFromNetParams(
		netparams.GovernanceProposalUpdateNetParamMinClose,
		netparams.GovernanceProposalUpdateNetParamMaxClose,
		netparams.GovernanceProposalUpdateNetParamMinEnact,
		netparams.GovernanceProposalUpdateNetParamMaxEnact,
		netparams.GovernanceProposalUpdateNetParamRequiredParticipation,
		netparams.GovernanceProposalUpdateNetParamRequiredMajority,
		netparams.GovernanceProposalUpdateNetParamMinProposerBalance,
		netparams.GovernanceProposalUpdateNetParamMinVoterBalance,
		"0",
		"0",
		"0",
	)
}

func (e *Engine) getNewFreeformProposalParameters() *types.ProposalParameters {
	return e.getProposalParametersFromNetParams(
		netparams.GovernanceProposalFreeformMinClose,
		netparams.GovernanceProposalFreeformMaxClose,
		"0s",
		"0s",
		netparams.GovernanceProposalFreeformRequiredParticipation,
		netparams.GovernanceProposalFreeformRequiredMajority,
		netparams.GovernanceProposalFreeformMinProposerBalance,
		netparams.GovernanceProposalFreeformMinVoterBalance,
		"0",
		"0",
		"0",
	)
}

func (e *Engine) getNewTransferProposalParameters() *types.ProposalParameters {
	return e.getProposalParametersFromNetParams(
		netparams.GovernanceProposalTransferMinClose,
		netparams.GovernanceProposalTransferMaxClose,
		netparams.GovernanceProposalAssetMinEnact,
		netparams.GovernanceProposalAssetMaxEnact,
		netparams.GovernanceProposalTransferRequiredParticipation,
		netparams.GovernanceProposalTransferRequiredMajority,
		netparams.GovernanceProposalTransferMinProposerBalance,
		netparams.GovernanceProposalTransferMinVoterBalance,
		"0",
		"0",
		"0",
	)
}

func (e *Engine) getProposalParametersFromNetParams(
	minCloseKey, maxCloseKey, minEnactKey, maxEnactKey, requiredParticipationKey,
	requiredMajorityKey, minProposerBalanceKey, minVoterBalanceKey,
	requiredParticipationLPKey, requiredMajorityLPKey, minEquityLikeShareKey string,
) *types.ProposalParameters {
	pp := types.ProposalParameters{}
	pp.MinClose, _ = e.netp.GetDuration(minCloseKey)
	pp.MaxClose, _ = e.netp.GetDuration(maxCloseKey)
	pp.MinEnact, _ = e.netp.GetDuration(minEnactKey)
	pp.MaxEnact, _ = e.netp.GetDuration(maxEnactKey)
	pp.RequiredParticipation, _ = e.netp.GetDecimal(requiredParticipationKey)
	pp.RequiredMajority, _ = e.netp.GetDecimal(requiredMajorityKey)
	pp.MinProposerBalance, _ = e.netp.GetUint(minProposerBalanceKey)
	pp.MinVoterBalance, _ = e.netp.GetUint(minVoterBalanceKey)
	pp.RequiredParticipationLP, _ = e.netp.GetDecimal(requiredParticipationLPKey)
	pp.RequiredMajorityLP, _ = e.netp.GetDecimal(requiredMajorityLPKey)
	pp.MinEquityLikeShare, _ = e.netp.GetDecimal(minEquityLikeShareKey)
	return &pp
}

func validateNetworkParameterUpdate(
	netp NetParams, np *types.NetworkParameter,
) (types.ProposalError, error) {
	if len(np.Key) <= 0 {
		return types.ProposalErrorNetworkParameterInvalidKey, ErrEmptyNetParamKey
	}

	if len(np.Value) <= 0 {
		return types.ProposalErrorNetworkParameterInvalidValue, ErrEmptyNetParamValue
	}

	// so we seems to just need to call on validate in here.
	// no need to know what's the parameter really or anything else
	var (
		perr = types.ProposalErrorUnspecified
		err  = netp.Validate(np.Key, np.Value)
	)
	if err != nil {
		perr = types.ProposalErrorNetworkParameterValidationFailed
	}

	return perr, err
}
