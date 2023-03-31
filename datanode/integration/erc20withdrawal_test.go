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
	// This is a bit suboptimal as this needs updating when the event file is replaced; perhaps we can make it nicer in the future.
	queries := map[string]string{
		"ERC20WithdrawalApproval": `{ erc20WithdrawalApproval(withdrawalId: "692595a5049cebd114ed62265e300fb9252967e52beef35b42ff4e298059a954"){ assetSource amount nonce signatures targetAddress } }`,
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			assertGraphQLQueriesReturnSame(t, query)
		})
	}
}
