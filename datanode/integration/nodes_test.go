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

func TestNodes(t *testing.T) {
	queries := map[string]string{
		"Nodes": "{ nodesConnection { edges { node { id, pubkey, tmPubkey, ethereumAddress, infoUrl, location, name, avatarUrl, status, stakedByOperator, stakedByDelegates, stakedTotal, pendingStake, delegationsConnection{ edges { node { party { id }, epoch, amount } } }, rewardScore { validatorStatus, validatorScore }, rankingScore { previousStatus, status, votingPower, performanceScore, stakeScore, rankingScore }, epochData { total, offline, online } } } } }",
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			assertGraphQLQueriesReturnSame(t, query)
		})
	}
}

func TestNodeData(t *testing.T) {
	queries := map[string]string{
		"NodeData": "{ nodeData { stakedTotal, totalNodes, inactiveNodes, tendermintNodes {total, inactive, maximum }, ersatzNodes {total, inactive, maximum }, pendingNodes {total, inactive}, uptime } }",
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			assertGraphQLQueriesReturnSame(t, query)
		})
	}
}
