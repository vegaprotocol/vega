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

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/sqlstore"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestNode(t *testing.T, ps *sqlstore.Node, block entities.Block, id string) entities.Node {
	t.Helper()
	node := entities.Node{
		ID:              entities.NewNodeID(id),
		PubKey:          entities.VegaPublicKey(generateID()),
		TmPubKey:        entities.TendermintPublicKey(generateTendermintPublicKey()),
		EthereumAddress: entities.EthereumAddress(generateEthereumAddress()),
		VegaTime:        block.VegaTime,
		Status:          entities.NodeStatusNonValidator,
	}

	err := ps.UpsertNode(context.Background(), &node)
	require.NoError(t, err)
	return node
}

func addNodeAnnounced(t *testing.T, ps *sqlstore.Node, nodeID entities.NodeID, added bool, fromEpoch uint64, vegatime time.Time) {
	t.Helper()
	aux := entities.ValidatorUpdateAux{
		Added:     added,
		FromEpoch: fromEpoch,
	}
	err := ps.AddNodeAnnoucedEvent(context.Background(), nodeID, vegatime, &aux)
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

	now := time.Now()
	node1 := addTestNode(t, ns, block, generateID())
	addNodeAnnounced(t, ns, node1.ID, true, 0, now)
	addNodeAnnounced(t, ns, node1.ID, false, 7, now)
	addRankingScore(t, ns, block, node1, 3)

	// get all nodes
	found, _, err := ns.GetNodes(ctx, 3, entities.CursorPagination{})
	require.NoError(t, err)
	require.Len(t, found, 1)

	// get all nodes
	found, _, err = ns.GetNodes(ctx, 7, entities.CursorPagination{})
	require.NoError(t, err)
	require.Len(t, found, 0)

	// get single node in epoch where it had a ranking
	node, err := ns.GetNodeByID(ctx, node1.ID.String(), 3)
	require.NoError(t, err)
	require.NotNil(t, node)
	require.NotNil(t, node.RankingScore)

	node, err = ns.GetNodeByID(ctx, "DEADBEEF", 3)
	require.Error(t, err)

	// check the value can be changed, since this happens during a checkpoint restore
	// we were need to remove genesis validators if they aren't in the checkpoint
	addNodeAnnounced(t, ns, node1.ID, true, 7, now)
	// get all nodes
	found, _, err = ns.GetNodes(ctx, 7, entities.CursorPagination{})
	require.NoError(t, err)
	require.Len(t, found, 1)
}

