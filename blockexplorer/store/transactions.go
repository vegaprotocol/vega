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

package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"code.vegaprotocol.io/vega/blockexplorer/entities"
	pb "code.vegaprotocol.io/vega/protos/blockexplorer/api/v1"

	"github.com/georgysavva/scany/pgxscan"
)

var (
	ErrTxNotFound      = errors.New("transaction not found")
	ErrMultipleTxFound = errors.New("multiple transactions found")
)

func (s *Store) GetTransaction(ctx context.Context, txID string) (*pb.Transaction, error) {
	txID = strings.ToUpper(txID)

	query := `SELECT t.rowid, t.block_id, t.index, t.created_at, t.tx_hash, t.tx_result, t.cmd_type, t.submitter FROM tx_results t WHERE t.tx_hash=$1`
	var rows []entities.TxResultRow

	if err := pgxscan.Select(ctx, s.pool, &rows, query, txID); err != nil {
		return nil, fmt.Errorf("querying tx_results: %w", err)
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
	cmdTypes, exclCmdTypes, parties []string,
	first uint32,
	after *entities.TxCursor,
	last uint32,
	before *entities.TxCursor,
) ([]*pb.Transaction, error) {
	query := `SELECT t.rowid, t.block_height, t.index, t.created_at, t.tx_hash, t.tx_result, t.cmd_type, t.submitter FROM tx_results t`

	args := []interface{}{}
	predicates := []string{}

	// by default we want the most recent transactions so we'll set the limit to first
	// and sort order to desc
	limit := first
	sortOrder := "desc"

	// if we have a before cursor we want the results ordered earliest to latest
	// so the limit will be set to last and sort order to asc
	if before != nil {
		block := nextBindVar(&args, before.BlockNumber)
		index := nextBindVar(&args, before.TxIndex)
		predicate := fmt.Sprintf("(t.block_height, t.index) > (%s, %s)", block, index)
		predicates = append(predicates, predicate)
		limit = last
		sortOrder = "asc"
	}

	if after != nil {
		block := nextBindVar(&args, after.BlockNumber)
		index := nextBindVar(&args, after.TxIndex)
		predicate := fmt.Sprintf("(t.block_height, t.index) < (%s, %s)", block, index)
		predicates = append(predicates, predicate)
	}

	// just in case we have no before cursor, but we want to have the last N transactions in the data set
	// i.e. the earliest transactions, sorting ascending
	if last > 0 && first == 0 && after == nil && before == nil {
		limit = last
		sortOrder = "asc"
	}

	if len(cmdTypes) > 0 {
		predicates = append(predicates, fmt.Sprintf("t.cmd_type = ANY(%s)", nextBindVar(&args, cmdTypes)))
	}

	if len(exclCmdTypes) > 0 {
		predicates = append(predicates, fmt.Sprintf("t.cmd_type != ALL(%s)", nextBindVar(&args, exclCmdTypes)))
	}

	if len(parties) > 0 {
		partiesBytes := make([][]byte, len(parties))
		for i, p := range parties {
			partiesBytes[i] = []byte(p)
		}
		predicates = append(predicates, fmt.Sprintf("t.submitter = ANY(%s)", nextBindVar(&args, partiesBytes)))
	}

	for key, value := range filters {
		var predicate string

		if key == "tx.submitter" {
			// tx.submitter is lifted out of attributes and into tx_results by a trigger for faster access
			predicate = fmt.Sprintf("t.submitter= %s", nextBindVar(&args, value))
		} else if key == "cmd.type" {
			predicate = fmt.Sprintf("t.cmd_type= %s", nextBindVar(&args, value))
		} else if key == "block.height" {
			// much quicker to filter block height by joining to the block table than looking in attributes
			predicate = fmt.Sprintf("t.block_height = %s", nextBindVar(&args, value))
		} else {
			predicate = fmt.Sprintf(`
				EXISTS (SELECT 1 FROM events e JOIN attributes a ON e.rowid = a.event_id
						WHERE e.tx_id = t.rowid
						AND a.composite_key = %s
						AND a.value = %s)`, nextBindVar(&args, key), nextBindVar(&args, value))
		}
		predicates = append(predicates, predicate)
	}

	if len(predicates) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(predicates, " AND "))
	}

	query = fmt.Sprintf("%s ORDER BY t.block_height %s, t.index %s", query, sortOrder, sortOrder)
	query = fmt.Sprintf("%s LIMIT %d", query, limit)

	var rows []entities.TxResultRow
	if err := pgxscan.Select(ctx, s.pool, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("querying tx_results: %w", err)
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

	// make sure the results are always order in the same direction, i.e. newest first, regardless of the order of the
	// results from the database.
	sort.Slice(txs, func(i, j int) bool {
		if txs[i].Block == txs[j].Block {
			return txs[i].Index > txs[j].Index
		}
		return txs[i].Block > txs[j].Block
	})

	return txs, nil
}
