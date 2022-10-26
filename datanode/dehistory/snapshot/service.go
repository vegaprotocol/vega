package snapshot

import (
	"context"
	"fmt"
	"io/fs"
	"os"

	"code.vegaprotocol.io/vega/datanode/dehistory/snapshot/mutex"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/logging"
)

type Service struct {
	log    *logging.Logger
	config Config

	connConfig sqlstore.ConnectionConfig

	createSnapshotLock    mutex.CtxMutex
	snapshotsCopyFromPath string
	snapshotsCopyToPath   string
}

func NewSnapshotService(log *logging.Logger, config Config, connConfig sqlstore.ConnectionConfig,
	snapshotsCopyFromPath string,
	snapshotsCopyToPath string,
) (*Service, error) {
	s := &Service{
		log:                   log,
		config:                config,
		connConfig:            connConfig,
		createSnapshotLock:    mutex.New(),
		snapshotsCopyFromPath: snapshotsCopyFromPath,
		snapshotsCopyToPath:   snapshotsCopyToPath,
	}

	err := os.MkdirAll(s.snapshotsCopyToPath, fs.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create the snapshots dir %s: %w", s.snapshotsCopyToPath, err)
	}

	return s, nil
}

func (b *Service) SnapshotData(ctx context.Context, chainID string, toHeight int64, fromHeight int64) error {
	_, err := b.CreateSnapshot(ctx, chainID, fromHeight, toHeight)
	if err != nil {
		return fmt.Errorf("failed to create snapshot from height %d to %d: %w", fromHeight, toHeight, err)
	}

	return nil
}
