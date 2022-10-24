package snapshot

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"time"

	"code.vegaprotocol.io/vega/datanode/dehistory/fsutil"

	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/logging"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type CreateSnapshotResult struct {
	CurrentStateSnapshot     CurrentState
	HistorySnapshot          History
	CurrentStateSnapshotPath string
	HistorySnapshotPath      string
	DatabaseVersion          int64
}

func (b *Service) CreateSnapshotAsync(ctx context.Context, chainID string, fromHeight int64, toHeight int64) (CreateSnapshotResult, error) {
	return b.createSnapshot(ctx, chainID, fromHeight, toHeight, true)
}

func (b *Service) CreateSnapshotSynchronously(ctx context.Context, chainID string, fromHeight int64, toHeight int64) (CreateSnapshotResult, error) {
	return b.createSnapshot(ctx, chainID, fromHeight, toHeight, false)
}

func (b *Service) createSnapshot(ctx context.Context, chainID string, fromHeight int64, toHeight int64, async bool) (CreateSnapshotResult, error) {
	var err error
	if len(chainID) == 0 {
		return CreateSnapshotResult{}, fmt.Errorf("chain id is required")
	}

	currentSnapshot := NewCurrentSnapshot(chainID, toHeight)
	historySnapshot := NewHistorySnapshot(chainID, fromHeight, toHeight)
	b.log.Infof("creating snapshot for %+v", historySnapshot)

	dbMetaData, err := NewDatabaseMetaData(ctx, b.connConfig)
	if err != nil {
		return CreateSnapshotResult{}, fmt.Errorf("failed to get data dump metadata: %w", err)
	}

	var cleanUp []func()
	ctxWithTimeout, cancelFn := context.WithTimeout(ctx, b.config.WaitForCreationLockTimeout.Duration)
	defer cancelFn()
	if !b.createSnapshotLock.Lock(ctxWithTimeout) {
		return CreateSnapshotResult{}, fmt.Errorf("context cancelled whilst waiting for create snapshot lock")
	}

	cleanUp = append(cleanUp, func() { b.createSnapshotLock.Unlock() })

	conn, err := pgxpool.Connect(ctx, b.connConfig.GetConnectionString())
	if err != nil {
		runAllInReverseOrder(cleanUp)
		return CreateSnapshotResult{}, fmt.Errorf("unable to connect to database: %w", err)
	}
	cleanUp = append(cleanUp, func() { conn.Close() })

	copyDataTx, err := conn.Begin(ctx)
	if err != nil {
		runAllInReverseOrder(cleanUp)
		return CreateSnapshotResult{}, fmt.Errorf("failed to begin copy table data transaction: %w", err)
	}
	// Rolling back a committed transaction does nothing
	cleanUp = append(cleanUp, func() { copyDataTx.Rollback(ctx) })

	if _, err = copyDataTx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE"); err != nil {
		runAllInReverseOrder(cleanUp)
		return CreateSnapshotResult{}, fmt.Errorf("failed to set transaction isolation level to serilizable:%w", err)
	}

	snapshotInProgressFile := filepath.Join(b.snapshotsCopyToPath, InProgressFileName(chainID, toHeight))
	if _, err = os.Create(snapshotInProgressFile); err != nil {
		runAllInReverseOrder(cleanUp)
		return CreateSnapshotResult{}, fmt.Errorf("failed to create write lock file:%w", err)
	}
	cleanUp = append(cleanUp, func() { os.Remove(snapshotInProgressFile) })

	if async {
		go func() {
			defer func() { runAllInReverseOrder(cleanUp) }()
			err = b.snapshotData(ctx, copyDataTx, dbMetaData, currentSnapshot, historySnapshot)
			if err != nil {
				b.log.Error("failed to snapshot data", logging.Error(err))
				if b.config.PanicOnSnapshotCreationError {
					panic(fmt.Sprintf("failed to snapshot data:%s", err))
				}
			}
		}()
	} else {
		defer func() { runAllInReverseOrder(cleanUp) }()
		err = b.snapshotData(ctx, copyDataTx, dbMetaData, currentSnapshot, historySnapshot)
		if err != nil {
			return CreateSnapshotResult{}, fmt.Errorf("failed to snapshot data:%w", err)
		}
	}

	return CreateSnapshotResult{
		CurrentStateSnapshot:     currentSnapshot,
		HistorySnapshot:          historySnapshot,
		CurrentStateSnapshotPath: filepath.Join(b.snapshotsCopyToPath, currentSnapshot.CompressedFileName()),
		HistorySnapshotPath:      filepath.Join(b.snapshotsCopyToPath, historySnapshot.CompressedFileName()),
		DatabaseVersion:          dbMetaData.DatabaseVersion,
	}, nil
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
	allCopySQL := append(currentSnapshot.GetCopySQL(dbMetaData, b.config.DatabaseSnapshotsCopyToPath),
		historySnapshot.GetCopySQL(dbMetaData, b.config.DatabaseSnapshotsCopyToPath)...)
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
		logging.Int64("height", currentSnapshot.Height), logging.Duration("time taken", time.Since(start)),
		logging.Int64("rows copied", rowsCopied),
		logging.Int64("compressed current state data size", compressedCurrentStateByteCount),
		logging.Int64("compressed history data size", compressedHistoryByteCount),
		logging.String("md5 hash", snapshotHash))

	return nil
}

func GetHistoryMd5Hash(snapshot CreateSnapshotResult) (string, error) {
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
