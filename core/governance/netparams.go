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

func (e *Engine) getNewMarketProposalParameters() *ProposalParameters {
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

func (e *Engine) getUpdateMarketProposalParameters() *ProposalParameters {
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

func (e *Engine) getNewAssetProposalParameters() *ProposalParameters {
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

func (e *Engine) getUpdateAssetProposalParameters() *ProposalParameters {
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

func (e *Engine) getUpdateNetworkParameterProposalParameters() *ProposalParameters {
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

func (e *Engine) getNewFreeformProposalParameters() *ProposalParameters {
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

func (e *Engine) getProposalParametersFromNetParams(
	minCloseKey, maxCloseKey, minEnactKey, maxEnactKey, requiredParticipationKey,
	requiredMajorityKey, minProposerBalanceKey, minVoterBalanceKey,
	requiredParticipationLPKey, requiredMajorityLPKey, minEquityLikeShareKey string,
) *ProposalParameters {
	pp := ProposalParameters{}
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
