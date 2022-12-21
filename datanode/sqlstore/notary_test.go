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
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	v1 "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotary(t *testing.T) {
	t.Run("Adding a single signature", testAddSignatures)
	t.Run("Adding multiple signatures for multiple resources", testAddMultipleSignatures)
	t.Run("Getting a non-existing resource signatures", testNoResource)
}

func setupNotaryStoreTests(t *testing.T) (*sqlstore.Notary, *sqlstore.Blocks, sqlstore.Connection) {
	t.Helper()
	ns := sqlstore.NewNotary(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)
	return ns, bs, connectionSource.Connection
}

func testAddSignatures(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ws, bs, conn := setupNotaryStoreTests(t)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from withdrawals`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	ns := getTestNodeSignature(t, ctx, bs, "deadbeef", "iamsig")
	err = ws.Add(ctx, ns)
	require.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from node_signatures`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)
}

func testAddMultipleSignatures(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ws, bs, _ := setupNotaryStoreTests(t)

	nodeSig1 := getTestNodeSignature(t, ctx, bs, "deadbeef", "iamsig")
	nodeSig2 := getTestNodeSignature(t, ctx, bs, "deadbeef", "iamsig")         // this will have a different sig
	nodeSig3 := getTestNodeSignature(t, ctx, bs, "deadbeef", "iamsig")         // this will be a dupe of ns2
	nodeSig4 := getTestNodeSignature(t, ctx, bs, "deadbeefdeadbeef", "iamsig") // this will have a different sig and id

	nodeSig2.Sig = []byte("iamdifferentsig")
	nodeSig4.Sig = []byte("iamdifferentsigagain")

	err := ws.Add(ctx, nodeSig1)
	require.NoError(t, err)

	err = ws.Add(ctx, nodeSig2)
	require.NoError(t, err)

	err = ws.Add(ctx, nodeSig3)
	require.NoError(t, err)

	err = ws.Add(ctx, nodeSig4)
	require.NoError(t, err)

	res, _, err := ws.GetByResourceID(ctx, "deadbeef", entities.CursorPagination{})
	require.NoError(t, err)
	require.Len(t, res, 2)

	res, _, err = ws.GetByResourceID(ctx, "deadbeefdeadbeef", entities.CursorPagination{})
	require.NoError(t, err)
	require.Len(t, res, 1)
}

func getTestNodeSignature(t *testing.T, ctx context.Context, bs *sqlstore.Blocks, id string, sig string) *entities.NodeSignature {
	t.Helper()
	block := addTestBlock(t, ctx, bs)
	ns, err := entities.NodeSignatureFromProto(
		&v1.NodeSignature{
			Id:   id,
			Sig:  []byte(sig),
			Kind: v1.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_WITHDRAWAL,
		},
		generateTxHash(),
		block.VegaTime,
	)
	require.NoError(t, err)
	return ns
}

func testNoResource(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ws, _, _ := setupNotaryStoreTests(t)

	res, _, err := ws.GetByResourceID(ctx, "deadbeefdeadbeef", entities.CursorPagination{})
	require.NoError(t, err)
	require.Len(t, res, 0)
}

func TestNodeSignaturePagination(t *testing.T) {
	t.Run("should return all node signatures if no pagination is specified", testNodeSignaturePaginationNoPagination)
	t.Run("should return first page of node signatures if first pagination is specified", testNodeSignaturePaginationFirst)
	t.Run("should return last page of node signatures if last pagination is specified", testNodeSignaturePaginationLast)
	t.Run("should return specified page of node signatures if first and after pagination is specified", testNodeSignaturePaginationFirstAfter)
	t.Run("should return specified page of node signatures if last and before pagination is specified", testNodeSignaturePaginationLastBefore)

	t.Run("should return all node signatures if no pagination is specified - newest first", testNodeSignaturePaginationNoPaginationNewestFirst)
	t.Run("should return first page of node signatures if first pagination is specified - newest first", testNodeSignaturePaginationFirstNewestFirst)
	t.Run("should return last page of node signatures if last pagination is specified - newest first", testNodeSignaturePaginationLastNewestFirst)
	t.Run("should return specified page of node signatures if first and after pagination is specified - newest first", testNodeSignaturePaginationFirstAfterNewestFirst)
	t.Run("should return specified page of node signatures if last and before pagination is specified - newest first", testNodeSignaturePaginationLastBeforeNewestFirst)
}

