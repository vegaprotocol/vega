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

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgconn"
)

type Chain struct {
	*ConnectionSource
}

func NewChain(connectionSource *ConnectionSource) *Chain {
	return &Chain{
		ConnectionSource: connectionSource,
	}
}

func (c *Chain) Get(ctx context.Context) (entities.Chain, error) {
	defer metrics.StartSQLQuery("Chain", "Get")()
	chain := entities.Chain{}

	query := `SELECT id from chain`
	return chain, c.wrapE(pgxscan.Get(ctx, c.Connection, &chain, query))
}

func (c *Chain) Set(ctx context.Context, chain entities.Chain) error {
	defer metrics.StartSQLQuery("Chain", "Set")()
	query := `INSERT INTO chain(id) VALUES($1)`
	_, err := c.Connection.Exec(ctx, query, chain.ID)
	if e, ok := err.(*pgconn.PgError); ok {
		// 23505 is postgres error code for a unique constraint violation
		if e.Code == "23505" {
			return entities.ErrChainAlreadySet
		}
	}
	return err
}
