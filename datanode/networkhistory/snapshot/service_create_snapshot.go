package snapshot

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/datanode/networkhistory/segment"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/fs"
	vio "code.vegaprotocol.io/vega/libs/io"
	"github.com/georgysavva/scany/pgxscan"
	"golang.org/x/exp/maps"

	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/logging"
	"github.com/jackc/pgx/v4"
)

var (
	ErrSnapshotExists = errors.New("snapshot exists")
	ErrNoLastSnapshot = errors.New("no last snapshot")
)

func (b *Service) CreateSnapshot(ctx context.Context, chainID string, toHeight int64) (segment.Unpublished, error) {
	return b.createNewSnapshot(ctx, chainID, toHeight, false)
}

func (b *Service) CreateSnapshotAsynchronously(ctx context.Context, chainID string, toHeight int64) (segment.Unpublished, error) {
	return b.createNewSnapshot(ctx, chainID, toHeight, true)
}

func (b *Service) createNewSnapshot(ctx context.Context, chainID string, toHeight int64,
	async bool,
) (segment.Unpublished, error) {
	var err error
	if len(chainID) == 0 {
		return segment.Unpublished{}, fmt.Errorf("chain id is required")
	}

	dbMetaData, err := NewDatabaseMetaData(ctx, b.connPool)
	if err != nil {
		return segment.Unpublished{}, fmt.Errorf("failed to get data dump metadata: %w", err)
	}

	var cleanUp []func()
	ctxWithTimeout, cancelFn := context.WithTimeout(ctx, b.config.WaitForCreationLockTimeout.Duration)
	defer cancelFn()

	// This lock ensures snapshots cannot be created in parallel, during normal run this should never be an issue
	// as the time between snapshots is sufficiently large, however during event replay (and some testing/dev scenarios)
	// the time between snapshots can be sufficiently small to run the risk that snapshotting could overlap without this
	// lock.
	if !b.createSnapshotLock.Lock(ctxWithTimeout) {
		panic("context cancelled whilst waiting for create snapshot lock")
	}

	cleanUp = append(cleanUp, func() { b.createSnapshotLock.Unlock() })

	copyDataTx, err := b.connPool.Begin(ctx)
	if err != nil {
		runAllInReverseOrder(cleanUp)
		return segment.Unpublished{}, fmt.Errorf("failed to begin copy table data transaction: %w", err)
	}
	// Rolling back a committed transaction does nothing
	cleanUp = append(cleanUp, func() { _ = copyDataTx.Rollback(ctx) })

	if _, err = copyDataTx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE"); err != nil {
		runAllInReverseOrder(cleanUp)
		return segment.Unpublished{}, fmt.Errorf("failed to set transaction isolation level to serilizable: %w", err)
	}

	nextSpan, err := getNextSnapshotSpan(ctx, toHeight, copyDataTx)
	if err != nil {
		runAllInReverseOrder(cleanUp)
		if errors.Is(err, ErrSnapshotExists) {
			return segment.Unpublished{}, ErrSnapshotExists
		}
		return segment.Unpublished{}, fmt.Errorf("failed to get next snapshot span:%w", err)
	}

	s := segment.Unpublished{
		Base: segment.Base{
			HeightFrom:      nextSpan.FromHeight,
			HeightTo:        nextSpan.ToHeight,
			DatabaseVersion: dbMetaData.DatabaseVersion,
			ChainID:         chainID,
		},
		Directory: b.copyToPath,
	}

	b.log.Infof("creating snapshot for %+v", s)

	if _, err = os.Create(s.InProgressFilePath()); err != nil {
		runAllInReverseOrder(cleanUp)
		return segment.Unpublished{}, fmt.Errorf("failed to create write lock file:%w", err)
	}
	cleanUp = append(cleanUp, func() { _ = os.Remove(s.InProgressFilePath()) })

	// To ensure reads are isolated from this point forward execute a read on last block
	_, err = sqlstore.GetLastBlockUsingConnection(ctx, copyDataTx)
	if err != nil {
		runAllInReverseOrder(cleanUp)
		return segment.Unpublished{}, fmt.Errorf("failed to get last block using connection: %w", err)
	}

	snapshotData := func() {
		defer func() { runAllInReverseOrder(cleanUp) }()
		err = b.snapshotData(ctx, copyDataTx, dbMetaData, s)
		if err != nil {
			b.log.Panic("failed to snapshot data", logging.Error(err))
		}
	}

	if async {
		go snapshotData()
	} else {
		snapshotData()
	}

	return s, nil
}

