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

func TestERC20WithdrawalApproval(t *testing.T) {
	queries := map[string]string{
		"ERC20WithdrawalApproval": `{ erc20WithdrawalApproval(withdrawalId:"7ee15f2fc0d49687df4a791fce246d82a0b82c420d02a562e7d4bcc430e9a8c7") { assetSource amount nonce signatures targetAddress } }`,
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			assertGraphQLQueriesReturnSame[struct{ ERC20WithdrawalApproval ERC20WithdrawalApproval }](t, query)
		})
	}
}
