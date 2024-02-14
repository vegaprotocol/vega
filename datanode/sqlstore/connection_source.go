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
	"io"
	"strconv"
	"sync"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

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

type copyingConnection interface {
	Connection
	CopyTo(ctx context.Context, w io.Writer, sql string, args ...any) (pgconn.CommandTag, error)
}

type ConnectionSource struct {
	Connection      copyingConnection
	pool            *pgxpool.Pool
	log             *logging.Logger
	postCommitHooks []func()
	mu              sync.Mutex
}

type (
	transactionContextKey struct{}
	connectionContextKey  struct{}
)

func NewTransactionalConnectionSource(log *logging.Logger, connConfig ConnectionConfig) (*ConnectionSource, error) {
	pool, err := CreateConnectionPool(connConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	connectionSource := &ConnectionSource{
		log:        log.Named("connection-source"),
		pool:       pool,
		Connection: &delegatingConnection{pool: pool},
	}

	return connectionSource, nil
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

func (s *ConnectionSource) WithConnection(ctx context.Context) (context.Context, error) {
	poolConn, err := s.pool.Acquire(ctx)
	conn := poolConn.Hijack()
	if err != nil {
		return context.Background(), errors.Errorf("failed to acquire connection:%s", err)
	}

	return context.WithValue(ctx, connectionContextKey{}, conn), nil
}

func (s *ConnectionSource) WithTransaction(ctx context.Context) (context.Context, error) {
	var tx pgx.Tx
	var err error
	if outerTx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok {
		tx, err = outerTx.Begin(ctx)
	} else if conn, ok := ctx.Value(connectionContextKey{}).(*pgx.Conn); ok {
		tx, err = conn.Begin(ctx)
	} else {
		tx, err = s.pool.Begin(ctx)
	}

	if err != nil {
		return ctx, errors.Errorf("failed to start transaction:%s", err)
	}

	return context.WithValue(ctx, transactionContextKey{}, tx), nil
}

func (s *ConnectionSource) AfterCommit(ctx context.Context, f func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If we're in a transaction, defer calling f() until Commit() is called

	if _, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok {
		s.postCommitHooks = append(s.postCommitHooks, f)
		return
	}

	// If we're not in a transaction, call f() immediately
	f()
}

func (s *ConnectionSource) Commit(ctx context.Context) error {
	if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok {
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit transaction for context:%s, error:%w", ctx, err)
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		for _, f := range s.postCommitHooks {
			f()
		}
		s.postCommitHooks = s.postCommitHooks[:0]
	} else {
		return fmt.Errorf("no transaction is associated with the context")
	}

	return nil
}

func (s *ConnectionSource) Rollback(ctx context.Context) error {
	if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok {
		if err := tx.Rollback(ctx); err != nil {
			return fmt.Errorf("failed to rollback transaction for context:%s, error:%w", ctx, err)
		}
	} else {
		return fmt.Errorf("no transaction is associated with the context")
	}

	return nil
}

func (s *ConnectionSource) Close() {
	s.pool.Close()
}

func (s *ConnectionSource) RefreshMaterializedViews(ctx context.Context) error {
	conn := ctx.Value(connectionContextKey{}).(*pgx.Conn)
	materializedViewsToRefresh := []struct {
		name         string
		concurrently bool
	}{
		{"game_stats", false},
		{"game_stats_current", false},
	}

	for _, view := range materializedViewsToRefresh {
		sql := "REFRESH MATERIALIZED VIEW "
		if view.concurrently {
			sql += "CONCURRENTLY "
		}
		sql += view.name

		_, err := conn.Exec(ctx, sql)
		if err != nil {
			return fmt.Errorf("failed to refresh materialized view %s: %w", view.name, err)
		}
	}
	return nil
}

func (s *ConnectionSource) wrapE(err error) error {
	return wrapE(err)
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

type delegatingConnection struct {
	pool *pgxpool.Pool
}

func (t *delegatingConnection) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok {
		return tx.CopyFrom(ctx, tableName, columnNames, rowSrc)
	}
	if conn, ok := ctx.Value(connectionContextKey{}).(*pgx.Conn); ok {
		return conn.CopyFrom(ctx, tableName, columnNames, rowSrc)
	}
	return t.pool.CopyFrom(ctx, tableName, columnNames, rowSrc)
}

func (t *delegatingConnection) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok {
		return tx.SendBatch(ctx, b)
	}
	if conn, ok := ctx.Value(connectionContextKey{}).(*pgx.Conn); ok {
		return conn.SendBatch(ctx, b)
	}
	return t.pool.SendBatch(ctx, b)
}

func (t *delegatingConnection) Exec(ctx context.Context, sql string, arguments ...interface{}) (commandTag pgconn.CommandTag, err error) {
	if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok {
		return tx.Exec(ctx, sql, arguments...)
	}
	if conn, ok := ctx.Value(connectionContextKey{}).(*pgx.Conn); ok {
		return conn.Exec(ctx, sql, arguments...)
	}
	return t.pool.Exec(ctx, sql, arguments...)
}

func (t *delegatingConnection) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok {
		return tx.Query(ctx, sql, args...)
	}
	if conn, ok := ctx.Value(connectionContextKey{}).(*pgx.Conn); ok {
		return conn.Query(ctx, sql, args...)
	}
	return t.pool.Query(ctx, sql, args...)
}

func (t *delegatingConnection) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok {
		return tx.QueryRow(ctx, sql, args...)
	}
	if conn, ok := ctx.Value(connectionContextKey{}).(*pgx.Conn); ok {
		return conn.QueryRow(ctx, sql, args...)
	}
	return t.pool.QueryRow(ctx, sql, args...)
}

func (t *delegatingConnection) QueryFunc(ctx context.Context, sql string, args []interface{}, scans []interface{}, f func(pgx.QueryFuncRow) error) (pgconn.CommandTag, error) {
	if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok {
		return tx.QueryFunc(ctx, sql, args, scans, f)
	}
	if conn, ok := ctx.Value(connectionContextKey{}).(*pgx.Conn); ok {
		return conn.QueryFunc(ctx, sql, args, scans, f)
	}
	return t.pool.QueryFunc(ctx, sql, args, scans, f)
}

func (t *delegatingConnection) CopyTo(ctx context.Context, w io.Writer, sql string, args ...any) (pgconn.CommandTag, error) {
	var err error
	sql, err = SanitizeSql(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to sanitize sql: %w", err)
	}
	if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok {
		return tx.Conn().PgConn().CopyTo(ctx, w, sql)
	}
	if conn, ok := ctx.Value(connectionContextKey{}).(*pgx.Conn); ok {
		return conn.PgConn().CopyTo(ctx, w, sql)
	}
	conn, err := t.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	return conn.Conn().PgConn().CopyTo(ctx, w, sql)
}

func CreateConnectionPool(conf ConnectionConfig) (*pgxpool.Pool, error) {
	poolConfig, err := conf.GetPoolConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get pool config: %w", err)
	}

	setMaxPoolSize(context.Background(), poolConfig, conf)
	registerNumericType(poolConfig)

	poolConfig.MinConns = conf.MinConnPoolSize

	pool, err := pgxpool.ConnectConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	return pool, nil
}
