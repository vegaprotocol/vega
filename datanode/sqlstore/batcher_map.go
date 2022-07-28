// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlstore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
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

func (b *MapBatcher[K, V]) Flush(ctx context.Context, connection Connection) ([]V, error) {
	if len(b.pending) == 0 {
		return nil, nil
	}

	rows := make([][]interface{}, 0, len(b.pending))
	values := make([]V, 0, len(b.pending))
	for _, entity := range b.pending {
		rows = append(rows, entity.ToRow())
		values = append(values, entity)
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

	for k := range b.pending {
		delete(b.pending, k)
	}

	return values, nil
}
