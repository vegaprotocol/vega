package sqlstore

import (
	"context"
	"fmt"
	"strconv"

	"code.vegaprotocol.io/data-node/logging"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"

	shopspring "github.com/jackc/pgtype/ext/shopspring-numeric"
)

type Connection interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	QueryFunc(ctx context.Context, sql string, args []interface{}, scans []interface{}, f func(pgx.QueryFuncRow) error) (pgconn.CommandTag, error)
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
}

type ConnectionSource struct {
	conf       ConnectionConfig
	Connection Connection
	pool       *pgxpool.Pool
	log        *logging.Logger
}

type transactionContextKey struct{}
type connectionContextKey struct{}

func NewTransactionalConnectionSource(log *logging.Logger, conf ConnectionConfig) (*ConnectionSource, error) {
	poolConfig, err := conf.GetPoolConfig()
	if err != nil {
		return nil, errors.Wrap(err, "creating connection source")
	}

	setMaxPoolSize(context.Background(), poolConfig)
	registerNumericType(poolConfig)

	pool, err := pgxpool.ConnectConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	connectionSource := &ConnectionSource{
		conf:       conf,
		log:        log.Named("connection_source"),
		pool:       pool,
		Connection: &delegatingConnection{pool: pool},
	}

	return connectionSource, nil
}

func setMaxPoolSize(ctx context.Context, poolConfig *pgxpool.Config) error {
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

	if maxConnections < 6 {
		maxConnections = 6
	}

	poolConfig.MaxConns = int32(maxConnections) - 5
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
	if s.conf.UseTransactions {
		var tx pgx.Tx
		var err error
		if conn, ok := ctx.Value(connectionContextKey{}).(*pgx.Conn); ok {
			tx, err = conn.Begin(ctx)
		} else {
			tx, err = s.pool.Begin(ctx)
		}

		if err != nil {
			return context.Background(), errors.Errorf("failed to start transaction:%s", err)
		}

		return context.WithValue(ctx, transactionContextKey{}, tx), nil
	} else {
		return ctx, nil
	}
}

func (s *ConnectionSource) Commit(ctx context.Context) error {
	if s.conf.UseTransactions {
		if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok {
			if err := tx.Commit(ctx); err != nil {
				return fmt.Errorf("failed to commit transaction for context:%s, error:%w", ctx, err)
			}
		} else {
			return fmt.Errorf("no transaction is associated with the context")
		}
	}

	return nil
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
