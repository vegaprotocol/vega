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
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestDelegation(t *testing.T, ds *sqlstore.Delegations,
	party entities.Party,
	node entities.Node,
	epochID int64,
	block entities.Block,
) entities.Delegation {
	r := entities.Delegation{
		PartyID:  party.ID,
		NodeID:   node.ID,
		EpochID:  epochID,
		Amount:   decimal.NewFromInt(100),
		VegaTime: block.VegaTime,
	}
	err := ds.Add(context.Background(), r)
	require.NoError(t, err)
	return r
}

func delegationLessThan(x, y entities.Delegation) bool {
	if x.EpochID != y.EpochID {
		return x.EpochID < y.EpochID
	}
	if x.PartyID.String() != y.PartyID.String() {
		return x.PartyID.String() < y.PartyID.String()
	}
	if x.NodeID.String() != y.NodeID.String() {
		return x.NodeID.String() < y.NodeID.String()
	}
	return x.Amount.LessThan(y.Amount)
}

func assertDelegationsMatch(t *testing.T, expected, actual []entities.Delegation) {
	t.Helper()
	assert.Empty(t, cmp.Diff(expected, actual, cmpopts.SortSlices(delegationLessThan)))
}

func TestDelegations(t *testing.T) {
	defer DeleteEverything()
	ps := sqlstore.NewParties(connectionSource)
	ds := sqlstore.NewDelegations(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)
	ns := sqlstore.NewNode(connectionSource)
	block := addTestBlock(t, bs)

	node1 := addTestNode(t, ns, block, generateID())
	node2 := addTestNode(t, ns, block, generateID())

	node1ID := node1.ID.String()
	node2ID := node2.ID.String()

	party1 := addTestParty(t, ps, block)
	party2 := addTestParty(t, ps, block)

	party1ID := party1.ID.String()
	party2ID := party2.ID.String()

	delegation1 := addTestDelegation(t, ds, party1, node1, 1, block)
	delegation2 := addTestDelegation(t, ds, party1, node2, 2, block)
	delegation3 := addTestDelegation(t, ds, party2, node1, 3, block)
	delegation4 := addTestDelegation(t, ds, party2, node2, 4, block)
	delegation5 := addTestDelegation(t, ds, party2, node2, 5, block)

	t.Run("GetAll", func(t *testing.T) {
		expected := []entities.Delegation{delegation1, delegation2, delegation3, delegation4, delegation5}
		actual, err := ds.GetAll(context.Background())
		require.NoError(t, err)
		assertDelegationsMatch(t, expected, actual)
	})

	t.Run("GetByParty", func(t *testing.T) {
		expected := []entities.Delegation{delegation1, delegation2}
		actual, _, err := ds.Get(context.Background(), &party1ID, nil, nil, nil)
		require.NoError(t, err)
		assertDelegationsMatch(t, expected, actual)
	})

	t.Run("GetByNode", func(t *testing.T) {
		expected := []entities.Delegation{delegation1, delegation3}
		actual, _, err := ds.Get(context.Background(), nil, &node1ID, nil, nil)
		require.NoError(t, err)
		assertDelegationsMatch(t, expected, actual)
	})

	t.Run("GetByEpoch", func(t *testing.T) {
		expected := []entities.Delegation{delegation4}
		four := int64(4)
		actual, _, err := ds.Get(context.Background(), nil, nil, &four, nil)
		require.NoError(t, err)
		assertDelegationsMatch(t, expected, actual)
	})

	t.Run("GetByPartyAndNode", func(t *testing.T) {
		expected := []entities.Delegation{delegation4, delegation5}
		actual, _, err := ds.Get(context.Background(), &party2ID, &node2ID, nil, nil)
		require.NoError(t, err)
		assertDelegationsMatch(t, expected, actual)
	})

	t.Run("GetByPartyAndNodeAndEpoch", func(t *testing.T) {
		expected := []entities.Delegation{delegation4}
		four := int64(4)
		actual, _, err := ds.Get(context.Background(), &party2ID, &node2ID, &four, nil)
		require.NoError(t, err)
		assertDelegationsMatch(t, expected, actual)
	})

	t.Run("GetPagination", func(t *testing.T) {
		expected := []entities.Delegation{delegation4, delegation3, delegation2}
		p := entities.OffsetPagination{Skip: 1, Limit: 3, Descending: true}
		actual, _, err := ds.Get(context.Background(), nil, nil, nil, &p)
		require.NoError(t, err)
		assert.Equal(t, expected, actual) // Explicitly check the order on this one
	})
}

