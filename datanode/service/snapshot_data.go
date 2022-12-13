package service

import (
	"context"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"
)

type snapStore interface {
	Add(ctx context.Context, p entities.CoreSnapshotData) error
	List(ctx context.Context, pagination entities.CursorPagination) ([]entities.CoreSnapshotData, entities.PageInfo, error)
}
type SnapshotData struct {
	snapStore snapStore
	log       *logging.Logger
}

func NewSnapshotData(snapStore snapStore, log *logging.Logger) *SnapshotData {
	return &SnapshotData{
		snapStore: snapStore,
		log:       log,
	}
}

func (s *SnapshotData) AddSnapshot(ctx context.Context, snap entities.CoreSnapshotData) error {
	return s.snapStore.Add(ctx, snap)
}

func (s *SnapshotData) ListSnapshots(ctx context.Context, pagination entities.CursorPagination) ([]entities.CoreSnapshotData, entities.PageInfo, error) {
	return s.snapStore.List(ctx, pagination)
}
