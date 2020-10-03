package governance

import (
	"code.vegaprotocol.io/vega/netparams"
	types "code.vegaprotocol.io/vega/proto"
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
	pp.MinProposerBalance, _ = e.netp.GetFloat(minProposerBalanceKey)
	pp.MinVoterBalance, _ = e.netp.GetFloat(minVoterBalanceKey)
	return &pp, nil
}

func validateNetworkParameterUpdate(
	netp NetParams, np *types.NetworkParameter) error {
	if len(np.Key) <= 0 {
		return ErrEmptyNetParamKey
	}

	if len(np.Value) <= 0 {
		return ErrEmptyNetParamValue
	}

	// so we seems to just need to call on validate in here.
	// no need to know what's the parameter really or anything else
	return netp.Validate(np.Key, np.Value)
}
