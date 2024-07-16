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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/datanode/networkhistory/segment"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/fs"
	"code.vegaprotocol.io/vega/logging"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"golang.org/x/exp/maps"
	"golang.org/x/sync/errgroup"
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

func (b *Service) createNewSnapshot(
	ctx context.Context,
	chainID string,
	toHeight int64,
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
	// cleanUp = append(cleanUp, func() { _ = os.Remove(s.InProgressFilePath()) })

	// To ensure reads are isolated from this point forward execute a read on last block
	_, err = sqlstore.GetLastBlockUsingConnection(ctx, copyDataTx)
	if err != nil {
		_ = os.Remove(s.InProgressFilePath())
		runAllInReverseOrder(cleanUp)
		return segment.Unpublished{}, fmt.Errorf("failed to get last block using connection: %w", err)
	}

	snapshotData := func() {
		defer func() { runAllInReverseOrder(cleanUp) }()
		err = b.snapshotData(ctx, copyDataTx, dbMetaData, s, b.connPool)
		if err != nil {
			b.log.Panic("failed to snapshot data", logging.Error(err))
		}

		b.fw.AddLockFile(s.InProgressFilePath())
	}

	fmt.Printf("IS ASYNC???? > %v\n", async)

	// if async {
	// go snapshotData()
	// } else {
	snapshotData()
	// }

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

type txSnapshot struct {
	snapshotID string
}

func getExportSnapshot(ctx context.Context, tx pgx.Tx) (*txSnapshot, error) {
	row := tx.QueryRow(ctx, "SELECT pg_export_snapshot();")
	var s string
	err := row.Scan(&s)
	if err != nil {
		return nil, fmt.Errorf("couldn't scan pg_export_snapshot result: %w\n", err)
	}

	return &txSnapshot{s}, nil
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

func (b *Service) snapshotData(ctx context.Context, tx pgx.Tx, dbMetaData DatabaseMetadata, seg segment.Unpublished, pool *pgxpool.Pool) error {
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

	currentStateDir := path.Join(seg.UnpublishedSnapshotDataDirectory(), "currentstate")
	historyStateDir := path.Join(seg.UnpublishedSnapshotDataDirectory(), "history")

	err := os.MkdirAll(currentStateDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create current state directory:%w", err)
	}

	err = os.MkdirAll(historyStateDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create history state directory:%w", err)
	}

	fmt.Printf("\n\nCURRENT STATE\n\n")
	// Write Current State
	currentSQL, lastSpan := currentStateCopySQL(dbMetaData)

	lastSpanCopied, lastSpanBytesCopied, err := copyTablesDataLastSpan(ctx, tx, lastSpan, currentStateDir, b.fw, b.tableSnapshotFileSizesCached)
	if err != nil {
		return fmt.Errorf("failed to copy last span table data:%w", err)
	}

	// create snapshot so all others starts with the same state
	pgTxSnapshot, err := getExportSnapshot(ctx, tx)
	if err != nil {
		panic(err)
	}

	// currentRowsCopied, currentStateBytesCopied, err := copyTablesData(ctx, tx, currentSQL, currentStateDir, b.fw, b.tableSnapshotFileSizesCached)
	currentRowsCopied, currentStateBytesCopied, err := copyTablesDataAsync(ctx, pool, currentSQL, currentStateDir, b.fw, b.tableSnapshotFileSizesCached, pgTxSnapshot)
	if err != nil {
		return fmt.Errorf("failed to copy current state table data:%w", err)
	}

	fmt.Printf("\n\nCURRENT STATE DONE\n\n")

	fmt.Printf("\n\nHISTORY STATE\n\n")
	// Write History
	historySQL := historyCopySQL(dbMetaData, seg)
	// historyRowsCopied, historyBytesCopied, err := copyTablesData(ctx, tx, historySQL, historyStateDir, b.fw, b.tableSnapshotFileSizesCached)
	historyRowsCopied, historyBytesCopied, err := copyTablesDataAsync(ctx, pool, historySQL, historyStateDir, b.fw, b.tableSnapshotFileSizesCached, pgTxSnapshot)
	if err != nil {
		return fmt.Errorf("failed to copy history table data:%w", err)
	}
	fmt.Printf("\n\nHISTORY STATE DONE\n\n")

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit snapshot transaction:%w", err)
	}

	metrics.SetLastSnapshotRowcount(float64(currentRowsCopied + historyRowsCopied + lastSpanCopied))
	metrics.SetLastSnapshotCurrentStateBytes(float64(currentStateBytesCopied + lastSpanBytesCopied))
	metrics.SetLastSnapshotHistoryBytes(float64(historyBytesCopied))
	metrics.SetLastSnapshotSeconds(time.Since(start).Seconds())

	b.log.Info("finished creating snapshot for chain", logging.String("chain", seg.ChainID),
		logging.Int64("from height", seg.HeightFrom),
		logging.Int64("to height", seg.HeightTo), logging.Duration("time taken", time.Since(start)),
		logging.Int64("rows copied", currentRowsCopied+historyRowsCopied),
		logging.Int64("current state data size", currentStateBytesCopied),
		logging.Int64("history data size", historyBytesCopied),
	)

	return nil
}

func currentStateCopySQL(dbMetaData DatabaseMetadata) ([]TableCopySql, TableCopySql) {
	var copySQL []TableCopySql
	var lastSpan TableCopySql
	tablesNames := maps.Keys(dbMetaData.TableNameToMetaData)
	sort.Strings(tablesNames)

	for _, tableName := range tablesNames {
		meta := dbMetaData.TableNameToMetaData[tableName]
		if !dbMetaData.TableNameToMetaData[tableName].Hypertable {
			tableCopySQL := fmt.Sprintf(`copy (select * from %s order by %s) TO STDOUT WITH (FORMAT csv, HEADER) `, tableName,
				meta.SortOrder)

			if tableName == "last_snapshot_span" {
				lastSpan = TableCopySql{meta, tableCopySQL}
			} else {
				copySQL = append(copySQL, TableCopySql{meta, tableCopySQL})
			}
		}
	}
	return copySQL, lastSpan
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

// func copyTablesData(
// 	ctx context.Context,
// 	tx pgx.Tx,
// 	copySQL []TableCopySql,
// 	toDir string,
// 	fw *FileWorker,
// 	lenCache map[string]int,
// ) (int64, int64, error) {
// 	var totalRowsCopied int64
// 	var totalBytesCopied int64

// 	for _, tableSql := range copySQL {
// 		filePath := path.Join(toDir, tableSql.metaData.Name)
// 		// numRowsCopied, bytesCopied, err := writeTableToDataFile(ctx, tx, filePath, tableSql)
// 		numRowsCopied, bytesCopied, err := extractTableData(ctx, tx, filePath, tableSql, fw, lenCache, nil)
// 		if err != nil {
// 			return 0, 0, fmt.Errorf("failed to write table %s to file %s:%w", tableSql.metaData.Name, filePath, err)
// 		}

// 		totalRowsCopied += numRowsCopied
// 		totalBytesCopied += bytesCopied
// 	}

// 	return totalRowsCopied, totalBytesCopied, nil
// }

func copyTablesDataLastSpan(
	ctx context.Context,
	tx pgx.Tx,
	tableSql TableCopySql,
	toDir string,
	fw *FileWorker,
	lenCache map[string]int,
) (int64, int64, error) {

	filePath := path.Join(toDir, tableSql.metaData.Name)
	var mtx sync.Mutex
	return extractTableData2(ctx, tx, filePath, tableSql, fw, lenCache, &mtx)
}

func copyTablesDataAsync(
	ctx context.Context,
	pool *pgxpool.Pool,
	copySQL []TableCopySql,
	toDir string,
	fw *FileWorker,
	lenCache map[string]int,
	pgTxSnapshot *txSnapshot,
) (int64, int64, error) {
	var (
		totalRowsCopied  atomic.Int64
		totalBytesCopied atomic.Int64
		mtx              sync.Mutex
	)

	errg, newCtx := errgroup.WithContext(ctx)

	for _, tSql := range copySQL {
		tableSql := tSql

		errg.Go(func() error {

			filePath := path.Join(toDir, tableSql.metaData.Name)
			// numRowsCopied, bytesCopied, err := writeTableToDataFile(ctx, tx, filePath, tableSql)
			numRowsCopied, bytesCopied, err := extractTableData(newCtx, pool, filePath, tableSql, fw, lenCache, &mtx, pgTxSnapshot)
			if err != nil {
				return fmt.Errorf("failed to write table %s to file %s:%w", tableSql.metaData.Name, filePath, err)
			}

			totalRowsCopied.Add(numRowsCopied)
			totalBytesCopied.Add(bytesCopied)

			return nil
		})
	}

	fmt.Printf("\n\n\n\nWAITING\n\n\n\n")
	if err := errg.Wait(); err != nil {
		return 0, 0, err
	}

	fmt.Printf("ALL DONE BABY\n\n\n\n")

	return totalRowsCopied.Load(), totalBytesCopied.Load(), nil
}

func extractTableData(
	ctx context.Context,
	pool *pgxpool.Pool,
	filePath string,
	tableSql TableCopySql,
	fw *FileWorker,
	lenCache map[string]int,
	cacheMu *sync.Mutex,
	pgTxSnapshot *txSnapshot,
) (int64, int64, error) {

	if cacheMu != nil {
		cacheMu.Lock()
	}
	allocCap := lenCache[tableSql.metaData.Name]
	if cacheMu != nil {
		cacheMu.Unlock()
	}
	fmt.Printf("DEBUGTEMP: %v - initialAllocCap(%v)\n", tableSql.metaData.Name, allocCap)

	if allocCap == 0 {
		allocCap = 1000000 // roughly 1mb, because why not.
	} else {
		// if we already have something cached maybe this is growing
		// a grow of 30% is not unreasonnable?
		// should leave us some room
		allocCap += allocCap / 3
	}

	fmt.Printf("DEBUGTEMP: %v -   finalAllocCap(%v)\n", tableSql.metaData.Name, allocCap)

	b := bytes.NewBuffer(make([]byte, 0, allocCap))

	numRowsCopied, err := executeCopy(ctx, pool, tableSql, b, pgTxSnapshot)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to execute copy: %w", err)
	}

	fmt.Printf("===============>>>> %v - %v - %v\n", filePath, numRowsCopied, b.Len())

	len := int64(b.Len())
	// schedule it
	fw.Add(b, filePath)

	// save the new len for this table
	if cacheMu != nil {
		cacheMu.Lock()
	}
	lenCache[tableSql.metaData.Name] = int(len)
	if cacheMu != nil {
		cacheMu.Unlock()
	}
	fmt.Printf("DEBUGTEMP: %v -       actualLen(%v)\n", tableSql.metaData.Name, len)

	return numRowsCopied, len, nil
}

func executeCopy(ctx context.Context, pool *pgxpool.Pool, tableSql TableCopySql, w io.Writer, pgTxSnapshot *txSnapshot) (int64, error) {
	defer metrics.StartNetworkHistoryCopy(tableSql.metaData.Name)()

	// conn, err := pool.Acquire(ctx)
	// if err != nil {
	// 	fmt.Sprintf("\n\n\n\nBOIIIIIIIIIIIIIIIIIIII couldn't acquire connection: %v\n\n\n\n\n", err)
	// }
	// defer conn.Release()

	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin copy table data transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	if _, err = tx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE"); err != nil {
		return 0, fmt.Errorf("failed to set transaction isolation level to serilizable: %w", err)
	}

	if _, err = tx.Exec(ctx, fmt.Sprintf("SET TRANSACTION SNAPSHOT '%v'", pgTxSnapshot.snapshotID)); err != nil {
		fmt.Printf("COULDN'T IMPORT!!! %v", err)
		return 0, fmt.Errorf("failed to set transaction isolation level to serilizable: %w", err)
	}

	tag, err := tx.Conn().PgConn().CopyTo(ctx, w, tableSql.copySql)
	if err != nil {
		return 0, fmt.Errorf("failed to execute copy sql %s: %w", tableSql.copySql, err)
	}

	rowsCopied := tag.RowsAffected()

	fmt.Printf("SQL: %v / COPIED: %v\n", tableSql.copySql, rowsCopied)
	metrics.NetworkHistoryRowsCopied(tableSql.metaData.Name, rowsCopied)

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	return rowsCopied, nil
}

