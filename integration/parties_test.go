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

func TestParties(t *testing.T) {
	queries := map[string]string{
		//"Deposits":           "{ parties { id deposits{ id, party { id }, amount, asset { id }, status, createdTimestamp, creditedTimestamp, txHash } } }",
		//"Withdrawals":        "{ parties { id withdrawals { id, party { id }, amount, asset { id }, status, ref, expiry, txHash, createdTimestamp, withdrawnTimestamp } } }",
		//"Delegations":        "{ parties{ id delegations{ node { id }, party{ id }, epoch, amount } } }",
		//"Proposals":          "{ parties{ id proposals{ id votes{ yes { totalNumber } no { totalNumber } } } } }",
		//"Votes":              "{ parties{ id votes{ proposalId vote{ value } } } }",
		"Margin Levels": "{ parties{ id margins { market { id }, asset { id }, party { id }, maintenanceLevel, searchLevel, initialLevel, collateralReleaseLevel, timestamp } } }",
		//"LiquidityProvision": "{ parties{ id, orders { id, liquidityProvision { id, party { id }, createdAt, updatedAt, market { id }, commitmentAmount, fee, sells { order { id }, liquidityOrder { reference } }, buys { order { id }, liquidityOrder { reference } }, version, status, reference } }, liquidityProvisions { id, party { id }, createdAt, updatedAt, market { id }, commitmentAmount, fee, sells { order { id }, liquidityOrder { reference } }, buys { order { id }, liquidityOrder { reference } }, version, status, reference } } }",
		//"StakeLinking":       "{ parties { stake { currentStakeAvailable, linkings { id, type, timestamp, party { id }, amount, status, finalizedAt, txHash } } } }",
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			var new, old struct{ Parties []Party }
			assertGraphQLQueriesReturnSame(t, query, &new, &old)
		})
	}
}
