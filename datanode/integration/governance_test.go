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

package integration_test

import "testing"

func TestGovernance(t *testing.T) {
	queries := map[string]string{
		"Proposals":                  `{ proposals{ id, reference, party { id }, state, datetime, rejectionReason, errorDetails } }`,
		"ProposalVoteSummary":        `{ proposals{ id votes{ yes{ totalNumber totalWeight totalTokens } } } }`,
		"ProposalVoteDetails":        `{ proposals{ id votes{ yes{ votes{value party { id } datetime proposalId governanceTokenBalance governanceTokenWeight } } } } }`,
		"NewMarketProposals":         `{ proposals: newMarketProposals { id } }`,
		"NetworkParametersProposals": `{ proposals: networkParametersProposals { id } }`,
		"NewAssetProposals":          `{ proposals: newAssetProposals { id } }`,
		"NewFreeformProposals":       `{ proposals: newFreeformProposals { id } }`,
		// Don't currently have these in test data stream
		//"UpdateMarketProposals":      `{ updateMarketProposals { id } }`,
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			assertGraphQLQueriesReturnSame[struct{ Proposals []Proposal }](t, query)
		})
	}
}
