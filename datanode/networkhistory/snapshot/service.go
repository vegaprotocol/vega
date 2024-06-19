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

package snapshot

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"code.vegaprotocol.io/vega/datanode/networkhistory/segment"
	"code.vegaprotocol.io/vega/datanode/networkhistory/snapshot/mutex"
	"code.vegaprotocol.io/vega/logging"

	"github.com/jackc/pgx/v4/pgxpool"
)

type HistoryStore interface {
	StagedSegment(ctx context.Context, s segment.Full) (segment.Staged, error)
}

type Service struct {
	log      *logging.Logger
	config   Config
	connPool *pgxpool.Pool

	historyStore HistoryStore

	fw *FileWorker

	createSnapshotLock         mutex.CtxMutex
	copyToPath                 string
	migrateSchemaUpToVersion   func(version int64) error
	migrateSchemaDownToVersion func(version int64) error
}

func NewSnapshotService(log *logging.Logger, config Config, connPool *pgxpool.Pool,
	historyStore HistoryStore,
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
		historyStore:               historyStore,
		fw:                         NewFileWorker(),
	}

	err = os.MkdirAll(s.copyToPath, fs.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create the snapshots dir %s: %w", s.copyToPath, err)
	}

	return s, nil
}

func (b *Service) Flush() {
	for !b.fw.Empty() {
		if err := b.fw.Consume(); err != nil {
			b.log.Error("failed to write all files to disk", logging.Error(err))
		}
	}
}

func (b *Service) SnapshotData(ctx context.Context, chainID string, toHeight int64) error {
	_, err := b.CreateSnapshotAsynchronously(ctx, chainID, toHeight)
	if err != nil {
		return fmt.Errorf("failed to create snapshot for height %d: %w", toHeight, err)
	}

	return nil
}
