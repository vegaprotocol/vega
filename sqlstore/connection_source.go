package sqlstore

import (
	"context"
	"fmt"

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

func NewTransactionalConnectionSource(log *logging.Logger, conf ConnectionConfig) (*ConnectionSource, error) {
	poolConfig, err := conf.GetPoolConfig()
	if err != nil {
		return nil, errors.Wrap(err, "creating connection source")
	}

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

func (s *ConnectionSource) WithTransaction(ctx context.Context) (context.Context, error) {
	if s.conf.UseTransactions {
		tx, err := s.pool.Begin(ctx)
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
	} else {
		return t.pool.CopyFrom(ctx, tableName, columnNames, rowSrc)
	}
}

func (t *delegatingConnection) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok {
		return tx.SendBatch(ctx, b)
	} else {
		return t.pool.SendBatch(ctx, b)
	}
}

func (t *delegatingConnection) Exec(ctx context.Context, sql string, arguments ...interface{}) (commandTag pgconn.CommandTag, err error) {
	if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok {
		return tx.Exec(ctx, sql, arguments...)
	} else {
		return t.pool.Exec(ctx, sql, arguments...)
	}
}

func (t *delegatingConnection) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok {
		return tx.Query(ctx, sql, args...)
	} else {
		return t.pool.Query(ctx, sql, args...)
	}
}

func (t *delegatingConnection) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok {
		return tx.QueryRow(ctx, sql, args...)
	} else {
		return t.pool.QueryRow(ctx, sql, args...)
	}
}

func (t *delegatingConnection) QueryFunc(ctx context.Context, sql string, args []interface{}, scans []interface{}, f func(pgx.QueryFuncRow) error) (pgconn.CommandTag, error) {
	if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok {
		return tx.QueryFunc(ctx, sql, args, scans, f)
	} else {
		return t.pool.QueryFunc(ctx, sql, args, scans, f)
	}
}
