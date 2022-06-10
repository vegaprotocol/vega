package integration_test

import "testing"

func TestNodes(t *testing.T) {
	queries := map[string]string{
		"Nodes": "{ nodes { id, pubkey, tmPubkey, ethereumAdddress, infoUrl, location, name, avatarUrl, status, stakedByOperator, stakedByDelegates, stakedTotal, pendingStake, delegations { party { id }, epoch, amount }, rewardScore { validatorStatus, validatorScore }, rankingScore { previousStatus, status, votingPower, performanceScore, stakeScore, rankingScore }, epochData { total, offline, online } } }",
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			var new, old struct{ Nodes []Node }
			assertGraphQLQueriesReturnSame(t, query, &old, &new)
		})
	}
}

func TestNodeData(t *testing.T) {
	queries := map[string]string{
		"NodeData": "{ nodeData { stakedTotal, totalNodes, inactiveNodes, validatingNodes, uptime } }",
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			var new, old struct{ NodeData NodeData }
			assertGraphQLQueriesReturnSame(t, query, &old, &new)
		})
	}
}
