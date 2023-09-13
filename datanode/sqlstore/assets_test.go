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
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testAssetCount int

func addTestAsset(t *testing.T, ctx context.Context, as *sqlstore.Assets, block entities.Block, idPrefix ...string) entities.Asset {
	t.Helper()
	// Make an asset
	testAssetCount++
	quantum, _ := decimal.NewFromString("10")
	assetID := helpers.GenerateID()

	if len(idPrefix) > 0 && idPrefix[0] != "" {
		assetID = fmt.Sprintf("%s%02d", idPrefix[0], testAssetCount)
	}

	asset := entities.Asset{
		ID:                entities.AssetID(assetID),
		Name:              fmt.Sprint("my test asset", testAssetCount),
		Symbol:            fmt.Sprint("TEST", testAssetCount),
		Decimals:          5,
		Quantum:           quantum,
		ERC20Contract:     "0xdeadbeef",
		VegaTime:          block.VegaTime,
		LifetimeLimit:     decimal.New(42, 0),
		WithdrawThreshold: decimal.New(81, 0),
		Status:            entities.AssetStatusEnabled,
		TxHash:            generateTxHash(),
	}

	// Add it to the database
	err := as.Add(ctx, asset)
	require.NoError(t, err)
	return asset
}

func assetsEqual(t *testing.T, expected, actual entities.Asset) {
	t.Helper()

	assert.Equal(t, expected.ID, actual.ID)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Symbol, actual.Symbol)
	assert.Equal(t, expected.Decimals, actual.Decimals)
	assert.Equal(t, expected.Quantum, actual.Quantum)
	assert.Equal(t, expected.ERC20Contract, actual.ERC20Contract)
	assert.Equal(t, expected.VegaTime, actual.VegaTime)
	assert.True(t, expected.LifetimeLimit.Equal(actual.LifetimeLimit))
	assert.True(t, expected.WithdrawThreshold.Equal(actual.WithdrawThreshold))
}

// TestAssetCache tests for a bug which was discovered whereby fetching an asset by ID after
// it had been updated but before the transaction was committed led to a poisoned cache that
// returned stale values.
func TestAssetCache(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)
	as := sqlstore.NewAssets(connectionSource)
	block := addTestBlock(t, ctx, bs)

	// A make a lovely asset
	asset := addTestAsset(t, ctx, as, block, "")

	// Try updating the asset to have a new symbol in the top level transaction
	asset2 := asset
	asset2.Symbol = "TEST2"
	err := as.Add(ctx, asset2)
	require.NoError(t, err)

	// Should get new asset symbol immediately
	fetched, err := as.GetByID(ctx, string(asset.ID))
	require.NoError(t, err)
	require.Equal(t, asset2, fetched)

	// Now in a sub-transaction, update the asset to have another different symbol
	txCtx, err := connectionSource.WithTransaction(ctx)
	require.NoError(t, err)
	asset3 := asset
	asset3.Symbol = "TEST3"
	err = as.Add(txCtx, asset3)
	require.NoError(t, err)

	// Transaction hasn't committed yet, we should still get the old symbol when fetching that asset
	fetched, err = as.GetByID(ctx, string(asset.ID))
	require.NoError(t, err)
	assert.Equal(t, asset2, fetched)

	// Commit the transaction and fetch the asset, we should get the asset with the new symbol
	err = connectionSource.Commit(txCtx)
	require.NoError(t, err)
	fetched, err = as.GetByID(ctx, string(asset.ID))
	require.NoError(t, err)
	assert.Equal(t, asset3, fetched)
}

func TestAsset(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)
	block := addTestBlock(t, ctx, bs)

	as := sqlstore.NewAssets(connectionSource)

	// Get all assets, there shouldn't be any yet
	assets, err := as.GetAll(ctx)
	require.NoError(t, err)
	require.Empty(t, assets)

	asset := addTestAsset(t, ctx, as, block)
	asset2 := addTestAsset(t, ctx, as, block)

	// Query and check we've got back an asset the same as the one we put in
	fetchedAsset, err := as.GetByID(ctx, asset.ID.String())
	assert.NoError(t, err)
	assetsEqual(t, asset, fetchedAsset)

	// Get all assets and make sure there's one more than there was to begin with
	assets, err = as.GetAll(ctx)
	assert.NoError(t, err)
	assert.Len(t, assets, 2)

	fetchedAssets, err := as.GetByTxHash(ctx, asset.TxHash)
	assert.NoError(t, err)
	assetsEqual(t, asset, fetchedAssets[0])

	fetchedAssets, err = as.GetByTxHash(ctx, asset2.TxHash)
	assert.NoError(t, err)
	assetsEqual(t, asset2, fetchedAssets[0])
}

func setupAssetPaginationTest(t *testing.T, ctx context.Context) (*sqlstore.Assets, []entities.Asset) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	block := addTestBlock(t, ctx, bs)

	as := sqlstore.NewAssets(connectionSource)

	assets := make([]entities.Asset, 0, 10)

	testAssetCount = 0

	for i := 0; i < 10; i++ {
		asset := addTestAsset(t, ctx, as, block, "deadbeef")
		assets = append(assets, asset)
	}

	return as, assets
}

