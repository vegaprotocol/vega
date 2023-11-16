// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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

func addCheckpoint(t *testing.T, ctx context.Context, ns *sqlstore.Checkpoints, hash, blockHash string, blockHeight int64, block entities.Block,
	seqNum uint64,
) entities.Checkpoint {
	t.Helper()
	c := entities.Checkpoint{
		Hash:        hash,
		BlockHash:   blockHash,
		BlockHeight: blockHeight,
		VegaTime:    block.VegaTime,
		SeqNum:      seqNum,
	}
	ns.Add(ctx, c)
	return c
}

func TestCheckpoints(t *testing.T) {
	ctx := tempTransaction(t)

	checkpointStore := sqlstore.NewCheckpoints(connectionSource)
	blockStore := sqlstore.NewBlocks(connectionSource)
	block1 := addTestBlock(t, ctx, blockStore)
	block2 := addTestBlock(t, ctx, blockStore)

	checkpoint1 := addCheckpoint(t, ctx, checkpointStore, "myHash", "myBlockHash", 1, block1, 0)
	checkpoint2 := addCheckpoint(t, ctx, checkpointStore, "myOtherHash", "myOtherBlockHash", 2, block2, 0)

	t.Run("GetAll", func(t *testing.T) {
		expected := []entities.Checkpoint{checkpoint2, checkpoint1}
		pagination := entities.CursorPagination{NewestFirst: true}
		actual, _, err := checkpointStore.GetAll(ctx, pagination)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}

func TestCheckpointsSameHashAndBlock(t *testing.T) {
	ctx := tempTransaction(t)

	checkpointStore := sqlstore.NewCheckpoints(connectionSource)
	blockStore := sqlstore.NewBlocks(connectionSource)
	block1 := addTestBlock(t, ctx, blockStore)

	checkpoint1 := addCheckpoint(t, ctx, checkpointStore, "myHash", "myBlockHash", 1, block1, 0)
	checkpoint2 := addCheckpoint(t, ctx, checkpointStore, "myHash", "myBlockHash", 1, block1, 1)

	t.Run("GetAll", func(t *testing.T) {
		expected := []entities.Checkpoint{checkpoint1, checkpoint2}
		pagination := entities.CursorPagination{NewestFirst: true}
		actual, _, err := checkpointStore.GetAll(ctx, pagination)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}

func TestCheckpointPagination(t *testing.T) {
	t.Run("should return all checkpoints if no pagination is specified", testCheckpointPaginationNoPagination)
	t.Run("should return first page of checkpoints if first is provided", testCheckpointPaginationFirst)
	t.Run("should return last page of checkpoints if last is provided", testCheckpointPaginationLast)
	t.Run("should return specified page of checkpoints if first and after is specified", testCheckpointPaginationFirstAndAfter)
	t.Run("should return specified page of checkpoints if last and before is specified", testCheckpointPaginationLastAndBefore)

	t.Run("should return all checkpoints if no pagination is specified - newest first", testCheckpointPaginationNoPaginationNewestFirst)
	t.Run("should return first page of checkpoints if first is provided - newest first", testCheckpointPaginationFirstNewestFirst)
	t.Run("should return last page of checkpoints if last is provided - newest first", testCheckpointPaginationLastNewestFirst)
	t.Run("should return specified page of checkpoints if first and after is specified - newest first", testCheckpointPaginationFirstAndAfterNewestFirst)
	t.Run("should return specified page of checkpoints if last and before is specified - newest first", testCheckpointPaginationLastAndBeforeNewestFirst)
}

func testCheckpointPaginationNoPagination(t *testing.T) {
	ctx := tempTransaction(t)

	cs, checkpoints := setupCheckpointPaginationTest(t, ctx)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := cs.GetAll(ctx, pagination)
	require.NoError(t, err)
	want := checkpoints
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     checkpoints[0].Cursor().Encode(),
		EndCursor:       checkpoints[9].Cursor().Encode(),
	}, pageInfo)
}

func testCheckpointPaginationFirst(t *testing.T) {
	ctx := tempTransaction(t)

	cs, checkpoints := setupCheckpointPaginationTest(t, ctx)

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := cs.GetAll(ctx, pagination)
	require.NoError(t, err)
	want := checkpoints[:3]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     checkpoints[0].Cursor().Encode(),
		EndCursor:       checkpoints[2].Cursor().Encode(),
	}, pageInfo)
}