func getNextSnapshotSpan(ctx context.Context, toHeight int64, copyDataTx pgx.Tx) (Span, error) {
	lastSnapshotSpan, err := getLastSnapshotSpan(ctx, copyDataTx)

	var nextSpan Span
	if err != nil {
		if errors.Is(err, ErrNoLastSnapshot) {
			oldestHistoryBlock, err := sqlstore.GetOldestHistoryBlockUsingConnection(ctx, copyDataTx)
			if err != nil {
				return Span{}, fmt.Errorf("failed to get oldest history block:%w", err)
			}
			nextSpan = Span{
				FromHeight: oldestHistoryBlock.Height,
				ToHeight:   toHeight,
			}
		} else {
			return nextSpan, fmt.Errorf("failed to get last snapshot span:%w", err)
		}
	} else {
		if toHeight < lastSnapshotSpan.ToHeight {
			return Span{}, fmt.Errorf("toHeight %d is less than last snapshot span %+v", toHeight, lastSnapshotSpan)
		}

		if toHeight == lastSnapshotSpan.ToHeight {
			return Span{}, ErrSnapshotExists
		}

		nextSpan = Span{FromHeight: lastSnapshotSpan.ToHeight + 1, ToHeight: toHeight}
	}

	err = setLastSnapshotSpan(ctx, copyDataTx, nextSpan.FromHeight, nextSpan.ToHeight)
	if err != nil {
		return Span{}, fmt.Errorf("failed to set last snapshot span:%w", err)
	}

	return nextSpan, nil
}

type Span struct {
	FromHeight int64
	ToHeight   int64
}

func setLastSnapshotSpan(ctx context.Context, connection sqlstore.Connection, fromHeight, toHeight int64) error {
	_, err := connection.Exec(ctx, `Insert into last_snapshot_span (from_height, to_height) VALUES($1, $2)
	 on conflict(onerow_check) do update set from_height=EXCLUDED.from_height, to_height=EXCLUDED.to_height`,
		fromHeight, toHeight)
	if err != nil {
		return fmt.Errorf("failed to update last_snapshot_span table:%w", err)
	}
	return nil
}

func getLastSnapshotSpan(ctx context.Context, connection sqlstore.Connection) (*Span, error) {
	ls := &Span{}
	err := pgxscan.Get(ctx, connection, ls,
		`SELECT from_height, to_height
		FROM last_snapshot_span`)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoLastSnapshot
	}

	return ls, err
}

func runAllInReverseOrder(functions []func()) {
	for i := len(functions) - 1; i >= 0; i-- {
		functions[i]()
	}
}

func (b *Service) snapshotData(ctx context.Context, tx pgx.Tx, dbMetaData DatabaseMetadata, seg segment.Unpublished) error {
	defer func() {
		// Calling rollback on a committed transaction has no effect, hence we can rollback in defer to ensure
		// always rolled back if the transaction was not successfully committed
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, "SET TIME ZONE 0"); err != nil {
		return fmt.Errorf("failed to set timezone to UTC:%w", err)
	}

	start := time.Now()
	b.log.Infof("copying all table data....")

	f, err := os.Create(seg.ZipFilePath())
	if err != nil {
		return fmt.Errorf("failed to create file:%w", err)
	}

	countWriter := vio.NewCountWriter(f)
	zipWriter := zip.NewWriter(countWriter)
	defer zipWriter.Close()

	// Write Current State
	currentSQL := currentStateCopySQL(dbMetaData)
	currentRowsCopied, err := copyTableData(ctx, tx, currentSQL, zipWriter)
	if err != nil {
		return fmt.Errorf("failed to copy current state table data:%w", err)
	}
	compressedCurrentStateByteCount := countWriter.Count()

	// Write History
	historySQL := historyCopySQL(dbMetaData, seg)
	historyRowsCopied, err := copyTableData(ctx, tx, historySQL, zipWriter)
	if err != nil {
		return fmt.Errorf("failed to copy history table data:%w", err)
	}
	compressedHistoryByteCount := countWriter.Count() - compressedCurrentStateByteCount

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit snapshot transaction:%w", err)
	}

	metrics.SetLastSnapshotRowcount(float64(currentRowsCopied + historyRowsCopied))
	metrics.SetLastSnapshotCurrentStateBytes(float64(compressedCurrentStateByteCount))
	metrics.SetLastSnapshotHistoryBytes(float64(compressedHistoryByteCount))
	metrics.SetLastSnapshotSeconds(time.Since(start).Seconds())

	b.log.Info("finished creating snapshot for chain", logging.String("chain", seg.ChainID),
		logging.Int64("from height", seg.HeightFrom),
		logging.Int64("to height", seg.HeightTo), logging.Duration("time taken", time.Since(start)),
		logging.Int64("rows copied", currentRowsCopied+historyRowsCopied),
		logging.Int64("compressed current state data size", compressedCurrentStateByteCount),
		logging.Int64("compressed history data size", compressedHistoryByteCount),
	)

	return nil
}

