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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addCheckpoint(t *testing.T, ns *sqlstore.Checkpoints, hash, blockHash string, blockHeight int64, block entities.Block) entities.Checkpoint {
	c := entities.Checkpoint{
		Hash:        hash,
		BlockHash:   blockHash,
		BlockHeight: blockHeight,
		VegaTime:    block.VegaTime,
	}
	ns.Add(context.Background(), c)
	return c
}

func TestCheckpoints(t *testing.T) {
	defer DeleteEverything()
	ctx := context.Background()
	checkpointStore := sqlstore.NewCheckpoints(connectionSource)
	blockStore := sqlstore.NewBlocks(connectionSource)
	block1 := addTestBlock(t, blockStore)
	block2 := addTestBlock(t, blockStore)

	checkpoint1 := addCheckpoint(t, checkpointStore, "myHash", "myBlockHash", 1, block1)
	checkpoint2 := addCheckpoint(t, checkpointStore, "myOtherHash", "myOtherBlockHash", 2, block2)

	t.Run("GetAll", func(t *testing.T) {
		expected := []entities.Checkpoint{checkpoint2, checkpoint1}
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
	defer DeleteEverything()
	cs, checkpoints := setupCheckpointPaginationTest(t)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := cs.GetAll(timeoutCtx, pagination)
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
	defer DeleteEverything()
	cs, checkpoints := setupCheckpointPaginationTest(t)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := cs.GetAll(timeoutCtx, pagination)
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
	defer DeleteEverything()
	cs, checkpoints := setupCheckpointPaginationTest(t)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := cs.GetAll(timeoutCtx, pagination)
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
	defer DeleteEverything()
	cs, checkpoints := setupCheckpointPaginationTest(t)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	first := int32(3)
	after := checkpoints[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := cs.GetAll(timeoutCtx, pagination)
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
	defer DeleteEverything()
	cs, checkpoints := setupCheckpointPaginationTest(t)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	last := int32(3)
	before := checkpoints[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	got, pageInfo, err := cs.GetAll(timeoutCtx, pagination)
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
	defer DeleteEverything()
	cs, checkpoints := setupCheckpointPaginationTest(t)
	checkpoints = entities.ReverseSlice(checkpoints)
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := cs.GetAll(timeoutCtx, pagination)
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
	defer DeleteEverything()
	cs, checkpoints := setupCheckpointPaginationTest(t)
	checkpoints = entities.ReverseSlice(checkpoints)
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := cs.GetAll(timeoutCtx, pagination)
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
	defer DeleteEverything()
	cs, checkpoints := setupCheckpointPaginationTest(t)
	checkpoints = entities.ReverseSlice(checkpoints)
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := cs.GetAll(timeoutCtx, pagination)
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
	defer DeleteEverything()
	cs, checkpoints := setupCheckpointPaginationTest(t)
	checkpoints = entities.ReverseSlice(checkpoints)
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	first := int32(3)
	after := checkpoints[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := cs.GetAll(timeoutCtx, pagination)
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
	defer DeleteEverything()
	cs, checkpoints := setupCheckpointPaginationTest(t)
	checkpoints = entities.ReverseSlice(checkpoints)
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	last := int32(3)
	before := checkpoints[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)
	got, pageInfo, err := cs.GetAll(timeoutCtx, pagination)
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

func setupCheckpointPaginationTest(t *testing.T) (*sqlstore.Checkpoints, []entities.Checkpoint) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	cs := sqlstore.NewCheckpoints(connectionSource)

	blockTime := time.Date(2022, 7, 27, 8, 0, 0, 0, time.Local)
	checkPoints := make([]entities.Checkpoint, 10)

	for i := 0; i < 10; i++ {
		blockTime = blockTime.Add(time.Minute)
		block := addTestBlockForTime(t, bs, blockTime)
		hash := int64(i + 1)
		checkPoints[i] = addCheckpoint(t, cs, fmt.Sprintf("TestHash%02d", hash), fmt.Sprintf("TestBlockHash%02d", hash), hash, block)
	}

	return cs, checkPoints
}
