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

	if err := goose.Up(goose.SqlDbToGooseAdapter{Conn: db}, SQLMigrationsDir); err != nil {
		return fmt.Errorf("error migrating sql schema: %w", err)
	}
	return nil
}
