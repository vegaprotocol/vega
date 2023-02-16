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