func TestGetNodesJoiningAndLeaving(t *testing.T) {
	DeleteEverything()
	defer DeleteEverything()
	ctx := context.Background()
	bs := sqlstore.NewBlocks(connectionSource)
	ns := sqlstore.NewNode(connectionSource)
	block := addTestBlock(t, bs)

	node1 := addTestNode(t, ns, block, generateID())
	node2 := addTestNode(t, ns, block, generateID())

	// The node1 will exist int the epochs [2,3] and [6,7]
	exists := map[int]bool{2: true, 3: true, 6: true, 7: true}
	addNodeAnnounced(t, ns, node1.ID, true, 2, time.Now())
	addNodeAnnounced(t, ns, node1.ID, false, 4, time.Now())
	addNodeAnnounced(t, ns, node1.ID, true, 6, time.Now())
	addNodeAnnounced(t, ns, node1.ID, false, 8, time.Now())

	// node2 will always exist
	addNodeAnnounced(t, ns, node2.ID, true, 0, time.Now())

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
	node1 := addTestNode(t, ns, block, generateID())
	node2 := addTestNode(t, ns, block, generateID())

	addTestDelegation(t, ds, party1, node1, 3, block)
	addTestDelegation(t, ds, party1, node1, 4, block)
	addTestDelegation(t, ds, party1, node2, 3, block)
	addTestDelegation(t, ds, party1, node2, 4, block)

	// The node1 will exist int the epochs [2,3]
	addNodeAnnounced(t, ns, node1.ID, true, 2, time.Now())
	addNodeAnnounced(t, ns, node1.ID, false, 4, time.Now())

	// node2 will always exist
	addNodeAnnounced(t, ns, node2.ID, true, 0, time.Now())

	// move to epoch 3 both nodes should exist
	now := time.Unix(2000, 4)
	addTestEpoch(t, es, 3, now, now, &now, block)

	nodes, _, _ := ns.GetNodes(ctx, 3, entities.CursorPagination{})
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
	nodes, _, err := ns.GetNodes(ctx, epoch, entities.CursorPagination{})
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

func TestNodePagination(t *testing.T) {
	t.Run("Should return all nodes if no pagination is specified", testNodePaginationNoPagination)
	t.Run("Should return first page of results if first is provided", testNodePaginationFirst)
	t.Run("Should return last page of results if last is provided", testNodePaginationLast)
	t.Run("Should return requested page of results if first and after is provided", testNodePaginationFirstAfter)
	t.Run("Should return requested page of results if last and before is provided", testNodePaginationLastBefore)
}

func addPaginationTestNodes(t *testing.T, ns *sqlstore.Node) (nodes []entities.Node) {
	blockTime := time.Now().Add(-time.Hour)
	bs := sqlstore.NewBlocks(connectionSource)

	nodes = append(nodes, addTestNode(t, ns, addTestBlockForTime(t, bs, blockTime), "deadbeef01"))
	blockTime = blockTime.Add(time.Minute)
	nodes = append(nodes, addTestNode(t, ns, addTestBlockForTime(t, bs, blockTime), "deadbeef02"))
	blockTime = blockTime.Add(time.Minute)
	nodes = append(nodes, addTestNode(t, ns, addTestBlockForTime(t, bs, blockTime), "deadbeef03"))
	blockTime = blockTime.Add(time.Minute)
	nodes = append(nodes, addTestNode(t, ns, addTestBlockForTime(t, bs, blockTime), "deadbeef04"))
	blockTime = blockTime.Add(time.Minute)
	nodes = append(nodes, addTestNode(t, ns, addTestBlockForTime(t, bs, blockTime), "deadbeef05"))
	blockTime = blockTime.Add(time.Minute)
	nodes = append(nodes, addTestNode(t, ns, addTestBlockForTime(t, bs, blockTime), "deadbeef06"))
	blockTime = blockTime.Add(time.Minute)
	nodes = append(nodes, addTestNode(t, ns, addTestBlockForTime(t, bs, blockTime), "deadbeef07"))
	blockTime = blockTime.Add(time.Minute)
	nodes = append(nodes, addTestNode(t, ns, addTestBlockForTime(t, bs, blockTime), "deadbeef08"))
	blockTime = blockTime.Add(time.Minute)
	nodes = append(nodes, addTestNode(t, ns, addTestBlockForTime(t, bs, blockTime), "deadbeef09"))
	blockTime = blockTime.Add(time.Minute)
	nodes = append(nodes, addTestNode(t, ns, addTestBlockForTime(t, bs, blockTime), "deadbeef10"))
	addNodeAnnounced(t, ns, nodes[0].ID, true, 1, nodes[0].VegaTime)
	addNodeAnnounced(t, ns, nodes[1].ID, true, 1, nodes[1].VegaTime)
	addNodeAnnounced(t, ns, nodes[2].ID, true, 1, nodes[2].VegaTime)
	addNodeAnnounced(t, ns, nodes[3].ID, true, 1, nodes[3].VegaTime)
	addNodeAnnounced(t, ns, nodes[4].ID, true, 1, nodes[4].VegaTime)
	addNodeAnnounced(t, ns, nodes[5].ID, true, 1, nodes[5].VegaTime)
	addNodeAnnounced(t, ns, nodes[6].ID, true, 1, nodes[6].VegaTime)
	addNodeAnnounced(t, ns, nodes[7].ID, true, 1, nodes[7].VegaTime)
	addNodeAnnounced(t, ns, nodes[8].ID, true, 1, nodes[8].VegaTime)
	addNodeAnnounced(t, ns, nodes[9].ID, true, 1, nodes[9].VegaTime)

	return
}

func testNodePaginationNoPagination(t *testing.T) {
	defer DeleteEverything()
	ns := sqlstore.NewNode(connectionSource)
	nodes := addPaginationTestNodes(t, ns)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	got, pageInfo, err := ns.GetNodes(timeoutCtx, 1, pagination)
	require.NoError(t, err)
	assert.Len(t, got, len(nodes))
	assert.Equal(t, nodes[0].ID, got[0].ID)
	assert.Equal(t, nodes[1].ID, got[1].ID)
	assert.Equal(t, nodes[2].ID, got[2].ID)
	assert.Equal(t, nodes[3].ID, got[3].ID)
	assert.Equal(t, nodes[4].ID, got[4].ID)
	assert.Equal(t, nodes[5].ID, got[5].ID)
	assert.Equal(t, nodes[6].ID, got[6].ID)
	assert.Equal(t, nodes[7].ID, got[7].ID)
	assert.Equal(t, nodes[8].ID, got[8].ID)
	assert.Equal(t, nodes[9].ID, got[9].ID)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     nodes[0].Cursor().Encode(),
		EndCursor:       nodes[9].Cursor().Encode(),
	}, pageInfo)
}