func TestDelegationPagination(t *testing.T) {
	t.Run("Should return all delegations if no filter or pagination is provided", testDelegationPaginationNoFilterNoPagination)
	t.Run("Should return the first page if no filter but first is provided", testDelegationPaginationNoFilterFirstPage)
	t.Run("Should return the request page if no filter but first after is provided", testDelegationPaginationNoFilterFirstAfterPage)
	t.Run("Should return the last page if no filter but last is provided", testDelegationPaginationNoFilterLastPage)
	t.Run("Should return the request page if no filter but last before is provided", testDelegationPaginationNoFilterLastBeforePage)

	t.Run("Should return all delegations if no filter or pagination is provided - newest first", testDelegationPaginationNoFilterNoPaginationNewestFirst)
	t.Run("Should return the first page if no filter but first is provided - newest first", testDelegationPaginationNoFilterFirstPageNewestFirst)
	t.Run("Should return the request page if no filter but first after is provided - newest first", testDelegationPaginationNoFilterFirstAfterPageNewestFirst)
	t.Run("Should return the last page if no filter but last is provided - newest first", testDelegationPaginationNoFilterLastPageNewestFirst)
	t.Run("Should return the request page if no filter but last before is provided - newest first", testDelegationPaginationNoFilterLastBeforePageNewestFirst)

	t.Run("Should return all delegations if party filter is provided and pagination not provided", testDelegationPaginationPartyFilterNoPagination)
	t.Run("Should return the first page if party filter and first is provided", testDelegationPaginationPartyFilterFirstPage)
	t.Run("Should return the request page if party filter and first after is provided", testDelegationPaginationPartyFilterFirstAfterPage)
	t.Run("Should return the last page if party filter and last is provided", testDelegationPaginationPartyFilterLastPage)
	t.Run("Should return the request page if party filter and last before is provided", testDelegationPaginationPartyFilterLastBeforePage)

	t.Run("Should return all delegations if party/node filter is provided and pagination not provided", testDelegationPaginationPartyNodeFilterNoPagination)
	t.Run("Should return the first page if party/node filter and first is provided", testDelegationPaginationPartyNodeFilterFirstPage)
	t.Run("Should return the request page if party/node filter and first after is provided", testDelegationPaginationPartyNodeFilterFirstAfterPage)
	t.Run("Should return the last page if party/node filter and last is provided", testDelegationPaginationPartyNodeFilterLastPage)
	t.Run("Should return the request page if party/node filter and last before is provided", testDelegationPaginationPartyNodeFilterLastBeforePage)
}

func testDelegationPaginationNoFilterNoPagination(t *testing.T) {
	defer DeleteEverything()
	ds, delegations, _, _ := setupPaginatedDelegationsTests(t)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	got, pageInfo, err := ds.Get(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)

	want := delegations[:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     delegations[0].Cursor().Encode(),
		EndCursor:       delegations[19].Cursor().Encode(),
	}, pageInfo)
}

func testDelegationPaginationNoFilterFirstPage(t *testing.T) {
	defer DeleteEverything()
	ds, delegations, _, _ := setupPaginatedDelegationsTests(t)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	got, pageInfo, err := ds.Get(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)

	want := delegations[:3]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     delegations[0].Cursor().Encode(),
		EndCursor:       delegations[2].Cursor().Encode(),
	}, pageInfo)
}

func testDelegationPaginationNoFilterFirstAfterPage(t *testing.T) {
	defer DeleteEverything()
	ds, delegations, _, _ := setupPaginatedDelegationsTests(t)
	first := int32(3)
	after := delegations[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	got, pageInfo, err := ds.Get(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)

	want := delegations[3:6]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     delegations[3].Cursor().Encode(),
		EndCursor:       delegations[5].Cursor().Encode(),
	}, pageInfo)
}

func testDelegationPaginationNoFilterLastPage(t *testing.T) {
	defer DeleteEverything()
	ds, delegations, _, _ := setupPaginatedDelegationsTests(t)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	got, pageInfo, err := ds.Get(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)

	want := delegations[17:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     delegations[17].Cursor().Encode(),
		EndCursor:       delegations[19].Cursor().Encode(),
	}, pageInfo)
}

func testDelegationPaginationNoFilterLastBeforePage(t *testing.T) {
	defer DeleteEverything()
	ds, delegations, _, _ := setupPaginatedDelegationsTests(t)
	last := int32(3)
	before := delegations[17].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	got, pageInfo, err := ds.Get(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)

	want := delegations[14:17]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     delegations[14].Cursor().Encode(),
		EndCursor:       delegations[16].Cursor().Encode(),
	}, pageInfo)
}

