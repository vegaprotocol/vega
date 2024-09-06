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

	_, err := bs.Exec(ctx,
		`insert into blocks(vega_time, height, hash) values ($1, $2, $3)`,
		b.VegaTime, b.Height, b.Hash)
	if err != nil {
		return fmt.Errorf("adding block: %w", err)
	}

	bs.setLastBlock(b)
	return nil
}

func (bs *Blocks) GetAll(ctx context.Context) ([]entities.Block, error) {
	defer metrics.StartSQLQuery("Blocks", "GetAll")()
	blocks := []entities.Block{}
	err := pgxscan.Select(ctx, bs.ConnectionSource, &blocks,
		`SELECT vega_time, height, hash
		FROM blocks
		ORDER BY vega_time desc`)
	return blocks, err
}

func (bs *Blocks) GetAtHeight(ctx context.Context, height int64) (entities.Block, error) {
	connection := bs.ConnectionSource
	defer metrics.StartSQLQuery("Blocks", "GetAtHeight")()

	// Check if it's in our cache first
	block, err := bs.GetLastBlock(ctx)
	if err == nil && block.Height == height {
		return block, nil
	}

	return GetAtHeightUsingConnection(ctx, connection, height)
}

func GetAtHeightUsingConnection(ctx context.Context, connection Connection, height int64) (entities.Block, error) {
	block := entities.Block{}

	return block, wrapE(pgxscan.Get(ctx, connection, &block,
		`SELECT vega_time, height, hash
		FROM blocks
		WHERE height=$1`, height))
}

// GetLastBlock return the last block or ErrNoLastBlock if no block is found.
func (bs *Blocks) GetLastBlock(ctx context.Context) (entities.Block, error) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	if bs.lastBlock != nil {
		fmt.Println("got last block", bs.lastBlock.Height)
		return *bs.lastBlock, nil
	}
	defer metrics.StartSQLQuery("Blocks", "GetLastBlock")()

	lastBlock, err := bs.getLastBlockUsingConnection(ctx, bs.ConnectionSource)
	// FIXME(woot?): why do we set that before checking for error, that would clearly fuckup the cache or something innit?
	bs.lastBlock = lastBlock
	if err != nil {
		return entities.Block{}, err
	}

	return *lastBlock, err
}

func (bs *Blocks) setLastBlock(b entities.Block) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	fmt.Println("set last block", b.Height)
	bs.lastBlock = &b
}

func (bs *Blocks) GetOldestHistoryBlock(ctx context.Context) (entities.Block, error) {
	defer metrics.StartSQLQuery("Blocks", "GetOldestHistoryBlock")()

	return bs.getOldestHistoryBlockUsingConnection(ctx, bs.ConnectionSource)
}

func (bs *Blocks) getOldestHistoryBlockUsingConnection(ctx context.Context, connection Connection) (entities.Block, error) {
	block := &entities.Block{}
	if err := pgxscan.Get(ctx, connection, block, `SELECT vega_time, height, hash
		FROM blocks order by height asc limit 1`); err != nil {
		return entities.Block{}, bs.wrapE(err)
	}

	return *block, nil
}

func GetOldestHistoryBlockUsingConnection(ctx context.Context, connection Connection) (entities.Block, error) {
	block := &entities.Block{}
	err := pgxscan.Get(ctx, connection, block, `SELECT vega_time, height, hash
		FROM blocks order by height asc limit 1`)

	if errors.Is(err, pgx.ErrNoRows) {
		return entities.Block{}, entities.ErrNotFound
	}

	return *block, nil
}

func (bs *Blocks) getLastBlockUsingConnection(ctx context.Context, connection Connection) (*entities.Block, error) {
	block := &entities.Block{}
	if err := pgxscan.Get(ctx, connection, block,
		`SELECT vega_time, height, hash
		FROM last_block`); err != nil {
		return nil, bs.wrapE(err)
	}

	return block, nil
}

func GetLastBlockUsingConnection(ctx context.Context, connection Connection) (*entities.Block, error) {
	block := &entities.Block{}
	err := pgxscan.Get(ctx, connection, block,
		`SELECT vega_time, height, hash
		FROM last_block`)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, entities.ErrNotFound
	}

	return block, err
}
