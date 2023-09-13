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

package sqlstore_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestNode(t *testing.T, ctx context.Context, ps *sqlstore.Node, block entities.Block, id string) entities.Node {
	t.Helper()
	node := entities.Node{
		ID:              entities.NodeID(id),
		PubKey:          entities.VegaPublicKey(helpers.GenerateID()),
		TmPubKey:        entities.TendermintPublicKey(generateTendermintPublicKey()),
		EthereumAddress: entities.EthereumAddress(generateEthereumAddress()),
		VegaTime:        block.VegaTime,
		Status:          entities.NodeStatusNonValidator,
		TxHash:          generateTxHash(),
	}

	err := ps.UpsertNode(ctx, &node)
	require.NoError(t, err)
	return node
}

func addNodeAnnounced(t *testing.T, ctx context.Context, ps *sqlstore.Node, nodeID entities.NodeID, added bool, epochSeq uint64, vegatime time.Time) {
	t.Helper()
	aux := entities.ValidatorUpdateAux{
		Added:    added,
		EpochSeq: epochSeq,
	}
	err := ps.AddNodeAnnouncedEvent(ctx, nodeID.String(), vegatime, &aux)
	require.NoError(t, err)
}

func addRankingScore(t *testing.T, ctx context.Context, ps *sqlstore.Node, node entities.Node, r entities.RankingScore) {
	t.Helper()

	aux := entities.RankingScoreAux{
		NodeID:   node.ID,
		EpochSeq: r.EpochSeq,
	}

	err := ps.UpsertRanking(ctx, &r, &aux)
	require.NoError(t, err)
}

func TestUpdateNodePubKey(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)
	ns := sqlstore.NewNode(connectionSource)
	block := addTestBlock(t, ctx, bs)

	now := time.Now()
	node1 := addTestNode(t, ctx, ns, block, helpers.GenerateID())
	addNodeAnnounced(t, ctx, ns, node1.ID, true, 0, now)

	kr := entities.KeyRotation{
		NodeID:    node1.ID,
		OldPubKey: node1.PubKey,
		NewPubKey: entities.VegaPublicKey(helpers.GenerateID()),
		VegaTime:  block.VegaTime,
	}

	ns.UpdatePublicKey(ctx, &kr)

	fetched, err := ns.GetNodeByID(ctx, node1.ID.String(), 1)
	assert.NoError(t, err)
	assert.Equal(t, fetched.PubKey, kr.NewPubKey)
}

func TestGetNodes(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)
	ns := sqlstore.NewNode(connectionSource)
	block := addTestBlock(t, ctx, bs)

	now := time.Now()
	node1 := addTestNode(t, ctx, ns, block, helpers.GenerateID())
	addNodeAnnounced(t, ctx, ns, node1.ID, true, 0, now)
	addNodeAnnounced(t, ctx, ns, node1.ID, false, 7, now)
	addRankingScore(t, ctx, ns, node1,
		entities.RankingScore{
			StakeScore:       decimal.NewFromFloat(0.5),
			PerformanceScore: decimal.NewFromFloat(0.25),
			PreviousStatus:   entities.ValidatorNodeStatusErsatz,
			Status:           entities.ValidatorNodeStatusTendermint,
			EpochSeq:         3,
			VegaTime:         block.VegaTime,
		})

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
	addNodeAnnounced(t, ctx, ns, node1.ID, true, 7, now)
	// get all nodes
	found, _, err = ns.GetNodes(ctx, 7, entities.CursorPagination{})
	require.NoError(t, err)
	require.Len(t, found, 1)
}

func TestNodeGetByTxHash(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)
	ns := sqlstore.NewNode(connectionSource)
	block := addTestBlock(t, ctx, bs)

	now := time.Now()
	node1 := addTestNode(t, ctx, ns, block, helpers.GenerateID())
	node2 := addTestNode(t, ctx, ns, block, helpers.GenerateID())
	addNodeAnnounced(t, ctx, ns, node1.ID, true, 0, now)
	addNodeAnnounced(t, ctx, ns, node2.ID, false, 7, now)
	addNodeAnnounced(t, ctx, ns, node2.ID, false, 9, now)

	found, err := ns.GetByTxHash(ctx, node1.TxHash)
	require.NoError(t, err)
	require.Len(t, found, 1)
	require.Equal(t, node1.ID, found[0].ID)

	found, err = ns.GetByTxHash(ctx, node2.TxHash)
	require.NoError(t, err)
	require.Len(t, found, 1)
	require.Equal(t, node2.ID, found[0].ID)
}