func currentStateCopySQL(dbMetaData DatabaseMetadata) []TableCopySql {
	var copySQL []TableCopySql
	tablesNames := maps.Keys(dbMetaData.TableNameToMetaData)
	sort.Strings(tablesNames)

	for _, tableName := range tablesNames {
		meta := dbMetaData.TableNameToMetaData[tableName]
		if !dbMetaData.TableNameToMetaData[tableName].Hypertable {
			tableCopySQL := fmt.Sprintf(`copy (select * from %s order by %s) TO STDOUT WITH (FORMAT csv, HEADER) `, tableName,
				meta.SortOrder)
			copySQL = append(copySQL, TableCopySql{meta, tableCopySQL})
		}
	}
	return copySQL
}

func historyCopySQL(dbMetaData DatabaseMetadata, segment interface{ GetFromHeight() int64 }) []TableCopySql {
	var copySQL []TableCopySql
	tablesNames := maps.Keys(dbMetaData.TableNameToMetaData)
	sort.Strings(tablesNames)

	for _, tableName := range tablesNames {
		meta := dbMetaData.TableNameToMetaData[tableName]
		if dbMetaData.TableNameToMetaData[tableName].Hypertable {
			partitionColumn := dbMetaData.TableNameToMetaData[tableName].PartitionColumn
			hyperTableCopySQL := fmt.Sprintf(`copy (select * from %s where %s >= (SELECT vega_time from blocks where height = %d) order by %s) to STDOUT (FORMAT csv, HEADER)`,
				tableName,
				partitionColumn,
				segment.GetFromHeight(),
				meta.SortOrder)
			copySQL = append(copySQL, TableCopySql{meta, hyperTableCopySQL})
		}
	}
	return copySQL
}

func copyTableData(ctx context.Context, tx pgx.Tx, copySQL []TableCopySql, zipWriter *zip.Writer) (int64, error) {
	var totalRowsCopied int64
	for _, tableSql := range copySQL {
		dirName := "currentstate"
		if tableSql.metaData.Hypertable {
			dirName = "history"
		}

		fileWriter, err := zipWriter.Create(path.Join(dirName, tableSql.metaData.Name))
		if err != nil {
			return 0, fmt.Errorf("failed to create file in zip:%w", err)
		}

		numRowsCopied, err := executeCopy(ctx, tx, tableSql, fileWriter)
		if err != nil {
			return 0, fmt.Errorf("failed to execute copy: %w", err)
		}
		totalRowsCopied += numRowsCopied
	}

	return totalRowsCopied, nil
}

func executeCopy(ctx context.Context, tx pgx.Tx, tableSql TableCopySql, w io.Writer) (int64, error) {
	defer metrics.StartNetworkHistoryCopy(tableSql.metaData.Name)()

	tag, err := tx.Conn().PgConn().CopyTo(ctx, w, tableSql.copySql)
	if err != nil {
		return 0, fmt.Errorf("failed to execute copy sql %s: %w", tableSql.copySql, err)
	}

	rowsCopied := tag.RowsAffected()
	metrics.NetworkHistoryRowsCopied(tableSql.metaData.Name, rowsCopied)

	return rowsCopied, nil
}

func (b *Service) GetUnpublishedSnapshots() ([]segment.Unpublished, error) {
	files, err := os.ReadDir(b.copyToPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get files in snapshot directory:%w", err)
	}

	segments := []segment.Unpublished{}
	chainID := ""
	for _, file := range files {
		if !file.IsDir() {
			baseSegment, err := segment.NewFromZipFileName(file.Name())
			if err != nil {
				continue
			}
			segment := segment.Unpublished{
				Base:      baseSegment,
				Directory: b.copyToPath,
			}

			if len(chainID) == 0 {
				chainID = segment.ChainID
			}

			if segment.ChainID != chainID {
				return nil, fmt.Errorf("current state snapshots for multiple chain ids exist in snapshots directory %s", b.copyToPath)
			}

			lockFileExists, err := fs.FileExists(segment.InProgressFilePath())
			if err != nil {
				return nil, fmt.Errorf("failed to check for lock file:%w", err)
			}

			if lockFileExists {
				continue
			}
			segments = append(segments, segment)
		}
	}

	return segments, nil
}

type TableCopySql struct {
	metaData TableMetadata
	copySql  string
}
