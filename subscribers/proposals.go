package subscribers

import (
	"sync"

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
	UpdateNetworkProposal
)

type ProposalFilteredSub struct {
	*Base
	mu      sync.Mutex
	filters []ProposalFilter
	matched []types.Proposal
}

// ByProposalID - filter proposal events by proposal ID
func ProposalByID(id string) ProposalFilter {
	return func(p types.Proposal) bool {
		if p.ID == id {
			return true
		}
		return false
	}
}

// ProposalByPartyID - filter proposals submitted by given party
func ProposalByPartyID(id string) ProposalFilter {
	return func(p types.Proposal) bool {
		if p.PartyID == id {
			return true
		}
		return false
	}
}

// ProposalByState - filter proposals by state
func ProposalByState(s types.Proposal_State) ProposalFilter {
	return func(p types.Proposal) bool {
		if p.State == s {
			return true
		}
		return false
	}
}

// ProposalByReference - filter out proposals by reference
func ProposalByReference(ref string) ProposalFilter {
	return func(p types.Proposal) bool {
		if p.Reference == ref {
			return true
		}
		return false
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
			case UpdateNetworkProposal:
				if un := p.Terms.GetUpdateNetwork(); un != nil {
					return true
				}
			}
		}
		return false
	}
}
