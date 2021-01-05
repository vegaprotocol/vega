package governance

import (
	"code.vegaprotocol.io/vega/netparams"
	types "code.vegaprotocol.io/vega/proto/gen/golang"

	"github.com/pkg/errors"
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
		netparams.GovernanceProposalUpdateMarketMinClose,
		netparams.GovernanceProposalUpdateMarketMaxClose,
		netparams.GovernanceProposalUpdateMarketMinEnact,
		netparams.GovernanceProposalUpdateMarketMaxEnact,
		netparams.GovernanceProposalUpdateMarketRequiredParticipation,
		netparams.GovernanceProposalUpdateMarketRequiredMajority,
		netparams.GovernanceProposalUpdateMarketMinProposerBalance,
		netparams.GovernanceProposalUpdateMarketMinVoterBalance,
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
	pp.RequiredParticipation, _ = e.netp.GetFloat(requiredParticipationKey)
	pp.RequiredMajority, _ = e.netp.GetFloat(requiredMajorityKey)
	mpb, _ := e.netp.GetInt(minProposerBalanceKey)
	pp.MinProposerBalance = uint64(mpb)
	pp.MinVoterBalance, _ = e.netp.GetFloat(minVoterBalanceKey)
	return &pp, nil
}

func validateNetworkParameterUpdate(
	netp NetParams, np *types.NetworkParameter) (types.ProposalError, error) {
	if len(np.Key) <= 0 {
		return types.ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_KEY, ErrEmptyNetParamKey
	}

	if len(np.Value) <= 0 {
		return types.ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_VALUE, ErrEmptyNetParamValue
	}

	// so we seems to just need to call on validate in here.
	// no need to know what's the parameter really or anything else
	var (
		perr = types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED
		err  = netp.Validate(np.Key, np.Value)
	)
	if err != nil {
		perr = types.ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_VALIDATION_FAILED
	}

	return perr, err
}
