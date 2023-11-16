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

package sqlstore

import (
	"context"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"
)

type Delegations struct {
	*ConnectionSource
}

var delegationsOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
	ColumnOrdering{Name: "party_id", Sorting: ASC},
	ColumnOrdering{Name: "node_id", Sorting: ASC},
	ColumnOrdering{Name: "epoch_id", Sorting: ASC},
}

func NewDelegations(connectionSource *ConnectionSource) *Delegations {
	d := &Delegations{
		ConnectionSource: connectionSource,
	}
	return d
}

func (ds *Delegations) Add(ctx context.Context, d entities.Delegation) error {
	defer metrics.StartSQLQuery("Delegations", "Add")()
	_, err := ds.Connection.Exec(ctx,
		`INSERT INTO delegations(
			party_id,
			node_id,
			epoch_id,
			amount,
			tx_hash,
			vega_time,
			seq_num)
		 VALUES ($1,  $2,  $3,  $4,  $5, $6, $7);`,
		d.PartyID, d.NodeID, d.EpochID, d.Amount, d.TxHash, d.VegaTime, d.SeqNum)
	return err
}

func (ds *Delegations) GetAll(ctx context.Context) ([]entities.Delegation, error) {
	defer metrics.StartSQLQuery("Delegations", "GetAll")()
	delegations := []entities.Delegation{}
	err := pgxscan.Select(ctx, ds.Connection, &delegations, `
		SELECT * from delegations;`)
	return delegations, err
}

func (ds *Delegations) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Delegation, error) {
	defer metrics.StartSQLQuery("Delegations", "GetByTxHash")()

	var delegations []entities.Delegation
	query := `SELECT * FROM delegations WHERE tx_hash = $1`

	err := pgxscan.Select(ctx, ds.Connection, &delegations, query, txHash)
	if err != nil {
		return nil, err
	}

	return delegations, nil
}

func (ds *Delegations) Get(ctx context.Context,
	partyIDHex *string,
	nodeIDHex *string,
	epochID *int64,
	pagination entities.Pagination,
) ([]entities.Delegation, entities.PageInfo, error) {
	query := `SELECT * from delegations_current`
	var args []interface{}
	var pageInfo entities.PageInfo

	conditions := []string{}

	if partyIDHex != nil {
		partyID := entities.PartyID(*partyIDHex)
		conditions = append(conditions, fmt.Sprintf("party_id=%s", nextBindVar(&args, partyID)))
	}

	if nodeIDHex != nil {
		nodeID := entities.NodeID(*nodeIDHex)
		conditions = append(conditions, fmt.Sprintf("node_id=%s", nextBindVar(&args, nodeID)))
	}

	if epochID != nil {
		conditions = append(conditions, fmt.Sprintf("epoch_id=%s", nextBindVar(&args, *epochID)))
	}

	if len(conditions) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(conditions, " AND "))
	}

	defer metrics.StartSQLQuery("Delegations", "Get")()
	delegations := []entities.Delegation{}
	var err error
	if pagination != nil {
		switch p := pagination.(type) {
		case entities.CursorPagination:
			query, args, err = PaginateQuery[entities.DelegationCursor](query, args, delegationsOrdering, p)
			if err != nil {
				return nil, pageInfo, err
			}

			err := pgxscan.Select(ctx, ds.Connection, &delegations, query, args...)
			if err != nil {
				return nil, pageInfo, fmt.Errorf("querying delegations: %w", err)
			}

			delegations, pageInfo = entities.PageEntities[*v2.DelegationEdge](delegations, p)

			return delegations, pageInfo, nil
		default:
			panic("unsupported pagination")
		}
	}

	err = pgxscan.Select(ctx, ds.Connection, &delegations, query, args...)
	if err != nil {
		return nil, pageInfo, fmt.Errorf("querying delegations: %w", err)
	}

	return delegations, pageInfo, nil
}
