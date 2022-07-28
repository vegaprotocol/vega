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
	"encoding/hex"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/sqlstore"
	"github.com/stretchr/testify/assert"
)

func addTestBlock(t *testing.T, bs *sqlstore.Blocks) entities.Block {
	return addTestBlockForTime(t, bs, time.Now())
}

func addTestBlockForTime(t *testing.T, bs *sqlstore.Blocks, vegaTime time.Time) entities.Block {
	// Make a block
	hash, err := hex.DecodeString("deadbeef")
	assert.NoError(t, err)

	// Postgres only stores timestamps in microsecond resolution
	block1 := entities.Block{
		VegaTime: vegaTime.Truncate(time.Microsecond),
		Height:   2,
		Hash:     hash,
	}

	// Add it to the database
	err = bs.Add(context.Background(), block1)
	assert.NoError(t, err)

	return block1
}

func TestBlock(t *testing.T) {
	defer DeleteEverything()
	bs := sqlstore.NewBlocks(connectionSource)

	// See how many we have right now (it's possible that other tests added some)
	blocks, err := bs.GetAll()
	assert.NoError(t, err)
	blocks_len := len(blocks)

	block1 := addTestBlock(t, bs)

	// Add it again, we should get a primary key violation
	err = bs.Add(context.Background(), block1)
	assert.Error(t, err)

	// Query and check we've got back a block the same as the one we put in
	blocks, err = bs.GetAll()
	assert.NoError(t, err)
	assert.Len(t, blocks, blocks_len+1)
	assert.Equal(t, blocks[0], block1)
}

func TestGetLastBlock(t *testing.T) {
	defer DeleteEverything()
	bs := sqlstore.NewBlocks(connectionSource)

	now := time.Now()

	addTestBlockForTime(t, bs, now)
	block2 := addTestBlockForTime(t, bs, now.Add(1*time.Second))

	// Query the last block
	block, err := bs.GetLastBlock(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, block2, block)
}

func TestGetLastBlockAfterRecovery(t *testing.T) {
	defer DeleteEverything()
	bs := sqlstore.NewBlocks(connectionSource)

	now := time.Now()

	addTestBlockForTime(t, bs, now)
	block2 := addTestBlockForTime(t, bs, now.Add(1*time.Second))

	// Recreate the store
	bs = sqlstore.NewBlocks(connectionSource)

	// Query the last block
	block, err := bs.GetLastBlock(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, block2, block)
}

func TestGetLastBlockWhenNoBlocks(t *testing.T) {
	defer DeleteEverything()
	bs := sqlstore.NewBlocks(connectionSource)

	// Query the last block
	_, err := bs.GetLastBlock(context.Background())
	assert.Equal(t, sqlstore.ErrNoLastBlock, err)
}
