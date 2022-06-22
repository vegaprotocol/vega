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

package sqlstore_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func addTestNode(t *testing.T, ps *sqlstore.Node, block entities.Block) entities.Node {
	t.Helper()
	node := entities.Node{
		ID:              entities.NewNodeID(generateID()),
		PubKey:          entities.VegaPublicKey(generateID()),
		EthereumAddress: entities.EthereumAddress(generateEthereumAddress()),
		TmPubKey:        entities.TendermintPublicKey(generateTendermintPublicKey()),
		VegaTime:        block.VegaTime,
		Status:          entities.NodeStatusNonValidator,
	}

	err := ps.UpsertNode(context.Background(), &node)
	require.NoError(t, err)
	return node
}

func addNodeAnnounced(t *testing.T, ps *sqlstore.Node, nodeID entities.NodeID, added bool, fromEpoch uint64) {
	t.Helper()
	aux := entities.ValidatorUpdateAux{
		Added:     added,
		FromEpoch: fromEpoch,
	}
	err := ps.AddNodeAnnoucedEvent(context.Background(), nodeID, &aux)
	require.NoError(t, err)
}

func addRankingScore(t *testing.T, ps *sqlstore.Node, block entities.Block, node entities.Node, epoch uint64) entities.RankingScore {
	t.Helper()
	r := entities.RankingScore{
		StakeScore:       decimal.NewFromFloat(0.5),
		PerformanceScore: decimal.NewFromFloat(0.25),
		PreviousStatus:   entities.ValidatorNodeStatusErsatz,
		Status:           entities.ValidatorNodeStatusTendermint,
		EpochSeq:         epoch,
		VegaTime:         block.VegaTime,
	}

	aux := entities.RankingScoreAux{
		NodeId:   node.ID,
		EpochSeq: epoch,
	}

	err := ps.UpsertRanking(context.Background(), &r, &aux)
	require.NoError(t, err)
	return r
}

func TestGetNodes(t *testing.T) {
	DeleteEverything()
	defer DeleteEverything()
	ctx := context.Background()
	bs := sqlstore.NewBlocks(connectionSource)
	ns := sqlstore.NewNode(connectionSource)
	block := addTestBlock(t, bs)

	node1 := addTestNode(t, ns, block)
	addNodeAnnounced(t, ns, node1.ID, true, 0)
	addNodeAnnounced(t, ns, node1.ID, false, 7)
	addRankingScore(t, ns, block, node1, 3)

	// get all nodes
	found, err := ns.GetNodes(ctx, 3)
	require.NoError(t, err)
	require.Len(t, found, 1)

	// get all nodes
	found, err = ns.GetNodes(ctx, 7)
	require.NoError(t, err)
	require.Len(t, found, 0)

	// get single node in epoch where it had a ranking
	node, err := ns.GetNodeByID(ctx, node1.ID.String(), 3)
	require.NoError(t, err)
	require.NotNil(t, node)
	require.NotNil(t, node.RankingScore)

	node, err = ns.GetNodeByID(ctx, "DEADBEEF", 3)
	require.Error(t, err)
}

func TestGetNodesJoiningAndLeaving(t *testing.T) {
	DeleteEverything()
	defer DeleteEverything()
	ctx := context.Background()
	bs := sqlstore.NewBlocks(connectionSource)
	ns := sqlstore.NewNode(connectionSource)
	block := addTestBlock(t, bs)

	node1 := addTestNode(t, ns, block)
	node2 := addTestNode(t, ns, block)

	// The node1 will exist int the epochs [2,3] and [6,7]
	exists := map[int]bool{2: true, 3: true, 6: true, 7: true}
	addNodeAnnounced(t, ns, node1.ID, true, 2)
	addNodeAnnounced(t, ns, node1.ID, false, 4)
	addNodeAnnounced(t, ns, node1.ID, true, 6)
	addNodeAnnounced(t, ns, node1.ID, false, 8)

	// node2 will always exist
	addNodeAnnounced(t, ns, node2.ID, true, 0)

	nodeID1 := node1.ID.String()
	nodeID2 := node2.ID.String()

	assertNodeExistence(t, ctx, ns, nodeID1, 1, false)
	assertNodeExistence(t, ctx, ns, nodeID2, 1, true)
	for i := 1; i < 10; i++ {
		assertNodeExistence(t, ctx, ns, nodeID1, uint64(i), exists[i])
		assertNodeExistence(t, ctx, ns, nodeID2, uint64(i), true)
	}
}

func TestGetNodeData(t *testing.T) {
	DeleteEverything()
	defer DeleteEverything()
	ctx := context.Background()
	bs := sqlstore.NewBlocks(connectionSource)
	ns := sqlstore.NewNode(connectionSource)
	es := sqlstore.NewEpochs(connectionSource)
	ds := sqlstore.NewDelegations(connectionSource)
	ps := sqlstore.NewParties(connectionSource)

	block := addTestBlock(t, bs)
	party1 := addTestParty(t, ps, block)
	node1 := addTestNode(t, ns, block)
	node2 := addTestNode(t, ns, block)

	addTestDelegation(t, ds, party1, node1, 3, block)
	addTestDelegation(t, ds, party1, node1, 4, block)
	addTestDelegation(t, ds, party1, node2, 3, block)
	addTestDelegation(t, ds, party1, node2, 4, block)

	// The node1 will exist int the epochs [2,3]
	addNodeAnnounced(t, ns, node1.ID, true, 2)
	addNodeAnnounced(t, ns, node1.ID, false, 4)

	// node2 will always exist
	addNodeAnnounced(t, ns, node2.ID, true, 0)

	// move to epoch 3 both nodes should exist
	now := time.Unix(2000, 4)
	addTestEpoch(t, es, 3, now, now, &now, block)

	nodes, _ := ns.GetNodes(ctx, 3)
	require.Len(t, nodes, 2)
	nodeData, err := ns.GetNodeData(ctx)
	require.NoError(t, err)
	require.Equal(t, uint32(2), nodeData.TotalNodes)

	// move to epoch 4 and only one should exist
	now = now.Add(time.Hour)
	addTestEpoch(t, es, 4, now, now, &now, block)
	nodeData, err = ns.GetNodeData(ctx)
	require.NoError(t, err)
	require.Equal(t, uint32(1), nodeData.TotalNodes)
}

func assertNodeExistence(t *testing.T, ctx context.Context, ns *sqlstore.Node, nodeID string, epoch uint64, exists bool) {
	t.Helper()
	nodes, err := ns.GetNodes(ctx, epoch)
	require.NoError(t, err)
	node, err := ns.GetNodeByID(ctx, nodeID, epoch)

	found := false
	for _, n := range nodes {
		if n.ID.String() == nodeID {
			found = true
			break
		}
	}

	if !exists {
		require.ErrorIs(t, err, sqlstore.ErrNodeNotFound)
		require.False(t, found)
		return
	}

	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, node.ID.String(), nodeID)
}