func TestGetNodesJoiningAndLeaving(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)
	ns := sqlstore.NewNode(connectionSource)
	block := addTestBlock(t, ctx, bs)

	node1 := addTestNode(t, ctx, ns, block, helpers.GenerateID())
	node2 := addTestNode(t, ctx, ns, block, helpers.GenerateID())

	// The node1 will exist int the epochs [2,3] and [6,7]
	exists := map[int]bool{2: true, 3: true, 6: true, 7: true}
	addNodeAnnounced(t, ctx, ns, node1.ID, true, 2, time.Now())
	addNodeAnnounced(t, ctx, ns, node1.ID, false, 4, time.Now())
	addNodeAnnounced(t, ctx, ns, node1.ID, true, 6, time.Now())
	addNodeAnnounced(t, ctx, ns, node1.ID, false, 8, time.Now())

	// node2 will always exist
	addNodeAnnounced(t, ctx, ns, node2.ID, true, 0, time.Now())

	nodeID1 := node1.ID.String()
	nodeID2 := node2.ID.String()

	assertNodeExistence(ctx, t, ns, nodeID1, 1, false)
	assertNodeExistence(ctx, t, ns, nodeID2, 1, true)
	for i := 1; i < 10; i++ {
		assertNodeExistence(ctx, t, ns, nodeID1, uint64(i), exists[i])
		assertNodeExistence(ctx, t, ns, nodeID2, uint64(i), true)
	}
}

func TestGetNodeData(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)
	ns := sqlstore.NewNode(connectionSource)
	es := sqlstore.NewEpochs(connectionSource)
	ds := sqlstore.NewDelegations(connectionSource)
	ps := sqlstore.NewParties(connectionSource)

	block := addTestBlock(t, ctx, bs)
	party1 := addTestParty(t, ctx, ps, block)
	node1 := addTestNode(t, ctx, ns, block, helpers.GenerateID())
	node2 := addTestNode(t, ctx, ns, block, helpers.GenerateID())

	addTestDelegation(t, ctx, ds, party1, node1, 3, block, 0)
	addTestDelegation(t, ctx, ds, party1, node1, 4, block, 1)
	addTestDelegation(t, ctx, ds, party1, node2, 3, block, 2)
	addTestDelegation(t, ctx, ds, party1, node2, 4, block, 3)

	// node1 goes from pending -> ersatz -> tendermint
	// then gets demoted straight to pending with a zero perf score
	addRankingScore(t, ctx, ns, node1,
		entities.RankingScore{
			StakeScore:       decimal.NewFromFloat(0.5),
			PerformanceScore: decimal.NewFromFloat(0.25),
			PreviousStatus:   entities.ValidatorNodeStatusPending,
			Status:           entities.ValidatorNodeStatusErsatz,
			EpochSeq:         2,
			VegaTime:         block.VegaTime,
		})
	addRankingScore(t, ctx, ns, node1,
		entities.RankingScore{
			StakeScore:       decimal.NewFromFloat(0.5),
			PerformanceScore: decimal.NewFromFloat(0.25),
			PreviousStatus:   entities.ValidatorNodeStatusErsatz,
			Status:           entities.ValidatorNodeStatusTendermint,
			EpochSeq:         3,
			VegaTime:         block.VegaTime,
		})
	addRankingScore(t, ctx, ns, node1,
		entities.RankingScore{
			StakeScore:       decimal.NewFromFloat(0.5),
			PerformanceScore: decimal.Zero,
			PreviousStatus:   entities.ValidatorNodeStatusTendermint,
			Status:           entities.ValidatorNodeStatusPending,
			EpochSeq:         4,
			VegaTime:         block.VegaTime,
		})

	// node 2 is always a happy tendermint node
	for i := 0; i < 6; i++ {
		addRankingScore(t, ctx, ns, node2, entities.RankingScore{
			StakeScore:       decimal.NewFromFloat(0.5),
			PerformanceScore: decimal.NewFromFloat(0.25),
			PreviousStatus:   entities.ValidatorNodeStatusTendermint,
			Status:           entities.ValidatorNodeStatusTendermint,
			EpochSeq:         uint64(i),
			VegaTime:         block.VegaTime,
		})
	}

	// The node1 will exist int the epochs [2,3,4]
	addNodeAnnounced(t, ctx, ns, node1.ID, true, 2, time.Now())
	addNodeAnnounced(t, ctx, ns, node1.ID, false, 5, time.Now())

	// node2 will always exist
	addNodeAnnounced(t, ctx, ns, node2.ID, true, 0, time.Now())

	// move to epoch 2 both nodes should exist
	now := time.Unix(2000, 4)
	addTestEpoch(t, ctx, es, 2, now, now, &now, block)
	nodeData, err := ns.GetNodeData(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, uint32(2), nodeData.TotalNodes)
	require.Equal(t, entities.NodeSet{
		Total: 1,
	}, nodeData.TendermintNodes)
	require.Equal(t, entities.NodeSet{
		Total:    1,
		Promoted: []string{node1.ID.String()},
	}, nodeData.ErsatzNodes)
	require.Equal(t, entities.NodeSet{}, nodeData.PendingNodes)

	// move to epoch 3 and check promotions
	addTestEpoch(t, ctx, es, 3, now, now, &now, block)
	nodeData, err = ns.GetNodeData(ctx, 3)
	require.NoError(t, err)
	require.Equal(t, uint32(2), nodeData.TotalNodes)
	require.Equal(t, entities.NodeSet{
		Total:    2,
		Promoted: []string{node1.ID.String()},
	}, nodeData.TendermintNodes)
	require.Equal(t, entities.NodeSet{}, nodeData.ErsatzNodes)
	require.Equal(t, entities.NodeSet{}, nodeData.PendingNodes)

	// move to epoch 4 and check demotions
	now = now.Add(time.Hour)
	addTestEpoch(t, ctx, es, 4, now, now, &now, block)
	nodeData, err = ns.GetNodeData(ctx, 4)
	require.NoError(t, err)
	require.Equal(t, uint32(2), nodeData.TotalNodes)
	require.Equal(t, uint32(1), nodeData.InactiveNodes)
	require.Equal(t, entities.NodeSet{
		Total: 1,
	}, nodeData.TendermintNodes)
	require.Equal(t, entities.NodeSet{}, nodeData.ErsatzNodes)
	require.Equal(t, entities.NodeSet{
		Total:    1,
		Inactive: 1,
		Demoted:  []string{node1.ID.String()},
	}, nodeData.PendingNodes)

	// move to epoch 5 just have one tendermint node
	now = now.Add(time.Hour)
	addTestEpoch(t, ctx, es, 5, now, now, &now, block)
	nodeData, err = ns.GetNodeData(ctx, 5)
	require.NoError(t, err)
	require.Equal(t, uint32(1), nodeData.TotalNodes)
	require.Equal(t, entities.NodeSet{
		Total: 1,
	}, nodeData.TendermintNodes)
	require.Equal(t, entities.NodeSet{}, nodeData.ErsatzNodes)
	require.Equal(t, entities.NodeSet{}, nodeData.PendingNodes)
}

