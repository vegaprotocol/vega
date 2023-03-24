package sqlstore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/pressly/goose/v3"
)

type PgxTxToGooseConnectionAdapter struct {
	Conn pgx.Tx
}

func (p PgxTxToGooseConnectionAdapter) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	t, err := p.Conn.Exec(ctx, query, args...)
	return PgxToSqlResult{ct: t}, err
}

func (p PgxTxToGooseConnectionAdapter) Exec(query string, args ...any) (sql.Result, error) {
	return p.ExecContext(context.Background(), query, args...)
}

func (p PgxTxToGooseConnectionAdapter) Query(query string, args ...any) (goose.Rows, error) {
	return p.QueryContext(context.Background(), query, args...)
}

func (p PgxTxToGooseConnectionAdapter) QueryContext(ctx context.Context, query string, args ...any) (goose.Rows, error) {
	rows, err := p.Conn.Query(ctx, query, args...)
	return PgxToGooseRows{rows: rows}, err
}

func (p PgxTxToGooseConnectionAdapter) QueryRow(query string, args ...any) goose.Row {
	return p.QueryRowContext(context.Background(), query, args...)
}

func (p PgxTxToGooseConnectionAdapter) QueryRowContext(ctx context.Context, query string, args ...any) goose.Row {
	return p.Conn.QueryRow(ctx, query, args...)
}

func (p PgxTxToGooseConnectionAdapter) Close() error {
	return errors.New("closing a transaction is not supported")
}

func (p PgxTxToGooseConnectionAdapter) Begin() (goose.Tx, error) {
	tx, err := p.Conn.Begin(context.Background())
	return PgxToGooseTx{tx}, err
}

type PgxToGooseTx struct {
	tx pgx.Tx
}

func (p PgxToGooseTx) Commit() error {
	return p.tx.Commit(context.Background())
}

func (p PgxToGooseTx) Rollback() error {
	return p.tx.Rollback(context.Background())
}

func (p PgxToGooseTx) Exec(query string, args ...interface{}) (sql.Result, error) {
	t, err := p.tx.Exec(context.Background(), query, args...)
	return PgxToSqlResult{ct: t}, err
}

type PgxToSqlResult struct {
	ct pgconn.CommandTag
}

func (p PgxToSqlResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (p PgxToSqlResult) RowsAffected() (int64, error) {
	return p.ct.RowsAffected(), nil
}

type PgxToGooseRows struct {
	rows pgx.Rows
}

func (p PgxToGooseRows) Next() bool {
	return p.rows.Next()
}

func (p PgxToGooseRows) Err() error {
	return p.rows.Err()
}

func (p PgxToGooseRows) Scan(dest ...any) error {
	return p.rows.Scan(dest...)
}

func (p PgxToGooseRows) Close() error {
	p.rows.Close()
	return nil
}
