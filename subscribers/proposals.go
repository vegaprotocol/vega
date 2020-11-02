package subscribers

import (
	types "code.vegaprotocol.io/vega/proto"
)

type ProposalType int

type PropE interface {
	GovernanceEvent
	Proposal() types.Proposal
}

const (
	NewMarketProposal ProposalType = iota
	NewAssetPropopsal
	UpdateMarketProposal
	UpdateNetworkParameterProposal
)

type ProposalFilteredSub struct {
	*Base
}

// ProposalByID - filter proposal events by proposal ID
func ProposalByID(id string) ProposalFilter {
	return func(p types.Proposal) bool {
		return p.ID == id
	}
}

// ProposalByPartyID - filter proposals submitted by given party
func ProposalByPartyID(id string) ProposalFilter {
	return func(p types.Proposal) bool {
		return p.PartyID == id
	}
}

// ProposalByState - filter proposals by state
func ProposalByState(s types.Proposal_State) ProposalFilter {
	return func(p types.Proposal) bool {
		return p.State == s
	}
}

// ProposalByReference - filter out proposals by reference
func ProposalByReference(ref string) ProposalFilter {
	return func(p types.Proposal) bool {
		return p.Reference == ref
	}
}

func ProposalByChange(ptypes ...ProposalType) ProposalFilter {
	return func(p types.Proposal) bool {
		for _, t := range ptypes {
			switch t {
			case NewMarketProposal:
				if nm := p.Terms.GetNewMarket(); nm != nil {
					return true
				}
			case NewAssetPropopsal:
				if na := p.Terms.GetNewAsset(); na != nil {
					return true
				}
			case UpdateMarketProposal:
				if um := p.Terms.GetUpdateMarket(); um != nil {
					return true
				}
			case UpdateNetworkParameterProposal:
				if un := p.Terms.GetUpdateNetworkParameter(); un != nil {
					return true
				}
			}
		}
		return false
	}
}
