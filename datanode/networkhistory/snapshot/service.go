package snapshot

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v4/pgxpool"

	"code.vegaprotocol.io/vega/datanode/networkhistory/snapshot/mutex"
	"code.vegaprotocol.io/vega/logging"
)

type Service struct {
	log      *logging.Logger
	config   Config
	connPool *pgxpool.Pool

	createSnapshotLock         mutex.CtxMutex
	copyToPath                 string
	migrateSchemaUpToVersion   func(version int64) error
	migrateSchemaDownToVersion func(version int64) error
}

func NewSnapshotService(log *logging.Logger, config Config, connPool *pgxpool.Pool,
	snapshotsCopyToPath string,
	migrateDatabaseToVersion func(version int64) error,
	migrateSchemaDownToVersion func(version int64) error,
) (*Service, error) {
	var err error

	if snapshotsCopyToPath, err = filepath.Abs(snapshotsCopyToPath); err != nil {
		return nil, err
	}

	s := &Service{
		log:                        log,
		config:                     config,
		connPool:                   connPool,
		createSnapshotLock:         mutex.New(),
		copyToPath:                 snapshotsCopyToPath,
		migrateSchemaUpToVersion:   migrateDatabaseToVersion,
		migrateSchemaDownToVersion: migrateSchemaDownToVersion,
	}

	err = os.MkdirAll(s.copyToPath, fs.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create the snapshots dir %s: %w", s.copyToPath, err)
	}

	return s, nil
}

func (b *Service) SnapshotData(ctx context.Context, chainID string, toHeight int64) error {
	_, err := b.CreateSnapshotAsynchronously(ctx, chainID, toHeight)
	if err != nil {
		return fmt.Errorf("failed to create snapshot for height %d: %w", toHeight, err)
	}

	return nil
}
