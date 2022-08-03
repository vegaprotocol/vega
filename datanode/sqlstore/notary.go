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

func (n *Notary) GetByResourceID(ctx context.Context, id string, pagination entities.CursorPagination) ([]entities.NodeSignature, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("Notary", "GetByResourceID")()
	var pageInfo entities.PageInfo
	var args []interface{}

	sorting, cmp, cursor := extractPaginationInfo(pagination)

	nc := &entities.NodeSignatureCursor{}
	if err := nc.Parse(cursor); err != nil {
		return nil, pageInfo, fmt.Errorf("could not parse cursor information: %w", err)
	}

	cursorParams := []CursorQueryParameter{
		NewCursorQueryParameter("resource_id", sorting, EQ, entities.NewNodeSignatureID(id)),
		NewCursorQueryParameter("sig", sorting, cmp, nc.Sig),
	}

	ns := []entities.NodeSignature{}
	query := `SELECT resource_id, sig, kind FROM node_signatures`
	query, args = orderAndPaginateWithCursor(query, pagination, cursorParams, args...)

	if err := pgxscan.Select(ctx, n.Connection, &ns, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("could not get node signatures for resource: %w", err)
	}

	ns, pageInfo = entities.PageEntities[*v2.NodeSignatureEdge](ns, pagination)
	return ns, pageInfo, nil
}
