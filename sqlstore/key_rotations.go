package sqlstore

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
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
		ON CONFLICT DO UPDATE
			old_pub_key = EXCLUDED.old_pub_key,
			new_pub_key = EXCLUDED.new_pub_key,
			block_height = EXCLUDED.block_height
	`, kr.NodeID, kr.OldPubKey, kr.NewPubKey, kr.BlockHeight, kr.VegaTime)

	// TODO Update node table with new pubkey here?

	return err
}

func (store *KeyRotations) GetAllPubKeyRotations(ctx context.Context) ([]entities.KeyRotation, error) {
	defer metrics.StartSQLQuery("KeyRotations", "GetByID")()
	keyRotations := []entities.KeyRotation{}
	err := pgxscan.Select(ctx, store.pool, &keyRotations, `SELECT * FROM key_rotations ORDER BY vega_time, node_id desc`)

	return keyRotations, err
}

func (store *KeyRotations) GetPubKeyRotationsPerNode(ctx context.Context, nodeId string) ([]entities.KeyRotation, error) {
	defer metrics.StartSQLQuery("KeyRotations", "GetPubKeyRotationsPerNode")()
	id := entities.NewNodeID(nodeId)

	keyRotations := []entities.KeyRotation{}
	err := pgxscan.Select(ctx, store.pool, &keyRotations, `SELECT * FROM key_rotations where node_id = $1 ORDER BY vega_time, node_id desc`, id)

	return keyRotations, err
}
