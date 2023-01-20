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
	"errors"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/blockexplorer/entities"
	pb "code.vegaprotocol.io/vega/protos/blockexplorer"
	"github.com/georgysavva/scany/pgxscan"
)

var (
	ErrTxNotFound      = errors.New("transaction not found")
	ErrMultipleTxFound = errors.New("multiple transactions found")
)

func (s *Store) GetTransaction(ctx context.Context, txID string) (*pb.Transaction, error) {
	query := `SELECT * FROM tx_results where tx_hash=$1`
	var rows []entities.TxResultRow

	if err := pgxscan.Select(ctx, s.pool, &rows, query, txID); err != nil {
		return nil, fmt.Errorf("querying tx_results:%w", err)
	}

	if len(rows) == 0 {
		return nil, ErrTxNotFound
	}

	if len(rows) > 1 {
		return nil, ErrMultipleTxFound
	}

	tx, err := rows[0].ToProto()
	if err != nil {
		return nil, err
	}
	return tx, nil
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
		predicate := fmt.Sprintf("(block_id > %s OR (block_id = %s AND index > %s))", block, block, index)
		predicates = append(predicates, predicate)
	}

	for key, value := range filters {
		var predicate string

		if key == "tx.submitter" {
			// tx.submitter is lifted out of attributes and into tx_results by a trigger for faster access
			predicate = fmt.Sprintf("tx_results.submitter=%s", nextBindVar(&args, value))
		} else if key == "cmd.type" {
			predicate = fmt.Sprintf("tx_results.cmd_type=%s", nextBindVar(&args, value))
		} else if key == "block.height" {
			// much quicker to filter block height by joining to the block table than looking in attributes
			predicate = fmt.Sprintf("block_id = (select b.rowid from blocks b where b.height = %s)", nextBindVar(&args, value))
		} else {
			predicate = fmt.Sprintf(`
				EXISTS (SELECT 1 FROM events e JOIN attributes a ON e.rowid = a.event_id
						WHERE e.tx_id = tx_results.rowid
						AND a.composite_key = %s
						AND a.value = %s)`, nextBindVar(&args, key), nextBindVar(&args, value))
		}
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
			s.log.Warn(fmt.Sprintf("unable to decode transaction %s: %v", row.TxHash, err))
			continue
		}
		txs = append(txs, tx)
	}

	return txs, nil
}
