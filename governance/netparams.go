package governance

import (
	"errors"

	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/types/num"
)

var (
	ErrEmptyNetParamKey   = errors.New("empty network parameter key")
	ErrEmptyNetParamValue = errors.New("empty network parmater value")
)

func (e *Engine) getNewMarketProposalParameters() (*ProposalParameters, error) {
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

func (e *Engine) getNewAssetProposalParameters() (*ProposalParameters, error) {
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

func (e *Engine) getUpdateNetworkParameterProposalParameters() (*ProposalParameters, error) {
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

func (e *Engine) getProposalParametersFromNetParams(
	minCloseKey, maxCloseKey, minEnactKey, maxEnactKey, requiredParticipationKey,
	requiredMajorityKey, minProposerBalanceKey, minVoterBalanceKey string,
) (*ProposalParameters, error) {
	pp := ProposalParameters{}
	pp.MinClose, _ = e.netp.GetDuration(minCloseKey)
	pp.MaxClose, _ = e.netp.GetDuration(maxCloseKey)
	pp.MinEnact, _ = e.netp.GetDuration(minEnactKey)
	pp.MaxEnact, _ = e.netp.GetDuration(maxEnactKey)
	rp, _ := e.netp.GetFloat(requiredParticipationKey)
	pp.RequiredParticipation = num.NewDecimalFromFloat(rp)
	rm, _ := e.netp.GetFloat(requiredMajorityKey)
	pp.RequiredMajority = num.NewDecimalFromFloat(rm)
	mpb, _ := e.netp.GetInt(minProposerBalanceKey)
	pp.MinProposerBalance = num.NewUint(uint64(mpb))
	mvb, _ := e.netp.GetInt(minVoterBalanceKey)
	pp.MinVoterBalance = num.NewUint(uint64(mvb))
	return &pp, nil
}

func validateNetworkParameterUpdate(
	netp NetParams, np *proto.NetworkParameter) (proto.ProposalError, error) {
	if len(np.Key) <= 0 {
		return proto.ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_KEY, ErrEmptyNetParamKey
	}

	if len(np.Value) <= 0 {
		return proto.ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_VALUE, ErrEmptyNetParamValue
	}

	// so we seems to just need to call on validate in here.
	// no need to know what's the parameter really or anything else
	var (
		perr = proto.ProposalError_PROPOSAL_ERROR_UNSPECIFIED
		err  = netp.Validate(np.Key, np.Value)
	)
	if err != nil {
		perr = proto.ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_VALIDATION_FAILED
	}

	return perr, err
}
