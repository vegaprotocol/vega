package sqlstore_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyRotationsGetByTx(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keyRotations := setKeyRotationStoreTest(t, ctx)

	got, err := ks.GetByTxHash(ctx, keyRotations[0].TxHash)
	require.NoError(t, err)
	want := []entities.KeyRotation{keyRotations[0]}
	assert.Equal(t, want, got)

	got, err = ks.GetByTxHash(ctx, keyRotations[1].TxHash)
	require.NoError(t, err)
	want = []entities.KeyRotation{keyRotations[1]}
	assert.Equal(t, want, got)
}

func TestKeyRotationsCursorPagination(t *testing.T) {
	t.Run("should return all key rotations if no pagination is specified", testKeyRotationPaginationNoPagination)
	t.Run("should return first page of key rotations if first is provided", testKeyRotationPaginationFirstPage)
	t.Run("should return last page of key rotations if last is provided", testKeyRotationPaginationLastPage)
	t.Run("should return specified page of key rotations if first and after is provided", testKeyRotationPaginationFirstAndAfter)
	t.Run("should return specified page of key rotations if last and before is provided", testKeyRotationPaginationLastAndBefore)

	t.Run("should return all key rotations if no pagination is specified - newest first", testKeyRotationPaginationNoPaginationNewestFirst)
	t.Run("should return first page of key rotations if first is provided - newest first", testKeyRotationPaginationFirstPageNewestFirst)
	t.Run("should return last page of key rotations if last is provided - newest first", testKeyRotationPaginationLastPageNewestFirst)
	t.Run("should return specified page of key rotations if first and after is provided - newest first", testKeyRotationPaginationFirstAndAfterNewestFirst)
	t.Run("should return specified page of key rotations if last and before is provided - newest first", testKeyRotationPaginationLastAndBeforeNewestFirst)

	t.Run("should return all key rotations for specific node if no pagination is specified", testKeyRotationPaginationForNodeNoPagination)
	t.Run("should return first page of key rotations for specific node if first is provided", testKeyRotationPaginationForNodeFirstPage)
	t.Run("should return last page of key rotations for specific node if last is provided", testKeyRotationPaginationForNodeLastPage)
	t.Run("should return specified page of key rotations for specific node if first and after is provided", testKeyRotationPaginationForNodeFirstAndAfter)
	t.Run("should return specified page of key rotations for specific node if last and before is provided", testKeyRotationPaginationForNodeLastAndBefore)

	t.Run("should return all key rotations for specific node if no pagination is specified - newest first", testKeyRotationPaginationForNodeNoPaginationNewestFirst)
	t.Run("should return first page of key rotations for specific node if first is provided - newest first", testKeyRotationPaginationForNodeFirstPageNewestFirst)
	t.Run("should return last page of key rotations for specific node if last is provided - newest first", testKeyRotationPaginationForNodeLastPageNewestFirst)
	t.Run("should return specified page of key rotations for specific node if first and after is provided - newest first", testKeyRotationPaginationForNodeFirstAndAfterNewestFirst)
	t.Run("should return specified page of key rotations for specific node if last and before is provided - newest first", testKeyRotationPaginationForNodeLastAndBeforeNewestFirst)
}

func testKeyRotationPaginationNoPagination(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keys := setKeyRotationStoreTest(t, ctx)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ks.GetAllPubKeyRotations(ctx, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 20)
	want := keys
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[19].Cursor().Encode(),
	}, pageInfo)
}

