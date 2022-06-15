package sqlstore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
)

type ListBatcher[T simpleEntity] struct {
	pending     []T
	tableName   string
	columnNames []string
}

func NewListBatcher[T simpleEntity](tableName string, columnNames []string) ListBatcher[T] {
	return ListBatcher[T]{
		tableName:   tableName,
		columnNames: columnNames,
		pending:     make([]T, 0, 1000),
	}
}

type simpleEntity interface {
	ToRow() []interface{}
}

func (b *ListBatcher[T]) Add(entity T) {
	b.pending = append(b.pending, entity)
}

func (b *ListBatcher[T]) Flush(ctx context.Context, connection Connection) ([]T, error) {
	rows := make([][]interface{}, len(b.pending))
	for i := 0; i < len(b.pending); i++ {
		rows[i] = b.pending[i].ToRow()
	}

	copyCount, err := connection.CopyFrom(
		ctx,
		pgx.Identifier{b.tableName},
		b.columnNames,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to copy %s entries into database:%w", b.tableName, err)
	}

	if copyCount != int64(len(b.pending)) {
		return nil, fmt.Errorf("copied %d %s rows into the database, expected to copy %d",
			copyCount,
			b.tableName,
			len(b.pending))
	}

	flushed := b.pending
	b.pending = b.pending[:0]
	return flushed, nil
}
