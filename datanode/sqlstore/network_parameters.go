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
	"sync"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"
)

type NetworkParameters struct {
	*ConnectionSource
	cacheLock sync.Mutex
	cache     map[string]entities.NetworkParameter
}

var networkParameterOrdering = TableOrdering{
	ColumnOrdering{Name: "key", Sorting: ASC},
}

func NewNetworkParameters(connectionSource *ConnectionSource) *NetworkParameters {
	p := &NetworkParameters{
		ConnectionSource: connectionSource,
		cache:            map[string]entities.NetworkParameter{},
	}
	return p
}

func (np *NetworkParameters) Add(ctx context.Context, r entities.NetworkParameter) error {
	np.cacheLock.Lock()
	defer np.cacheLock.Unlock()

	defer metrics.StartSQLQuery("NetworkParameters", "Add")()
	_, err := np.Connection.Exec(ctx,
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

	np.cache[r.Key] = r

	return err
}

func (np *NetworkParameters) GetByKey(ctx context.Context, key string) (entities.NetworkParameter, error) {
	defer metrics.StartSQLQuery("NetworkParameters", "GetByKey")()
	np.cacheLock.Lock()
	defer np.cacheLock.Unlock()

	var parameter entities.NetworkParameter
	if value, ok := np.cache[key]; ok {
		return value, nil
	}

	query := `SELECT * FROM network_parameters_current where key = $1`
	defer metrics.StartSQLQuery("NetworkParameters", "GetByKey")()
	err := pgxscan.Get(ctx, np.Connection, &parameter, query, key)
	if err != nil {
		return entities.NetworkParameter{}, np.wrapE(err)
	}

	np.cache[parameter.Key] = parameter
	return parameter, nil
}

func (np *NetworkParameters) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.NetworkParameter, error) {
	defer metrics.StartSQLQuery("NetworkParameters", "GetByTxHash")()

	var parameters []entities.NetworkParameter
	query := `SELECT * FROM network_parameters WHERE tx_hash = $1`

	err := pgxscan.Select(ctx, np.Connection, &parameters, query, txHash)
	if err != nil {
		return nil, np.wrapE(err)
	}

	return parameters, nil
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
