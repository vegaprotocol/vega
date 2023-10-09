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
	"embed"
	"fmt"

	"code.vegaprotocol.io/vega/logging"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/pkg/errors"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var EmbedMigrations embed.FS

const SQLMigrationsDir = "migrations"

func MigrateToLatestSchema(log *logging.Logger, config Config) error {
	goose.SetBaseFS(EmbedMigrations)
	goose.SetLogger(log.Named("db migration").GooseLogger())

	poolConfig, err := config.Postgres.ToPgxPoolConfig()
	if err != nil {
		return errors.Wrap(err, "migrating schema")
	}

	db := stdlib.OpenDB(*poolConfig.ConnConfig)
	defer db.Close()

	if err := goose.Up(db, SQLMigrationsDir); err != nil {
		return fmt.Errorf("error migrating sql schema: %w", err)
	}
	return nil
}
