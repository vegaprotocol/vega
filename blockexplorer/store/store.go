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

package store

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/logging"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Store struct {
	log  *logging.Logger
	pool *pgxpool.Pool
}

func NewStore(config Config, log *logging.Logger) (*Store, error) {
	log = log.Named(namedLogger)

	poolConfig, err := config.Postgres.ToPgxPoolConfig()
	if err != nil {
		return nil, fmt.Errorf("creating connection source: %w", err)
	}

	pool, err := pgxpool.ConnectConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	store := &Store{
		log:  log,
		pool: pool,
	}
	return store, nil
}

func MustNewStore(config Config, log *logging.Logger) *Store {
	store, err := NewStore(config, log)
	if err != nil {
		log.Fatal("creating store", logging.Error(err))
	}
	return store
}