func testNodeSignaturePaginationNoPagination(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ns, sigs := setupNodeSignaturePaginationTest(t, ctx)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ns.GetByResourceID(ctx, "deadbeef", pagination)
	require.NoError(t, err)
	want := sigs
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[9].Cursor().Encode(),
	}, pageInfo)
}

func testNodeSignaturePaginationFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ns, sigs := setupNodeSignaturePaginationTest(t, ctx)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ns.GetByResourceID(ctx, "deadbeef", pagination)
	require.NoError(t, err)
	want := sigs[:3]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testNodeSignaturePaginationLast(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ns, sigs := setupNodeSignaturePaginationTest(t, ctx)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ns.GetByResourceID(ctx, "deadbeef", pagination)
	require.NoError(t, err)
	want := sigs[7:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testNodeSignaturePaginationFirstAfter(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ns, sigs := setupNodeSignaturePaginationTest(t, ctx)
	first := int32(3)
	after := sigs[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ns.GetByResourceID(ctx, "deadbeef", pagination)
	require.NoError(t, err)
	want := sigs[3:6]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testNodeSignaturePaginationLastBefore(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ns, sigs := setupNodeSignaturePaginationTest(t, ctx)
	last := int32(3)
	before := sigs[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)

	got, pageInfo, err := ns.GetByResourceID(ctx, "deadbeef", pagination)
	require.NoError(t, err)
	want := sigs[4:7]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testNodeSignaturePaginationNoPaginationNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ns, sigs := setupNodeSignaturePaginationTest(t, ctx)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := ns.GetByResourceID(ctx, "deadbeef", pagination)
	require.NoError(t, err)
	want := entities.ReverseSlice(sigs)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[9].Cursor().Encode(),
	}, pageInfo)
}

func testNodeSignaturePaginationFirstNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ns, sigs := setupNodeSignaturePaginationTest(t, ctx)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := ns.GetByResourceID(ctx, "deadbeef", pagination)
	require.NoError(t, err)
	want := entities.ReverseSlice(sigs)[:3]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testNodeSignaturePaginationLastNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ns, sigs := setupNodeSignaturePaginationTest(t, ctx)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := ns.GetByResourceID(ctx, "deadbeef", pagination)
	require.NoError(t, err)
	want := entities.ReverseSlice(sigs)[7:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testNodeSignaturePaginationFirstAfterNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ns, sigs := setupNodeSignaturePaginationTest(t, ctx)
	first := int32(3)
	after := sigs[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := ns.GetByResourceID(ctx, "deadbeef", pagination)
	require.NoError(t, err)
	want := entities.ReverseSlice(sigs)[3:6]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testNodeSignaturePaginationLastBeforeNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ns, sigs := setupNodeSignaturePaginationTest(t, ctx)
	last := int32(3)
	before := sigs[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)

	got, pageInfo, err := ns.GetByResourceID(ctx, "deadbeef", pagination)
	require.NoError(t, err)
	want := entities.ReverseSlice(sigs)[4:7]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func setupNodeSignaturePaginationTest(t *testing.T, ctx context.Context) (*sqlstore.Notary, []entities.NodeSignature) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	ns := sqlstore.NewNotary(connectionSource)
	signatures := make([]entities.NodeSignature, 10)

	for i := 0; i < 10; i++ {
		signature := getTestNodeSignature(t, ctx, bs, "deadbeef", fmt.Sprintf("sig%02d", i+1))
		signatures[i] = *signature
		err := ns.Add(ctx, signature)
		require.NoError(t, err)
	}

	return ns, signatures
}
