// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package integration_test

import "testing"

func TestGovernance(t *testing.T) {
	queries := map[string]string{
		"Proposals":           `{ proposalsConnection{ edges { node { id, reference, party { id }, state, datetime, rejectionReason, errorDetails } } } }`,
		"ProposalVoteSummary": `{ proposalsConnection{ edges { node { id votes{ yes{ totalNumber totalWeight totalTokens } } } } } }`,
		"ProposalVoteDetails": `{ proposalsConnection{ edges { node { id votes{ yes{ votes{value party { id } datetime proposalId governanceTokenBalance governanceTokenWeight } } } } } } }`,
		"ProposalNewMarket":   `{ proposalsConnection { edges { node { id terms { change { ... on NewMarket { instrument { name } decimalPlaces riskParameters { ... on SimpleRiskModel { params { factorLong factorShort } } } metadata priceMonitoringParameters { triggers { horizonSecs probability auctionExtensionSecs } } liquidityMonitoringParameters { targetStakeParameters { timeWindow scalingFactor } } positionDecimalPlaces } } } } } } }`,
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			assertGraphQLQueriesReturnSame(t, query)
		})
	}
}
