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
	"sync"
	"sync/atomic"

	"code.vegaprotocol.io/vega/logging"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
)

type ConnectionSource struct {
	pool   *pgxpool.Pool
	log    *logging.Logger
	isTest bool
}

type wrappedTx struct {
	parent    *wrappedTx
	mu        sync.Mutex
	postHooks []func()
	id        int64
	idgen     *atomic.Int64
	tx        pgx.Tx
	subTx     map[int64]*wrappedTx
}

type (
	txKey   struct{}
	connKey struct{}
)

func NewTransactionalConnectionSource(ctx context.Context, log *logging.Logger, connConfig ConnectionConfig) (*ConnectionSource, error) {
	pool, err := CreateConnectionPool(ctx, connConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}
	return &ConnectionSource{
		pool: pool,
		log:  log.Named("connection-source"),
	}, nil
}

func (c *ConnectionSource) ToggleTest() {
	c.isTest = true
}

func (c *ConnectionSource) WithConnection(ctx context.Context) (context.Context, error) {
	poolConn, err := c.pool.Acquire(ctx)
	if err != nil {
		return context.Background(), errors.Errorf("failed to acquire connection:%s", err)
	}
	return context.WithValue(ctx, connKey{}, &wrappedConn{
		Conn: poolConn.Hijack(),
	}), nil
}

func (c *ConnectionSource) WithTransaction(ctx context.Context) (context.Context, error) {
	var tx pgx.Tx
	var err error
	nTx := &wrappedTx{
		postHooks: []func(){},
		subTx:     map[int64]*wrappedTx{},
		idgen:     &atomic.Int64{},
	}
	// start id at 0
	nTx.idgen.Store(0)
	if ctxTx, ok := ctx.Value(txKey{}).(*wrappedTx); ok {
		// register sub-transactions
		nTx.id = ctxTx.idgen.Add(1)
		tx, err = ctxTx.tx.Begin(ctx)
		nTx.parent = ctxTx
		if err == nil {
			ctxTx.mu.Lock()
			ctxTx.subTx[nTx.id] = nTx
			ctxTx.mu.Unlock()
		}
	} else if conn, ok := ctx.Value(connKey{}).(*wrappedConn); ok {
		tx, err = conn.Begin(ctx)
	} else {
		tx, err = c.pool.Begin(ctx)
	}
	if err != nil {
		return ctx, errors.Wrapf(err, "failed to start transaction:%s", err)
	}
	nTx.tx = tx
	return context.WithValue(ctx, txKey{}, nTx), nil
}

func (c *ConnectionSource) AfterCommit(ctx context.Context, f func()) {
	// if the context references an ongoing transaction, append the callback to be invoked on commit
	if cTx, ok := ctx.Value(txKey{}).(*wrappedTx); ok {
		cTx.mu.Lock()
		cTx.postHooks = append(cTx.postHooks, f)
		cTx.mu.Unlock()
		return
	}
	// not in transaction, just call immediately.
	f()
}

func (c *ConnectionSource) Rollback(ctx context.Context) error {
	// if we're in a transaction, roll it back starting with the sub-transactions.
	tx, ok := ctx.Value(txKey{}).(*wrappedTx)
	if !ok {
		// no tx ongoing
		return fmt.Errorf("no transaction is associated with the context")
	}
	return tx.Rollback(ctx)
}

func (c *ConnectionSource) Commit(ctx context.Context) error {
	tx, ok := ctx.Value(txKey{}).(*wrappedTx)
	if !ok {
		return fmt.Errorf("no transaction is associated with the context")
	}
	tx.mu.Lock()
	defer tx.mu.Unlock()
	post, err := tx.commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction for context: %s, error: %w", ctx, err)
	}
	// invoke all post-commit hooks once the transaction (and its sub transactions) have been committed
	// make an exception for unit tests, so we don't need to commit DB transactions for hooks on the nested transaction.
	if !c.isTest && tx.parent != nil {
		// this is a nested transaction, don't invoke hooks until the parent is committed
		// instead prepend the hooks and return.
		tx.parent.mu.Lock()
		tx.parent.postHooks = append(post, tx.parent.postHooks...)
		// remove the reference to this transaction from its parent
		delete(tx.parent.subTx, tx.id)
		tx.parent.mu.Unlock()
		return nil
	}
	// this is the main transactions, invoke all hooks now
	for _, f := range post {
		f()
	}
	if tx.parent != nil {
		tx.parent.mu.Lock()
		delete(tx.parent.subTx, tx.id)
		tx.parent.mu.Unlock()
	}
	return nil
}

func (c *ConnectionSource) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	// this is nasty, but required for the API tests currently.
	if c.isTest && c.pool == nil {
		return nil, pgx.ErrNoRows
	}
	if tx, ok := ctx.Value(txKey{}).(*wrappedTx); ok {
		return tx.tx.Query(ctx, sql, args...)
	}
	if conn, ok := ctx.Value(connKey{}).(*wrappedConn); ok {
		return conn.Query(ctx, sql, args...)
	}
	return c.pool.Query(ctx, sql, args...)
}

