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

func TestParties(t *testing.T) {
	queries := map[string]string{
		"Deposits":           "{ partiesConnection{ edges { node { id depositsConnection{ edges { node { id, party { id }, amount, asset { id }, status, createdTimestamp, creditedTimestamp, txHash } } } } } } }",
		"Withdrawals":        "{ partiesConnection{ edges { node { id withdrawalsConnection { edges { node { id, party { id }, amount, asset { id }, status, ref, txHash, createdTimestamp, withdrawnTimestamp } } } } } } }",
		"Delegations":        "{ partiesConnection{ edges { node { id delegationsConnection{ edges { node { node { id }, party{ id }, epoch, amount } } } } } } }",
		"Proposals":          "{ partiesConnection{ edges { node { id proposalsConnection{ edges { node { id votes{ yes { totalNumber } no { totalNumber } } } } } } } } }",
		"Votes":              "{ partiesConnection{ edges { node { id votesConnection{ edges { node { proposalId vote{ value } } } } } } } }",
		"Margin Levels":      "{ partiesConnection{ edges { node { id marginsConnection{ edges { node { market { id }, asset { id }, party { id }, maintenanceLevel, searchLevel, initialLevel, collateralReleaseLevel, timestamp } } } } } } }",
		"LiquidityProvision": "{ partiesConnection{ edges { node { id, ordersConnection { edges { node { id, liquidityProvision { id, party { id }, createdAt, updatedAt, market { id }, commitmentAmount, fee, sells { order { id }, liquidityOrder { reference } }, buys { order { id }, liquidityOrder { reference } }, version, status, reference } } } } } } } }",
		"StakeLinking":       "{ partiesConnection{ edges { node { stakingSummary { currentStakeAvailable, linkings { edges { node { id, type, timestamp, party { id }, amount, status, finalizedAt, txHash } } } } } } } }",
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			assertGraphQLQueriesReturnSame(t, query)
		})
	}
}
