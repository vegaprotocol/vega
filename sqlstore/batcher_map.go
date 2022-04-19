package sqlstore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type MapBatcher[K entityKey, V entity[K]] struct {
	pending     map[K]V
	tableName   string
	columnNames []string
}

func NewMapBatcher[K entityKey, V entity[K]](tableName string, columnNames []string) MapBatcher[K, V] {
	return MapBatcher[K, V]{
		tableName:   tableName,
		columnNames: columnNames,
		pending:     make(map[K]V),
	}
}

type entityKey interface {
	comparable
}

type entity[K entityKey] interface {
	ToRow() []interface{}
	Key() K
}

func (b *MapBatcher[K, V]) Add(e V) {
	b.pending[e.Key()] = e
}

func (b *MapBatcher[K, V]) Flush(ctx context.Context, pool *pgxpool.Pool) error {
	if len(b.pending) == 0 {
		return nil
	}

	// if b.tableName == "margin_levels" {
	// 	fmt.Println("yay")
	// }
	rows := make([][]interface{}, 0, len(b.pending))
	for _, entity := range b.pending {
		rows = append(rows, entity.ToRow())
	}

	copyCount, err := pool.CopyFrom(
		ctx,
		pgx.Identifier{b.tableName},
		b.columnNames,
		pgx.CopyFromRows(rows),
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

	//b.pending = make(map[K]V)
	for k := range b.pending {
		delete(b.pending, k)
	}

	return nil
}
