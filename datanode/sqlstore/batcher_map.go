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
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

type MapBatcher[K entityKey, V entity[K]] struct {
	pending     *orderedmap.OrderedMap[K, V]
	tableName   string
	columnNames []string
}

func NewMapBatcher[K entityKey, V entity[K]](tableName string, columnNames []string) MapBatcher[K, V] {
	return MapBatcher[K, V]{
		tableName:   tableName,
		columnNames: columnNames,
		pending:     orderedmap.New[K, V](),
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
	key := e.Key()
	_, present := b.pending.Set(key, e)
	if present {
		b.pending.MoveToBack(key)
	}
}

func (b *MapBatcher[K, V]) Flush(ctx context.Context, connection Connection) ([]V, error) {
	nPending := b.pending.Len()
	if nPending == 0 {
		return nil, nil
	}

	rows := make([][]interface{}, 0, nPending)
	values := make([]V, 0, nPending)
	for kv := b.pending.Oldest(); kv != nil; kv = kv.Next() {
		rows = append(rows, kv.Value.ToRow())
		values = append(values, kv.Value)
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

	if copyCount != int64(nPending) {
		return nil, fmt.Errorf("copied %d %s rows into the database, expected to copy %d",
			copyCount,
			b.tableName,
			nPending)
	}

	b.pending = orderedmap.New[K, V]()
	return values, nil
}
