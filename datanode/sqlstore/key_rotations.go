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

func NewKeyRotations(connectionSource *ConnectionSource) *KeyRotations {
	return &KeyRotations{
		ConnectionSource: connectionSource,
	}
}

func (store *KeyRotations) Upsert(ctx context.Context, kr *entities.KeyRotation) error {
	defer metrics.StartSQLQuery("KeyRotations", "Upsert")()
	_, err := store.pool.Exec(ctx, `
		INSERT INTO key_rotations(node_id, old_pub_key, new_pub_key, block_height, vega_time)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (node_id, vega_time) DO UPDATE SET
			old_pub_key = EXCLUDED.old_pub_key,
			new_pub_key = EXCLUDED.new_pub_key,
			block_height = EXCLUDED.block_height
	`, kr.NodeID, kr.OldPubKey, kr.NewPubKey, kr.BlockHeight, kr.VegaTime)

	// TODO Update node table with new pubkey here?

	return err
}

func (store *KeyRotations) GetAllPubKeyRotations(ctx context.Context, pagination entities.CursorPagination) ([]entities.KeyRotation, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("KeyRotations", "GetAll")()
	var pageInfo entities.PageInfo
	keyRotations := []entities.KeyRotation{}

	sorting, cmp, cursor := extractPaginationInfo(pagination)

	kc := &entities.KeyRotationCursor{}
	if err := kc.Parse(cursor); err != nil {
		return nil, pageInfo, fmt.Errorf("could not parse key rotation cursor: %w", err)
	}

	cursorParams := []CursorQueryParameter{
		NewCursorQueryParameter("vega_time", sorting, cmp, kc.VegaTime),
		NewCursorQueryParameter("node_id", sorting, cmp, entities.NodeID(kc.NodeID)),
	}

	var args []interface{}
	query := `SELECT * FROM key_rotations`
	query, args = orderAndPaginateWithCursor(query, pagination, cursorParams, args...)

	if err := pgxscan.Select(ctx, store.pool, &keyRotations, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("failed to retrieve key rotations: %w", err)
	}

	keyRotations, pageInfo = entities.PageEntities[*v2.KeyRotationEdge](keyRotations, pagination)

	return keyRotations, pageInfo, nil
}

func (store *KeyRotations) GetPubKeyRotationsPerNode(ctx context.Context, nodeId string, pagination entities.CursorPagination) ([]entities.KeyRotation, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("KeyRotations", "GetPubKeyRotationsPerNode")()
	var pageInfo entities.PageInfo
	id := entities.NodeID(nodeId)
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

	if err := pgxscan.Select(ctx, store.pool, &keyRotations, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("failed to retrieve key rotations: %w", err)
	}

	keyRotations, pageInfo = entities.PageEntities[*v2.KeyRotationEdge](keyRotations, pagination)

	return keyRotations, pageInfo, nil
}