func testNodePaginationFirst(t *testing.T) {
	defer DeleteEverything()
	ns := sqlstore.NewNode(connectionSource)
	nodes := addPaginationTestNodes(t, ns)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	got, pageInfo, err := ns.GetNodes(timeoutCtx, 1, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	assert.Equal(t, nodes[0].ID, got[0].ID)
	assert.Equal(t, nodes[1].ID, got[1].ID)
	assert.Equal(t, nodes[2].ID, got[2].ID)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     nodes[0].Cursor().Encode(),
		EndCursor:       nodes[2].Cursor().Encode(),
	}, pageInfo)
}

func testNodePaginationLast(t *testing.T) {
	defer DeleteEverything()
	ns := sqlstore.NewNode(connectionSource)
	nodes := addPaginationTestNodes(t, ns)

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	got, pageInfo, err := ns.GetNodes(timeoutCtx, 1, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	assert.Equal(t, nodes[7].ID, got[0].ID)
	assert.Equal(t, nodes[8].ID, got[1].ID)
	assert.Equal(t, nodes[9].ID, got[2].ID)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     nodes[7].Cursor().Encode(),
		EndCursor:       nodes[9].Cursor().Encode(),
	}, pageInfo)
}

func testNodePaginationFirstAfter(t *testing.T) {
	defer DeleteEverything()
	ns := sqlstore.NewNode(connectionSource)
	nodes := addPaginationTestNodes(t, ns)

	first := int32(3)
	after := nodes[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	got, pageInfo, err := ns.GetNodes(timeoutCtx, 1, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	assert.Equal(t, nodes[3].ID, got[0].ID)
	assert.Equal(t, nodes[4].ID, got[1].ID)
	assert.Equal(t, nodes[5].ID, got[2].ID)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     nodes[3].Cursor().Encode(),
		EndCursor:       nodes[5].Cursor().Encode(),
	}, pageInfo)
}

func testNodePaginationLastBefore(t *testing.T) {
	defer DeleteEverything()
	ns := sqlstore.NewNode(connectionSource)
	nodes := addPaginationTestNodes(t, ns)

	last := int32(3)
	before := nodes[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	got, pageInfo, err := ns.GetNodes(timeoutCtx, 1, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	assert.Equal(t, nodes[4].ID, got[0].ID)
	assert.Equal(t, nodes[5].ID, got[1].ID)
	assert.Equal(t, nodes[6].ID, got[2].ID)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     nodes[4].Cursor().Encode(),
		EndCursor:       nodes[6].Cursor().Encode(),
	}, pageInfo)
}
