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
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
)

type testBlockSource struct {
	blockStore *sqlstore.Blocks
	blockTime  time.Time
}

func (bs *testBlockSource) getNextBlock(t *testing.T, ctx context.Context) entities.Block {
	t.Helper()
	bs.blockTime = bs.blockTime.Add(1 * time.Second)
	return addTestBlockForTime(t, ctx, bs.blockStore, bs.blockTime)
}

func addTestBlock(t *testing.T, ctx context.Context, bs *sqlstore.Blocks) entities.Block {
	t.Helper()
	return addTestBlockForTime(t, ctx, bs, time.Now())
}

func addTestBlockForTime(t *testing.T, ctx context.Context, bs *sqlstore.Blocks, vegaTime time.Time) entities.Block {
	t.Helper()
	return addTestBlockForHeightAndTime(t, ctx, bs, 2, vegaTime)
}

func addTestBlockForHeightAndTime(t *testing.T, ctx context.Context, bs *sqlstore.Blocks, height int64, vegaTime time.Time) entities.Block {
	t.Helper()
	// Make a block
	hash, err := hex.DecodeString("deadbeef")
	assert.NoError(t, err)

	// Postgres only stores timestamps in microsecond resolution
	block1 := entities.Block{
		VegaTime: vegaTime.Truncate(time.Microsecond),
		Height:   height,
		Hash:     hash,
	}

	// Add it to the database
	err = bs.Add(ctx, block1)
	assert.NoError(t, err)

	return block1
}

func TestBlock(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)

	// See how many we have right now (it's possible that other tests added some)
	blocks, err := bs.GetAll(ctx)
	assert.NoError(t, err)
	blocksLen := len(blocks)

	block1 := addTestBlock(t, ctx, bs)

	// Query and check we've got back a block the same as the one we put in
	blocks, err = bs.GetAll(ctx)
	assert.NoError(t, err)
	assert.Len(t, blocks, blocksLen+1)
	assert.Equal(t, blocks[0], block1)

	// Add it again, we should get a primary key violation [do this last as it invalidates tx]
	err = bs.Add(ctx, block1)
	assert.Error(t, err)
}

func TestGetLastBlock(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)

	now := time.Now()

	addTestBlockForTime(t, ctx, bs, now)
	block2 := addTestBlockForTime(t, ctx, bs, now.Add(1*time.Second))

	// Query the last block
	block, err := bs.GetLastBlock(ctx)
	assert.NoError(t, err)
	assert.Equal(t, block2, block)
}

func TestGetOldestHistoryBlock(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)

	now := time.Now()

	block1 := addTestBlockForTime(t, ctx, bs, now)
	addTestBlockForTime(t, ctx, bs, now.Add(1*time.Second))

	// Query the first block
	block, err := bs.GetOldestHistoryBlock(ctx)
	assert.NoError(t, err)
	assert.Equal(t, block1, block)
}

func TestGetOldestHistoryBlockWhenNoHistoryBlocks(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)
	// Query the first block
	_, err := bs.GetOldestHistoryBlock(ctx)
	assert.Equal(t, entities.ErrNotFound, err)
}

func TestGetLastBlockAfterRecovery(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)

	now := time.Now()

	addTestBlockForTime(t, ctx, bs, now)
	block2 := addTestBlockForTime(t, ctx, bs, now.Add(1*time.Second))

	// Recreate the store
	bs = sqlstore.NewBlocks(connectionSource)

	// Query the last block
	block, err := bs.GetLastBlock(ctx)
	assert.NoError(t, err)
	assert.Equal(t, block2, block)
}

func TestGetLastBlockWhenNoBlocks(t *testing.T) {
	ctx := tempTransaction(t)

	bs := sqlstore.NewBlocks(connectionSource)

	// Query the last block
	_, err := bs.GetLastBlock(ctx)
	assert.Equal(t, entities.ErrNotFound, err)
}
