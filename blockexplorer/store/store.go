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

package store

import (
	"context"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/blockexplorer/entities"
	"code.vegaprotocol.io/vega/logging"
	pb "code.vegaprotocol.io/vega/protos/blockexplorer"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Store struct {
	log  *logging.Logger
	pool *pgxpool.Pool
}

func NewStore(config Config, log *logging.Logger) (*Store, error) {
	log = log.Named(namedLogger)

	poolConfig, err := config.Postgres.ToPgxPoolConfig()
	if err != nil {
		return nil, fmt.Errorf("creating connection source: %w", err)
	}

	pool, err := pgxpool.ConnectConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	store := &Store{
		log:  log,
		pool: pool,
	}
	return store, nil
}

func MustNewStore(config Config, log *logging.Logger) *Store {
	store, err := NewStore(config, log)
	if err != nil {
		log.Fatal("creating store", logging.Error(err))
	}
	return store
}

func (s *Store) ListTransactions(ctx context.Context,
	filters map[string]string,
	limit uint32,
	before *entities.TxCursor,
	after *entities.TxCursor,
) ([]*pb.Transaction, error) {
	query := `SELECT * FROM tx_results`

	args := []interface{}{}
	predicates := []string{}
	if before != nil {
		block := nextBindVar(&args, before.BlockNumber)
		index := nextBindVar(&args, before.TxIndex)
		predicate := fmt.Sprintf("(block_id < %s OR (block_id = %s AND index < %s))", block, block, index)
		predicates = append(predicates, predicate)
	}

	if after != nil {
		block := nextBindVar(&args, after.BlockNumber)
		index := nextBindVar(&args, after.TxIndex)
		predicate := fmt.Sprintf("(block_id > %s OR (block_id = %s AND index ? %s))", block, block, index)
		predicates = append(predicates, predicate)
	}

	for key, value := range filters {
		predicate := fmt.Sprintf(`
			EXISTS (SELECT 1 FROM events e JOIN attributes a ON e.rowid = a.event_id
			        WHERE e.tx_id = tx_results.rowid
			        AND a.composite_key = %s
		 	        AND a.value = %s)`, nextBindVar(&args, key), nextBindVar(&args, value))
		predicates = append(predicates, predicate)
	}

	if len(predicates) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(predicates, " AND "))
	}

	query = fmt.Sprintf("%s ORDER BY block_id desc, index desc", query)
	query = fmt.Sprintf("%s LIMIT %d", query, limit)

	var rows []entities.TxResultRow
	if err := pgxscan.Select(ctx, s.pool, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("querying tx_results:%w", err)
	}

	txs := make([]*pb.Transaction, 0, len(rows))
	for _, row := range rows {
		tx, err := row.ToProto()
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}

	return txs, nil
}
