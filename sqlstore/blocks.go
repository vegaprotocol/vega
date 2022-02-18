package sqlstore

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

var (
	ErrNoLastBlock       = errors.New("No last block")
	ErrBlockWaitTimedout = errors.New("Timed out waiting for TimeUpdate event")
	BlockWaitTimeout     = 5 * time.Second
)

type Blocks struct {
	*SQLStore
	lastBlock        *entities.Block
	lastBlockChanged chan struct{}
	mu               sync.Mutex
}

func NewBlocks(sqlStore *SQLStore) *Blocks {
	b := &Blocks{
		SQLStore:         sqlStore,
		lastBlockChanged: make(chan struct{}),
	}
	return b
}

func (bs *Blocks) Add(b entities.Block) error {
	ctx := context.Background()
	_, err := bs.pool.Exec(ctx,
		`insert into blocks(vega_time, height, hash) values ($1, $2, $3)`,
		b.VegaTime, b.Height, b.Hash)

	if err != nil {
		return fmt.Errorf("adding block: %w", err)
	}

	bs.setLastBlock(b)
	return nil
}

func (bs *Blocks) GetAll() ([]entities.Block, error) {
	ctx := context.Background()
	blocks := []entities.Block{}
	err := pgxscan.Select(ctx, bs.pool, &blocks,
		`SELECT vega_time, height, hash
		FROM blocks
		ORDER BY vega_time desc`)
	return blocks, err
}

func (bs *Blocks) GetAtHeight(height int64) (entities.Block, error) {
	// Check if it's in our cache first
	block, err := bs.getLastBlock()
	if err == nil && block.Height == height {
		return block, nil
	}

	// Else query the database
	err = pgxscan.Get(context.Background(), bs.pool, &block,
		`SELECT vega_time, height, hash
		FROM blocks
		WHERE height=$1`, height)
	return block, err
}

// WaitForBlockHeight will block until we have received an TimeUpdate message for this block
// This is a bit fiddly; because we can't control much about which order messages are sent or
// received on the event bus, sometimes we process events before the corresponding TimeUpdate
// message for that block has been processed. This is a  problem because we want to store the
// timestamp of the block on each row in the database. This function will block until we have
// the block time information for the specified block height, or fail after BlockWaitTimeout.
func (bs *Blocks) WaitForBlockHeight(height int64) (entities.Block, error) {
	ctx, cancel := context.WithTimeout(context.Background(), BlockWaitTimeout)
	defer cancel()
	for {
		if block, err := bs.GetAtHeight(height); err == nil {
			return block, nil
		}
		select {
		case <-bs.lastBlockChanged:
		case <-ctx.Done():
			return entities.Block{}, ErrBlockWaitTimedout
		}
	}
}

func (bs *Blocks) getLastBlock() (entities.Block, error) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	if bs.lastBlock != nil {
		return *bs.lastBlock, nil
	}
	return entities.Block{}, ErrNoLastBlock
}

func (bs *Blocks) setLastBlock(b entities.Block) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.lastBlock = &b
	close(bs.lastBlockChanged)
	bs.lastBlockChanged = make(chan struct{})
}
