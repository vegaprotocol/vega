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
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/sqlstore"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testAssetCount int = 0

func addTestAsset(t *testing.T, as *sqlstore.Assets, block entities.Block, idPrefix ...string) entities.Asset {
	// Make an asset
	testAssetCount += 1
	totalSupply, _ := decimal.NewFromString("1000000000000000000001")
	quantum, _ := decimal.NewFromString("10")
	assetID := generateID()

	if len(idPrefix) > 0 && idPrefix[0] != "" {
		assetID = fmt.Sprintf("%s%02d", idPrefix[0], testAssetCount)
	}

	asset := entities.Asset{
		ID:                entities.NewAssetID(assetID),
		Name:              fmt.Sprint("my test asset", testAssetCount),
		Symbol:            fmt.Sprint("TEST", testAssetCount),
		TotalSupply:       totalSupply,
		Decimals:          5,
		Quantum:           quantum,
		ERC20Contract:     "0xdeadbeef",
		VegaTime:          block.VegaTime,
		LifetimeLimit:     decimal.New(42, 0),
		WithdrawThreshold: decimal.New(81, 0),
		Status:            entities.AssetStatusEnabled,
	}

	// Add it to the database
	err := as.Add(context.Background(), asset)
	require.NoError(t, err)
	return asset
}

func TestAsset(t *testing.T) {
	defer DeleteEverything()

	bs := sqlstore.NewBlocks(connectionSource)
	block := addTestBlock(t, bs)

	as := sqlstore.NewAssets(connectionSource)
	ctx := context.Background()

	// Get all assets, there shouldn't be any yet
	assets, err := as.GetAll(ctx)
	require.NoError(t, err)
	require.Empty(t, assets)

	asset := addTestAsset(t, as, block)

	// Query and check we've got back an asset the same as the one we put in
	fetchedAsset, err := as.GetByID(ctx, asset.ID.String())
	assert.NoError(t, err)
	assert.Equal(t, asset.ID, fetchedAsset.ID)
	assert.Equal(t, asset.Name, fetchedAsset.Name)
	assert.Equal(t, asset.Symbol, fetchedAsset.Symbol)
	assert.Equal(t, asset.TotalSupply, fetchedAsset.TotalSupply)
	assert.Equal(t, asset.Decimals, fetchedAsset.Decimals)
	assert.Equal(t, asset.Quantum, fetchedAsset.Quantum)
	assert.Equal(t, asset.ERC20Contract, fetchedAsset.ERC20Contract)
	assert.Equal(t, asset.VegaTime, fetchedAsset.VegaTime)
	assert.True(t, asset.LifetimeLimit.Equal(fetchedAsset.LifetimeLimit))
	assert.True(t, asset.WithdrawThreshold.Equal(fetchedAsset.WithdrawThreshold))

	// Get all assets and make sure there's one more than there was to begin with
	assets, err = as.GetAll(ctx)
	assert.NoError(t, err)
	assert.Len(t, assets, 1)
}

func setupAssetPaginationTest(t *testing.T) (*sqlstore.Assets, []entities.Asset) {
	bs := sqlstore.NewBlocks(connectionSource)
	block := addTestBlock(t, bs)

	as := sqlstore.NewAssets(connectionSource)

	assets := make([]entities.Asset, 0, 10)

	testAssetCount = 0

	for i := 0; i < 10; i++ {
		asset := addTestAsset(t, as, block, "deadbeef")
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
	defer DeleteEverything()

	as, assets := setupAssetPaginationTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	assert.NoError(t, err)

	got, pageInfo, err := as.GetAllWithCursorPagination(ctx, pagination)
	assert.NoError(t, err)
	assert.Equal(t, assets, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     entities.NewCursor(assets[0].ID.String()).Encode(),
		EndCursor:       entities.NewCursor(assets[9].ID.String()).Encode(),
	}, pageInfo)
}

func testAssetPaginationFirst(t *testing.T) {
	defer DeleteEverything()
	as, assets := setupAssetPaginationTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	assert.NoError(t, err)

	got, pageInfo, err := as.GetAllWithCursorPagination(ctx, pagination)
	assert.NoError(t, err)
	assert.Equal(t, assets[:3], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     entities.NewCursor(assets[0].ID.String()).Encode(),
		EndCursor:       entities.NewCursor(assets[2].ID.String()).Encode(),
	}, pageInfo)
}

