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
	"fmt"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
)

type KeyRotations struct {
	*ConnectionSource
}

var keyRotationsOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
	ColumnOrdering{Name: "node_id", Sorting: ASC},
	ColumnOrdering{Name: "old_pub_key", Sorting: ASC},
	ColumnOrdering{Name: "new_pub_key", Sorting: ASC},
}

func NewKeyRotations(connectionSource *ConnectionSource) *KeyRotations {
	return &KeyRotations{
		ConnectionSource: connectionSource,
	}
}

func (store *KeyRotations) Upsert(ctx context.Context, kr *entities.KeyRotation) error {
	defer metrics.StartSQLQuery("KeyRotations", "Upsert")()
	_, err := store.Connection.Exec(ctx, `
		INSERT INTO key_rotations(node_id, old_pub_key, new_pub_key, block_height, tx_hash, vega_time)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (node_id, vega_time) DO UPDATE SET
			old_pub_key = EXCLUDED.old_pub_key,
			new_pub_key = EXCLUDED.new_pub_key,
			block_height = EXCLUDED.block_height,
			tx_hash = EXCLUDED.tx_hash
	`, kr.NodeID, kr.OldPubKey, kr.NewPubKey, kr.BlockHeight, kr.TxHash, kr.VegaTime)

	// TODO Update node table with new pubkey here?

	return err
}

func (store *KeyRotations) GetAllPubKeyRotations(ctx context.Context, pagination entities.CursorPagination) ([]entities.KeyRotation, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("KeyRotations", "GetAll")()
	var pageInfo entities.PageInfo
	keyRotations := []entities.KeyRotation{}

	var args []interface{}
	var err error
	query := `SELECT * FROM key_rotations`
	query, args, err = PaginateQuery[entities.KeyRotationCursor](query, args, keyRotationsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err = pgxscan.Select(ctx, store.Connection, &keyRotations, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("failed to retrieve key rotations: %w", err)
	}

	keyRotations, pageInfo = entities.PageEntities[*v2.KeyRotationEdge](keyRotations, pagination)

	return keyRotations, pageInfo, nil
}

func (store *KeyRotations) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.KeyRotation, error) {
	defer metrics.StartSQLQuery("KeyRotations", "GetByTxHash")()

	var keyRotations []entities.KeyRotation
	query := `SELECT * FROM key_rotations WHERE tx_hash = $1`

	if err := pgxscan.Select(ctx, store.Connection, &keyRotations, query, txHash); err != nil {
		return nil, fmt.Errorf("failed to retrieve key rotations: %w", err)
	}

	return keyRotations, nil
}

func (store *KeyRotations) GetPubKeyRotationsPerNode(ctx context.Context, nodeID string, pagination entities.CursorPagination) ([]entities.KeyRotation, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("KeyRotations", "GetPubKeyRotationsPerNode")()
	var pageInfo entities.PageInfo
	id := entities.NodeID(nodeID)
	keyRotations := []entities.KeyRotation{}

	sorting, cmp, cursor := extractPaginationInfo(pagination)

	kc := &entities.KeyRotationCursor{}
	if err := kc.Parse(cursor); err != nil {
		return nil, pageInfo, fmt.Errorf("could not parse key rotation cursor: %w", err)
	}

	cursorParams := []CursorQueryParameter{
		NewCursorQueryParameter("vega_time", sorting, cmp, kc.VegaTime),
	}

	var args []interface{}
	query := fmt.Sprintf(`SELECT * FROM key_rotations WHERE node_id = %s`, nextBindVar(&args, id))
	query, args = orderAndPaginateWithCursor(query, pagination, cursorParams, args...)

	if err := pgxscan.Select(ctx, store.Connection, &keyRotations, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("failed to retrieve key rotations: %w", err)
	}

	keyRotations, pageInfo = entities.PageEntities[*v2.KeyRotationEdge](keyRotations, pagination)

	return keyRotations, pageInfo, nil
}
