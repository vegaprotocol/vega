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
	"strconv"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgtype"
	shopspring "github.com/jackc/pgtype/ext/shopspring-numeric"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
)

var (
	numSpareConnections = 15 // If possible, the pool size will be (max_connections - numSpareConnections).
	poolSizeLowerBound  = 10 // But it will never be lower than this.
)

type Connection interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	QueryFunc(ctx context.Context, sql string, args []interface{}, scans []interface{}, f func(pgx.QueryFuncRow) error) (pgconn.CommandTag, error)
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
}

func setMaxPoolSize(ctx context.Context, poolConfig *pgxpool.Config, conf ConnectionConfig) error {
	conn, err := pgx.Connect(ctx, poolConfig.ConnString())
	if err != nil {
		return fmt.Errorf("connecting to db: %w", err)
	}
	defer conn.Close(ctx)

	var maxConnectionsStr string
	if err := conn.QueryRow(ctx, "SHOW max_connections;").Scan(&maxConnectionsStr); err != nil {
		return fmt.Errorf("querying max_connections: %w", err)
	}

	maxConnections, err := strconv.Atoi(maxConnectionsStr)
	if err != nil {
		return fmt.Errorf("max_connections was not an integer: %w", err)
	}

	maxConnections = num.MaxV(maxConnections-numSpareConnections, poolSizeLowerBound)
	if conf.MaxConnPoolSize > 0 && maxConnections > conf.MaxConnPoolSize {
		maxConnections = conf.MaxConnPoolSize
	}

	poolConfig.MaxConns = int32(maxConnections)
	return nil
}

func wrapE(err error) error {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return entities.ErrNotFound
	case errors.Is(err, entities.ErrInvalidID):
		return entities.ErrInvalidID
	default:
		return err
	}
}

func registerNumericType(poolConfig *pgxpool.Config) {
	// Cause postgres numeric types to be loaded as shopspring decimals and vice-versa
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		conn.ConnInfo().RegisterDataType(pgtype.DataType{
			Value: &shopspring.Numeric{},
			Name:  "numeric",
			OID:   pgtype.NumericOID,
		})
		return nil
	}
}

func CreateConnectionPool(ctx context.Context, conf ConnectionConfig) (*pgxpool.Pool, error) {
	poolConfig, err := conf.GetPoolConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get pool config: %w", err)
	}

	setMaxPoolSize(ctx, poolConfig, conf)
	registerNumericType(poolConfig)

	poolConfig.MinConns = conf.MinConnPoolSize

	pool, err := pgxpool.ConnectConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	return pool, nil
}
