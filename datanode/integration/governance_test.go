// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
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
		"Proposals":           `{ proposalsConnection{ edges { node { id, reference, party { id }, state, datetime, rejectionReason, errorDetails } } } }`,
		"ProposalVoteSummary": `{ proposalsConnection{ edges { node { id votes{ yes{ totalNumber totalWeight totalTokens } } } } } }`,
		"ProposalVoteDetails": `{ proposalsConnection{ edges { node { id votes{ yes{ votes{value party { id } datetime proposalId governanceTokenBalance governanceTokenWeight } } } } } } }`,
		"ProposalNewMarket":   `{ proposalsConnection { edges { node { id terms { change { ... on NewMarket { instrument { name } decimalPlaces riskParameters { ... on SimpleRiskModel { params { factorLong factorShort } } } metadata priceMonitoringParameters { triggers { horizonSecs probability auctionExtensionSecs } } liquidityMonitoringParameters { targetStakeParameters { timeWindow scalingFactor } triggeringRatio auctionExtensionSecs } positionDecimalPlaces lpPriceRange } } } } } } }`,
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			assertGraphQLQueriesReturnSame(t, query)
		})
	}
}