func testAssetPaginationLast(t *testing.T) {
	defer DeleteEverything()
	as, assets := setupAssetPaginationTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	assert.NoError(t, err)

	got, pageInfo, err := as.GetAllWithCursorPagination(ctx, pagination)
	assert.NoError(t, err)
	assert.Equal(t, assets[7:], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(assets[7].ID.String()).Encode(),
		EndCursor:       entities.NewCursor(assets[9].ID.String()).Encode(),
	}, pageInfo)
}

func testAssetPaginationFirstAndAfter(t *testing.T) {
	defer DeleteEverything()
	as, assets := setupAssetPaginationTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	first := int32(3)
	after := entities.NewCursor(assets[2].ID.String()).Encode()

	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	assert.NoError(t, err)

	got, pageInfo, err := as.GetAllWithCursorPagination(ctx, pagination)
	assert.NoError(t, err)
	assert.Equal(t, assets[3:6], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(assets[3].ID.String()).Encode(),
		EndCursor:       entities.NewCursor(assets[5].ID.String()).Encode(),
	}, pageInfo)
}

func testAssetPaginationLastAndBefore(t *testing.T) {
	defer DeleteEverything()
	as, assets := setupAssetPaginationTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	last := int32(3)
	before := entities.NewCursor(assets[7].ID.String()).Encode()

	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	assert.NoError(t, err)

	got, pageInfo, err := as.GetAllWithCursorPagination(ctx, pagination)
	assert.NoError(t, err)
	assert.Equal(t, assets[4:7], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(assets[4].ID.String()).Encode(),
		EndCursor:       entities.NewCursor(assets[6].ID.String()).Encode(),
	}, pageInfo)
}

func testAssetsPaginationNoPaginationNewestFirst(t *testing.T) {
	defer DeleteEverything()

	as, assets := setupAssetPaginationTest(t)
	assets = entities.ReverseSlice(assets)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	assert.NoError(t, err)

	got, pageInfo, err := as.GetAllWithCursorPagination(ctx, pagination)
	assert.NoError(t, err)
	assert.Equal(t, assets, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     entities.NewCursor(assets[0].ID.String()).Encode(),
		EndCursor:       entities.NewCursor(assets[9].ID.String()).Encode(),
	}, pageInfo)
}

func testAssetPaginationFirstNewestFirst(t *testing.T) {
	defer DeleteEverything()
	as, assets := setupAssetPaginationTest(t)
	assets = entities.ReverseSlice(assets)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	assert.NoError(t, err)

	got, pageInfo, err := as.GetAllWithCursorPagination(ctx, pagination)
	assert.NoError(t, err)
	assert.Equal(t, assets[:3], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     entities.NewCursor(assets[0].ID.String()).Encode(),
		EndCursor:       entities.NewCursor(assets[2].ID.String()).Encode(),
	}, pageInfo)
}

func testAssetPaginationLastNewestFirst(t *testing.T) {
	defer DeleteEverything()
	as, assets := setupAssetPaginationTest(t)
	assets = entities.ReverseSlice(assets)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	assert.NoError(t, err)

	got, pageInfo, err := as.GetAllWithCursorPagination(ctx, pagination)
	assert.NoError(t, err)
	assert.Equal(t, assets[7:], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(assets[7].ID.String()).Encode(),
		EndCursor:       entities.NewCursor(assets[9].ID.String()).Encode(),
	}, pageInfo)
}

func testAssetPaginationFirstAndAfterNewestFirst(t *testing.T) {
	defer DeleteEverything()
	as, assets := setupAssetPaginationTest(t)
	assets = entities.ReverseSlice(assets)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	first := int32(3)
	after := entities.NewCursor(assets[2].ID.String()).Encode()

	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	assert.NoError(t, err)

	got, pageInfo, err := as.GetAllWithCursorPagination(ctx, pagination)
	assert.NoError(t, err)
	assert.Equal(t, assets[3:6], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(assets[3].ID.String()).Encode(),
		EndCursor:       entities.NewCursor(assets[5].ID.String()).Encode(),
	}, pageInfo)
}

func testAssetPaginationLastAndBeforeNewestFirst(t *testing.T) {
	defer DeleteEverything()
	as, assets := setupAssetPaginationTest(t)
	assets = entities.ReverseSlice(assets)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	last := int32(3)
	before := entities.NewCursor(assets[7].ID.String()).Encode()

	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	assert.NoError(t, err)

	got, pageInfo, err := as.GetAllWithCursorPagination(ctx, pagination)
	assert.NoError(t, err)
	assert.Equal(t, assets[4:7], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     entities.NewCursor(assets[4].ID.String()).Encode(),
		EndCursor:       entities.NewCursor(assets[6].ID.String()).Encode(),
	}, pageInfo)
}
