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

	"code.vegaprotocol.io/vega/datanode/metrics"

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
	metrics.IncrementBatcherAddedEntities(b.tableName)
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

	metrics.BatcherFlushedEntitiesAdd(b.tableName, len(rows))

	return values, nil
}
