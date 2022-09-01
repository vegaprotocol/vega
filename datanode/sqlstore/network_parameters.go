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

type NetworkParameters struct {
	*ConnectionSource
}

var networkParameterOrdering = TableOrdering{
	ColumnOrdering{Name: "key", Sorting: ASC},
}

func NewNetworkParameters(connectionSource *ConnectionSource) *NetworkParameters {
	p := &NetworkParameters{
		ConnectionSource: connectionSource,
	}
	return p
}

func (ps *NetworkParameters) Add(ctx context.Context, r entities.NetworkParameter) error {
	defer metrics.StartSQLQuery("NetworkParameters", "Add")()
	_, err := ps.Connection.Exec(ctx,
		`INSERT INTO network_parameters(
			key,
			value,
			tx_hash,
			vega_time)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (key, vega_time) DO UPDATE SET
			value = EXCLUDED.value,
			tx_hash = EXCLUDED.tx_hash;
		 `,
		r.Key, r.Value, r.TxHash, r.VegaTime)
	return err
}

func (np *NetworkParameters) GetAll(ctx context.Context, pagination entities.CursorPagination) ([]entities.NetworkParameter, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("NetworkParameters", "GetAll")()
	var pageInfo entities.PageInfo

	// we are ordering by key so we aren't going to change the sort order for newest first
	// therefore we just set it to default to false in case it's true in the request
	if pagination.NewestFirst {
		pagination.NewestFirst = false
	}

	var (
		nps  []entities.NetworkParameter
		args []interface{}
		err  error
	)
	query := `SELECT * FROM network_parameters_current`
	query, args, err = PaginateQuery[entities.NetworkParameterCursor](query, args, networkParameterOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err = pgxscan.Select(ctx, np.Connection, &nps, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("could not get network parameters: %w", err)
	}

	nps, pageInfo = entities.PageEntities[*v2.NetworkParameterEdge](nps, pagination)
	return nps, pageInfo, nil
}
