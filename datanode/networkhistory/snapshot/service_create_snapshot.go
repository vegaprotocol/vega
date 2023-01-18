package snapshot

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/georgysavva/scany/pgxscan"

	"code.vegaprotocol.io/vega/datanode/sqlstore"

	"code.vegaprotocol.io/vega/datanode/networkhistory/fsutil"

	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/logging"
	"github.com/jackc/pgx/v4"
)

type MetaData struct {
	CurrentStateSnapshot     CurrentState
	HistorySnapshot          History
	CurrentStateSnapshotPath string
	HistorySnapshotPath      string
	DatabaseVersion          int64
}

var (
	ErrSnapshotExists = errors.New("Snapshot exists")
	ErrNoLastSnapshot = errors.New("No last snapshot")
)

func (b *Service) CreateSnapshot(ctx context.Context, chainID string, toHeight int64) (MetaData, error) {
	return b.createNewSnapshot(ctx, chainID, toHeight, false)
}

func (b *Service) CreateSnapshotAsynchronously(ctx context.Context, chainID string, toHeight int64) (MetaData, error) {
	return b.createNewSnapshot(ctx, chainID, toHeight, true)
}

func (b *Service) createNewSnapshot(ctx context.Context, chainID string, toHeight int64,
	async bool,
) (MetaData, error) {
	var err error
	if len(chainID) == 0 {
		return MetaData{}, fmt.Errorf("chain id is required")
	}

	dbMetaData, err := NewDatabaseMetaData(ctx, b.connPool)
	if err != nil {
		return MetaData{}, fmt.Errorf("failed to get data dump metadata: %w", err)
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
		return MetaData{}, fmt.Errorf("failed to begin copy table data transaction: %w", err)
	}
	// Rolling back a committed transaction does nothing
	cleanUp = append(cleanUp, func() { _ = copyDataTx.Rollback(ctx) })

	if _, err = copyDataTx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE"); err != nil {
		runAllInReverseOrder(cleanUp)
		return MetaData{}, fmt.Errorf("failed to set transaction isolation level to serilizable: %w", err)
	}

	snapshotInProgressFile := filepath.Join(b.snapshotsCopyToPath, InProgressFileName(chainID, toHeight))
	if _, err = os.Create(snapshotInProgressFile); err != nil {
		runAllInReverseOrder(cleanUp)
		return MetaData{}, fmt.Errorf("failed to create write lock file:%w", err)
	}
	cleanUp = append(cleanUp, func() { _ = os.Remove(snapshotInProgressFile) })

	nextSpan, err := getNextSnapshotSpan(ctx, toHeight, copyDataTx)
	if err != nil {
		runAllInReverseOrder(cleanUp)
		if errors.Is(err, ErrSnapshotExists) {
			return MetaData{}, ErrSnapshotExists
		}
		return MetaData{}, fmt.Errorf("failed to get next snapshot span:%w", err)
	}

	historySnapshot := NewHistorySnapshot(chainID, nextSpan.FromHeight, nextSpan.ToHeight)
	currentSnapshot := NewCurrentSnapshot(chainID, nextSpan.ToHeight)

	b.log.Infof("creating snapshot for %+v", historySnapshot)

	// To ensure reads are isolated from this point forward execute a read on last block
	_, err = sqlstore.GetLastBlockUsingConnection(ctx, copyDataTx)
	if err != nil {
		runAllInReverseOrder(cleanUp)
		return MetaData{}, fmt.Errorf("failed to get last block using connection: %w", err)
	}

	snapshotData := func() {
		defer func() { runAllInReverseOrder(cleanUp) }()
		err = b.snapshotData(ctx, copyDataTx, dbMetaData, currentSnapshot, historySnapshot)
		if err != nil {
			b.log.Panic("failed to snapshot data", logging.Error(err))
		}
	}

	if async {
		go snapshotData()
	} else {
		snapshotData()
	}

	return MetaData{
		CurrentStateSnapshot:     currentSnapshot,
		HistorySnapshot:          historySnapshot,
		CurrentStateSnapshotPath: filepath.Join(b.snapshotsCopyToPath, currentSnapshot.CompressedFileName()),
		HistorySnapshotPath:      filepath.Join(b.snapshotsCopyToPath, historySnapshot.CompressedFileName()),
		DatabaseVersion:          dbMetaData.DatabaseVersion,
	}, nil
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

