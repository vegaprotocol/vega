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

func TestAccounts(t *testing.T) {
	queries := map[string]string{
		"PartyAccounts":       `{ partiesConnection{ edges { node { id accountsConnection{ edges { node { asset{ id } market { id } type balance } } } } } } }`,
		"MarketAccounts":      `{ marketsConnection{ edges { node { id accountsConnection{ edges { node { asset{ id } market { id } type balance } } } } } } }`,
		"AssetFeeAccounts":    `{ assetsConnection{ edges { node { id infrastructureFeeAccount{ asset{ id } market { id } type balance } } } } }`,
		"AssetRewardAccounts": `{ assetsConnection{ edges { node { id globalRewardPoolAccount{ asset{ id } market { id } type balance } } } } }`,
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			assertGraphQLQueriesReturnSame(t, query)
		})
	}
}
