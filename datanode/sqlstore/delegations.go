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
	"errors"
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
		case *entities.OffsetPagination:
			if p != nil {
				orderCols := []string{"epoch_id", "party_id", "node_id"}
				query, args = orderAndPaginateQuery(query, orderCols, *p, args...)
			}
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
			// invalid pagination type
			return nil, pageInfo, errors.New("invalid cursor")
		}
	}

	err = pgxscan.Select(ctx, ds.Connection, &delegations, query, args...)
	if err != nil {
		return nil, pageInfo, fmt.Errorf("querying delegations: %w", err)
	}

	return delegations, pageInfo, nil
}
