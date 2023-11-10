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
	metrics.IncrementBatcherAddedEntities(b.tableName)
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
		return nil, fmt.Errorf("failed to copy %q entries into database: %w", b.tableName, err)
	}

	if copyCount != int64(len(b.pending)) {
		return nil, fmt.Errorf("copied %d %s rows into the database, expected to copy %d",
			copyCount,
			b.tableName,
			len(b.pending))
	}

	flushed := b.pending
	b.pending = b.pending[:0]

	metrics.BatcherFlushedEntitiesAdd(b.tableName, len(rows))
	return flushed, nil
}
