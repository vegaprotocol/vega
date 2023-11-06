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

func (s *Store) Close() {
	if s.pool != nil {
		s.log.Info("Closing connection to database")
		s.pool.Close()
	}
}

func NewStore(config Config, log *logging.Logger) (*Store, error) {
	log = log.Named(namedLogger)

	log.Info("Initiating connection to database")

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
