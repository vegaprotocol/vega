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

	createSnapshotLock       mutex.CtxMutex
	absSnapshotsCopyFromPath string
	absSnapshotsCopyToPath   string
	migrateSchemaToVersion   func(version int64) error
}

func NewSnapshotService(log *logging.Logger, config Config, connPool *pgxpool.Pool,
	snapshotsCopyFromPath string,
	snapshotsCopyToPath string,
	migrateDatabaseToVersion func(version int64) error,
) (*Service, error) {
	var err error
	// As these paths are passed to postgres, they need to be absolute as it will likely have
	// a different current working directory than the datanode process. Note; if postgres is
	// containerized, it is up to the container launcher to ensure that the snapshotsCopy{From|To}Path
	// is accessible with the same path inside and outside the container.
	if snapshotsCopyFromPath, err = filepath.Abs(snapshotsCopyFromPath); err != nil {
		return nil, err
	}

	if snapshotsCopyToPath, err = filepath.Abs(snapshotsCopyToPath); err != nil {
		return nil, err
	}

	s := &Service{
		log:                      log,
		config:                   config,
		connPool:                 connPool,
		createSnapshotLock:       mutex.New(),
		absSnapshotsCopyFromPath: snapshotsCopyFromPath,
		absSnapshotsCopyToPath:   snapshotsCopyToPath,
		migrateSchemaToVersion:   migrateDatabaseToVersion,
	}

	err = os.MkdirAll(s.absSnapshotsCopyToPath, fs.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create the snapshots dir %s: %w", s.absSnapshotsCopyToPath, err)
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