func (b *Service) snapshotData(ctx context.Context, copyDataTx pgx.Tx, dbMetaData DatabaseMetadata,
	currentSnapshot CurrentState,
	historySnapshot History,
) (err error) {
	uncompressedCurrentDataDir := filepath.Join(b.snapshotsCopyToPath, currentSnapshot.UncompressedDataDir())
	uncompressedHistoryDataDir := filepath.Join(b.snapshotsCopyToPath, historySnapshot.UncompressedDataDir())

	defer func() {
		// Calling rollback on a committed transaction has no effect, hence we can rollback in defer to ensure
		// always rolled back if the transaction was not successfully committed
		_ = copyDataTx.Rollback(ctx)
		_ = os.RemoveAll(uncompressedCurrentDataDir)
		_ = os.RemoveAll(uncompressedHistoryDataDir)
	}()

	if _, err = copyDataTx.Exec(ctx, "SET TIME ZONE 0"); err != nil {
		return fmt.Errorf("failed to set timezone to UTC:%w", err)
	}

	if err = fsutil.MkdirAllIgnoringUMask(uncompressedCurrentDataDir); err != nil {
		return fmt.Errorf("failed to create uncompressed data directory for current snapshot:%w", err)
	}

	if err = fsutil.MkdirAllIgnoringUMask(uncompressedHistoryDataDir); err != nil {
		return fmt.Errorf("failed to create uncompressed data directory for history snapshot:%w", err)
	}

	start := time.Now()
	b.log.Infof("copying all table data....")
	allCopySQL := append(currentSnapshot.GetCopySQL(dbMetaData, b.snapshotsCopyToPath),
		historySnapshot.GetCopySQL(dbMetaData, b.snapshotsCopyToPath)...)
	rowsCopied, err := copyTableData(ctx, copyDataTx, allCopySQL)
	if err != nil {
		return fmt.Errorf("failed to copy table data:%w", err)
	}

	err = copyDataTx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit snapshot transaction:%w", err)
	}

	b.log.Infof("compressing current state snapshot data")

	compressedCurrentStateFile := filepath.Join(b.snapshotsCopyToPath, currentSnapshot.CompressedFileName())
	compressedCurrentStateByteCount, err := fsutil.TarAndCompressDirWithDeterministicHeader(uncompressedCurrentDataDir, compressedCurrentStateFile, dbMetaData.DatabaseVersion)
	if err != nil {
		return fmt.Errorf("failed to compress snapshot:%w", err)
	}
	b.log.Infof("compressed current state snapshot data size: %dB", compressedCurrentStateByteCount)

	b.log.Infof("compressing history data")
	compressedHistoryStateFile := filepath.Join(b.snapshotsCopyToPath, historySnapshot.CompressedFileName())
	compressedHistoryByteCount, err := fsutil.TarAndCompressDirWithDeterministicHeader(uncompressedHistoryDataDir, compressedHistoryStateFile, dbMetaData.DatabaseVersion)
	if err != nil {
		return fmt.Errorf("failed to compress history snapshot:%w", err)
	}
	b.log.Infof("compressed history snapshot data size: %dB", compressedHistoryByteCount)

	snapshotHash, err := GetSnapshotMd5Hash(compressedCurrentStateFile, compressedHistoryStateFile)
	if err != nil {
		b.log.Errorf("failed to get md5 hash of snapshot:%w", err)
	}

	metrics.SetLastSnapshotRowcount(float64(rowsCopied))
	metrics.SetLastSnapshotCurrentStateBytes(float64(compressedCurrentStateByteCount))
	metrics.SetLastSnapshotHistoryBytes(float64(compressedHistoryByteCount))
	metrics.SetLastSnapshotSeconds(time.Since(start).Seconds())

	b.log.Info("finished creating snapshot for chain", logging.String("chain", currentSnapshot.ChainID),
		logging.Int64("from height", historySnapshot.HeightFrom),
		logging.Int64("to height", currentSnapshot.Height), logging.Duration("time taken", time.Since(start)),
		logging.Int64("rows copied", rowsCopied),
		logging.Int64("compressed current state data size", compressedCurrentStateByteCount),
		logging.Int64("compressed history data size", compressedHistoryByteCount),
		logging.String("md5 hash", snapshotHash))

	return nil
}

func GetHistoryMd5Hash(snapshot MetaData) (string, error) {
	snapshotHash := md5.New()

	err := hashFile(snapshotHash, snapshot.HistorySnapshotPath)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(snapshotHash.Sum(nil)), nil
}

func GetSnapshotMd5Hash(currentStateSnapshotFile string, historySnapshotFile string) (string, error) {
	snapshotHash := md5.New()
	err := hashFile(snapshotHash, currentStateSnapshotFile)
	if err != nil {
		return "", err
	}

	err = hashFile(snapshotHash, historySnapshotFile)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(snapshotHash.Sum(nil)), nil
}

func hashFile(hash hash.Hash, filepath string) (err error) {
	filePath := filepath
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	_, err = io.Copy(hash, file)
	if err != nil {
		return err
	}

	return nil
}

func InProgressFileName(chainID string, height int64) string {
	return fmt.Sprintf("%s-%d.snapshotinprogress", chainID, height)
}

func copyTableData(ctx context.Context, tx pgx.Tx, copySQL []string) (int64, error) {
	var numRowsCopied int64
	for _, sql := range copySQL {
		tag, err := tx.Exec(ctx, sql)
		numRowsCopied += tag.RowsAffected()

		if err != nil {
			return 0, fmt.Errorf("failed to execute copy %s: %w", sql, err)
		}
	}

	return numRowsCopied, nil
}
