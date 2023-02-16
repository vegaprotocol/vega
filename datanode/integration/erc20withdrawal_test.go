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

func TestERC20WithdrawalApproval(t *testing.T) {
	queries := map[string]string{
		"ERC20WithdrawalApproval": `{ erc20WithdrawalApproval(withdrawalId: "7EB9B511E4DA3397DFC2A71D05A1A0DD4CC1782AA5080D4FE30C2EC1E31622E6"){ assetSource amount nonce signatures targetAddress } }`,
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			assertGraphQLQueriesReturnSame(t, query)
		})
	}
}
