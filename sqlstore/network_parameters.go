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

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
	"github.com/georgysavva/scany/pgxscan"
)

type NetworkParameters struct {
	*ConnectionSource
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
			vega_time)
		 VALUES ($1,  $2,  $3)
		 ON CONFLICT (key, vega_time) DO UPDATE SET
			value = EXCLUDED.value;
		 `,
		r.Key, r.Value, r.VegaTime)
	return err
}

func (np *NetworkParameters) GetAll(ctx context.Context) ([]entities.NetworkParameter, error) {
	defer metrics.StartSQLQuery("NetworkParameters", "GetAll")()
	var nps []entities.NetworkParameter
	query := `SELECT DISTINCT ON (key) * FROM network_parameters ORDER BY key, vega_time DESC`
	err := pgxscan.Select(ctx, np.Connection, &nps, query)
	return nps, err
}
