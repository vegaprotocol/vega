// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package subscribers

import (
	types "code.vegaprotocol.io/protos/vega"
)

type ProposalType int

type PropE interface {
	GovernanceEvent
	Proposal() types.Proposal
}

const (
	NewMarketProposal ProposalType = iota
	NewAssetProposal
	UpdateMarketProposal
	UpdateNetworkParameterProposal
	NewFreeformProposal
)

type ProposalFilteredSub struct {
	*Base
}

// ProposalByID - filter proposal events by proposal ID
func ProposalByID(id string) ProposalFilter {
	return func(p types.Proposal) bool {
		return p.Id == id
	}
}

// ProposalByPartyID - filter proposals submitted by given party
func ProposalByPartyID(id string) ProposalFilter {
	return func(p types.Proposal) bool {
		return p.PartyId == id
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
			case NewAssetProposal:
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
			case NewFreeformProposal:
				if un := p.Terms.GetNewFreeform(); un != nil {
					return true
				}
			}
		}
		return false
	}
}
