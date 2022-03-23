package integration_test

import "testing"

func TestGovernance(t *testing.T) {
	queries := map[string]string{
		"Proposals":                  `{ proposals{ id, reference, party { id }, state, datetime, rejectionReason, errorDetails } }`,
		"ProposalVoteSummary":        `{ proposals{ id votes{ yes{ totalNumber totalWeight totalTokens } } } }`,
		"ProposalVoteDetails":        `{ proposals{ id votes{ yes{ votes{value party { id } datetime proposalId governanceTokenBalance governanceTokenWeight } } } } }`,
		"NewMarketProposals":         `{ newMarketProposals { id } }`,
		"UpdateMarketProposals":      `{ updateMarketProposals { id } }`,
		"NetworkParametersProposals": `{ networkParametersProposals { id } }`,
		"NewAssetProposals":          `{ newAssetProposals { id } }`,
		"NewFreeformProposals":       `{ newFreeformProposals { id } }`,
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			var new, old struct{ Proposals []Proposal }
			assertGraphQLQueriesReturnSame(t, query, &new, &old)
		})
	}
}