func extractTableData2(
	ctx context.Context,
	tx pgx.Tx,
	filePath string,
	tableSql TableCopySql,
	fw *FileWorker,
	lenCache map[string]int,
	cacheMu *sync.Mutex,
) (int64, int64, error) {

	if cacheMu != nil {
		cacheMu.Lock()
	}
	allocCap := lenCache[tableSql.metaData.Name]
	if cacheMu != nil {
		cacheMu.Unlock()
	}
	fmt.Printf("DEBUGTEMP: %v - initialAllocCap(%v)\n", tableSql.metaData.Name, allocCap)

	if allocCap == 0 {
		allocCap = 1000000 // roughly 1mb, because why not.
	} else {
		// if we already have something cached maybe this is growing
		// a grow of 30% is not unreasonnable?
		// should leave us some room
		allocCap += allocCap / 3
	}

	fmt.Printf("DEBUGTEMP: %v -   finalAllocCap(%v)\n", tableSql.metaData.Name, allocCap)

	b := bytes.NewBuffer(make([]byte, 0, allocCap))

	numRowsCopied, err := executeCopy2(ctx, tx, tableSql, b)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to execute copy: %w", err)
	}

	len := int64(b.Len())
	// schedule it
	fw.Add(b, filePath)

	// save the new len for this table
	if cacheMu != nil {
		cacheMu.Lock()
	}
	lenCache[tableSql.metaData.Name] = int(len)
	if cacheMu != nil {
		cacheMu.Unlock()
	}
	fmt.Printf("DEBUGTEMP: %v -       actualLen(%v)\n", tableSql.metaData.Name, len)

	return numRowsCopied, len, nil
}

func executeCopy2(ctx context.Context, tx pgx.Tx, tableSql TableCopySql, w io.Writer) (int64, error) {
	defer metrics.StartNetworkHistoryCopy(tableSql.metaData.Name)()

	tag, err := tx.Conn().PgConn().CopyTo(ctx, w, tableSql.copySql)
	if err != nil {
		// fmt.Printf("YOLOFAILURE: %v\n", err)
		return 0, fmt.Errorf("failed to execute copy sql %s: %w", tableSql.copySql, err)
	}

	// fmt.Printf("YOLO: %v\n", string(tag))

	rowsCopied := tag.RowsAffected()
	metrics.NetworkHistoryRowsCopied(tableSql.metaData.Name, rowsCopied)

	// _, err = w.Write(tag)
	// if err != nil {
	// 	return 0, fmt.Errorf("failed to execute copy sql %s: %w", tableSql.copySql, err)
	// }

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
		if file.IsDir() {
			baseSegment, err := segment.NewFromSnapshotDataDirectory(file.Name())
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
