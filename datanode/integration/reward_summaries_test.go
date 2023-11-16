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

func TestRewardSummaries(t *testing.T) {
	queries := map[string]string{
		"RewardSummaries": `{ epochRewardSummaries(filter: { assetIds:["41498ab9aca53472efe12c37209689f755e30f681170cba3a5bd7012a1ef2001"], marketIds:[""], fromEpoch: 266, toEpoch: 276 }) { edges { node { epoch marketId assetId rewardType amount } } } }`,
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			assertGraphQLQueriesReturnSame(t, query)
		})
	}
}