func assertNodeExistence(ctx context.Context, t *testing.T, ns *sqlstore.Node, nodeID string, epoch uint64, exists bool) {
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
		require.ErrorIs(t, err, entities.ErrNotFound)
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

func addPaginationTestNodes(t *testing.T, ctx context.Context, ns *sqlstore.Node) (nodes []entities.Node) {
	t.Helper()
	blockTime := time.Now().Add(-time.Hour)
	bs := sqlstore.NewBlocks(connectionSource)

	nodes = append(nodes, addTestNode(t, ctx, ns, addTestBlockForTime(t, ctx, bs, blockTime), "deadbeef01"))
	blockTime = blockTime.Add(time.Minute)
	nodes = append(nodes, addTestNode(t, ctx, ns, addTestBlockForTime(t, ctx, bs, blockTime), "deadbeef02"))
	blockTime = blockTime.Add(time.Minute)
	nodes = append(nodes, addTestNode(t, ctx, ns, addTestBlockForTime(t, ctx, bs, blockTime), "deadbeef03"))
	blockTime = blockTime.Add(time.Minute)
	nodes = append(nodes, addTestNode(t, ctx, ns, addTestBlockForTime(t, ctx, bs, blockTime), "deadbeef04"))
	blockTime = blockTime.Add(time.Minute)
	nodes = append(nodes, addTestNode(t, ctx, ns, addTestBlockForTime(t, ctx, bs, blockTime), "deadbeef05"))
	blockTime = blockTime.Add(time.Minute)
	nodes = append(nodes, addTestNode(t, ctx, ns, addTestBlockForTime(t, ctx, bs, blockTime), "deadbeef06"))
	blockTime = blockTime.Add(time.Minute)
	nodes = append(nodes, addTestNode(t, ctx, ns, addTestBlockForTime(t, ctx, bs, blockTime), "deadbeef07"))
	blockTime = blockTime.Add(time.Minute)
	nodes = append(nodes, addTestNode(t, ctx, ns, addTestBlockForTime(t, ctx, bs, blockTime), "deadbeef08"))
	blockTime = blockTime.Add(time.Minute)
	nodes = append(nodes, addTestNode(t, ctx, ns, addTestBlockForTime(t, ctx, bs, blockTime), "deadbeef09"))
	blockTime = blockTime.Add(time.Minute)
	nodes = append(nodes, addTestNode(t, ctx, ns, addTestBlockForTime(t, ctx, bs, blockTime), "deadbeef10"))
	addNodeAnnounced(t, ctx, ns, nodes[0].ID, true, 1, nodes[0].VegaTime)
	addNodeAnnounced(t, ctx, ns, nodes[1].ID, true, 1, nodes[1].VegaTime)
	addNodeAnnounced(t, ctx, ns, nodes[2].ID, true, 1, nodes[2].VegaTime)
	addNodeAnnounced(t, ctx, ns, nodes[3].ID, true, 1, nodes[3].VegaTime)
	addNodeAnnounced(t, ctx, ns, nodes[4].ID, true, 1, nodes[4].VegaTime)
	addNodeAnnounced(t, ctx, ns, nodes[5].ID, true, 1, nodes[5].VegaTime)
	addNodeAnnounced(t, ctx, ns, nodes[6].ID, true, 1, nodes[6].VegaTime)
	addNodeAnnounced(t, ctx, ns, nodes[7].ID, true, 1, nodes[7].VegaTime)
	addNodeAnnounced(t, ctx, ns, nodes[8].ID, true, 1, nodes[8].VegaTime)
	addNodeAnnounced(t, ctx, ns, nodes[9].ID, true, 1, nodes[9].VegaTime)

	return nodes
}

func testNodePaginationNoPagination(t *testing.T) {
	ctx := tempTransaction(t)

	ns := sqlstore.NewNode(connectionSource)
	nodes := addPaginationTestNodes(t, ctx, ns)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ns.GetNodes(ctx, 1, pagination)
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
	ctx := tempTransaction(t)

	ns := sqlstore.NewNode(connectionSource)
	nodes := addPaginationTestNodes(t, ctx, ns)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ns.GetNodes(ctx, 1, pagination)
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
	ctx := tempTransaction(t)

	ns := sqlstore.NewNode(connectionSource)
	nodes := addPaginationTestNodes(t, ctx, ns)

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ns.GetNodes(ctx, 1, pagination)
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
	ctx := tempTransaction(t)

	ns := sqlstore.NewNode(connectionSource)
	nodes := addPaginationTestNodes(t, ctx, ns)

	first := int32(3)
	after := nodes[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ns.GetNodes(ctx, 1, pagination)
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
	ctx := tempTransaction(t)

	ns := sqlstore.NewNode(connectionSource)
	nodes := addPaginationTestNodes(t, ctx, ns)

	last := int32(3)
	before := nodes[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)

	got, pageInfo, err := ns.GetNodes(ctx, 1, pagination)
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

func TestNode_AddRankingScoreInSameEpoch(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)
	ns := sqlstore.NewNode(connectionSource)

	block := addTestBlock(t, ctx, bs)
	node1 := addTestNode(t, ctx, ns, block, helpers.GenerateID())

	// node1 goes from pending -> ersatz -> tendermint
	// then gets demoted straight to pending with a zero perf score
	addRankingScore(t, ctx, ns, node1,
		entities.RankingScore{
			StakeScore:       decimal.NewFromFloat(0.5),
			PerformanceScore: decimal.NewFromFloat(0.25),
			PreviousStatus:   entities.ValidatorNodeStatusPending,
			Status:           entities.ValidatorNodeStatusErsatz,
			EpochSeq:         2,
			VegaTime:         block.VegaTime,
		})
	addRankingScore(t, ctx, ns, node1,
		entities.RankingScore{
			StakeScore:       decimal.NewFromFloat(0.5),
			PerformanceScore: decimal.Zero,
			PreviousStatus:   entities.ValidatorNodeStatusTendermint,
			Status:           entities.ValidatorNodeStatusPending,
			EpochSeq:         2,
			VegaTime:         block.VegaTime,
		})
}
