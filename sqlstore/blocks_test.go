package sqlstore_test

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
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
