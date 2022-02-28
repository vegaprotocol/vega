package governance

import (
	"errors"

	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/types"
)

var (
	ErrEmptyNetParamKey   = errors.New("empty network parameter key")
	ErrEmptyNetParamValue = errors.New("empty network parmater value")
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
	)
}

func (e *Engine) getNewFreeformProposalarameters() *ProposalParameters {
	return e.getProposalParametersFromNetParams(
		netparams.GovernanceProposalFreeformMinClose,
		netparams.GovernanceProposalFreeformMaxClose,
		"0s",
		"0s",
		netparams.GovernanceProposalFreeformRequiredParticipation,
		netparams.GovernanceProposalFreeformRequiredMajority,
		netparams.GovernanceProposalFreeformMinProposerBalance,
		netparams.GovernanceProposalFreeformMinVoterBalance,
	)
}

func (e *Engine) getProposalParametersFromNetParams(
	minCloseKey, maxCloseKey, minEnactKey, maxEnactKey, requiredParticipationKey,
	requiredMajorityKey, minProposerBalanceKey, minVoterBalanceKey string,
) *ProposalParameters {
	pp := ProposalParameters{}
	pp.MinClose, _ = e.netp.GetDuration(minCloseKey)
	pp.MaxClose, _ = e.netp.GetDuration(maxCloseKey)
	pp.MinEnact, _ = e.netp.GetDuration(minEnactKey)
	pp.MaxEnact, _ = e.netp.GetDuration(maxEnactKey)
	pp.RequiredParticipation, _ = e.netp.GetDecimal(requiredParticipationKey)
	pp.RequiredMajority, _ = e.netp.GetDecimal(requiredMajorityKey)
	mpb, _ := e.netp.GetUint(minProposerBalanceKey)
	pp.MinProposerBalance = mpb
	mvb, _ := e.netp.GetUint(minVoterBalanceKey)
	pp.MinVoterBalance = mvb
	return &pp
}

func validateNetworkParameterUpdate(
	netp NetParams, np *types.NetworkParameter) (types.ProposalError, error) {
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
