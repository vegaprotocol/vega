package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

type Notary struct {
	*ConnectionSource
}

func NewNotary(connectionSource *ConnectionSource) *Notary {
	return &Notary{
		ConnectionSource: connectionSource,
	}
}

func (n *Notary) Add(ctx context.Context, ns *entities.NodeSignature) error {

	query := `INSERT INTO node_signatures (resource_id, sig, kind)
		VALUES ($1, $2, $3)
		ON CONFLICT (resource_id, sig) DO NOTHING`

	if _, err := n.pool.Exec(ctx, query,
		ns.ResourceID,
		ns.Sig,
		ns.Kind,
	); err != nil {
		err = fmt.Errorf("could not insert node-signature into database: %w", err)
		return err
	}

	return nil
}

func (n *Notary) GetByResourceID(ctx context.Context, id string) ([]entities.NodeSignature, error) {
	ns := []entities.NodeSignature{}
	query := `SELECT resource_id, sig, kind FROM node_signatures WHERE resource_id=$1`
	err := pgxscan.Select(ctx, n.Connection, &ns, query, entities.NewNodeSignatureID(id))
	return ns, err
}
