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
	"bytes"
	"context"
	"embed"
	"fmt"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/pkg/errors"
	"github.com/pressly/goose/v3"
)

var ErrBadID = errors.New("Bad ID (must be hex string)")

//go:embed migrations/*.sql
var EmbedMigrations embed.FS

const SQLMigrationsDir = "migrations"

func MigrateToLatestSchema(log *logging.Logger, config Config) error {
	goose.SetBaseFS(EmbedMigrations)
	goose.SetLogger(log.Named("db migration").GooseLogger())

	poolConfig, err := config.ConnectionConfig.GetPoolConfig()
	if err != nil {
		return errors.Wrap(err, "migrating schema")
	}

	db := stdlib.OpenDB(*poolConfig.ConnConfig)
	defer db.Close()

	currentVersion, err := goose.GetDBVersion(db)
	if err != nil {
		return err
	}

	if currentVersion > 0 && config.WipeOnStartup {
		if err := goose.Down(db, SQLMigrationsDir); err != nil {
			return fmt.Errorf("error clearing sql schema: %w", err)
		}
	}

	if err := goose.Up(db, SQLMigrationsDir); err != nil {
		return fmt.Errorf("error migrating sql schema: %w", err)
	}
	return nil
}

func CreateVegaSchema(log *logging.Logger, connConfig ConnectionConfig) error {
	goose.SetBaseFS(EmbedMigrations)
	goose.SetLogger(log.Named("snapshot schema creation").GooseLogger())

	poolConfig, err := connConfig.GetPoolConfig()
	if err != nil {
		return fmt.Errorf("failed to get connection configuration: %w", err)
	}

	db := stdlib.OpenDB(*poolConfig.ConnConfig)
	defer func() {
		err := db.Close()
		if err != nil {
			log.Errorf("error when closing connection used to create vega schema:%w", err)
		}
	}()

	if err := goose.Up(db, SQLMigrationsDir); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

func RecreateVegaDatabase(ctx context.Context, log *logging.Logger, connConfig ConnectionConfig) error {
	postgresDbConn, err := pgx.Connect(context.Background(), connConfig.GetConnectionStringForPostgresDatabase())
	if err != nil {
		return fmt.Errorf("unable to connect to database:%w", err)
	}

	defer func() {
		err := postgresDbConn.Close(ctx)
		if err != nil {
			log.Errorf("error closing database connection after loading snapshot:%v", err)
		}
	}()

	_, err = postgresDbConn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH ( FORCE )", connConfig.Database))
	if err != nil {
		return fmt.Errorf("unable to drop existing database:%w", err)
	}

	_, err = postgresDbConn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s TEMPLATE template0 LC_COLLATE 'C' LC_CTYPE 'C';", connConfig.Database))
	if err != nil {
		return fmt.Errorf("unable to create database:%w", err)
	}
	return nil
}

func ApplyDataRetentionPolicies(config Config) error {
	poolConfig, err := config.ConnectionConfig.GetPoolConfig()
	if err != nil {
		return errors.Wrap(err, "applying data retention policy")
	}

	db := stdlib.OpenDB(*poolConfig.ConnConfig)
	defer db.Close()
	for _, policy := range config.RetentionPolicies {
		if _, err := db.Exec(fmt.Sprintf("SELECT remove_retention_policy('%s', true);", policy.HypertableOrCaggName)); err != nil {
			return errors.Wrapf(err, "removing retention policy from %s", policy.HypertableOrCaggName)
		}

		if _, err := db.Exec(fmt.Sprintf("SELECT add_retention_policy('%s', INTERVAL '%s');", policy.HypertableOrCaggName, policy.DataRetentionPeriod)); err != nil {
			return errors.Wrapf(err, "adding retention policy to %s", policy.HypertableOrCaggName)
		}
	}

	return nil
}

func StartEmbeddedPostgres(log *logging.Logger, config Config, stateDir string, postgresLog *bytes.Buffer) (*embeddedpostgres.EmbeddedPostgres, string, error) {
	embeddedPostgresRuntimePath := paths.JoinStatePath(paths.StatePath(stateDir), "sqlstore")
	embeddedPostgresDataPath := paths.JoinStatePath(paths.StatePath(stateDir), "sqlstore", "node-data")

	embeddedPostgres := createEmbeddedPostgres(&embeddedPostgresRuntimePath, &embeddedPostgresDataPath,
		postgresLog, config.ConnectionConfig)

	if err := embeddedPostgres.Start(); err != nil {
		log.Errorf("postgres log: \n%s", postgresLog.String())
		return nil, "", fmt.Errorf("use embedded database was true, but failed to start: %w", err)
	}

	return embeddedPostgres, embeddedPostgresRuntimePath.String(), nil
}

func createEmbeddedPostgres(runtimePath *paths.StatePath, dataPath *paths.StatePath, postgresLog *bytes.Buffer, conf ConnectionConfig) *embeddedpostgres.EmbeddedPostgres {
	dbConfig := embeddedpostgres.DefaultConfig().
		Username(conf.Username).
		Password(conf.Password).
		Database(conf.Database).
		Port(uint32(conf.Port))

	if postgresLog != nil {
		dbConfig = dbConfig.Logger(postgresLog)
	}

	if runtimePath != nil {
		dbConfig = dbConfig.RuntimePath(runtimePath.String()).BinariesPath(runtimePath.String())
	}

	if dataPath != nil {
		dbConfig = dbConfig.DataPath(dataPath.String())
	}

	return embeddedpostgres.NewDatabase(dbConfig)
}
