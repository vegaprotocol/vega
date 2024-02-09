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