func (c *ConnectionSource) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	if tx, ok := ctx.Value(txKey{}).(*wrappedTx); ok {
		return tx.tx.QueryRow(ctx, sql, args...)
	}
	if conn, ok := ctx.Value(connKey{}).(*wrappedConn); ok {
		return conn.QueryRow(ctx, sql, args...)
	}
	return c.pool.QueryRow(ctx, sql, args...)
}

func (c *ConnectionSource) QueryFunc(ctx context.Context, sql string, args []interface{}, scans []interface{}, f func(pgx.QueryFuncRow) error) (pgconn.CommandTag, error) {
	if tx, ok := ctx.Value(txKey{}).(*wrappedTx); ok {
		return tx.tx.QueryFunc(ctx, sql, args, scans, f)
	}
	if conn, ok := ctx.Value(connKey{}).(*wrappedConn); ok {
		return conn.QueryFunc(ctx, sql, args, scans, f)
	}
	return c.pool.QueryFunc(ctx, sql, args, scans, f)
}

func (c *ConnectionSource) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	if tx, ok := ctx.Value(txKey{}).(*wrappedTx); ok {
		return tx.tx.SendBatch(ctx, b)
	}
	if conn, ok := ctx.Value(connKey{}).(*wrappedConn); ok {
		return conn.SendBatch(ctx, b)
	}
	return c.pool.SendBatch(ctx, b)
}

func (c *ConnectionSource) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	if tx, ok := ctx.Value(txKey{}).(*wrappedTx); ok {
		return tx.tx.CopyFrom(ctx, tableName, columnNames, rowSrc)
	}
	if conn, ok := ctx.Value(connKey{}).(*wrappedConn); ok {
		return conn.CopyFrom(ctx, tableName, columnNames, rowSrc)
	}
	return c.pool.CopyFrom(ctx, tableName, columnNames, rowSrc)
}

func (c *ConnectionSource) CopyTo(ctx context.Context, w io.Writer, sql string, args ...any) (pgconn.CommandTag, error) {
	// this is nasty, but required for the API tests currently.
	if c.isTest && c.pool == nil {
		return pgconn.CommandTag{}, nil
	}
	var err error
	sql, err = SanitizeSql(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to sanitize sql: %w", err)
	}
	if tx, ok := ctx.Value(txKey{}).(*wrappedTx); ok {
		return tx.tx.Conn().PgConn().CopyTo(ctx, w, sql)
	}
	if conn, ok := ctx.Value(connKey{}).(*wrappedConn); ok {
		return conn.PgConn().CopyTo(ctx, w, sql)
	}
	conn, err := c.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	return conn.Conn().PgConn().CopyTo(ctx, w, sql)
}

func (c *ConnectionSource) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if tx, ok := ctx.Value(txKey{}).(*wrappedTx); ok {
		return tx.tx.Exec(ctx, sql, args...)
	}
	if conn, ok := ctx.Value(connKey{}).(*wrappedConn); ok {
		return conn.Exec(ctx, sql, args...)
	}
	return c.pool.Exec(ctx, sql, args...)
}

type wrappedConn struct {
	*pgx.Conn
}

func (c *ConnectionSource) RefreshMaterializedViews(ctx context.Context) error {
	conn := ctx.Value(connKey{}).(*wrappedConn)
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

func (c *ConnectionSource) Close() {
	c.pool.Close()
}

func (c *ConnectionSource) wrapE(err error) error {
	return wrapE(err)
}

func (t *wrappedTx) commit(ctx context.Context) ([]func(), error) {
	// return callbacks so we only invoke them if no errors occurred
	ret := t.postHooks
	for id, sTx := range t.subTx {
		// acquire the lock, release it as soon as possible
		sTx.mu.Lock()
		subCB, err := sTx.commit(ctx)
		if err != nil {
			sTx.mu.Unlock()
			return nil, err
		}
		sTx.mu.Unlock()
		delete(t.subTx, id)
		// prepend callbacks from sub transactions
		ret = append(subCB, ret...)
	}
	// actually commit this transaction
	if err := t.tx.Commit(ctx); err != nil {
		return nil, err
	}
	return ret, nil
}

func (t *wrappedTx) Rollback(ctx context.Context) error {
	for _, sTx := range t.subTx {
		if err := sTx.Rollback(ctx); err != nil {
			return err
		}
	}
	if err := t.tx.Rollback(ctx); err != nil {
		return fmt.Errorf("failed to rollback transaction for context:%s, error:%w", ctx, err)
	}
	if t.parent != nil {
		t.parent.rmSubTx(t.id)
	}
	return nil
}

func (t *wrappedTx) rmSubTx(id int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	// this is called from Rollback, which is recursive already.
	// no need to recursively remove the sub-tx
	delete(t.subTx, id)
}
