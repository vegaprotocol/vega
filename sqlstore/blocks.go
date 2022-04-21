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
	*ConnectionSource
	lastBlock        *entities.Block
	lastBlockChanged chan struct{}
	mu               sync.Mutex
}

func NewBlocks(connectionSource *ConnectionSource) *Blocks {
	b := &Blocks{
		ConnectionSource: connectionSource,
		lastBlockChanged: make(chan struct{}),
	}
	return b
}

func (bs *Blocks) Add(ctx context.Context, b entities.Block) error {
	_, err := bs.Connection.Exec(ctx,
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
	err := pgxscan.Select(ctx, bs.Connection, &blocks,
		`SELECT vega_time, height, hash
		FROM blocks
		ORDER BY vega_time desc`)
	return blocks, err
}

func (bs *Blocks) GetAtHeight(height int64) (entities.Block, error) {
	// Check if it's in our cache first
	block, err := bs.GetLastBlock()
	if err == nil && block.Height == height {
		return block, nil
	}

	// Else query the database
	err = pgxscan.Get(context.Background(), bs.Connection, &block,
		`SELECT vega_time, height, hash
		FROM blocks
		WHERE height=$1`, height)
	return block, err
}

func (bs *Blocks) GetLastBlock() (entities.Block, error) {
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
