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
	"github.com/jackc/pgx/v4"
)

func DropAllTablesAndViews(log *logging.Logger, config Config) error {
	var err error
	ctx := context.Background()

	poolConfig, err := config.Postgres.ToPgxPoolConfig()
	if err != nil {
		return fmt.Errorf("determining database connection params: %w", err)
	}

	conn, err := pgx.ConnectConfig(ctx, poolConfig.ConnConfig)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}

	views, err := queryViews(ctx, conn)
	if err != nil {
		return fmt.Errorf("fetching view list: %w", err)
	}

	tables, err := queryTables(ctx, conn)
	if err != nil {
		return fmt.Errorf("fetching table list: %w", err)
	}

	if len(views) == 0 && len(tables) == 0 {
		log.Info("database already empty")
	}

	for _, view := range views {
		if err := dropView(ctx, conn, view); err != nil {
			return err
		}
		log.Info("dropped view", logging.String("schema", view[0]), logging.String("name", view[1]))
	}

	for _, table := range tables {
		if err := dropTable(ctx, conn, table); err != nil {
			return err
		}
		log.Info("dropped table", logging.String("schema", table[0]), logging.String("name", table[1]))
	}

	return nil
}

func queryViews(ctx context.Context, conn *pgx.Conn) ([]pgx.Identifier, error) {
	return queryIdentifierList(
		ctx,
		conn,
		`select schemaname, viewname from pg_catalog.pg_views where schemaname=current_schema()`,
	)
}

func queryTables(ctx context.Context, conn *pgx.Conn) ([]pgx.Identifier, error) {
	return queryIdentifierList(
		ctx,
		conn,
		`select schemaname, tablename from pg_catalog.pg_tables where schemaname=current_schema()`,
	)
}

func dropView(ctx context.Context, conn *pgx.Conn, view pgx.Identifier) error {
	statement := fmt.Sprintf("DROP VIEW IF EXISTS %s CASCADE", view.Sanitize())
	_, err := conn.Exec(ctx, statement)
	if err != nil {
		return fmt.Errorf("dropping view %s: %w", view.Sanitize(), err)
	}
	return nil
}

func dropTable(ctx context.Context, conn *pgx.Conn, table pgx.Identifier) error {
	statement := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table.Sanitize())
	_, err := conn.Exec(ctx, statement)
	if err != nil {
		return fmt.Errorf("dropping table %s: %w", table.Sanitize(), err)
	}
	return nil
}

func queryIdentifierList(ctx context.Context, conn *pgx.Conn, query string) ([]pgx.Identifier, error) {
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	var identifiers []pgx.Identifier
	for rows.Next() {
		var schema, relation string
		if err := rows.Scan(&schema, &relation); err != nil {
			return nil, err
		}
		identifiers = append(identifiers, pgx.Identifier{schema, relation})
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return identifiers, nil
}
