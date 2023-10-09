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

package service

import (
	"context"

	"code.vegaprotocol.io/vega/datanode/entities"
)

type snapStore interface {
	Add(ctx context.Context, p entities.CoreSnapshotData) error
	List(ctx context.Context, pagination entities.CursorPagination) ([]entities.CoreSnapshotData, entities.PageInfo, error)
}
type SnapshotData struct {
	snapStore snapStore
}

func NewSnapshotData(snapStore snapStore) *SnapshotData {
	return &SnapshotData{
		snapStore: snapStore,
	}
}

func (s *SnapshotData) AddSnapshot(ctx context.Context, snap entities.CoreSnapshotData) error {
	return s.snapStore.Add(ctx, snap)
}

func (s *SnapshotData) ListSnapshots(ctx context.Context, pagination entities.CursorPagination) ([]entities.CoreSnapshotData, entities.PageInfo, error) {
	return s.snapStore.List(ctx, pagination)
}
