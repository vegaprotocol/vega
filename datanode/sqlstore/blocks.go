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

package sqlstore

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
)

var (
	ErrNoHistoryBlock    = errors.New("No history block")
	ErrNoLastBlock       = errors.New("No last block")
	ErrBlockWaitTimedout = errors.New("Timed out waiting for TimeUpdate event")
	BlockWaitTimeout     = 5 * time.Second
)

type Blocks struct {
	*ConnectionSource
	lastBlock *entities.Block
	mu        sync.Mutex
}

func NewBlocks(connectionSource *ConnectionSource) *Blocks {
	b := &Blocks{
		ConnectionSource: connectionSource,
	}
	return b
}

func (bs *Blocks) Add(ctx context.Context, b entities.Block) error {
	defer metrics.StartSQLQuery("Blocks", "Add")()
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
	defer metrics.StartSQLQuery("Blocks", "GetAll")()
	ctx := context.Background()
	blocks := []entities.Block{}
	err := pgxscan.Select(ctx, bs.Connection, &blocks,
		`SELECT vega_time, height, hash
		FROM blocks
		ORDER BY vega_time desc`)
	return blocks, err
}

func (bs *Blocks) GetAtHeight(ctx context.Context, height int64) (entities.Block, error) {
	// Check if it's in our cache first
	block, err := bs.GetLastBlock(ctx)
	if err == nil && block.Height == height {
		return block, nil
	}

	// Else query the database
	defer metrics.StartSQLQuery("Blocks", "GetAtHeight")()
	err = pgxscan.Get(context.Background(), bs.Connection, &block,
		`SELECT vega_time, height, hash
		FROM blocks
		WHERE height=$1`, height)
	return block, err
}

// GetLastBlock return the last block or ErrNoLastBlock if no block is found.
func (bs *Blocks) GetLastBlock(ctx context.Context) (entities.Block, error) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	if bs.lastBlock != nil {
		return *bs.lastBlock, nil
	}
	defer metrics.StartSQLQuery("Blocks", "GetLastBlock")()

	lastBlock, err := GetLastBlockUsingConnection(ctx, bs.Connection)
	bs.lastBlock = lastBlock
	if err != nil {
		return entities.Block{}, err
	}

	return *lastBlock, err
}

func (bs *Blocks) setLastBlock(b entities.Block) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.lastBlock = &b
}

func (bs *Blocks) GetOldestHistoryBlock(ctx context.Context) (entities.Block, error) {
	defer metrics.StartSQLQuery("Blocks", "GetOldestHistoryBlock")()

	return GetOldestHistoryBlockUsingConnection(ctx, bs.Connection)
}

func GetOldestHistoryBlockUsingConnection(ctx context.Context, connection Connection) (entities.Block, error) {
	block := &entities.Block{}
	err := pgxscan.Get(ctx, connection, block, `SELECT vega_time, height, hash
		FROM blocks order by height asc limit 1`)

	if errors.Is(err, pgx.ErrNoRows) {
		return entities.Block{}, ErrNoHistoryBlock
	}

	if err != nil {
		return entities.Block{}, err
	}

	return *block, nil
}

func GetLastBlockUsingConnection(ctx context.Context, connection Connection) (*entities.Block, error) {
	block := &entities.Block{}
	err := pgxscan.Get(ctx, connection, block,
		`SELECT vega_time, height, hash
		FROM last_block`)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoLastBlock
	}

	return block, err
}