func TestAssets_GetAllWithCursorPagination(t *testing.T) {
	t.Run("should return all deposits if no pagination is specified", testAssetsPaginationNoPagination)
	t.Run("should return the first page of results if first is provided", testAssetPaginationFirst)
	t.Run("should return the last page of results if last is provided", testAssetPaginationLast)
	t.Run("should return the specified page of results if first and after is provided", testAssetPaginationFirstAndAfter)
	t.Run("should return the specified page of results if last and before is provided", testAssetPaginationLastAndBefore)

	t.Run("should return all deposits if no pagination is specified - newest first", testAssetsPaginationNoPaginationNewestFirst)
	t.Run("should return the first page of results if first is provided - newest first", testAssetPaginationFirstNewestFirst)
	t.Run("should return the last page of results if last is provided - newest first", testAssetPaginationLastNewestFirst)
	t.Run("should return the specified page of results if first and after is provided - newest first", testAssetPaginationFirstAndAfterNewestFirst)
	t.Run("should return the specified page of results if last and before is provided - newest first", testAssetPaginationLastAndBeforeNewestFirst)
}

func testAssetsPaginationNoPagination(t *testing.T) {
	ctx := tempTransaction(t)

	as, assets := setupAssetPaginationTest(t, ctx)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	assert.NoError(t, err)

	got, pageInfo, err := as.GetAllWithCursorPagination(ctx, pagination)
	assert.NoError(t, err)
	assert.Equal(t, assets, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     assets[0].Cursor().Encode(),
		EndCursor:       assets[9].Cursor().Encode(),
	}, pageInfo)
}

func testAssetPaginationFirst(t *testing.T) {
	ctx := tempTransaction(t)

	as, assets := setupAssetPaginationTest(t, ctx)

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	assert.NoError(t, err)

	got, pageInfo, err := as.GetAllWithCursorPagination(ctx, pagination)
	assert.NoError(t, err)
	assert.Equal(t, assets[:3], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     assets[0].Cursor().Encode(),
		EndCursor:       assets[2].Cursor().Encode(),
	}, pageInfo)
}

func testAssetPaginationLast(t *testing.T) {
	ctx := tempTransaction(t)

	as, assets := setupAssetPaginationTest(t, ctx)

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	assert.NoError(t, err)

	got, pageInfo, err := as.GetAllWithCursorPagination(ctx, pagination)
	assert.NoError(t, err)
	assert.Equal(t, assets[7:], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     assets[7].Cursor().Encode(),
		EndCursor:       assets[9].Cursor().Encode(),
	}, pageInfo)
}

func testAssetPaginationFirstAndAfter(t *testing.T) {
	ctx := tempTransaction(t)

	as, assets := setupAssetPaginationTest(t, ctx)

	first := int32(3)
	after := assets[2].Cursor().Encode()

	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	assert.NoError(t, err)

	got, pageInfo, err := as.GetAllWithCursorPagination(ctx, pagination)
	assert.NoError(t, err)
	assert.Equal(t, assets[3:6], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     assets[3].Cursor().Encode(),
		EndCursor:       assets[5].Cursor().Encode(),
	}, pageInfo)
}

func testAssetPaginationLastAndBefore(t *testing.T) {
	ctx := tempTransaction(t)

	as, assets := setupAssetPaginationTest(t, ctx)

	last := int32(3)
	before := assets[7].Cursor().Encode()

	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	assert.NoError(t, err)

	got, pageInfo, err := as.GetAllWithCursorPagination(ctx, pagination)
	assert.NoError(t, err)
	assert.Equal(t, assets[4:7], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     assets[4].Cursor().Encode(),
		EndCursor:       assets[6].Cursor().Encode(),
	}, pageInfo)
}

func testAssetsPaginationNoPaginationNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	as, assets := setupAssetPaginationTest(t, ctx)
	assets = entities.ReverseSlice(assets)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	assert.NoError(t, err)

	got, pageInfo, err := as.GetAllWithCursorPagination(ctx, pagination)
	assert.NoError(t, err)
	assert.Equal(t, assets, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     assets[0].Cursor().Encode(),
		EndCursor:       assets[9].Cursor().Encode(),
	}, pageInfo)
}

func testAssetPaginationFirstNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	as, assets := setupAssetPaginationTest(t, ctx)
	assets = entities.ReverseSlice(assets)

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	assert.NoError(t, err)

	got, pageInfo, err := as.GetAllWithCursorPagination(ctx, pagination)
	assert.NoError(t, err)
	assert.Equal(t, assets[:3], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     assets[0].Cursor().Encode(),
		EndCursor:       assets[2].Cursor().Encode(),
	}, pageInfo)
}

func testAssetPaginationLastNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	as, assets := setupAssetPaginationTest(t, ctx)
	assets = entities.ReverseSlice(assets)

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	assert.NoError(t, err)

	got, pageInfo, err := as.GetAllWithCursorPagination(ctx, pagination)
	assert.NoError(t, err)
	assert.Equal(t, assets[7:], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     assets[7].Cursor().Encode(),
		EndCursor:       assets[9].Cursor().Encode(),
	}, pageInfo)
}

func testAssetPaginationFirstAndAfterNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	as, assets := setupAssetPaginationTest(t, ctx)
	assets = entities.ReverseSlice(assets)

	first := int32(3)
	after := assets[2].Cursor().Encode()

	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	assert.NoError(t, err)

	got, pageInfo, err := as.GetAllWithCursorPagination(ctx, pagination)
	assert.NoError(t, err)
	assert.Equal(t, assets[3:6], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     assets[3].Cursor().Encode(),
		EndCursor:       assets[5].Cursor().Encode(),
	}, pageInfo)
}

func testAssetPaginationLastAndBeforeNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	as, assets := setupAssetPaginationTest(t, ctx)
	assets = entities.ReverseSlice(assets)

	last := int32(3)
	before := assets[7].Cursor().Encode()

	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	assert.NoError(t, err)

	got, pageInfo, err := as.GetAllWithCursorPagination(ctx, pagination)
	assert.NoError(t, err)
	assert.Equal(t, assets[4:7], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     assets[4].Cursor().Encode(),
		EndCursor:       assets[6].Cursor().Encode(),
	}, pageInfo)
}
