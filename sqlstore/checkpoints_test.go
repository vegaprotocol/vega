package sqlstore_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
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
		actual, err := checkpointStore.GetAll(ctx)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}