func testCheckpointPaginationLast(t *testing.T) {
	ctx := tempTransaction(t)

	cs, checkpoints := setupCheckpointPaginationTest(t, ctx)

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := cs.GetAll(ctx, pagination)
	require.NoError(t, err)
	want := checkpoints[7:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     checkpoints[7].Cursor().Encode(),
		EndCursor:       checkpoints[9].Cursor().Encode(),
	}, pageInfo)
}

func testCheckpointPaginationFirstAndAfter(t *testing.T) {
	ctx := tempTransaction(t)

	cs, checkpoints := setupCheckpointPaginationTest(t, ctx)

	first := int32(3)
	after := checkpoints[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := cs.GetAll(ctx, pagination)
	require.NoError(t, err)
	want := checkpoints[3:6]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     checkpoints[3].Cursor().Encode(),
		EndCursor:       checkpoints[5].Cursor().Encode(),
	}, pageInfo)
}

func testCheckpointPaginationLastAndBefore(t *testing.T) {
	ctx := tempTransaction(t)

	cs, checkpoints := setupCheckpointPaginationTest(t, ctx)

	last := int32(3)
	before := checkpoints[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	got, pageInfo, err := cs.GetAll(ctx, pagination)
	require.NoError(t, err)
	want := checkpoints[4:7]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     checkpoints[4].Cursor().Encode(),
		EndCursor:       checkpoints[6].Cursor().Encode(),
	}, pageInfo)
}

func testCheckpointPaginationNoPaginationNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	cs, checkpoints := setupCheckpointPaginationTest(t, ctx)
	checkpoints = entities.ReverseSlice(checkpoints)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := cs.GetAll(ctx, pagination)
	require.NoError(t, err)
	want := checkpoints
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     checkpoints[0].Cursor().Encode(),
		EndCursor:       checkpoints[9].Cursor().Encode(),
	}, pageInfo)
}

func testCheckpointPaginationFirstNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	cs, checkpoints := setupCheckpointPaginationTest(t, ctx)
	checkpoints = entities.ReverseSlice(checkpoints)

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := cs.GetAll(ctx, pagination)
	require.NoError(t, err)
	want := checkpoints[:3]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     checkpoints[0].Cursor().Encode(),
		EndCursor:       checkpoints[2].Cursor().Encode(),
	}, pageInfo)
}

func testCheckpointPaginationLastNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	cs, checkpoints := setupCheckpointPaginationTest(t, ctx)
	checkpoints = entities.ReverseSlice(checkpoints)

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := cs.GetAll(ctx, pagination)
	require.NoError(t, err)
	want := checkpoints[7:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     checkpoints[7].Cursor().Encode(),
		EndCursor:       checkpoints[9].Cursor().Encode(),
	}, pageInfo)
}

func testCheckpointPaginationFirstAndAfterNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	cs, checkpoints := setupCheckpointPaginationTest(t, ctx)
	checkpoints = entities.ReverseSlice(checkpoints)

	first := int32(3)
	after := checkpoints[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := cs.GetAll(ctx, pagination)
	require.NoError(t, err)
	want := checkpoints[3:6]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     checkpoints[3].Cursor().Encode(),
		EndCursor:       checkpoints[5].Cursor().Encode(),
	}, pageInfo)
}

func testCheckpointPaginationLastAndBeforeNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	cs, checkpoints := setupCheckpointPaginationTest(t, ctx)
	checkpoints = entities.ReverseSlice(checkpoints)

	last := int32(3)
	before := checkpoints[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)
	got, pageInfo, err := cs.GetAll(ctx, pagination)
	require.NoError(t, err)
	want := checkpoints[4:7]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     checkpoints[4].Cursor().Encode(),
		EndCursor:       checkpoints[6].Cursor().Encode(),
	}, pageInfo)
}

func setupCheckpointPaginationTest(t *testing.T, ctx context.Context) (*sqlstore.Checkpoints, []entities.Checkpoint) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	cs := sqlstore.NewCheckpoints(connectionSource)
	blockTime := time.Date(2022, 7, 27, 8, 0, 0, 0, time.Local)
	checkPoints := make([]entities.Checkpoint, 10)

	for i := 0; i < 10; i++ {
		blockTime = blockTime.Add(time.Minute)
		block := addTestBlockForTime(t, ctx, bs, blockTime)
		hash := int64(i + 1)
		checkPoints[i] = addCheckpoint(t, ctx, cs, fmt.Sprintf("TestHash%02d", hash), fmt.Sprintf("TestBlockHash%02d", hash), hash, block, uint64(i))
	}

	return cs, checkPoints
}
