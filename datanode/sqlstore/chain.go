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
	"errors"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/metrics"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
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
	err := pgxscan.Get(ctx, c.pool, &chain, query)

	if errors.Is(err, pgx.ErrNoRows) {
		return entities.Chain{}, entities.ErrChainNotFound
	}

	return chain, err
}

func (c *Chain) Set(ctx context.Context, chain entities.Chain) error {
	defer metrics.StartSQLQuery("Chain", "Set")()
	query := `INSERT INTO chain(id) VALUES($1)`
	_, err := c.pool.Exec(ctx, query, chain.ID)
	if e, ok := err.(*pgconn.PgError); ok {
		// 23505 is postgres error code for a unique constraint violation
		if e.Code == "23505" {
			return entities.ErrChainAlreadySet
		}
	}
	return err
}
