package sqlstore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type ListBatcher struct {
	pending     [][]interface{}
	tableName   string
	columnNames []string
}

func NewListBatcher(tableName string, columnNames []string) ListBatcher {
	return ListBatcher{
		tableName:   tableName,
		columnNames: columnNames,
		pending:     make([][]interface{}, 0, 1000),
	}
}

type simpleEntity interface {
	ToRow() []interface{}
}

func (b *ListBatcher) Add(entity simpleEntity) {
	row := entity.ToRow()
	b.pending = append(b.pending, row)
}

func (b *ListBatcher) Flush(ctx context.Context, pool *pgxpool.Pool) error {
	copyCount, err := pool.CopyFrom(
		ctx,
		pgx.Identifier{b.tableName},
		b.columnNames,
		pgx.CopyFromRows(b.pending),
	)
	if err != nil {
		return fmt.Errorf("failed to copy %s entries into database:%w", b.tableName, err)
	}

	if copyCount != int64(len(b.pending)) {
		return fmt.Errorf("copied %d %s rows into the database, expected to copy %d",
			copyCount,
			b.tableName,
			len(b.pending))
	}

	b.pending = b.pending[:0]
	return nil
}
