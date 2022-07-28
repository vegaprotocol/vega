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

package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/metrics"
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
	defer metrics.StartSQLQuery("Notary", "Add")()
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
	defer metrics.StartSQLQuery("Notary", "GetByResourceID")()
	ns := []entities.NodeSignature{}
	query := `SELECT resource_id, sig, kind FROM node_signatures WHERE resource_id=$1`
	err := pgxscan.Select(ctx, n.Connection, &ns, query, entities.NewNodeSignatureID(id))
	return ns, err
}