func testKeyRotationPaginationFirstPage(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keys := setKeyRotationStoreTest(t, ctx)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ks.GetAllPubKeyRotations(ctx, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	want := keys[:3]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testKeyRotationPaginationLastPage(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keys := setKeyRotationStoreTest(t, ctx)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ks.GetAllPubKeyRotations(ctx, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	want := keys[17:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testKeyRotationPaginationFirstAndAfter(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keys := setKeyRotationStoreTest(t, ctx)
	first := int32(3)
	after := keys[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ks.GetAllPubKeyRotations(ctx, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	want := keys[3:6]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testKeyRotationPaginationLastAndBefore(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keys := setKeyRotationStoreTest(t, ctx)
	last := int32(3)
	before := keys[17].Cursor().Encode()

	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)

	got, pageInfo, err := ks.GetAllPubKeyRotations(ctx, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	want := keys[14:17]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testKeyRotationPaginationNoPaginationNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keys := setKeyRotationStoreTest(t, ctx)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := ks.GetAllPubKeyRotations(ctx, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 20)
	want := entities.ReverseSlice(keys)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[19].Cursor().Encode(),
	}, pageInfo)
}

func testKeyRotationPaginationFirstPageNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keys := setKeyRotationStoreTest(t, ctx)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := ks.GetAllPubKeyRotations(ctx, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	want := entities.ReverseSlice(keys)[:3]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testKeyRotationPaginationLastPageNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keys := setKeyRotationStoreTest(t, ctx)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := ks.GetAllPubKeyRotations(ctx, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	want := entities.ReverseSlice(keys)[17:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testKeyRotationPaginationFirstAndAfterNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keys := setKeyRotationStoreTest(t, ctx)
	first := int32(3)
	after := keys[17].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := ks.GetAllPubKeyRotations(ctx, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	want := entities.ReverseSlice(keys)[3:6]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testKeyRotationPaginationLastAndBeforeNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keys := setKeyRotationStoreTest(t, ctx)
	last := int32(3)
	before := keys[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)

	got, pageInfo, err := ks.GetAllPubKeyRotations(ctx, pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	want := entities.ReverseSlice(keys)[14:17]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testKeyRotationPaginationForNodeNoPagination(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keys := setKeyRotationStoreTest(t, ctx)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ks.GetPubKeyRotationsPerNode(ctx, "deadbeef01", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 10)
	want := keys[:10]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[9].Cursor().Encode(),
	}, pageInfo)
}

func testKeyRotationPaginationForNodeFirstPage(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keys := setKeyRotationStoreTest(t, ctx)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ks.GetPubKeyRotationsPerNode(ctx, "deadbeef01", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	want := keys[:3]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testKeyRotationPaginationForNodeLastPage(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keys := setKeyRotationStoreTest(t, ctx)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ks.GetPubKeyRotationsPerNode(ctx, "deadbeef01", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	want := keys[7:10]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testKeyRotationPaginationForNodeFirstAndAfter(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keys := setKeyRotationStoreTest(t, ctx)
	first := int32(3)
	after := keys[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ks.GetPubKeyRotationsPerNode(ctx, "deadbeef01", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	want := keys[3:6]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testKeyRotationPaginationForNodeLastAndBefore(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keys := setKeyRotationStoreTest(t, ctx)
	last := int32(3)
	before := keys[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)

	got, pageInfo, err := ks.GetPubKeyRotationsPerNode(ctx, "deadbeef01", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	want := keys[4:7]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testKeyRotationPaginationForNodeNoPaginationNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keys := setKeyRotationStoreTest(t, ctx)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := ks.GetPubKeyRotationsPerNode(ctx, "deadbeef01", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 10)
	want := entities.ReverseSlice(keys[0:10])
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[9].Cursor().Encode(),
	}, pageInfo)
}

func testKeyRotationPaginationForNodeFirstPageNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keys := setKeyRotationStoreTest(t, ctx)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := ks.GetPubKeyRotationsPerNode(ctx, "deadbeef01", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	want := entities.ReverseSlice(keys[0:10])[:3]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testKeyRotationPaginationForNodeLastPageNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keys := setKeyRotationStoreTest(t, ctx)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := ks.GetPubKeyRotationsPerNode(ctx, "deadbeef01", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	want := entities.ReverseSlice(keys[0:10])[7:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testKeyRotationPaginationForNodeFirstAndAfterNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keys := setKeyRotationStoreTest(t, ctx)
	first := int32(3)
	after := keys[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := ks.GetPubKeyRotationsPerNode(ctx, "deadbeef01", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	want := entities.ReverseSlice(keys[0:10])[3:6]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testKeyRotationPaginationForNodeLastAndBeforeNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	ks, keys := setKeyRotationStoreTest(t, ctx)
	last := int32(3)
	before := keys[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)

	got, pageInfo, err := ks.GetPubKeyRotationsPerNode(ctx, "deadbeef01", pagination)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	want := entities.ReverseSlice(keys[0:10])[4:7]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func setKeyRotationStoreTest(t *testing.T, ctx context.Context) (*sqlstore.KeyRotations, []entities.KeyRotation) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	ns := sqlstore.NewNode(connectionSource)
	ks := sqlstore.NewKeyRotations(connectionSource)

	keyRotations := make([]entities.KeyRotation, 20)
	blockTime := time.Date(2022, 8, 2, 9, 0, 0, 0, time.Local)
	block := addTestBlockForTime(t, ctx, bs, blockTime)

	addTestNode(t, ctx, ns, block, "deadbeef01")
	addTestNode(t, ctx, ns, block, "deadbeef02")

	for i := 0; i < 2; i++ {
		for j := 0; j < 10; j++ {
			blockTime = blockTime.Add(time.Minute)
			block := addTestBlockForTime(t, ctx, bs, blockTime)

			kr := entities.KeyRotation{
				NodeID:      entities.NodeID(fmt.Sprintf("deadbeef%02d", i+1)),
				OldPubKey:   entities.VegaPublicKey(fmt.Sprintf("cafed00d%02d", j+1)),
				NewPubKey:   entities.VegaPublicKey(fmt.Sprintf("cafed00d%02d", j+2)),
				BlockHeight: uint64((i * 10) + j + 1),
				VegaTime:    block.VegaTime,
				TxHash:      generateTxHash(),
			}
			if err := ks.Upsert(ctx, &kr); err != nil {
				t.Fatalf("creating key rotation test data: %v", err)
			}

			keyRotations[(i*10 + j)] = kr
		}
	}

	return ks, keyRotations
}