func testDelegationPaginationNoFilterNoPaginationNewestFirst(t *testing.T) {
	defer DeleteEverything()
	ds, delegations, _, _ := setupPaginatedDelegationsTests(t)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	got, pageInfo, err := ds.Get(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)

	delegations = entities.ReverseSlice(delegations)
	want := delegations[:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     delegations[0].Cursor().Encode(),
		EndCursor:       delegations[19].Cursor().Encode(),
	}, pageInfo)

}

func testDelegationPaginationNoFilterFirstPageNewestFirst(t *testing.T) {
	defer DeleteEverything()
	ds, delegations, _, _ := setupPaginatedDelegationsTests(t)
	delegations = entities.ReverseSlice(delegations)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	got, pageInfo, err := ds.Get(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)

	want := delegations[:3]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     delegations[0].Cursor().Encode(),
		EndCursor:       delegations[2].Cursor().Encode(),
	}, pageInfo)
}

func testDelegationPaginationNoFilterFirstAfterPageNewestFirst(t *testing.T) {
	defer DeleteEverything()
	ds, delegations, _, _ := setupPaginatedDelegationsTests(t)
	delegations = entities.ReverseSlice(delegations)
	first := int32(3)
	after := delegations[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	got, pageInfo, err := ds.Get(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)

	want := delegations[3:6]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     delegations[3].Cursor().Encode(),
		EndCursor:       delegations[5].Cursor().Encode(),
	}, pageInfo)
}

func testDelegationPaginationNoFilterLastPageNewestFirst(t *testing.T) {
	defer DeleteEverything()
	ds, delegations, _, _ := setupPaginatedDelegationsTests(t)
	delegations = entities.ReverseSlice(delegations)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	got, pageInfo, err := ds.Get(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)

	want := delegations[17:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     delegations[17].Cursor().Encode(),
		EndCursor:       delegations[19].Cursor().Encode(),
	}, pageInfo)
}

func testDelegationPaginationNoFilterLastBeforePageNewestFirst(t *testing.T) {
	defer DeleteEverything()
	ds, delegations, _, _ := setupPaginatedDelegationsTests(t)
	delegations = entities.ReverseSlice(delegations)
	last := int32(3)
	before := delegations[17].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	got, pageInfo, err := ds.Get(ctx, nil, nil, nil, pagination)
	require.NoError(t, err)

	want := delegations[14:17]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     delegations[14].Cursor().Encode(),
		EndCursor:       delegations[16].Cursor().Encode(),
	}, pageInfo)
}

func testDelegationPaginationPartyFilterNoPagination(t *testing.T) {
	defer DeleteEverything()
	ds, delegations, parties, _ := setupPaginatedDelegationsTests(t)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	partyID := parties[0].ID.String()
	got, pageInfo, err := ds.Get(ctx, &partyID, nil, nil, pagination)
	require.NoError(t, err)

	want := delegations[0:10]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     delegations[0].Cursor().Encode(),
		EndCursor:       delegations[9].Cursor().Encode(),
	}, pageInfo)
}

func testDelegationPaginationPartyFilterFirstPage(t *testing.T) {
	defer DeleteEverything()
	ds, delegations, parties, _ := setupPaginatedDelegationsTests(t)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	partyID := parties[0].ID.String()
	got, pageInfo, err := ds.Get(ctx, &partyID, nil, nil, pagination)
	require.NoError(t, err)

	want := delegations[0:3]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     delegations[0].Cursor().Encode(),
		EndCursor:       delegations[2].Cursor().Encode(),
	}, pageInfo)
}

func testDelegationPaginationPartyFilterFirstAfterPage(t *testing.T) {
	defer DeleteEverything()
	ds, delegations, parties, _ := setupPaginatedDelegationsTests(t)
	first := int32(3)
	after := delegations[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	partyID := parties[0].ID.String()
	got, pageInfo, err := ds.Get(ctx, &partyID, nil, nil, pagination)
	require.NoError(t, err)

	want := delegations[3:6]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     delegations[3].Cursor().Encode(),
		EndCursor:       delegations[5].Cursor().Encode(),
	}, pageInfo)
}

func testDelegationPaginationPartyFilterLastPage(t *testing.T) {
	defer DeleteEverything()
	ds, delegations, parties, _ := setupPaginatedDelegationsTests(t)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	partyID := parties[0].ID.String()
	got, pageInfo, err := ds.Get(ctx, &partyID, nil, nil, pagination)
	require.NoError(t, err)

	want := delegations[7:10]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     delegations[7].Cursor().Encode(),
		EndCursor:       delegations[9].Cursor().Encode(),
	}, pageInfo)
}

