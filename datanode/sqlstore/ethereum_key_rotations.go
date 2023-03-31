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

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
)

var ekrOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
	ColumnOrdering{Name: "node_id", Sorting: ASC},
	ColumnOrdering{Name: "old_address", Sorting: ASC},
	ColumnOrdering{Name: "new_address", Sorting: ASC},
}

type EthereumKeyRotations struct {
	*ConnectionSource
}

func NewEthereumKeyRotations(connectionSource *ConnectionSource) *EthereumKeyRotations {
	return &EthereumKeyRotations{
		ConnectionSource: connectionSource,
	}
}

func (store *EthereumKeyRotations) Add(ctx context.Context, kr entities.EthereumKeyRotation) error {
	defer metrics.StartSQLQuery("EthereumKeyRotations", "Add")()
	_, err := store.Connection.Exec(ctx, `
		INSERT INTO ethereum_key_rotations(node_id, old_address, new_address, block_height, tx_hash, vega_time, seq_num)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, kr.NodeID, kr.OldAddress, kr.NewAddress, kr.BlockHeight, kr.TxHash, kr.VegaTime, kr.SeqNum)

	return err
}

func (store *EthereumKeyRotations) List(ctx context.Context,
	nodeID entities.NodeID,
	pagination entities.CursorPagination,
) ([]entities.EthereumKeyRotation, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("EthereumKeyRotations", "List")()

	args := []interface{}{}
	whereClause := ""

	if nodeID.String() != "" {
		whereClause = `WHERE node_id = $1`
		args = append(args, nodeID)
	}

	query := `SELECT * FROM ethereum_key_rotations ` + whereClause

	query, args, err := PaginateQuery[entities.EthereumKeyRotationCursor](query, args, ekrOrdering, pagination)
	if err != nil {
		return nil, entities.PageInfo{}, err
	}

	ethereumKeyRotations := []entities.EthereumKeyRotation{}

	if err = pgxscan.Select(ctx, store.Connection, &ethereumKeyRotations, query, args...); err != nil {
		return nil, entities.PageInfo{}, err
	}

	paged, pageInfo := entities.PageEntities[*v2.EthereumKeyRotationEdge](ethereumKeyRotations, pagination)
	return paged, pageInfo, nil
}

func (store *EthereumKeyRotations) GetByTxHash(
	ctx context.Context,
	txHash entities.TxHash,
) ([]entities.EthereumKeyRotation, error) {
	defer metrics.StartSQLQuery("EthereumKeyRotations", "GetByTxHash")()

	var ethereumKeyRotations []entities.EthereumKeyRotation
	query := `SELECT * FROM ethereum_key_rotations WHERE tx_hash = $1`

	if err := pgxscan.Select(ctx, store.Connection, &ethereumKeyRotations, query, txHash); err != nil {
		return nil, err
	}

	return ethereumKeyRotations, nil
}