func testDelegationPaginationPartyFilterLastBeforePage(t *testing.T) {
	defer DeleteEverything()
	ds, delegations, parties, _ := setupPaginatedDelegationsTests(t)
	last := int32(3)
	before := delegations[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	partyID := parties[0].ID.String()
	got, pageInfo, err := ds.Get(ctx, &partyID, nil, nil, pagination)
	require.NoError(t, err)

	want := delegations[4:7]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     delegations[4].Cursor().Encode(),
		EndCursor:       delegations[6].Cursor().Encode(),
	}, pageInfo)
}

func testDelegationPaginationPartyNodeFilterNoPagination(t *testing.T) {
	defer DeleteEverything()
	ds, delegations, parties, nodes := setupPaginatedDelegationsTests(t)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	partyID := parties[1].ID.String()
	nodeID := nodes[1].ID.String()
	got, pageInfo, err := ds.Get(ctx, &partyID, &nodeID, nil, pagination)
	require.NoError(t, err)

	want := delegations[10:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     delegations[10].Cursor().Encode(),
		EndCursor:       delegations[19].Cursor().Encode(),
	}, pageInfo)
}

func testDelegationPaginationPartyNodeFilterFirstPage(t *testing.T) {
	defer DeleteEverything()
	ds, delegations, parties, nodes := setupPaginatedDelegationsTests(t)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	partyID := parties[1].ID.String()
	nodeID := nodes[1].ID.String()
	got, pageInfo, err := ds.Get(ctx, &partyID, &nodeID, nil, pagination)
	require.NoError(t, err)

	want := delegations[10:13]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     delegations[10].Cursor().Encode(),
		EndCursor:       delegations[12].Cursor().Encode(),
	}, pageInfo)
}

func testDelegationPaginationPartyNodeFilterFirstAfterPage(t *testing.T) {
	defer DeleteEverything()
	ds, delegations, parties, nodes := setupPaginatedDelegationsTests(t)
	first := int32(3)
	after := delegations[12].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	partyID := parties[1].ID.String()
	nodeID := nodes[1].ID.String()
	got, pageInfo, err := ds.Get(ctx, &partyID, &nodeID, nil, pagination)
	require.NoError(t, err)

	want := delegations[13:16]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     delegations[13].Cursor().Encode(),
		EndCursor:       delegations[15].Cursor().Encode(),
	}, pageInfo)
}

func testDelegationPaginationPartyNodeFilterLastPage(t *testing.T) {
	defer DeleteEverything()
	ds, delegations, parties, nodes := setupPaginatedDelegationsTests(t)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	partyID := parties[1].ID.String()
	nodeID := nodes[1].ID.String()
	got, pageInfo, err := ds.Get(ctx, &partyID, &nodeID, nil, pagination)
	require.NoError(t, err)

	want := delegations[17:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     delegations[17].Cursor().Encode(),
		EndCursor:       delegations[19].Cursor().Encode(),
	}, pageInfo)
}

func testDelegationPaginationPartyNodeFilterLastBeforePage(t *testing.T) {
	defer DeleteEverything()
	ds, delegations, parties, nodes := setupPaginatedDelegationsTests(t)
	last := int32(3)
	before := delegations[17].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	partyID := parties[1].ID.String()
	nodeID := nodes[1].ID.String()
	got, pageInfo, err := ds.Get(ctx, &partyID, &nodeID, nil, pagination)
	require.NoError(t, err)

	want := delegations[14:17]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     delegations[14].Cursor().Encode(),
		EndCursor:       delegations[16].Cursor().Encode(),
	}, pageInfo)
}

func setupPaginatedDelegationsTests(t *testing.T) (*sqlstore.Delegations,
	[]entities.Delegation, []entities.Party, []entities.Node) {
	t.Helper()

	ps := sqlstore.NewParties(connectionSource)
	ns := sqlstore.NewNode(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)
	ds := sqlstore.NewDelegations(connectionSource)

	delegations := make([]entities.Delegation, 0)

	blockTime := time.Date(2022, 7, 15, 8, 0, 0, 0, time.Local)
	block := addTestBlockForTime(t, bs, blockTime)

	nodes := []entities.Node{
		addTestNode(t, ns, block, generateID()),
		addTestNode(t, ns, block, generateID()),
	}

	parties := []entities.Party{
		addTestParty(t, ps, block),
		addTestParty(t, ps, block),
	}

	for i := 0; i < 2; i++ {
		for j := 0; j < 10; j++ {
			blockTime = blockTime.Add(time.Minute)
			block = addTestBlockForTime(t, bs, blockTime)
			delegations = append(delegations, addTestDelegation(t, ds, parties[i], nodes[i], int64((i*10)+j), block))
		}
	}

	return ds, delegations, parties, nodes
}
